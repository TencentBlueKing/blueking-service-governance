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

package alertstrategysync

import (
	"context"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// handleSync 同步应用在指定环境/泳道下的告警策略到蓝鲸监控远端。
// 先校验 registry 与 workspace，再定位 env；env 不存在属不可恢复，直接 StopRetry。
func handleSync(ctx context.Context, args Args) error {
	reg, ws, err := resolveContext(ctx, args)
	if err != nil {
		return err
	}

	env, err := reg.EnvStore.GetByName(ctx, args.WorkspaceID, args.AppID, args.EnvName)
	if err != nil {
		return errors.Wrapf(err, "get env %s", args.EnvName)
	}
	if env == nil {
		log.Warnf(ctx, "alert strategy sync: env not found, skip: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "env not found")
	}

	log.Infof(ctx, "alert strategy sync start: %s envID=%s", args, env.ID.Hex())
	alertstrategy.NewService(
		reg.AlertStrategyStore, reg.EnvStore, reg.AppStore, reg.ResourceSnapshotStore,
	).SyncStrategiesForAppInEnv(ctx, ws, args.AppID, env.ID, args.TrafficLaneName, operatorFromCtx(ctx))
	return nil
}

// handleCleanup 清理应用在指定环境/泳道下的告警策略远端引用。
// 与 handleSync 共用上下文解析逻辑；env 不存在时直接 StopRetry，避免无意义重试。
func handleCleanup(ctx context.Context, args Args) error {
	reg, ws, err := resolveContext(ctx, args)
	if err != nil {
		return err
	}

	env, err := reg.EnvStore.GetByName(ctx, args.WorkspaceID, args.AppID, args.EnvName)
	if err != nil {
		return errors.Wrapf(err, "get env %s", args.EnvName)
	}
	if env == nil {
		log.Warnf(ctx, "alert strategy cleanup: env not found, skip: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "env not found")
	}

	log.Infof(ctx, "alert strategy cleanup start: %s envID=%s", args, env.ID.Hex())
	alertstrategy.NewService(
		reg.AlertStrategyStore, reg.EnvStore, reg.AppStore, reg.ResourceSnapshotStore,
	).CleanupStrategiesForAppInEnv(ctx, ws, args.AppID, env.ID, args.TrafficLaneName, operatorFromCtx(ctx))
	return nil
}

// onSyncExhausted 在同步任务重试耗尽后记录最终失败日志。
func onSyncExhausted(ctx context.Context, args Args, lastErr error) {
	log.Errorf(ctx, "alert strategy sync exhausted, giving up: %s err=%v", args, lastErr)
}

// onCleanupExhausted 在清理任务重试耗尽后记录最终失败日志。
func onCleanupExhausted(ctx context.Context, args Args, lastErr error) {
	log.Errorf(ctx, "alert strategy cleanup exhausted, giving up: %s err=%v", args, lastErr)
}

// resolveContext 获取 store registry 和 workspace，返回 ErrStopRetry 代表不可恢复
func resolveContext(ctx context.Context, args Args) (*storereg.Registry, *workspace.Workspace, error) {
	reg := storereg.G()
	if reg == nil || reg.AlertStrategyStore == nil || reg.EnvStore == nil ||
		reg.AppStore == nil || reg.ResourceSnapshotStore == nil || reg.WorkspaceStore == nil {
		return nil, nil, errors.Wrap(taskq.ErrStopRetry, "required store not initialized")
	}

	ws, err := reg.WorkspaceStore.Get(ctx, args.WorkspaceID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get workspace %s", args.WorkspaceID)
	}
	if ws == nil {
		return nil, nil, errors.Wrap(taskq.ErrStopRetry, "workspace not found")
	}
	return reg, ws, nil
}

// operatorFromCtx 从 asynq context 中获取操作人（由 taskq envelope 恢复）
func operatorFromCtx(ctx context.Context) string {
	user, err := auth.GetUser(ctx)
	if err != nil || user.ID == "" {
		return "system"
	}
	return user.ID
}
