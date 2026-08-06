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

package types

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
)

type ScopeType string

// 当前支持的环境变量作用域层级。
// 未来可能将其他作用域的环境变量迁移到这个模块中，比如单一环境。
const (
	// ScopeTypeEnv 单一环境级
	ScopeTypeEnv ScopeType = "env"
	// ScopeTypeEnvType 环境类型级
	ScopeTypeEnvType ScopeType = "envType"
	// ScopeTypeWorkspace 空间级
	ScopeTypeWorkspace ScopeType = "workspace"
)

// ScopedEnvVarScope identifies a scope by type and value.
type ScopedEnvVarScope struct {
	ScopeType  ScopeType
	ScopeValue string
}

func (s ScopedEnvVarScope) String() string {
	return fmt.Sprintf("%s:%s", s.ScopeType, s.ScopeValue)
}

// ScopeWorkspace is the predefined scope for workspace-level env vars.
var ScopeWorkspace = ScopedEnvVarScope{
	ScopeType:  ScopeTypeWorkspace,
	ScopeValue: "",
}

// ScopeEnvType creates a scope for env-type-level env vars with the given env type value.
func ScopeEnvType(envType string) ScopedEnvVarScope {
	return ScopedEnvVarScope{
		ScopeType:  ScopeTypeEnvType,
		ScopeValue: envType,
	}
}

// ScopeEnv creates a scope for env-level env vars with the given environment name.
func ScopeEnv(envName string) ScopedEnvVarScope {
	return ScopedEnvVarScope{
		ScopeType:  ScopeTypeEnv,
		ScopeValue: envName,
	}
}

// ParseScopedEnvVarScope parses the given scope type and value into a ScopedEnvVarScope.
func ParseScopedEnvVarScope(scopeType, scopeValue string) (ScopedEnvVarScope, error) {
	switch ScopeType(scopeType) {
	case ScopeTypeWorkspace:
		if scopeValue != "" {
			return ScopedEnvVarScope{}, errors.New("scopeValue must be empty for workspace scope")
		}
		return ScopeWorkspace, nil
	case ScopeTypeEnvType:
		if env.IsValidEnvType(scopeValue) {
			return ScopeEnvType(scopeValue), nil
		}
		return ScopedEnvVarScope{}, errors.New(
			"scopeValue must be one of development, test, staging or production for envType scope",
		)
	case ScopeTypeEnv:
		if scopeValue == "" {
			return ScopedEnvVarScope{}, errors.New("scopeValue is required for env scope")
		}
		return ScopeEnv(scopeValue), nil
	default:
		return ScopedEnvVarScope{}, errors.Errorf("unsupported scopeType %s", scopeType)
	}
}
