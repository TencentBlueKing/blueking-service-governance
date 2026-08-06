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

// Package provider selects and constructs dependency service providers.
package provider

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/fake"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

// ServiceProvider 定义了服务实例提供者的接口。
//
// 服务实例提供者负责在底层服务平台（如 Polaris）上创建、查询和删除服务实例。
// 每个 ServiceProvider 实现对应一种特定的服务类型，例如 Polaris 服务。
type ServiceProvider interface {
	// CreateInstance 创建服务实例
	//
	// params 为各 provider 自定义的强类型参数，需实现 types.ProvisionParams 接口。
	// 例如 Polaris provider 接受 *polaris.CreateParams。
	CreateInstance(
		ctx context.Context,
		config *types.ServicePlanConfig,
		params types.ProvisionParams,
	) (*types.CreateInstanceResult, error)

	// QueryInstance 查询服务实例
	//
	// instConfig 为实例的持久化配置。其中 instConfig 取值自 ServiceInstance.Config
	QueryInstance(
		ctx context.Context,
		config *types.ServicePlanConfig,
		instConfig map[string]any,
	) (*types.QueryInstanceResult, error)

	// DeleteInstance 删除服务实例
	//
	// instConfig 为实例的持久化配置。其中 instConfig 取值自 ServiceInstance.Config
	DeleteInstance(
		ctx context.Context,
		config *types.ServicePlanConfig,
		instConfig map[string]any,
	) error
}

// New creates a new service provider
func New(serviceName string, plan *model.ServicePlan) (ServiceProvider, error) {
	switch serviceName {
	case "polaris":
		switch plan.ProviderType {
		case model.ProviderTypeSystemAllocated:
			return polaris.NewProvider(plan.Config)
		default:
			return nil, errors.Errorf("unknown providerType: %s for service(name:%s)", plan.ProviderType, serviceName)
		}
	// fake 仅用于测试，不应在生产流量中使用。
	case "fake":
		return fake.NewProvider(), nil
	default:
		return nil, errors.Errorf("provider not found for service(name:%s)", serviceName)
	}
}
