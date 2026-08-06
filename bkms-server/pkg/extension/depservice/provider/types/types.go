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

// Package types defines shared dependency service provider types and conversion helpers.
package types

import (
	"github.com/mitchellh/mapstructure"
)

// ProvisionParams 是所有 provider 创建参数的公共接口。
// 各 provider 定义自己的强类型结构体并实现此接口。
type ProvisionParams interface {
	Validate() error
}

// ToMap 将强类型实例配置/凭证序列化为 map 以持久化到 db。
// 与 ParseInstConfig 互为反向操作。
func ToMap(v any) (map[string]any, error) {
	var m map[string]any
	if err := mapstructure.Decode(v, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ParseInstConfig 从 map 中反序列化为强类型实例配置
func ParseInstConfig[T any](raw map[string]any) (*T, error) {
	cfg := new(T)
	if err := mapstructure.Decode(raw, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ServicePlanConfig represents the config of service plan
type ServicePlanConfig struct {
	Config map[string]any
}

// CreateInstanceResult represents the result of create instance
type CreateInstanceResult struct {
	// InstConfig represents the instance config
	InstConfig map[string]any
	// Credentials represents the instance credentials
	Credentials map[string]any

	// Async 为 true 表示创建请求已提交但尚未完成，
	// 实例应保持 provisioning 状态，后续由异步 task 推进完成。
	Async bool
}

// DeleteInstanceResult represents the result of delete instance
type DeleteInstanceResult struct {
	// Async 为 true 表示删除请求已提交但尚未完成，
	// 实例应保持 deleting 状态，后续由异步 task 推进完成。
	Async bool
}
