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
	"slices"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

// use a single instance of Validate, it caches struct info
var validate = validator.New(validator.WithRequiredStructEnabled())

// ServiceManager is the manager for app dependency services
type ServiceManager struct {
	svcStore  model.ServiceStore
	instStore model.ServiceInstanceStore
}

// New creates a new app dependency services manager
func New(svcStore model.ServiceStore, instStore model.ServiceInstanceStore) *ServiceManager {
	return &ServiceManager{svcStore: svcStore, instStore: instStore}
}

// CreateServiceInstance 创建服务实例
func (m *ServiceManager) CreateServiceInstance(
	ctx context.Context,
	params *CreateServiceInstanceParams,
) (bson.ObjectID, error) {
	if err := validate.Struct(params); err != nil {
		return bson.NilObjectID, errors.Wrap(err, "validate params")
	}

	// 先构造 provider，失败时无需写库，避免残留无效记录
	plan, svcProvider, err := m.getPlanAndProvider(ctx, params.ServiceName, params.PlanName)
	if err != nil {
		return bson.NilObjectID, err
	}

	// 初始化服务实例数据到 db 中（异步 CreateInstance 需要先拿到 instID）
	attachedApps := params.AttachedApps
	if attachedApps == nil {
		attachedApps = make([]string, 0)
	}
	instID, err := m.instStore.Create(
		ctx,
		&model.ServiceInstance{
			Name:          params.Name,
			ServiceName:   params.ServiceName,
			PlanName:      params.PlanName,
			ProviderType:  plan.ProviderType,
			ScopeType:     params.ScopeType,
			ScopeValue:    params.ScopeValue,
			WorkspaceID:   params.WorkspaceID,
			AttachedApps:  attachedApps,
			CustomEnvVars: params.CustomEnvVars,
			Operator:      params.Operator,
			Description:   params.Description,
			Status:        model.ProvisioningStatus,
		},
	)
	if err != nil {
		return instID, errors.Wrap(err, "init service instance to db")
	}

	createResult, err := svcProvider.CreateInstance(ctx, instID.Hex(), &types.ServicePlanConfig{
		Config: plan.Config,
	}, params.Params)
	if err != nil {
		createErr := errors.Wrap(err, "create service instance by provider")
		return instID, m.updateInstanceStatus(ctx, instID, model.CreateFailedStatus, createErr)
	}

	// 异步 provider 已投递 task，实例保持 provisioning 状态
	if createResult.Async {
		return instID, nil
	}

	// 同步 provider 直接更新配置和凭证
	if len(createResult.InstConfig) > 0 {
		if err = m.instStore.UpdateConfig(ctx, instID, createResult.InstConfig); err != nil {
			updateErr := errors.Wrap(err, "update service instance config")
			return instID, m.updateInstanceStatus(ctx, instID, model.UnavailableStatus, updateErr)
		}
	}
	if len(createResult.Credentials) > 0 {
		if err = m.instStore.UpdateCredentials(ctx, instID, createResult.Credentials); err != nil {
			updateErr := errors.Wrap(err, "update service instance credentials")
			return instID, m.updateInstanceStatus(ctx, instID, model.UnavailableStatus, updateErr)
		}
	}

	return instID, m.updateInstanceStatus(ctx, instID, model.AvailableStatus, nil)
}

// getPlanAndProvider 按服务名与 plan 名解析 plan 并构造对应 provider。
func (m *ServiceManager) getPlanAndProvider(
	ctx context.Context,
	serviceName, planName string,
) (*model.ServicePlan, provider.ServiceProvider, error) {
	svc, err := m.svcStore.Get(ctx, serviceName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get service %s", serviceName)
	}
	plan, err := svc.GetPlanByName(planName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get service plan %s", planName)
	}
	svcProvider, err := provider.New(svc.Name, plan)
	if err != nil {
		return nil, nil, errors.Wrap(err, "create service provider")
	}
	return plan, svcProvider, nil
}

// GetServiceInstance 查询服务实例
func (m *ServiceManager) GetServiceInstance(
	ctx context.Context,
	instID bson.ObjectID,
) (*model.ServiceInstance, error) {
	return m.instStore.Get(ctx, instID)
}

// DeleteServiceInstance 删除服务实例. 用于需要销毁实例的场景.
// NOTE: 如果实例被某个应用使用, 则无法删除
func (m *ServiceManager) DeleteServiceInstance(ctx context.Context, instID bson.ObjectID) error {
	inst, err := m.instStore.Get(ctx, instID)
	if err != nil {
		return errors.Wrap(err, "get service instance")
	}
	if len(inst.AttachedApps) != 0 {
		return errors.Errorf("service instance is used by apps(%v), cannot delete", inst.AttachedApps)
	}

	if inst.Status == model.DeletingStatus {
		return nil
	}

	if inst.Status == model.CreateFailedStatus {
		return m.instStore.Delete(ctx, instID)
	}

	if !isDeletableStatus(inst.Status) {
		return errors.Errorf(
			"service instance status %q does not allow delete, wait until it becomes available/unavailable/createFailed/deleteFailed",
			inst.Status,
		)
	}

	plan, svcProvider, err := m.getPlanAndProvider(ctx, inst.ServiceName, inst.PlanName)
	if err != nil {
		return err
	}

	if err = m.updateInstanceStatus(ctx, instID, model.DeletingStatus, nil); err != nil {
		return err
	}

	deleteResult, err := svcProvider.DeleteInstance(ctx, instID.Hex(), &types.ServicePlanConfig{
		Config: plan.Config,
	}, inst.Config)
	if err != nil {
		deleteErr := errors.Wrap(err, "delete service instance by provider")
		return m.updateInstanceStatus(ctx, instID, model.DeleteFailedStatus, deleteErr)
	}

	if deleteResult.Async {
		return nil
	}

	return m.instStore.Delete(ctx, instID)
}

// isDeletableStatus 判断是否允许走 provider 删除路径的稳定态。
func isDeletableStatus(status model.InstanceStatus) bool {
	return slices.Contains([]model.InstanceStatus{
		model.AvailableStatus,
		model.UnavailableStatus,
		model.DeleteFailedStatus,
	}, status)
}

// AttachInstanceToApp 将服务实例附加到应用，建立应用对服务实例的使用关系
func (m *ServiceManager) AttachInstanceToApp(ctx context.Context, instID bson.ObjectID, appID string) error {
	return m.instStore.AttachApp(ctx, instID, appID)
}

// DetachInstanceFromApp 将服务实例从应用分离，解除应用对服务实例的使用关系
func (m *ServiceManager) DetachInstanceFromApp(ctx context.Context, instID bson.ObjectID, appID string) error {
	return m.instStore.DetachApp(ctx, instID, appID)
}

func (m *ServiceManager) updateInstanceStatus(
	ctx context.Context,
	instID bson.ObjectID,
	status model.InstanceStatus,
	statusError error,
) error {
	var message string

	if statusError != nil {
		message = statusError.Error()
	}

	if err := m.instStore.UpdateStatus(ctx, instID, status, message); err != nil {
		// 有业务错误时优先返回业务错误
		log.Errorf(ctx, "update service instance status failed: %s", err)
		if statusError != nil {
			return statusError
		}
		return errors.Wrap(err, "update service instance status")
	}

	return statusError
}

// TODO 后续补齐依赖服务相关的其他方法(如 ListServiceInstances 等), 目前方法供北极星使用即可
