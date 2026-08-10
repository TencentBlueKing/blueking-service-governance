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

// Package usergroup 提供蓝鲸监控告警组管理能力
package usergroup

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// saveUserGroupDefaultTimezone 保存告警组时的默认时区。
const saveUserGroupDefaultTimezone = "Asia/Shanghai"

// ErrUserGroupNotInWorkspace 表示告警组不属于当前 workspace 对应的监控空间。
var ErrUserGroupNotInWorkspace = errors.New("user group does not belong to workspace")

// Service 提供告警组管理能力，不影响既有 APM 告警组同步链路。
type Service struct {
	newClient func(operator string) (bkmapi.MonitorClient, error)
}

// SaveParams 保存告警组参数。
type SaveParams struct {
	ID           int64
	Name         string                 `json:"name" binding:"required" validate:"required"`
	Channels     []string               `json:"channels" binding:"required,min=1" validate:"min=1"`
	Desc         string                 `json:"desc"`
	AlertNotice  []bkmapi.AlertNotice   `json:"alertNotice" binding:"required,min=1" validate:"min=1"`
	ActionNotice []bkmapi.ActionNotice  `json:"actionNotice" binding:"required,min=1" validate:"min=1"`
	Users        []bkmapi.UserGroupUser `json:"users" binding:"required,min=1" validate:"min=1"`
	Operator     string                 `validate:"required"`
}

// New 创建告警组管理服务。
func New() *Service {
	return &Service{newClient: bkmapi.NewMonitorClient}
}

// List 列出 workspace 对应监控空间下的全部告警组。
func (s *Service) List(ctx context.Context, ws *workspace.Workspace, operator string) ([]*bkmapi.UserGroup, error) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	groups, err := client.SearchUserGroups(ctx, &bkmapi.SearchUserGroupsReq{
		BkBizIDs: []int64{bkMonitorProjectID},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "search user groups, bk_biz_id=%d", bkMonitorProjectID)
	}
	return groups, nil
}

// FindByName 按名称查询 workspace 对应监控空间下的告警组。
func (s *Service) FindByName(
	ctx context.Context,
	ws *workspace.Workspace,
	name, operator string,
) (*bkmapi.UserGroup, error) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	groups, err := client.SearchUserGroups(ctx, &bkmapi.SearchUserGroupsReq{
		BkBizIDs: []int64{bkMonitorProjectID},
		Name:     name,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "search user groups by name, bk_biz_id=%d, name=%s", bkMonitorProjectID, name)
	}
	for _, group := range groups {
		if group != nil && group.Name == name {
			return group, nil
		}
	}
	return nil, nil
}

// Get 获取告警组详情，并校验其属于 workspace 对应监控空间。
func (s *Service) Get(
	ctx context.Context,
	ws *workspace.Workspace,
	groupID int64,
	operator string,
) (*bkmapi.UserGroupDetail, error) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return s.getWithClient(ctx, client, groupID, bkMonitorProjectID)
}

func (s *Service) getWithClient(
	ctx context.Context,
	client bkmapi.MonitorClient,
	groupID, bkMonitorProjectID int64,
) (*bkmapi.UserGroupDetail, error) {
	detail, err := client.SearchUserGroupDetail(ctx, &bkmapi.SearchUserGroupDetailReq{ID: groupID})
	if err != nil {
		return nil, errors.Wrapf(err, "search user group detail, id=%d", groupID)
	}
	if detail == nil || detail.BkBizID != bkMonitorProjectID {
		return nil, errors.Wrapf(
			ErrUserGroupNotInWorkspace,
			"group_id=%d, expected_bk_biz_id=%d",
			groupID,
			bkMonitorProjectID,
		)
	}
	return detail, nil
}

// Save 创建或更新告警组。更新时会先校验告警组归属。
func (s *Service) Save(
	ctx context.Context,
	ws *workspace.Workspace,
	params *SaveParams,
) (*bkmapi.UserGroupDetail, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(params.Operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}

	var requestID *int64
	if params.ID > 0 {
		if _, err = s.getWithClient(ctx, client, params.ID, bkMonitorProjectID); err != nil {
			return nil, errors.Wrap(err, "validate user group ownership before save")
		}
		requestID = &params.ID
	}

	detail, err := client.SaveUserGroup(ctx, buildSaveUserGroupReq(bkMonitorProjectID, requestID, params))
	if err != nil {
		return nil, errors.Wrapf(err, "save user group, bk_biz_id=%d, id=%d", bkMonitorProjectID, params.ID)
	}
	return detail, nil
}

