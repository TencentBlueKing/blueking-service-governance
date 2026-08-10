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

// Package hooks 注册告警策略依赖的环境领域事件 Hook。
package hooks

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

const ReconcileEnvTypeChangeHookName = "alert_strategy.reconcile_env_type_change"

// RegisterUpdateHooks 注册告警策略相关的环境更新 Hook，并显式传入所需的存储依赖。
func RegisterUpdateHooks(
	workspaceStore workspace.WorkspaceStore,
	alertStrategyStore alertstrategy.Store,
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	snapshotStore topology.ResourceSnapshotStore,
) {
	// 注册环境类型变更时协调告警策略的 Hook。
	bkmsenv.RegisterUpdateHook(
		ReconcileEnvTypeChangeHookName,
		NewReconcileEnvTypeChangeHook(workspaceStore, alertStrategyStore, envStore, appStore, snapshotStore),
	)
}

// NewReconcileEnvTypeChangeHook 创建一个 Hook，用于在环境类型发生变更后 reconcile 告警策略。
func NewReconcileEnvTypeChangeHook(
	workspaceStore workspace.WorkspaceStore,
	alertStrategyStore alertstrategy.Store,
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	snapshotStore topology.ResourceSnapshotStore,
) bkmsenv.UpdateHook {
	return func(ctx context.Context, before, after envmodel.Environment) error {
		// 环境类型未变化时无需处理，直接跳过。
		if before.Type == after.Type {
			return nil
		}

		// 查询环境所属的工作空间，用于后续的告警策略协调。
		ws, err := workspaceStore.Get(ctx, after.WorkspaceID)
		if err != nil {
			return errors.Wrapf(err, "get workspace %s", after.WorkspaceID)
		}
		// 默认以运维用户作为操作人；若上下文中存在真实用户则优先使用真实用户。
		operatorID := auth.MaintenanceUserID
		if user, userErr := auth.GetUser(ctx); userErr == nil && user.ID != "" {
			operatorID = user.ID
		}

		// 构建告警策略服务，并针对本次环境类型变更协调相关告警策略。
		svc := alertstrategy.NewService(alertStrategyStore, envStore, appStore, snapshotStore)
		if err := svc.ReconcileStrategiesForEnvTypeChange(
			ctx,
			ws,
			before,
			after,
			operatorID,
		); err != nil {
			return errors.Wrapf(
				err,
				"reconcile alert strategies for workspace %s env %s type change %s->%s",
				after.WorkspaceID,
				after.Name,
				before.Type,
				after.Type,
			)
		}
		return nil
	}
}
