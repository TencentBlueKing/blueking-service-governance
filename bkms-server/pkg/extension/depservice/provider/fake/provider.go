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

// Package fake 提供 ServiceProvider 的测试替身实现。
//
// 注意：fake provider 被注册在 provider.New 工厂中，
// 仅用于测试环境，不应在生产流量中使用。
package fake

import (
	"context"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

// Provider 是 provider.ServiceProvider 的测试替身。
type Provider struct{}

// NewProvider 创建一个 fake Provider。
func NewProvider() *Provider {
	return &Provider{}
}

// CreateInstance implements provider.ServiceProvider.
func (p *Provider) CreateInstance(
	_ context.Context,
	_ *types.ServicePlanConfig,
	_ types.ProvisionParams,
) (*types.CreateInstanceResult, error) {
	return nil, nil
}

// QueryInstance implements provider.ServiceProvider.
func (p *Provider) QueryInstance(
	_ context.Context,
	_ *types.ServicePlanConfig,
	_ map[string]any,
) (*types.QueryInstanceResult, error) {
	return &types.QueryInstanceResult{Status: types.AvailableStatus}, nil
}

// DeleteInstance implements provider.ServiceProvider.
func (p *Provider) DeleteInstance(
	_ context.Context,
	_ *types.ServicePlanConfig,
	_ map[string]any,
) error {
	return nil
}
