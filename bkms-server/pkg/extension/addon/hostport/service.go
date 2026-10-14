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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Service provides HostPort mapping CRUD and federated env state queries.
type Service struct {
	db       *mongo.Database
	envStore *envmodel.EnvironmentStore
}

// NewService creates a HostPort Service.
func NewService(db *mongo.Database) *Service {
	return &Service{db: db, envStore: store.Of(db, envmodel.EnvMeta)}
}

// GetHostPorts returns declared ports and federated env pending-deploy views.
func (s *Service) GetHostPorts(ctx context.Context, app *bkmsapp.Application) (*HostPorts, error) {
	ports, err := s.ListPorts(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	envStates, err := s.ListFederatedEnvStates(ctx, app)
	if err != nil {
		return nil, err
	}
	return &HostPorts{Ports: ports, EnvStates: envStates}, nil
}

// ListPorts returns declared container ports for an app.
func (s *Service) ListPorts(ctx context.Context, appID string) ([]int32, error) {
	return ListPorts(ctx, s.db, appID)
}

// ReplacePorts replaces the declared container ports for an app.
func (s *Service) ReplacePorts(ctx context.Context, appID string, ports []int32) ([]int32, error) {
	return ReplacePorts(ctx, s.db, appID, ports)
}

// ListFederatedEnvStates returns HostPort pending-deploy status for federated envs only.
func (s *Service) ListFederatedEnvStates(
	ctx context.Context,
	app *bkmsapp.Application,
) (map[string]EnvStateView, error) {
	envs, err := s.envStore.List(
		ctx,
		envmodel.InWorkspace(app.WorkspaceID),
		envmodel.VisibleToApp(app.ID),
		envmodel.NewestFirst(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "list app envs")
	}

	config, err := store.Of(s.db, HostPortConfigMeta).Get(ctx, ByApp(app.ID))
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return nil, errors.Wrap(err, "get hostport config")
	}

	desired := []int32{}
	if config != nil {
		desired = config.Ports
	}

	result := make(map[string]EnvStateView)
	for i := range envs {
		env := &envs[i]
		if !env.Cluster.IsFederation {
			continue
		}
		var statePtr *HostPortEnvState
		if config != nil {
			if state, ok := config.EnvStates[env.Name]; ok {
				stateCopy := state
				statePtr = &stateCopy
			}
		}
		result[env.Name] = ComputeEnvState(desired, statePtr)
	}
	return result, nil
}
