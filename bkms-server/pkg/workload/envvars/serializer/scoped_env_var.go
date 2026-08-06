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

// Package serializer defines Gin input and output serializers for envvars APIs.
package serializer

import (
	"time"

	"github.com/samber/lo"

	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// -----------------------------------------------------------------------------
// Path inputs
//
// These serializers only bind Gin URI parameters. Resource existence and
// permission checks stay in handler / perm helpers because they need registry and
// request context.
// -----------------------------------------------------------------------------

// WorkspaceURIInput is the path input for APIs scoped by workspace.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
}

// ScopedEnvVarURIInput is the path input for APIs scoped by workspace and scoped env var.
type ScopedEnvVarURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
	// Scoped EnvVar ID
	ScopedEnvVarID string `uri:"scopedEnvVarID" binding:"required,mongodb"`
}

// EnvURIInput is the path input for APIs scoped by environment.
type EnvURIInput struct {
	// 环境 ID
	EnvID string `uri:"envID" binding:"required,mongodb"`
}

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// AppEnvVarURIInput is the path input for APIs scoped by application and env var key.
type AppEnvVarURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 环境变量 Key
	Key string `uri:"key" binding:"required,max=256,envvar_key"`
}

// -----------------------------------------------------------------------------
// Shared outputs
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// ScopedEnvVarOutputObj is the JSON representation of a persisted scoped env var.
//
// It is used by create/update/list-public APIs. Unlike app model env vars, this
// object includes persistence fields, scope metadata, and sensitivity metadata.
type ScopedEnvVarOutputObj struct {
	// 环境变量 ID
	ID string `json:"id"`
	// 工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 作用域类型，目前支持 workspace、envType、env
	ScopeType string `json:"scopeType"`
	// 作用域值
	// - 当 scopeType 为 workspace 时，固定为空字符串
	// - 当 scopeType 为 envType 时，可选值为 development、test、staging、production
	// - 当 scopeType 为 env 时，值为具体环境名称
	ScopeValue string `json:"scopeValue"`
	// 环境变量 Key
	Key string `json:"key"`
	// 环境变量值
	Value string `json:"value"`
	// 描述
	Description string `json:"description"`
	// 是否敏感
	IsSensitive bool `json:"isSensitive"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from a scoped env var model.
func (o *ScopedEnvVarOutputObj) FromModel(obj envvars.ScopedEnvVar) *ScopedEnvVarOutputObj {
	value := obj.Value
	if obj.IsSensitive {
		value = envvartypes.SensitiveValueMask
	}

	*o = ScopedEnvVarOutputObj{
		ID:          obj.ID.Hex(),
		WorkspaceID: obj.WorkspaceID,
		ScopeType:   string(obj.ScopeType),
		ScopeValue:  obj.ScopeValue,
		Key:         obj.Key,
		Value:       value,
		Description: obj.Description,
		IsSensitive: obj.IsSensitive,
		CreatedAt:   obj.CreatedAt,
		UpdatedAt:   obj.UpdatedAt,
	}
	return o
}

// EnvVarConflictedSourceOutputObj is the JSON representation of a conflict source.
type EnvVarConflictedSourceOutputObj struct {
	// 冲突来源
	Source string `json:"source"`
	// 冲突来源值
	SourceValue string `json:"sourceValue"`
}

// EnvVarConflictedInfoOutputObj is the JSON representation of env var conflict info.
//
// It is only emitted by detailed-list APIs. A nil value means the env var has no
// conflicts and should be omitted from JSON by the containing serializer.
type EnvVarConflictedInfoOutputObj struct {
	// 冲突来源列表
	ConflictedSources []*EnvVarConflictedSourceOutputObj `json:"conflictedSources"`
	// 当前变量是否覆盖冲突变量并生效
	OverrideConflicted bool `json:"overrideConflicted"`
	// 冲突详情
	ConflictedDetail string `json:"conflictedDetail"`
}

// NewEnvVarConflictedInfoOutputObj builds output fields from env var conflict info.
func NewEnvVarConflictedInfoOutputObj(info envvartypes.EnvVarConflictedInfo) *EnvVarConflictedInfoOutputObj {
	// 当不存在任何冲突信息时，返回 nil
	if len(info.ConflictedSources) == 0 {
		return nil
	}

	return &EnvVarConflictedInfoOutputObj{
		ConflictedSources: lo.Map(
			info.ConflictedSources,
			func(source envvartypes.ConflictedSource, _ int) *EnvVarConflictedSourceOutputObj {
				return &EnvVarConflictedSourceOutputObj{
					Source:      string(source.Source),
					SourceValue: source.SourceValue,
				}
			},
		),
		OverrideConflicted: info.OverrideConflicted,
		ConflictedDetail:   info.ConflictedDetail,
	}
}

// -----------------------------------------------------------------------------
// Create / update / delete scoped env var API serializers
//
// CreateScopedEnvVarInput chooses a target scope. UpdateScopedEnvVarInput does
// not include scope fields because scope is immutable; it also uses *bool for
// isSensitive so an omitted field can preserve the stored value.
// -----------------------------------------------------------------------------

// CreateScopedEnvVarInput is the JSON body for creating a scoped env var.
type CreateScopedEnvVarInput struct {
	// 作用域类型，目前支持 workspace、envType、env
	ScopeType string `json:"scopeType" binding:"required,oneof=workspace envType env"`
	// 作用域值
	ScopeValue string `json:"scopeValue"`
	// 环境变量 Key
	Key string `json:"key" binding:"required,envvar_key"`
	// 环境变量值，允许为空
	Value string `json:"value" binding:"envvar_value"`
	// 描述
	Description string `json:"description"`
	// 是否敏感
	IsSensitive bool `json:"isSensitive"`
}

// CreateScopedEnvVarOutput is the JSON response for creating a scoped env var.
type CreateScopedEnvVarOutput struct {
	// 作用域级别环境变量
	Data *ScopedEnvVarOutputObj `json:"data"`
}

// UpdateScopedEnvVarInput is the JSON body for updating a scoped env var.
type UpdateScopedEnvVarInput struct {
	// 环境变量 Key
	Key string `json:"key" binding:"required,envvar_key"`
	// 环境变量值，未传时保持原值，允许显式传空字符串
	Value *string `json:"value" binding:"omitempty,envvar_value"`
	// 描述
	Description string `json:"description"`
	// 是否敏感，未传时保持原值不变
	IsSensitive *bool `json:"isSensitive"`
}

// UpdateScopedEnvVarOutput is the JSON response for updating a scoped env var.
type UpdateScopedEnvVarOutput struct {
	// 作用域级别环境变量
	Data *ScopedEnvVarOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// List public scoped env vars API serializers
//
// Public scoped env vars are persisted scoped env vars whose scope is workspace
// or envType. The response intentionally uses the basic scoped env var object and
// does not include conflict fields.
// -----------------------------------------------------------------------------

// ListPublicScopedEnvVarsOutput is the JSON response for listing public scoped env vars.
type ListPublicScopedEnvVarsOutput struct {
	// 作用域为 workspace 和 envType 的环境变量列表
	Data []*ScopedEnvVarOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// Detailed env/app env var list API serializers
//
// ScopedEnvVarDetailedOutputObj wraps a persisted scoped env var. AppEnvVarDetailedOutputObj
// wraps an app model variable, which only has key/value/description and therefore
// intentionally does not expose id, scope, timestamps, or isSensitive.
// -----------------------------------------------------------------------------

// ScopedEnvVarDetailedOutputObj is the JSON representation of a detailed scoped env var.
type ScopedEnvVarDetailedOutputObj struct {
	// 环境变量基础信息
	ScopedEnvVar *ScopedEnvVarOutputObj `json:"scopedEnvVar"`
	// 冲突信息
	ConflictedInfo *EnvVarConflictedInfoOutputObj `json:"conflictedInfo,omitempty"`
}

// FromModel fills output fields from a scoped env var and its conflict info.
func (o *ScopedEnvVarDetailedOutputObj) FromModel(
	obj envvars.ScopedEnvVar,
	conflictedInfo envvartypes.EnvVarConflictedInfo,
) *ScopedEnvVarDetailedOutputObj {
	*o = ScopedEnvVarDetailedOutputObj{
		ScopedEnvVar:   new(ScopedEnvVarOutputObj).FromModel(obj),
		ConflictedInfo: NewEnvVarConflictedInfoOutputObj(conflictedInfo),
	}
	return o
}

// ListDetailedEnvScopedEnvVarsOutput is the JSON response for listing detailed env scoped env vars.
type ListDetailedEnvScopedEnvVarsOutput struct {
	// 作用域为当前环境的环境变量详情列表
	Data []*ScopedEnvVarDetailedOutputObj `json:"data"`
}

// DetailedAppEnvVarOutputObj is the JSON representation of an app env var.
type DetailedAppEnvVarOutputObj struct {
	// 环境变量 Key
	Key string `json:"key"`
	// 环境变量值
	Value string `json:"value"`
	// 描述
	Description string `json:"description"`
	// 是否敏感
	IsSensitive bool `json:"isSensitive"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from an app model variable.
func (o *DetailedAppEnvVarOutputObj) FromModel(item appmodel.Variable) *DetailedAppEnvVarOutputObj {
	value := item.Value
	if item.IsSensitive {
		value = envvartypes.SensitiveValueMask
	}
	*o = DetailedAppEnvVarOutputObj{
		Key:         item.Key,
		Value:       value,
		Description: item.Description,
		IsSensitive: item.IsSensitive,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
	return o
}

// AppEnvVarDetailedOutputObj is the JSON representation of a detailed app env var.
type AppEnvVarDetailedOutputObj struct {
	// 应用环境变量基础信息
	AppEnvVar *DetailedAppEnvVarOutputObj `json:"appEnvVar"`
	// 冲突信息
	ConflictedInfo *EnvVarConflictedInfoOutputObj `json:"conflictedInfo,omitempty"`
}

// FromModel fills output fields from an app env var and its conflict info.
func (o *AppEnvVarDetailedOutputObj) FromModel(
	item appmodel.Variable,
	conflictedInfo envvartypes.EnvVarConflictedInfo,
) *AppEnvVarDetailedOutputObj {
	*o = AppEnvVarDetailedOutputObj{
		AppEnvVar:      new(DetailedAppEnvVarOutputObj).FromModel(item),
		ConflictedInfo: NewEnvVarConflictedInfoOutputObj(conflictedInfo),
	}
	return o
}

// ListDetailedAppEnvVarsOutput is the JSON response for listing detailed app env vars.
type ListDetailedAppEnvVarsOutput struct {
	// 应用环境变量详情列表
	Data []*AppEnvVarDetailedOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// App-defined env var CRUD API serializers
// -----------------------------------------------------------------------------

// CreateAppDefinedEnvVarInput is the JSON body for creating an app-defined env var.
type CreateAppDefinedEnvVarInput struct {
	// 环境变量 Key
	Key string `json:"key" binding:"required,envvar_key"`
	// 环境变量值，允许为空
	Value string `json:"value" binding:"envvar_value"`
	// 描述
	Description string `json:"description"`
	// 是否敏感
	IsSensitive bool `json:"isSensitive"`
}

// CreateAppDefinedEnvVarOutput is the JSON response for creating an app-defined env var.
type CreateAppDefinedEnvVarOutput struct {
	// 应用直接定义的环境变量
	Data *AppDefinedEnvVarOutputObj `json:"data"`
}

// UpdateAppDefinedEnvVarInput is the JSON body for updating an app-defined env var.
type UpdateAppDefinedEnvVarInput struct {
	// 更新后的环境变量 Key
	UpdatedKey string `json:"updatedKey" binding:"required,envvar_key"`
	// 环境变量值，未传时保持原值，允许显式传空字符串
	Value *string `json:"value" binding:"omitempty,envvar_value"`
	// 描述
	Description string `json:"description"`
	// 是否敏感，未传时保持原值不变
	IsSensitive *bool `json:"isSensitive"`
}

// UpdateAppDefinedEnvVarOutput is the JSON response for updating an app-defined env var.
type UpdateAppDefinedEnvVarOutput struct {
	// 应用直接定义的环境变量
	Data *AppDefinedEnvVarOutputObj `json:"data"`
}

// AppDefinedEnvVarOutputObj is the JSON representation of an app-defined env var.
type AppDefinedEnvVarOutputObj struct {
	// 环境变量 Key
	Key string `json:"key"`
	// 环境变量值
	Value string `json:"value"`
	// 描述
	Description string `json:"description"`
	// 是否敏感
	IsSensitive bool `json:"isSensitive"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from an app model variable.
func (o *AppDefinedEnvVarOutputObj) FromModel(item appmodel.Variable) *AppDefinedEnvVarOutputObj {
	value := item.Value
	if item.IsSensitive {
		value = envvartypes.SensitiveValueMask
	}
	*o = AppDefinedEnvVarOutputObj{
		Key:         item.Key,
		Value:       value,
		Description: item.Description,
		IsSensitive: item.IsSensitive,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
	return o
}

// ListAppDefinedEnvVarsOutput is the JSON response for listing app-defined env vars.
type ListAppDefinedEnvVarsOutput struct {
	// 应用直接定义的环境变量列表
	Data []*AppDefinedEnvVarOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// List environment available env vars API serializers
//
// EnvVarOutputObj represents the effective variable set for an environment.
// It is not necessarily backed by a scoped env var row; values may come from
// builtin variables, public scoped vars, or env-scoped vars after deduplication.
// -----------------------------------------------------------------------------

// ListEnvAvailableEnvVarsOutput is the JSON response for listing available env vars.
type ListEnvAvailableEnvVarsOutput struct {
	// 环境下所有可用的环境变量列表
	Data []*EnvVarOutputObj `json:"data"`
}
