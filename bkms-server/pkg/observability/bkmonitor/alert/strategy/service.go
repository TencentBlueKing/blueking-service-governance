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
	stderrors "errors"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// ErrDisplayNameAlreadyExists 表示同一应用下已存在相同策略名称。
var ErrDisplayNameAlreadyExists = errors.New("alert strategy displayName already exists in app")

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
	if err := s.ensureDisplayNameUnique(ctx, req.WorkspaceID, req.AppID, req.DisplayName, nil); err != nil {
		return nil, err
	}
	strategy := buildStrategyFromCreateReq(req)
	id, err := s.store.Create(ctx, strategy)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDisplayNameAlreadyExists
		}
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
	if err := s.ensureDisplayNameUnique(ctx, req.WorkspaceID, req.AppID, req.DisplayName, nil); err != nil {
		return nil, err
	}

	strategy := buildStrategyFromCreateReq(req)
	strategy.ID = bson.NewObjectID()

	remoteRefs, err := s.syncStrategyStateToRemote(ctx, ws, strategy, req.Operator)
	if err != nil {
		return nil, err
	}
	strategy.RemoteRefs = remoteRefs

	if _, err = s.store.Create(ctx, strategy); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDisplayNameAlreadyExists
		}
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
		if next.DisplayName != current.DisplayName {
			if uniqueErr := s.ensureDisplayNameUnique(
				ctx,
				current.WorkspaceID,
				current.AppID,
				next.DisplayName,
				&id,
			); uniqueErr != nil {
				return uniqueErr
			}
		}
		if !next.Enabled && len(current.RemoteRefs) == 0 {
			updateData["updater"] = req.Operator
			if updateErr := s.store.Update(ctx, id, updateData); updateErr != nil {
				if mongo.IsDuplicateKeyError(updateErr) {
					return ErrDisplayNameAlreadyExists
				}
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
			if mongo.IsDuplicateKeyError(updateErr) {
				return ErrDisplayNameAlreadyExists
			}
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

// DeleteByApp 删除应用下的所有告警策略。
//
// 该方法用于应用删除后的异步补偿清理，会尽量删除完全部策略并聚合失败信息返回，
// 以便调用方记录日志或做后续补偿，而不会因为首条失败就停止后续清理。
func (s *Service) DeleteByApp(
	ctx context.Context,
	ws *workspace.Workspace,
	workspaceID, appID string,
	operator string,
) error {
	strategies, err := s.store.ListByApp(ctx, workspaceID, appID)
	if err != nil {
		return errors.Wrap(err, "list alert strategies by app")
	}

	var joinedErr error
	for _, strategy := range strategies {
		if err := s.Delete(ctx, ws, strategy.ID, operator); err != nil {
			log.Errorf(
				ctx,
				"delete alert strategy during app cleanup failed, workspaceID=%s appID=%s strategyID=%s code=%s err=%v",
				workspaceID,
				appID,
				strategy.ID.Hex(),
				strategy.StrategyCode,
				err,
			)
			joinedErr = stderrors.Join(joinedErr, errors.Wrapf(err, "delete alert strategy %s", strategy.ID.Hex()))
		}
	}
	return joinedErr
}

// InitDefaultAlertStrategiesForApp 为应用初始化预置告警策略（仅本地，不自动同步远端）。
func (s *Service) InitDefaultAlertStrategiesForApp(
	ctx context.Context,
	workspaceID, appID, appName string,
	operator string,
	noticeGroupIDs []int64,
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
			NoticeGroupIDs:     append([]int64(nil), noticeGroupIDs...),
			Enabled:            true,
			Creator:            operator,
			Updater:            operator,
		}

		if err := s.ensureDisplayNameUnique(ctx, workspaceID, appID, rule.DisplayName, nil); err != nil {
			return err
		}
		if _, createErr := s.store.Create(ctx, rule); createErr != nil {
			if mongo.IsDuplicateKeyError(createErr) {
				return ErrDisplayNameAlreadyExists
			}
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
		MonitorMetric:      MonitorMetricForStrategyCode(req.StrategyCode),
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

func (s *Service) ensureDisplayNameUnique(
	ctx context.Context,
	workspaceID, appID, displayName string,
	excludeID *bson.ObjectID,
) error {
	exists, err := s.store.ExistsByAppAndDisplayName(ctx, workspaceID, appID, displayName, excludeID)
	if err != nil {
		return errors.Wrap(err, "check alert strategy displayName uniqueness")
	}
	if exists {
		return ErrDisplayNameAlreadyExists
	}
	return nil
}
