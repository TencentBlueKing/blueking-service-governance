/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package dashboard

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Service 提供应用仪表盘绑定管理能力。
type Service struct {
	dashboards *Store
	newClient  func(operator string) (bkmapi.MonitorClient, error)
}

// NewService 创建应用仪表盘绑定服务。
func NewService(db *mongo.Database) *Service {
	return &Service{dashboards: store.Of(db, Meta), newClient: bkmapi.NewMonitorClient}
}

// List 获取应用绑定的仪表盘列表
func (s *Service) List(
	ctx context.Context,
	ws *workspace.Workspace,
	appID, operator string,
) ([]*bkmapi.DashboardItem, error) {
	records, err := s.dashboards.List(ctx, ByApp(appID), store.Desc("createdAt"))
	if err != nil {
		return nil, errors.Wrapf(err, "list app dashboard bindings, appID=%s", appID)
	}
	if len(records) == 0 {
		return []*bkmapi.DashboardItem{}, nil
	}

	tree, err := s.fetchDirectoryTree(ctx, ws, operator)
	if err != nil {
		return nil, err
	}
	return assembleDashboards(records, dashboardInfoMap(tree)), nil
}

// Create 创建应用仪表盘绑定，创建前会校验 uid 在 bkmonitor 侧真实存在，并以 bkm 侧标题为准。
func (s *Service) Create(
	ctx context.Context,
	ws *workspace.Workspace,
	appID, uid, operator string,
) error {
	detail, err := s.ensureDashboardExists(ctx, ws, operator, uid)
	if err != nil {
		return err
	}
	if err := s.dashboards.Create(ctx, &AppDashboard{
		AppID:   appID,
		UID:     uid,
		Title:   detail.Title,
		Creator: operator,
	}); err != nil {
		return errors.Wrapf(err, "create app dashboard binding, appID=%s, uid=%s", appID, uid)
	}
	return nil
}

// Update 更新应用仪表盘绑定，变更 uid 时会校验新 uid 在 bkmonitor 侧真实存在。
func (s *Service) Update(
	ctx context.Context,
	ws *workspace.Workspace,
	appID, uid, operator string,
	updateData *AppDashboardUpdateData,
) error {
	if updateData.UID != nil && *updateData.UID != uid {
		detail, err := s.ensureDashboardExists(ctx, ws, operator, *updateData.UID)
		if err != nil {
			return err
		}
		// 变更 uid 时同步刷新标题，避免与新仪表盘不一致
		updateData.Title = &detail.Title
	}
	update, isEmpty := updateData.ToBSON()
	if isEmpty {
		return nil
	}
	update["updater"] = operator
	if err := s.dashboards.Update(ctx, update, ByAppAndUID(appID, uid)); err != nil {
		return errors.Wrapf(err, "update app dashboard binding, appID=%s, uid=%s", appID, uid)
	}
	return nil
}

// Delete 删除应用仪表盘绑定。
func (s *Service) Delete(ctx context.Context, appID, uid string) error {
	if err := s.dashboards.Delete(ctx, ByAppAndUID(appID, uid)); err != nil {
		return errors.Wrapf(err, "delete app dashboard binding, appID=%s, uid=%s", appID, uid)
	}
	return nil
}

// fetchDirectoryTree 解析 bkmonitor 项目并拉取仪表盘目录树。
func (s *Service) fetchDirectoryTree(
	ctx context.Context,
	ws *workspace.Workspace,
	operator string,
) ([]*bkmapi.DashboardDirectoryNode, error) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	tree, err := client.GetDashboardDirectoryTree(ctx, bkMonitorProjectID)
	if err != nil {
		return nil, errors.Wrapf(err, "get dashboard directory tree, bk_biz_id=%d", bkMonitorProjectID)
	}
	return tree, nil
}

// ensureDashboardExists 校验仪表盘 uid 在 bkmonitor 侧真实存在，并返回其详情。
func (s *Service) ensureDashboardExists(
	ctx context.Context,
	ws *workspace.Workspace,
	operator, uid string,
) (*bkmapi.DashboardDetail, error) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	detail, err := client.GetDashboardDetail(ctx, bkMonitorProjectID, uid)
	if err != nil {
		return nil, errors.Wrapf(err, "get dashboard detail, bk_biz_id=%d, uid=%s", bkMonitorProjectID, uid)
	}
	if detail == nil {
		return nil, errors.Wrapf(ErrDashboardNotExist, "uid=%s", uid)
	}
	return detail, nil
}

// dashboardInfoMap 转换
func dashboardInfoMap(tree []*bkmapi.DashboardDirectoryNode) map[string]bkmapi.DashboardItem {
	result := make(map[string]bkmapi.DashboardItem)
	lo.ForEach(tree, func(node *bkmapi.DashboardDirectoryNode, _ int) {
		if node == nil {
			return
		}
		lo.ForEach(node.Dashboards, func(item bkmapi.DashboardItem, _ int) {
			result[item.UID] = item
		})
	})
	return result
}

// assembleDashboards 将绑定记录与 bkm 仪表盘信息组装为结果列表。
func assembleDashboards(
	records []AppDashboard,
	infoMap map[string]bkmapi.DashboardItem,
) []*bkmapi.DashboardItem {
	return lo.Map(records, func(r AppDashboard, _ int) *bkmapi.DashboardItem {
		item := &bkmapi.DashboardItem{UID: r.UID, Title: r.Title}
		if info, ok := infoMap[r.UID]; ok {
			item.URL = info.URL
		}
		return item
	})
}
