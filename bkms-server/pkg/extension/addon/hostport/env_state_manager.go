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

package hostport

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// EnvStateManager reconciles HostPort applied-port snapshots after deploy/uninstall.
type EnvStateManager struct {
	db *mongo.Database
}

// NewEnvStateManager creates an EnvStateManager.
func NewEnvStateManager(db *mongo.Database) *EnvStateManager {
	return &EnvStateManager{db: db}
}

// ReconcileAfterDeploy records the ports that were actually injected/deployed
// for a federated env. appliedPorts must be the build-time snapshot (not a fresh
// ListPorts), so mid-deploy config changes do not skew pending state.
func (m *EnvStateManager) ReconcileAfterDeploy(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	appliedPorts []int32,
) error {
	if !env.Cluster.IsFederation {
		return nil
	}
	return UpsertEnvState(ctx, m.db, app.ID, env.Name, appliedPorts)
}

// ReconcileAfterUninstall removes the env HostPort snapshot for federated envs.
func (m *EnvStateManager) ReconcileAfterUninstall(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
) error {
	if !env.Cluster.IsFederation {
		return nil
	}
	return RemoveEnvState(ctx, m.db, app.ID, env.Name)
}
