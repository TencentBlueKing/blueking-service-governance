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

// Package serializer 定义组件定义 Gin API 的输入和输出结构。
package serializer

import (
	"time"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// EmptyOutput 是无数据接口的 JSON 响应。
type EmptyOutput struct{}

// ComponentDefURIInput 是组件定义 API 的路径参数。
type ComponentDefURIInput struct {
	// 组件定义名称
	CompDefName string `uri:"compDefName" binding:"required,uri_slug"`
}

// ListComponentDefsQueryInput 是组件定义列表的 query 参数。
type ListComponentDefsQueryInput struct {
	// 按可使用该组件定义的工作空间 ID 过滤
	ScopeWorkspaceID *string `form:"scopeWorkspaceID"`
	// 按可管理该组件定义的工作空间 ID 过滤
	ManagedByWorkspaceID *string `form:"managedByWorkspaceID"`
	// 搜索关键词
	Keyword *string `form:"keyword" binding:"omitempty,max=64"`
}

// ToModel 将 query 参数转换为列表查询选项。
func (i ListComponentDefsQueryInput) ToModel() *component.ListOptions {
	opts := &component.ListOptions{ExcludeInvisible: true}
	if i.ScopeWorkspaceID != nil {
		opts.ScopeWorkspaceID = *i.ScopeWorkspaceID
	}
	if i.ManagedByWorkspaceID != nil {
		opts.ManagedByWorkspaceID = *i.ManagedByWorkspaceID
	}
	if i.Keyword != nil {
		opts.Keyword = *i.Keyword
	}
	return opts
}

// CreateComponentDefInput 是创建组件定义的请求体。
type CreateComponentDefInput struct {
	// 组件名称
	CompDefName string `json:"compDefName" binding:"required,component_def_name"`
	// 组件展示名称
	DisplayName string `json:"displayName" binding:"max=64"`
	// 组件描述
	Description string `json:"description" binding:"max=512"`
	// 属性定义列表
	Properties []PropertyDefInput `json:"properties" binding:"dive"`
	// 根节点 YAML Patch 模板列表
	Patchers []string `json:"patchers" binding:"omitempty,dive,component_fragment"`
	// 额外 Kubernetes 资源 YAML 模板列表
	Specs []string `json:"specs" binding:"omitempty,dive,component_fragment"`
	// 生效范围类型: global / workspace
	ScopeType string `json:"scopeType" binding:"required,oneof=global workspace"`
	// 生效的工作空间列表
	ScopeWorkspaceIDs []string `json:"scopeWorkspaceIDs"`
	// 标记在哪些工作空间下可以管理该组件定义
	ManagedByWorkspaceIDs []string `json:"managedByWorkspaceIDs"`
}

// ToModel 将请求体转换为组件定义模型。
func (i CreateComponentDefInput) ToModel(userID string) *component.ComponentDef {
	return &component.ComponentDef{
		Name:        i.CompDefName,
		Version:     component.DefaultComponentDefVersion,
		DisplayName: i.DisplayName,
		Description: i.Description,
		Properties: lo.Map(i.Properties, func(p PropertyDefInput, _ int) component.Property {
			return p.ToModel()
		}),
		Patchers:              i.Patchers,
		Specs:                 i.Specs,
		ScopeType:             component.ScopeType(i.ScopeType),
		ScopeWorkspaceIDs:     i.ScopeWorkspaceIDs,
		IsBuiltin:             false,
		ManagedByWorkspaceIDs: i.ManagedByWorkspaceIDs,
		Creator:               userID,
		Updater:               userID,
	}
}

// PatchComponentDefInput 是更新组件定义的请求体。
type PatchComponentDefInput struct {
	// 组件展示名称
	DisplayName *string `json:"displayName" binding:"omitempty,max=64"`
	// 组件描述
	Description *string `json:"description" binding:"omitempty,max=512"`
	// 属性定义列表（传入时全量替换）
	PropertiesInput *ComponentDefPropertiesInput `json:"propertiesInput"`
	// 根节点 YAML Patch 模板列表（传入时全量替换）
	Patchers *[]string `json:"patchers" binding:"omitempty,dive,component_fragment"`
	// 额外 Kubernetes 资源 YAML 模板列表（传入时全量替换）
	Specs *[]string `json:"specs" binding:"omitempty,dive,component_fragment"`
	// 生效范围类型
	ScopeType *string `json:"scopeType" binding:"omitempty,oneof=global workspace"`
	// 生效的工作空间列表
	ScopeWorkspaceIDs []string `json:"scopeWorkspaceIDs"`
	// 标记在哪些工作空间下可以管理该组件定义
	ManagedByWorkspaceIDs []string `json:"managedByWorkspaceIDs"`
}

// ToModel 基于已有组件定义模型应用 patch 请求，返回新的组件定义模型。
func (i PatchComponentDefInput) ToModel(
	model *component.ComponentDef,
	userID string,
) *component.ComponentDef {
	next := *model
	if i.DisplayName != nil {
		next.DisplayName = *i.DisplayName
	}
	if i.Description != nil {
		next.Description = *i.Description
	}
	if i.PropertiesInput != nil {
		next.Properties = i.PropertiesInput.ToModel()
	}
	if i.Patchers != nil {
		next.Patchers = *i.Patchers
	}
	if i.Specs != nil {
		next.Specs = *i.Specs
	}
	if i.ScopeType != nil {
		next.ScopeType = component.ScopeType(*i.ScopeType)
	}
	if len(i.ScopeWorkspaceIDs) > 0 {
		next.ScopeWorkspaceIDs = i.ScopeWorkspaceIDs
	}
	if len(i.ManagedByWorkspaceIDs) > 0 {
		next.ManagedByWorkspaceIDs = i.ManagedByWorkspaceIDs
	}
	next.Updater = userID
	return &next
}

// ComponentDefPropertiesInput 包装 patch 时全量替换的属性定义列表。
type ComponentDefPropertiesInput struct {
	Properties []PropertyDefInput `json:"properties" binding:"dive"`
}

// ToModel 将请求结构转换为领域属性定义列表。
func (i *ComponentDefPropertiesInput) ToModel() []component.Property {
	if i == nil {
		return []component.Property{}
	}
	return lo.Map(i.Properties, func(p PropertyDefInput, _ int) component.Property {
		return p.ToModel()
	})
}

// PropertyDefInput 是单个组件属性定义的请求结构。
type PropertyDefInput struct {
	// 属性名称
	Name string `json:"name" binding:"required"`
	// 属性类型
	Type string `json:"type" binding:"required,oneof=STRING INT TEXT SELECT BOOL MAP"`
	// SELECT 类型可配置的候选项
	Options []PropertyOptionInput `json:"options" binding:"dive"`
	// 属性默认值
	DefaultValue any `json:"defaultValue"`
	// 属性描述
	Description string `json:"description" binding:"max=256"`
}

// ToModel 将请求结构转换为领域属性定义。
func (i PropertyDefInput) ToModel() component.Property {
	return component.Property{
		Name: i.Name,
		Type: component.PropType(i.Type),
		Options: lo.Map(i.Options, func(opt PropertyOptionInput, _ int) component.PropertyOption {
			return opt.ToModel()
		}),
		DefaultValue: i.DefaultValue,
		Description:  i.Description,
	}
}

// PropertyOptionInput 是 SELECT 属性单个选项的请求结构。
type PropertyOptionInput struct {
	Label string `json:"label" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// ToModel 将请求结构转换为领域属性选项。
func (i PropertyOptionInput) ToModel() component.PropertyOption {
	return component.PropertyOption{
		Label: i.Label,
		Value: i.Value,
	}
}

// ComponentDefOutputObj 是组件定义的响应对象。
type ComponentDefOutputObj struct {
	Name                       string                 `json:"name"`
	Version                    string                 `json:"version"`
	DisplayName                string                 `json:"displayName"`
	Description                string                 `json:"description"`
	Properties                 []PropertyDefOutputObj `json:"properties"`
	Patchers                   []string               `json:"patchers"`
	Specs                      []string               `json:"specs"`
	ScopeType                  string                 `json:"scopeType"`
	ScopeWorkspaceIDs          []string               `json:"scopeWorkspaceIDs"`
	IsBuiltin                  bool                   `json:"isBuiltin"`
	ManagedByWorkspaceIDs      []string               `json:"managedByWorkspaceIDs"`
	Creator                    string                 `json:"creator"`
	CreatedAt                  time.Time              `json:"createdAt"`
	Updater                    string                 `json:"updater"`
	UpdatedAt                  time.Time              `json:"updatedAt"`
	AppCompInstanceCount       int32                  `json:"appCompInstanceCount"`
	WorkspaceCompInstanceCount int32                  `json:"workspaceCompInstanceCount"`
}

// FromModel 将组件定义模型转换为响应对象。
func (o *ComponentDefOutputObj) FromModel(compDef *component.ComponentDef) *ComponentDefOutputObj {
	*o = ComponentDefOutputObj{
		Name:    compDef.Name,
		Version: compDef.Version,

		DisplayName: compDef.DisplayName,
		Description: compDef.Description,
		Properties: lo.Map(compDef.Properties, func(p component.Property, _ int) PropertyDefOutputObj {
			return *new(PropertyDefOutputObj).FromModel(p)
		}),
		Patchers:                   emptySliceIfNil(compDef.Patchers),
		Specs:                      emptySliceIfNil(compDef.Specs),
		ScopeType:                  string(compDef.ScopeType),
		ScopeWorkspaceIDs:          emptySliceIfNil(compDef.ScopeWorkspaceIDs),
		IsBuiltin:                  compDef.IsBuiltin,
		ManagedByWorkspaceIDs:      emptySliceIfNil(compDef.ManagedByWorkspaceIDs),
		Creator:                    compDef.Creator,
		CreatedAt:                  compDef.CreatedAt,
		Updater:                    compDef.Updater,
		UpdatedAt:                  compDef.UpdatedAt,
		AppCompInstanceCount:       compDef.AppCompInstanceCount,
		WorkspaceCompInstanceCount: compDef.WorkspaceCompInstanceCount,
	}
	return o
}

// ListComponentDefsOutput 是组件定义列表响应。
type ListComponentDefsOutput struct {
	Data []*ComponentDefOutputObj `json:"data"`
}

// BuiltinVarOutputObj 是组件模板系统变量的响应对象。
type BuiltinVarOutputObj struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

// FromModel 将领域层系统变量转换为响应对象。
func (o *BuiltinVarOutputObj) FromModel(
	item component.BuiltinVar,
) *BuiltinVarOutputObj {
	*o = BuiltinVarOutputObj{
		Key:         item.Key,
		Description: item.Description,
	}
	return o
}

// ListBuiltinVarsOutput 是组件模板系统变量列表响应。
type ListBuiltinVarsOutput struct {
	Data []*BuiltinVarOutputObj `json:"data"`
}

// FromModels 将领域层系统变量列表转换为响应对象。
func (o *ListBuiltinVarsOutput) FromModels(
	items []component.BuiltinVar,
) *ListBuiltinVarsOutput {
	o.Data = lo.Map(items, func(item component.BuiltinVar, _ int) *BuiltinVarOutputObj {
		return new(BuiltinVarOutputObj).FromModel(item)
	})
	return o
}

// PropertyDefOutputObj 是单个属性定义的响应对象。
type PropertyDefOutputObj struct {
	Name         string                    `json:"name"`
	Type         string                    `json:"type"`
	Options      []PropertyOptionOutputObj `json:"options"`
	DefaultValue any                       `json:"defaultValue,omitempty"`
	Description  string                    `json:"description"`
}

// FromModel 将领域属性定义转换为响应对象。
func (o *PropertyDefOutputObj) FromModel(p component.Property) *PropertyDefOutputObj {
	*o = PropertyDefOutputObj{
		Name: p.Name,
		Type: string(p.Type),
		Options: lo.Map(p.Options, func(opt component.PropertyOption, _ int) PropertyOptionOutputObj {
			return *new(PropertyOptionOutputObj).FromModel(opt)
		}),
		DefaultValue: p.NormalizedDefaultValue(),
		Description:  p.Description,
	}
	return o
}

// PropertyOptionOutputObj 是 SELECT 属性单个选项的响应对象。
type PropertyOptionOutputObj struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// FromModel 将领域属性选项转换为响应对象。
func (o *PropertyOptionOutputObj) FromModel(opt component.PropertyOption) *PropertyOptionOutputObj {
	*o = PropertyOptionOutputObj{
		Label: opt.Label,
		Value: opt.Value,
	}
	return o
}

func emptySliceIfNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
