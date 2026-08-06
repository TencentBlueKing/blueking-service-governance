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

// Package strategy 提供蓝鲸监控告警策略相关功能
package strategy

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// NewService 创建 Service 实例。
func NewService(
	store Store,
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	snapshotStore topology.ResourceSnapshotStore,
) *Service {
	return newServiceWithClientFactory(store, envStore, appStore, snapshotStore, bkmapi.NewMonitorClient)
}

func newServiceWithClientFactory(
	store Store,
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	snapshotStore topology.ResourceSnapshotStore,
	newClient alert.ClientFactory,
) *Service {
	return &Service{
		store:         store,
		envStore:      envStore,
		appStore:      appStore,
		snapshotStore: snapshotStore,
		newClient:     newClient,
	}
}

// Create 创建告警策略。
func (s *Service) Create(ctx context.Context, req *CreateReq) (*AlertStrategy, error) {
	if err := req.EffectiveScope.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate effectiveScope")
	}
	strategy := buildStrategyFromCreateReq(req)
	id, err := s.store.Create(ctx, strategy)
	if err != nil {
		return nil, errors.Wrap(err, "create alert strategy")
	}
	strategy.ID = id
	return strategy, nil
}

// CreateAndSync 创建告警策略；当策略启用时，先同步远端，再持久化本地记录。
func (s *Service) CreateAndSync(
	ctx context.Context,
	ws *workspace.Workspace,
	req *CreateReq,
) (*AlertStrategy, error) {
	if !req.Enabled {
		return s.Create(ctx, req)
	}
	if err := req.EffectiveScope.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate effectiveScope")
	}

	strategy := buildStrategyFromCreateReq(req)
	strategy.ID = bson.NewObjectID()

	remoteRefs, err := s.syncStrategyStateToRemote(ctx, ws, strategy, req.Operator)
	if err != nil {
		return nil, err
	}
	strategy.RemoteRefs = remoteRefs

	if _, err = s.store.Create(ctx, strategy); err != nil {
		return nil, errors.Wrap(err, "create alert strategy")
	}
	return strategy, nil
}

// Get 获取指定的告警策略。
func (s *Service) Get(ctx context.Context, id bson.ObjectID) (*AlertStrategy, error) {
	return s.store.Get(ctx, id)
}

// ListByWorkspace 获取工作空间下的告警策略列表。
func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string) ([]AlertStrategy, error) {
	return s.store.ListByWorkspace(ctx, workspaceID)
}

// ListByApp 获取应用下的告警策略列表。
func (s *Service) ListByApp(ctx context.Context, workspaceID, appID string) ([]AlertStrategy, error) {
	return s.store.ListByApp(ctx, workspaceID, appID)
}

// Update 更新告警策略；若请求未产生变更则返回 false。
func (s *Service) Update(ctx context.Context, id bson.ObjectID, req *UpdateReq) (bool, error) {
	updateData, changed, err := req.ToBSON()
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	updateData["updater"] = req.Operator
	return true, s.store.Update(ctx, id, updateData)
}

// UpdateAndSync 更新告警策略；若有变更，则先同步远端，再持久化本地记录。
func (s *Service) UpdateAndSync(
	ctx context.Context,
	ws *workspace.Workspace,
	id bson.ObjectID,
	req *UpdateReq,
) (bool, error) {
	updateData, changed, err := req.ToBSON()
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	if err = s.withLockedStrategy(ctx, id, "get alert strategy before update", func(current *AlertStrategy) error {
		next := cloneStrategy(current)
		req.ApplyTo(next)
		if !next.Enabled && len(current.RemoteRefs) == 0 {
			updateData["updater"] = req.Operator
			if updateErr := s.store.Update(ctx, id, updateData); updateErr != nil {
				return errors.Wrap(updateErr, "update alert strategy")
			}
			return nil
		}

		remoteRefs, syncErr := s.syncStrategyStateToRemote(ctx, ws, next, req.Operator)
		if syncErr != nil {
			return syncErr
		}

		updateData["remoteRefs"] = remoteRefs
		updateData["updater"] = req.Operator
		if updateErr := s.store.Update(ctx, id, updateData); updateErr != nil {
			return errors.Wrap(updateErr, "update alert strategy")
		}
		return nil
	}); err != nil {
		return false, err
	}
	return true, nil
}

