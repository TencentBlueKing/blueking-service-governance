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

// Package serializer defines Gin input and output serializers for workspace APIs.
package serializer

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// WorkspaceComponentURIInput 是工作空间组件 API 的路径参数。
type WorkspaceComponentURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 组件名称
	CompName string `uri:"compName" binding:"required,uri_slug"`
}

// CreateWorkspaceComponentInput 是创建工作空间组件的请求体。
type CreateWorkspaceComponentInput struct {
	// 组件名称, 用户不传时后端自动生成
	CompName *string `json:"compName" binding:"omitempty,component_name"`
	// 组件类型，即组件在市场中的名字，等同于 ComponentDef 的 name
	Type string `json:"type" binding:"required"`
	// 组件版本
	Version *string `json:"version"`
	// 组件属性
	Properties map[string]any `json:"properties"`
	// 组件生效范围类型: global 或 environment
	ScopeType string `json:"scopeType" binding:"required,oneof=global environment"`
	// 组件生效的环境列表，当 scopeType 为 environment 时有效
	ScopeEnvNames []string `json:"scopeEnvNames"`
}

// ToModel 将请求体转换为工作空间组件模型，不做查询或上下文相关校验。
func (i CreateWorkspaceComponentInput) ToModel(workspaceID string) *workspace.Component {
	version := component.DefaultComponentDefVersion
	if i.Version != nil {
		version = *i.Version
	}

	model := &workspace.Component{
		ComponentInst: component.ComponentInst{
			Type:       i.Type,
			Version:    version,
			Properties: i.Properties,
		},
		WorkspaceID:   workspaceID,
		ScopeType:     component.ScopeType(i.ScopeType),
		ScopeEnvNames: i.ScopeEnvNames,
	}
	if i.CompName != nil && *i.CompName != "" {
		model.Name = *i.CompName
	}
	return model
}

// PatchWorkspaceComponentInput 是更新工作空间组件的请求体。
type PatchWorkspaceComponentInput struct {
	// 修改组件名称
	Name *string `json:"name" binding:"omitempty,component_name"`
	// 组件属性
	Properties map[string]any `json:"properties"`
	// 组件生效范围类型
	ScopeType *string `json:"scopeType" binding:"omitempty,oneof=global environment"`
	// 组件生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames"`
}

// ToModel 将请求体转换为工作空间组件更新数据，不做查询或上下文相关校验。
func (i PatchWorkspaceComponentInput) ToModel() *workspace.ComponentUpdateData {
	updateData := &workspace.ComponentUpdateData{
		Properties:    i.Properties,
		Name:          i.Name,
		ScopeEnvNames: i.ScopeEnvNames,
	}
	if i.ScopeType != nil {
		scopeType := component.ScopeType(*i.ScopeType)
		updateData.ScopeType = &scopeType
	}
	return updateData
}

// WorkspaceComponentNameOutputObj 是包含组件名称的响应对象。
type WorkspaceComponentNameOutputObj struct {
	Name string `json:"name"`
}

// CreateWorkspaceComponentOutput 是创建工作空间组件的响应。
type CreateWorkspaceComponentOutput struct {
	Data *WorkspaceComponentNameOutputObj `json:"data"`
}

// WorkspaceComponentOutputObj 是工作空间组件的响应对象。
type WorkspaceComponentOutputObj struct {
	// 组件名称
	Name string `json:"name"`
	// 所属工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 组件类型
	Type string `json:"type"`
	// 组件版本
	Version string `json:"version"`
	// 组件属性
	Properties map[string]any `json:"properties"`
	// 组件生效范围类型
	ScopeType string `json:"scopeType"`
	// 组件生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames"`
	// 标记哪些应用引用了该空间组件
	RefAppIDs []string `json:"refAppIDs"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel 将工作空间组件模型转换为响应对象。
func (o *WorkspaceComponentOutputObj) FromModel(
	comp *workspace.Component,
	refAppIDs []string,
) *WorkspaceComponentOutputObj {
	*o = WorkspaceComponentOutputObj{
		Name:          comp.Name,
		WorkspaceID:   comp.WorkspaceID,
		Type:          comp.Type,
		Version:       comp.Version,
		Properties:    comp.Properties,
		ScopeType:     string(comp.ScopeType),
		ScopeEnvNames: emptySliceIfNil(comp.ScopeEnvNames),
		RefAppIDs:     emptySliceIfNil(refAppIDs),
		CreatedAt:     comp.CreatedAt,
		UpdatedAt:     comp.UpdatedAt,
	}
	return o
}

// ListWorkspaceComponentsOutput 是工作空间组件列表响应。
type ListWorkspaceComponentsOutput struct {
	Data []*WorkspaceComponentOutputObj `json:"data"`
}

func emptySliceIfNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
