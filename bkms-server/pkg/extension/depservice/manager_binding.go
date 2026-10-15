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

package depservice

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// CreateServiceBinding 创建一个依赖服务和应用间的绑定
func (m *ServiceManager) CreateServiceBinding(
	ctx context.Context,
	params *CreateServiceBindingParams,
) (*model.ServiceBinding, error) {
	if err := validate.Struct(params); err != nil {
		return nil, errors.Wrap(err, "validate params")
	}

	envMap := params.EnvInstanceMap
	if envMap == nil {
		envMap = map[string]bson.ObjectID{}
	}
	envVars := params.EnvVars
	if envVars == nil {
		envVars = map[string]string{}
	}

	if err := m.validateEnvInstanceMap(ctx, params.AppID, params.WorkspaceID, params.ServiceName, envMap); err != nil {
		return nil, err
	}

	binding := &model.ServiceBinding{
		Name:           params.Name,
		AppID:          params.AppID,
		WorkspaceID:    params.WorkspaceID,
		ServiceName:    params.ServiceName,
		EnvInstanceMap: envMap,
		EnvVars:        envVars,
		Description:    params.Description,
	}
	if err := m.bindings.Create(ctx, binding); err != nil {
		if errors.Is(err, model.ErrBindingNameExists) {
			return nil, errors.Wrapf(
				err,
				"binding %q already exists for service %s in app %s",
				params.Name, params.ServiceName, params.AppID,
			)
		}
		return nil, errors.Wrap(err, "create service binding")
	}
	return binding, nil
}

// GetServiceBinding 按应用 + 服务 + 名称获取绑定。
func (m *ServiceManager) GetServiceBinding(
	ctx context.Context,
	appID, serviceName, name string,
) (*model.ServiceBinding, error) {
	return m.bindings.Get(ctx, model.ByBindingName(appID, serviceName, name))
}

// ListServiceBindings 列出应用下某服务的全部绑定。
func (m *ServiceManager) ListServiceBindings(
	ctx context.Context,
	appID, serviceName string,
) ([]*model.ServiceBinding, error) {
	bindings, err := m.bindings.List(ctx, model.ByBindingQuery(&model.BindingQueryOptions{
		AppID:       appID,
		ServiceName: serviceName,
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "list service bindings of app %s service %s", appID, serviceName)
	}
	return lo.ToSlicePtr(bindings), nil
}

// UpdateServiceBinding 全量更新绑定的环境映射与环境变量。
func (m *ServiceManager) UpdateServiceBinding(
	ctx context.Context,
	appID, serviceName, name string,
	params *UpdateServiceBindingParams,
) (*model.ServiceBinding, error) {
	if err := validate.Struct(params); err != nil {
		return nil, errors.Wrap(err, "validate params")
	}

	existing, err := m.bindings.Get(ctx, model.ByBindingName(appID, serviceName, name))
	if err != nil {
		return nil, errors.Wrap(err, "get service binding")
	}

	envMap := params.EnvInstanceMap
	if envMap == nil {
		envMap = map[string]bson.ObjectID{}
	}
	envVars := params.EnvVars
	if envVars == nil {
		envVars = map[string]string{}
	}

	if err = m.validateEnvInstanceMap(ctx, appID, existing.WorkspaceID, serviceName, envMap); err != nil {
		return nil, err
	}

	if err = model.UpdateBinding(ctx, m.bindings, appID, serviceName, name, &model.ServiceBindingUpdateData{
		EnvInstanceMap: envMap,
		EnvVars:        envVars,
		Description:    params.Description,
	}); err != nil {
		return nil, errors.Wrap(err, "update service binding")
	}
	return m.bindings.Get(ctx, model.ByBindingName(appID, serviceName, name))
}

// DeleteServiceBinding 删除应用侧绑定。
func (m *ServiceManager) DeleteServiceBinding(ctx context.Context, appID, serviceName, name string) error {
	return m.bindings.Delete(ctx, model.ByBindingName(appID, serviceName, name))
}

func (m *ServiceManager) listBindingsByInstance(
	ctx context.Context,
	instID bson.ObjectID,
) ([]model.ServiceBinding, error) {
	bindings, err := m.bindings.List(ctx, model.ByBindingQuery(&model.BindingQueryOptions{InstanceID: instID}))
	if err != nil {
		return nil, errors.Wrapf(err, "list service bindings of instance %s", instID.Hex())
	}
	return bindings, nil
}

func (m *ServiceManager) validateEnvInstanceMap(
	ctx context.Context,
	appID, workspaceID, serviceName string,
	envMap map[string]bson.ObjectID,
) error {
	for envName, instID := range envMap {
		if envName == "" {
			return errors.Wrap(ErrInvalidArgument, "env name in envInstanceMap must not be empty")
		}
		if instID.IsZero() {
			return errors.Wrapf(ErrInvalidArgument, "instance id for env %q must not be empty", envName)
		}

		environment, err := m.envs.Get(ctx, envmodel.AppEnvByName(workspaceID, appID, envName))
		if err != nil {
			if errors.Is(err, envmodel.ErrEnvNotFound) {
				return errors.Wrapf(ErrInvalidArgument, "environment %q not found", envName)
			}
			return errors.Wrapf(err, "get environment %s", envName)
		}
		envType := environment.Type

		inst, err := m.insts.Get(ctx, store.ByID(instID))
		if err != nil {
			if model.AsNotFoundError(err) {
				return errors.Wrapf(ErrInvalidArgument, "service instance %s not found", instID.Hex())
			}
			return errors.Wrap(err, "get service instance")
		}
		if inst.WorkspaceID != workspaceID {
			return errors.Wrapf(
				ErrInvalidArgument,
				"instance %s does not belong to workspace %s",
				instID.Hex(),
				workspaceID,
			)
		}
		if inst.ServiceName != serviceName {
			return errors.Wrapf(ErrInvalidArgument, "instance %s is not a %s instance", instID.Hex(), serviceName)
		}
		if !inst.MatchesEnv(envName, envType) {
			return errors.Wrapf(
				ErrInvalidArgument,
				"instance %s is not available in environment %q",
				instID.Hex(),
				envName,
			)
		}
	}
	return nil
}