// buildSaveUserGroupReq 将前端简化请求模型转换为 BKMonitor 写接口所需的最小合法请求。
// BKMonitor 虽然暴露了大量轮值相关字段，但当前场景只需要固定组装一个非轮值 duty arrange，
// 以避免把底层结构细节暴露给前端。
func buildSaveUserGroupReq(
	bkMonitorProjectID int64,
	requestID *int64,
	params *SaveParams,
) *bkmapi.SaveUserGroupReq {
	return &bkmapi.SaveUserGroupReq{
		ID:           requestID,
		BkBizID:      bkMonitorProjectID,
		Name:         params.Name,
		Timezone:     saveUserGroupDefaultTimezone,
		NeedDuty:     false,
		Channels:     params.Channels,
		Desc:         params.Desc,
		AlertNotice:  normalizeAlertNotices(params.AlertNotice),
		ActionNotice: normalizeActionNotices(params.ActionNotice),
		Operator:     params.Operator,
		DutyArranges: buildSaveDutyArranges(params.Users),
	}
}

// buildSaveDutyArranges 组装 BKMonitor 非轮值场景下最小可用的 duty arrange。
// 把列表/对象字段显式初始化为空结构，避免被序列化成 null 后被监控接口拒绝。
func buildSaveDutyArranges(users []bkmapi.UserGroupUser) []bkmapi.DutyArrange {
	return []bkmapi.DutyArrange{{
		GroupType:    "specified",
		GroupNumber:  0,
		NeedRotation: false,
		DutyTime:     []map[string]any{},
		HandoffTime:  map[string]any{},
		DutyUsers:    [][]bkmapi.UserGroupUser{},
		Users:        append([]bkmapi.UserGroupUser(nil), users...),
		Backups:      []map[string]any{},
		Order:        1,
	}}
}

// normalizeAlertNotices 将 alert notice 中可能出现的 nil slice 统一改成空数组。
// 这么做是因为 BKMonitor 写接口会把 nil 序列化成 null，而像 notify_config/type/notice_ways/receivers
// 这类字段传 null 会被监控侧直接判为非法；由于这些字段嵌套在多层 slice 里，只能逐层遍历补齐。
func normalizeAlertNotices(notices []bkmapi.AlertNotice) []bkmapi.AlertNotice {
	normalized := append([]bkmapi.AlertNotice(nil), notices...)
	for i := range normalized {
		notice := &normalized[i]
		if notice.NotifyConfig == nil {
			notice.NotifyConfig = []bkmapi.AlertNoticeConfig{}
		}
		for j := range notice.NotifyConfig {
			config := &notice.NotifyConfig[j]
			if config.Type == nil {
				config.Type = []string{}
			}
			if config.NoticeWays == nil {
				config.NoticeWays = []bkmapi.NoticeWay{}
			}
			for k := range config.NoticeWays {
				noticeWay := &config.NoticeWays[k]
				if noticeWay.Receivers == nil {
					noticeWay.Receivers = []string{}
				}
			}
		}
	}
	return normalized
}

// normalizeActionNotices 与 normalizeAlertNotices 同理，
// 需要在下发前把 action notice 中的 nil slice 规范化成空数组，避免被 BKMonitor 拒绝。
func normalizeActionNotices(notices []bkmapi.ActionNotice) []bkmapi.ActionNotice {
	normalized := append([]bkmapi.ActionNotice(nil), notices...)
	for i := range normalized {
		notice := &normalized[i]
		if notice.NotifyConfig == nil {
			notice.NotifyConfig = []bkmapi.ActionNoticeConfig{}
		}
		for j := range notice.NotifyConfig {
			config := &notice.NotifyConfig[j]
			if config.Type == nil {
				config.Type = []string{}
			}
			if config.NoticeWays == nil {
				config.NoticeWays = []bkmapi.NoticeWay{}
			}
			for k := range config.NoticeWays {
				noticeWay := &config.NoticeWays[k]
				if noticeWay.Receivers == nil {
					noticeWay.Receivers = []string{}
				}
			}
		}
	}
	return normalized
}

// Delete 删除告警组。删除前会先校验告警组归属。
func (s *Service) Delete(
	ctx context.Context,
	ws *workspace.Workspace,
	groupID int64,
	operator string,
) error {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return errors.Wrap(err, "new bkmonitor client")
	}
	if _, err = s.getWithClient(ctx, client, groupID, bkMonitorProjectID); err != nil {
		return errors.Wrap(err, "validate user group ownership before delete")
	}
	if err = client.DeleteUserGroup(ctx, &bkmapi.DeleteUserGroupReq{
		IDs:      []int64{groupID},
		BkBizIDs: []int64{bkMonitorProjectID},
		Operator: operator,
	}); err != nil {
		return errors.Wrapf(err, "delete user group, bk_biz_id=%d, id=%d", bkMonitorProjectID, groupID)
	}
	return nil
}
