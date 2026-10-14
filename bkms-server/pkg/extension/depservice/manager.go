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
	"fmt"
	"slices"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// use a single instance of Validate, it caches struct info
var validate = validator.New(validator.WithRequiredStructEnabled())

// ServiceManager is the manager for app dependency services
type ServiceManager struct {
	db       *mongo.Database
	services *model.ServiceStore
	insts    *model.ServiceInstanceStore
	bindings *model.ServiceBindingStore
	envs     *envmodel.EnvironmentStore
}

// New creates a new app dependency services manager.
func New(db *mongo.Database) *ServiceManager {
	return &ServiceManager{
		db:       db,
		services: store.Of(db, model.ServiceMeta),
		insts:    store.Of(db, model.ServiceInstanceMeta),
		bindings: store.Of(db, model.ServiceBindingMeta),
		envs:     store.Of(db, envmodel.EnvMeta),
	}
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

	inst := &model.ServiceInstance{
		Name:         params.Name,
		ServiceName:  params.ServiceName,
		PlanName:     params.PlanName,
		ProviderType: plan.ProviderType,
		ScopeType:    params.ScopeType,
		ScopeValue:   params.ScopeValue,
		WorkspaceID:  params.WorkspaceID,
		Operator:     params.Operator,
		Description:  params.Description,
		Status:       model.ProvisioningStatus,
	}
	if err = m.insts.Create(ctx, inst); err != nil {
		if errors.Is(err, model.ErrInstanceNameExists) {
			return bson.NilObjectID, errors.Wrapf(
				err,
				"service instance %q already exists for service %s in workspace %s",
				params.Name, params.ServiceName, params.WorkspaceID,
			)
		}
		return bson.NilObjectID, errors.Wrap(err, "init service instance to db")
	}
	instID := inst.ID

	createResult, err := svcProvider.CreateInstance(ctx, instID.Hex(), &types.ServicePlanConfig{
		Config: plan.Config,
	}, params.Params)
	if err != nil {
		createErr := errors.Wrap(err, "create service instance by provider")
		return instID, m.updateInstanceStatus(ctx, instID, model.CreateFailedStatus, createErr)
	}

	// 异步 provider 已投递 task，实例保持 provisioning，Config 由后续 task 回写
	if createResult.Async {
		return instID, nil
	}

	// 同步 provider 直接更新配置和凭证
	if len(createResult.InstConfig) > 0 {
		err = m.insts.Update(ctx, map[string]any{"config": createResult.InstConfig}, store.ByID(instID))
		if err != nil {
			updateErr := errors.Wrap(err, "update service instance config")
			return instID, m.updateInstanceStatus(ctx, instID, model.UnavailableStatus, updateErr)
		}
	}
	if len(createResult.Credentials) > 0 {
		err = m.insts.Update(ctx, map[string]any{"credentials": createResult.Credentials}, store.ByID(instID))
		if err != nil {
			updateErr := errors.Wrap(err, "update service instance credentials")
			return instID, m.updateInstanceStatus(ctx, instID, model.UnavailableStatus, updateErr)
		}
	}

	return instID, m.updateInstanceStatus(ctx, instID, model.AvailableStatus, nil)
}

// UpdateServiceInstance 通过对应 provider 更新服务实例。
func (m *ServiceManager) UpdateServiceInstance(
	ctx context.Context,
	instID bson.ObjectID,
	params types.ProvisionParams,
) error {
	inst, err := m.insts.Get(ctx, store.ByID(instID))
	if err != nil {
		return errors.Wrap(err, "get service instance")
	}

	plan, svcProvider, err := m.getPlanAndProvider(ctx, inst.ServiceName, inst.PlanName)
	if err != nil {
		return err
	}

	updater, ok := svcProvider.(provider.InstanceUpdater)
	if !ok {
		return errors.Wrapf(ErrUpdateNotSupported, "service %s", inst.ServiceName)
	}

	err = updater.UpdateInstance(ctx, instID.Hex(), &types.ServicePlanConfig{
		Config: plan.Config,
	}, inst.Config, params)
	if err != nil {
		return errors.Wrap(err, "update service instance by provider")
	}
	return nil
}

// getPlanAndProvider 按服务名与 plan 名解析 plan 并构造对应 provider。
func (m *ServiceManager) getPlanAndProvider(
	ctx context.Context,
	serviceName, planName string,
) (*model.ServicePlan, provider.ServiceProvider, error) {
	svc, err := m.services.Get(ctx, model.ByName(serviceName))
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
	return m.insts.Get(ctx, store.ByID(instID))
}

// ListServiceInstances 按条件列出服务实例。
func (m *ServiceManager) ListServiceInstances(
	ctx context.Context,
	params *ListServiceInstancesParams,
) ([]*model.ServiceInstance, error) {
	if params == nil {
		params = &ListServiceInstancesParams{}
	}

	query := &model.SvcInstQueryOptions{
		WorkspaceID: params.WorkspaceID,
		ServiceName: params.ServiceName,
		Status:      params.Status,
		ScopeType:   params.ScopeType,
	}
	insts, err := m.insts.List(ctx, query.Cond(), model.RecentlyUpdatedFirst())
	if err != nil {
		return nil, errors.Wrap(err, "list service instances")
	}
	return lo.ToSlicePtr(insts), nil
}

// DeleteServiceInstance 删除服务实例. 用于需要销毁实例的场景.
// NOTE: 如果实例被绑定引用, 则无法删除
func (m *ServiceManager) DeleteServiceInstance(ctx context.Context, instID bson.ObjectID) error {
	inst, err := m.insts.Get(ctx, store.ByID(instID))
	if err != nil {
		return errors.Wrap(err, "get service instance")
	}

	bindings, err := m.listBindingsByInstance(ctx, instID)
	if err != nil {
		return errors.Wrap(err, "list bindings by instance")
	}
	if len(bindings) != 0 {
		names := make([]string, 0, len(bindings))
		for _, b := range bindings {
			names = append(names, fmt.Sprintf("%s/%s", b.AppID, b.Name))
		}
		return errors.Wrapf(ErrInvalidArgument, "service instance is used by bindings(%v), cannot delete", names)
	}

	if inst.Status == model.ProvisioningStatus {
		return errors.Wrap(ErrInvalidArgument, "service instance is provisioning, cannot delete")
	}
	if inst.Status == model.DeletingStatus {
		return errors.Wrap(ErrInvalidArgument, "service instance is deleting, cannot delete")
	}

	if inst.Status == model.CreateFailedStatus {
		return m.insts.DeleteIfExists(ctx, store.ByID(instID))
	}

	if !isDeletableStatus(inst.Status) {
		return errors.Wrapf(
			ErrInvalidArgument,
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

	return m.insts.DeleteIfExists(ctx, store.ByID(instID))
}

// isDeletableStatus 判断是否允许走 provider 删除路径的稳定态。
func isDeletableStatus(status model.InstanceStatus) bool {
	return slices.Contains([]model.InstanceStatus{
		model.AvailableStatus,
		model.UnavailableStatus,
		model.DeleteFailedStatus,
	}, status)
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

	if err := model.UpdateInstanceStatus(ctx, m.db, instID, status, message); err != nil {
		// 有业务错误时优先返回业务错误
		log.Errorf(ctx, "update service instance status failed: %s", err)
		if statusError != nil {
			return statusError
		}
		return errors.Wrap(err, "update service instance status")
	}

	return statusError
}
