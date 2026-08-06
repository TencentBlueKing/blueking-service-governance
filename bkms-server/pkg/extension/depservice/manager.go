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
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider"
	ptypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
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

	// 1. 根据服务和指定的 plan, 获取 provider
	svc, err := m.svcStore.Get(ctx, params.ServiceName)
	if err != nil {
		return bson.NilObjectID, errors.Wrapf(err, "get service %s", params.ServiceName)
	}

	plan, err := svc.GetPlanByName(params.PlanName)
	if err != nil {
		return bson.NilObjectID, errors.Wrapf(err, "get service plan %s", params.PlanName)
	}

	svcProvider, err := provider.New(svc.Name, plan)
	if err != nil {
		return bson.NilObjectID, errors.Wrap(err, "create service provider")
	}

	// 2. 初始化服务实例数据到 db 中
	AttachedApps := params.AttachedApps
	if AttachedApps == nil {
		AttachedApps = make([]string, 0)
	}
	instID, err := m.instStore.Create(
		ctx,
		&model.ServiceInstance{
			Name:         params.Name,
			ServiceName:  params.ServiceName,
			PlanName:     params.PlanName,
			ProviderType: plan.ProviderType,

			ScopeType:   params.ScopeType,
			ScopeValue:  params.ScopeValue,
			WorkspaceID: params.WorkspaceID,

			AttachedApps: AttachedApps,

			CustomEnvVars: params.CustomEnvVars,

			Operator:    params.Operator,
			Description: params.Description,
			Status:      model.ProvisioningStatus,
		},
	)
	if err != nil {
		return instID, errors.Wrap(err, "init service instance to db")
	}

	// 3. 调用 provider, 创建服务实例
	createResult, err := svcProvider.CreateInstance(ctx, &ptypes.ServicePlanConfig{
		Config: plan.Config,
	}, params.Params)
	if err != nil {
		createErr := errors.Wrap(err, "create service instance by provider")
		return instID, m.updateInstanceStatus(ctx, instID, model.CreateFailedStatus, createErr)
	}

	// 4. 更新实例配置和凭证
	if err = m.instStore.UpdateConfig(ctx, instID, createResult.InstConfig); err != nil {
		return instID, errors.Wrap(err, "update service instance config")
	}
	if len(createResult.Credentials) > 0 {
		if err = m.instStore.UpdateCredentials(ctx, instID, createResult.Credentials); err != nil {
			return instID, errors.Wrap(err, "update service instance credentials")
		}
	}

	// TODO 改成异步任务
	go func(ctx context.Context) {
		// 5. 轮询实例状态, 直到实例就绪, 并更新实例状态和最终凭证
		if err = m.startPollingInstance(ctx, instID, svcProvider, plan.Config, 60*time.Second); err != nil {
			log.Errorf(ctx, "start polling instance(id:%s) failed: %s", instID, err)
		}
	}(context.WithoutCancel(ctx))

	return instID, nil
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
	// 1. 确认是否有应用使用服务实例, 如果有, 则提示无法删除
	inst, err := m.instStore.Get(ctx, instID)
	if err != nil {
		return errors.Wrap(err, "get service instance")
	}
	if len(inst.AttachedApps) != 0 {
		return errors.Errorf("service instance is used by apps(%v), cannot delete", inst.AttachedApps)
	}

	// 2. 调用 provider, 删除服务实例
	// 如果实例本身没有被 provider 成功创建(即未实际产生有效的服务资源), 可以直接删除 db 中的服务实例记录
	if slices.Contains([]model.InstanceStatus{model.CreateFailedStatus, model.ProvisioningStatus}, inst.Status) {
		return m.instStore.Delete(ctx, instID)
	}

	svc, err := m.svcStore.Get(ctx, inst.ServiceName)
	if err != nil {
		return errors.Wrap(err, "get service")
	}

	plan, err := svc.GetPlanByName(inst.PlanName)
	if err != nil {
		return errors.Wrap(err, "get service plan")
	}

	svcProvider, err := provider.New(svc.Name, plan)
	if err != nil {
		return errors.Wrap(err, "new service provider")
	}

	err = svcProvider.DeleteInstance(ctx, &ptypes.ServicePlanConfig{
		Config: plan.Config,
	}, inst.Config)
	if err != nil {
		deleteErr := errors.Wrap(err, "delete service instance by provider")
		return m.updateInstanceStatus(ctx, instID, model.DeleteFailedStatus, deleteErr)
	}

	// 3. 删除 db 中的服务实例记录
	return m.instStore.Delete(ctx, instID)
}

