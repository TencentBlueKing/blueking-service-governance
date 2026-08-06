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
// 注意：fake provider 被注册在 provider.New 工厂中（serviceName == "fake"），
// 仅用于测试环境，不应在生产流量中使用。
package fake

import (
	"context"
	"sync"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

// Provider 是 provider.ServiceProvider 的测试替身。
// 可通过字段控制 Create/Delete 的返回行为。
type Provider struct {
	// CreateAsync 为 true 时 CreateInstance 返回 Async=true
	CreateAsync bool
	// CreateErr 非空时 CreateInstance 直接返回该错误
	CreateErr error
	// CreateResult 非空时作为 CreateInstance 的成功返回值（优先级高于 CreateAsync）
	CreateResult *types.CreateInstanceResult

	// DeleteAsync 为 true 时 DeleteInstance 返回 Async=true
	DeleteAsync bool
	// DeleteErr 非空时 DeleteInstance 直接返回该错误
	DeleteErr error
}

var (
	factoryMu sync.Mutex
	// nextProvider 若非空，NewProvider 将返回该实例（一次性），供测试定制行为。
	nextProvider *Provider
)

// Use 指定下一次 NewProvider（即 provider.New("fake", ...)）返回的实例。
// 调用方应在用例结束时 Reset，避免泄漏到其他测试。
func Use(p *Provider) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	nextProvider = p
}

// Reset 清除 Use 设置的定制实例。
func Reset() {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	nextProvider = nil
}

// NewProvider 创建一个 fake Provider。
// 若此前调用过 Use，则返回该定制实例并清空；否则返回默认同步成功行为。
func NewProvider() *Provider {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	if nextProvider != nil {
		p := nextProvider
		nextProvider = nil
		return p
	}
	return &Provider{}
}

// CreateInstance implements provider.ServiceProvider.
func (p *Provider) CreateInstance(
	_ context.Context,
	_ string,
	_ *types.ServicePlanConfig,
	_ types.ProvisionParams,
) (*types.CreateInstanceResult, error) {
	if p.CreateErr != nil {
		return nil, p.CreateErr
	}
	if p.CreateResult != nil {
		return p.CreateResult, nil
	}
	return &types.CreateInstanceResult{Async: p.CreateAsync}, nil
}

// DeleteInstance implements provider.ServiceProvider.
func (p *Provider) DeleteInstance(
	_ context.Context,
	_ string,
	_ *types.ServicePlanConfig,
	_ map[string]any,
) (*types.DeleteInstanceResult, error) {
	if p.DeleteErr != nil {
		return nil, p.DeleteErr
	}
	return &types.DeleteInstanceResult{Async: p.DeleteAsync}, nil
}
