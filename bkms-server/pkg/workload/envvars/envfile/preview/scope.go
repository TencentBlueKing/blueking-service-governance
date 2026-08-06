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

package preview

import (
	"strings"

	pkgerrors "github.com/pkg/errors"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	parserpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/parser"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// RecordResolution 承载单条记录的元数据解析结果。
type RecordResolution struct {
	// EffectiveScope 实际生效的作用域。
	EffectiveScope envvartypes.ScopedEnvVarScope
	// EffectStatus scope 指令的生效状态。
	EffectStatus ImportEffectScope
	// Messages 附加提示信息。
	Messages []string
}

// RecordResolver 定义记录解析策略的函数签名，不同导入上下文提供不同实现。
type RecordResolver func(record parserpkg.ParsedEnvVarRecord) (*RecordResolution, error)

// ResolvePublicRecord 公共导入场景只允许 workspace 与 envType 两类 scope。
func ResolvePublicRecord(record parserpkg.ParsedEnvVarRecord) (*RecordResolution, error) {
	if !record.ScopeTypeSpecified() {
		return nil, pkgerrors.New("scopeType is required in public import")
	}
	scope, err := parseDeclaredRecordScope(record)
	if err != nil {
		return nil, err
	}
	switch scope.ScopeType {
	case envvartypes.ScopeTypeWorkspace, envvartypes.ScopeTypeEnvType:
		return &RecordResolution{
			EffectiveScope: *scope,
			EffectStatus:   ImportEffectScopeApplied,
		}, nil
	default:
		return nil, pkgerrors.Errorf(
			"scopeType %q is not allowed in public import",
			recordScopeType(record),
		)
	}
}

// NewEnvRecordResolver 返回单环境导入场景的记录解析策略。
// 导入目标完全由当前页面上下文决定，因此文件中不允许声明任何 scope 元数据。
func NewEnvRecordResolver(environment envmodel.Environment) RecordResolver {
	scope := envvartypes.ScopeEnv(environment.Name)
	return func(record parserpkg.ParsedEnvVarRecord) (*RecordResolution, error) {
		if record.ScopeTypeSpecified() || record.ScopeValueSpecified() {
			return nil, pkgerrors.New("env import does not allow scope metadata")
		}
		return &RecordResolution{
			EffectiveScope: scope,
			EffectStatus:   ImportEffectScopeApplied,
		}, nil
	}
}

// ResolveAppRecord 应用导入场景不允许声明任何 scope 元数据。
func ResolveAppRecord(record parserpkg.ParsedEnvVarRecord) (*RecordResolution, error) {
	if record.ScopeTypeSpecified() || record.ScopeValueSpecified() {
		return nil, pkgerrors.New("app import does not allow scope metadata")
	}
	return &RecordResolution{
		EffectStatus: ImportEffectScopeNone,
	}, nil
}

// parseDeclaredRecordScope 只负责把输入里声明的原始 scope 字段转成结构化 scope。
// 返回指针是为了在调用方语义上明确区分“成功解析到一个 scope”和“解析失败无有效 scope”。
func parseDeclaredRecordScope(record parserpkg.ParsedEnvVarRecord) (*envvartypes.ScopedEnvVarScope, error) {
	scopeType := strings.TrimSpace(recordScopeType(record))
	scopeValue := strings.TrimSpace(recordScopeValue(record))

	if !record.ScopeTypeSpecified() {
		return nil, pkgerrors.New("scopeType is required")
	}
	if record.ScopeValueSpecified() &&
		strings.EqualFold(scopeType, string(envvartypes.ScopeTypeWorkspace)) &&
		scopeValue == "" {
		// 对 workspace scope，显式写出空 scopeValue 会被视为“声明了不该出现的字段”，
		// 这里和完全省略 scopeValue 做区分，便于把用户输入错误反馈清楚。
		return nil, pkgerrors.New("scopeValue must be omitted for workspace scope")
	}

	scope, err := envvartypes.ParseScopedEnvVarScope(scopeType, scopeValue)
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

func recordScopeType(record parserpkg.ParsedEnvVarRecord) string {
	if record.DeclaredScopeType == nil {
		return ""
	}
	return *record.DeclaredScopeType
}

func recordScopeValue(record parserpkg.ParsedEnvVarRecord) string {
	if record.DeclaredScopeValue == nil {
		return ""
	}
	return *record.DeclaredScopeValue
}