// Delete 删除指定的告警策略。
//
// 删除前若策略已同步到蓝鲸监控远端（存在 remoteRefs），必须先清理对应的远端策略；
// 远端清理失败则直接返回错误，避免本地与远端状态不一致。
func (s *Service) Delete(ctx context.Context, ws *workspace.Workspace, id bson.ObjectID, operator string) error {
	return s.withLockedStrategy(ctx, id, "get alert strategy before delete", func(strategy *AlertStrategy) error {
		if len(strategy.RemoteRefs) > 0 {
			if err := s.deleteRemoteStrategies(ctx, ws, strategy, operator); err != nil {
				return errors.Wrap(err, "delete remote strategies")
			}
		}
		if err := s.store.Delete(ctx, id); err != nil {
			return errors.Wrap(err, "delete alert strategy")
		}
		return nil
	})
}

// InitDefaultAlertStrategiesForApp 为应用初始化预置告警策略（仅本地，不自动同步远端）。
func (s *Service) InitDefaultAlertStrategiesForApp(
	ctx context.Context,
	workspaceID, appID, appName string,
	operator string,
) error {
	for _, tmpl := range defaultTemplates {
		rule := &AlertStrategy{
			WorkspaceID:        workspaceID,
			AppID:              appID,
			AppName:            appName,
			StrategyCode:       tmpl.StrategyCode,
			DisplayName:        tmpl.DisplayName,
			MonitorMetric:      tmpl.MonitorMetric,
			Severity:           tmpl.Severity,
			Threshold:          tmpl.Threshold,
			TriggerCondition:   tmpl.TriggerCondition,
			RecoverCondition:   tmpl.RecoverCondition,
			EffectiveTimeRange: tmpl.EffectiveTimeRange,
			EffectiveScope:     EffectiveScope{Type: EffectiveScopeAll},
			Enabled:            true,
			Creator:            operator,
			Updater:            operator,
		}

		if _, createErr := s.store.Create(ctx, rule); createErr != nil {
			log.Errorf(ctx, "failed to create default alert strategy %s for workspace %s app %s: %v",
				tmpl.StrategyCode, workspaceID, appID, createErr)
			return createErr
		}
		log.Infof(
			ctx,
			"created default alert strategy %s for workspace %s app %s",
			tmpl.StrategyCode,
			workspaceID,
			appID,
		)
	}
	return nil
}

func buildStrategyFromCreateReq(req *CreateReq) *AlertStrategy {
	return &AlertStrategy{
		WorkspaceID:        req.WorkspaceID,
		AppID:              req.AppID,
		AppName:            req.AppName,
		StrategyCode:       req.StrategyCode,
		DisplayName:        req.DisplayName,
		MonitorMetric:      req.MonitorMetric,
		Severity:           req.Severity,
		Threshold:          req.Threshold,
		TriggerCondition:   req.TriggerCondition,
		RecoverCondition:   req.RecoverCondition,
		EffectiveTimeRange: req.EffectiveTimeRange,
		EffectiveScope:     req.EffectiveScope,
		NoticeGroupIDs:     req.NoticeGroupIDs,
		Enabled:            req.Enabled,
		Creator:            req.Operator,
		Updater:            req.Operator,
	}
}

func cloneStrategy(strategy *AlertStrategy) *AlertStrategy {
	if strategy == nil {
		return nil
	}
	cloned := *strategy
	cloned.NoticeGroupIDs = append([]int64(nil), strategy.NoticeGroupIDs...)
	cloned.RemoteRefs = append([]RemoteStrategyRef(nil), strategy.RemoteRefs...)
	cloned.EffectiveScope.EnvTypes = append([]string(nil), strategy.EffectiveScope.EnvTypes...)
	cloned.EffectiveScope.EnvIDs = append([]bson.ObjectID(nil), strategy.EffectiveScope.EnvIDs...)
	return &cloned
}
