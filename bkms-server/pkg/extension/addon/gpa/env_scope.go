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

package gpa

import (
	"github.com/samber/lo"
)

// ScopeType 生效范围类型
type ScopeType string

const (
	// ScopeTypeGlobal 全局生效
	ScopeTypeGlobal ScopeType = "global"
	// ScopeTypeEnvironment 环境生效
	ScopeTypeEnvironment ScopeType = "environment"
)

// EnvScope 描述一个配置/资源的「环境生效范围」，是一个可被内嵌（embed）到 gpa 实体中的
// 通用结构。
//
// 使用方式（bson 内嵌，字段与方法会被提升到外层结构）：
//
//	type SomeConfig struct {
//	    Name string `bson:"name"`
//	    EnvScope `bson:",inline"`
//	}
type EnvScope struct {
	// ScopeType 生效范围类型: global, environment
	ScopeType ScopeType `bson:"scopeType"`
	// ScopeEnvNames 生效的环境列表，当 ScopeType 为 environment 时有效
	ScopeEnvNames []string `bson:"scopeEnvNames"`
}

// IsAvailableInEnv 检查配置是否在指定环境中可用。
//
// 判断规则：
//   - global 或空值（未设置）: 对所有环境可用，返回 true
//   - environment: 仅当 envName 在 ScopeEnvNames 列表中时可用
//   - 其他未知类型: 为向后兼容，默认可用，返回 true
func (s EnvScope) IsAvailableInEnv(envName string) bool {
	switch s.ScopeType {
	case ScopeTypeEnvironment:
		// 仅对指定环境列表生效
		return lo.Contains(s.ScopeEnvNames, envName)
	case ScopeTypeGlobal, "":
		// 全局生效或未设置，对所有环境可用
		return true
	default:
		// 未知类型，默认可用（与历史行为保持一致）
		return true
	}
}