// AttachInstanceToApp 将服务实例附加到应用，建立应用对服务实例的使用关系
func (m *ServiceManager) AttachInstanceToApp(ctx context.Context, instID bson.ObjectID, appID string) error {
	return m.instStore.AttachApp(ctx, instID, appID)
}

// DetachInstanceFromApp 将服务实例从应用分离，解除应用对服务实例的使用关系
func (m *ServiceManager) DetachInstanceFromApp(ctx context.Context, instID bson.ObjectID, appID string) error {
	return m.instStore.DetachApp(ctx, instID, appID)
}

// startPollingInstance 轮询查询服务实例的创建状态，直到实例就绪或超时
func (m *ServiceManager) startPollingInstance(
	ctx context.Context,
	instID bson.ObjectID,
	svcProvider provider.ServiceProvider,
	planConfig map[string]any,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	d := 1 * time.Second
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	log.Debugf(ctx, "start polling service instance(id:%s) every %v in %v", instID, d, timeout)

	inst, err := m.instStore.Get(ctx, instID)
	if err != nil {
		log.Errorf(ctx, "get service instance(id:%s) failed: %s", instID, err)
		return err
	}

	for {
		select {
		case <-ctx.Done():
			log.Debugf(ctx, "polling service instance(id:%s) by provider exceeded %v, stop polling", instID, timeout)
			return m.updateInstanceStatus(
				context.Background(),
				instID,
				model.CreateFailedStatus,
				errors.Errorf("polling service instance by provider exceeded %v", timeout),
			)
		case <-ticker.C:
			log.Debugf(ctx, "polling service instance(id:%s) once by provider", instID)
			queryResult, err := svcProvider.QueryInstance(ctx, &ptypes.ServicePlanConfig{
				Config: planConfig,
			}, inst.Config)
			if err != nil {
				return m.updateInstanceStatus(ctx, instID, model.CreateFailedStatus, err)
			}

			if queryResult.IsProvisioningComplete() {
				log.Debugf(ctx, "service instance(id:%s) provisioning complete, stop polling", instID)
				return m.updateInstanceByResult(
					ctx,
					queryResult,
					instID,
					lo.Assign(inst.Credentials, queryResult.Credentials),
				)
			}
		}
	}
}

// updateInstanceByResult updates instance credentials when the instance is active
func (m *ServiceManager) updateInstanceByResult(
	ctx context.Context,
	result *ptypes.QueryInstanceResult,
	instID bson.ObjectID,
	credentials map[string]any,
) error {
	if result.Status == ptypes.AvailableStatus {
		if err := m.instStore.UpdateCredentials(ctx, instID, credentials); err != nil {
			updateErr := errors.Wrap(err, "update service instance credentials")
			return m.updateInstanceStatus(ctx, instID, model.UnavailableStatus, updateErr)
		}

		return m.updateInstanceStatus(ctx, instID, model.AvailableStatus, nil)
	}

	if result.Status == ptypes.UnavailableStatus {
		resultErr := errors.New("service instance create failed by provider")
		return m.updateInstanceStatus(ctx, instID, model.UnavailableStatus, resultErr)
	}

	return nil
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
		log.Errorf(ctx, "update service instance status failed: %s", err)
	}

	return statusError
}

// TODO 后续补齐依赖服务相关的其他方法(如 ListServiceInstances 等), 目前方法供北极星使用即可
