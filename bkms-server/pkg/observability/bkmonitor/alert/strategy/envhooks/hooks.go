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

// Package envhooks 注册告警策略依赖的环境领域事件 Hook。
package envhooks

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

// RegisterUpdateHooks registers alert strategy hooks with explicit store dependencies.
func RegisterUpdateHooks(
	workspaceStore workspace.WorkspaceStore,
	alertStrategyStore alertstrategy.Store,
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	snapshotStore topology.ResourceSnapshotStore,
) {
	bkmsenv.RegisterUpdateHook(
		ReconcileEnvTypeChangeHookName,
		NewReconcileEnvTypeChangeHook(workspaceStore, alertStrategyStore, envStore, appStore, snapshotStore),
	)
}

// NewReconcileEnvTypeChangeHook creates a hook that reconciles alert strategies after env type changes.
func NewReconcileEnvTypeChangeHook(
	workspaceStore workspace.WorkspaceStore,
	alertStrategyStore alertstrategy.Store,
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	snapshotStore topology.ResourceSnapshotStore,
) bkmsenv.UpdateHook {
	return func(ctx context.Context, before, after envmodel.Environment) error {
		if before.Type == after.Type {
			return nil
		}

		ws, err := workspaceStore.Get(ctx, after.WorkspaceID)
		if err != nil {
			return errors.Wrapf(err, "get workspace %s", after.WorkspaceID)
		}
		operatorID := auth.MaintenanceUserID
		if user, userErr := auth.GetUser(ctx); userErr == nil && user.ID != "" {
			operatorID = user.ID
		}

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
