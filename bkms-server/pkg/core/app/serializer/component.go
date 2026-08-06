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

package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// AppComponentURIInput 是应用组件 API 的路径参数。
type AppComponentURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 组件名称
	CompName string `uri:"compName" binding:"required,uri_slug"`
}

// CreateAppComponentInput 是创建应用组件的请求体。
type CreateAppComponentInput struct {
	// 组件名称, 用户不传时后端自动生成
	CompName *string `json:"compName" binding:"omitempty,component_name"`
	// 组件类型，即组件在市场中的名字，等同于 ComponentDef 的 name
	Type string `json:"type"`
	// 组件版本
	Version *string `json:"version"`
	// 组件属性
	Properties map[string]any `json:"properties"`
	// 引用的空间组件名称，传入时将忽略 type/version/properties 字段
	RefWorkspaceCompName *string `json:"refWorkspaceCompName"`
}

// ToModel 将请求体转换为应用组件模型，不做查询或上下文相关校验。
func (i CreateAppComponentInput) ToModel() *component.Component {
	model := &component.Component{}
	if i.CompName != nil && *i.CompName != "" {
		model.Name = *i.CompName
	}
	if i.RefWorkspaceCompName != nil && *i.RefWorkspaceCompName != "" {
		model.ComponentRef = component.ComponentRef{RefWorkspaceCompName: *i.RefWorkspaceCompName}
		return model
	}

	version := component.DefaultComponentDefVersion
	if i.Version != nil && *i.Version != "" {
		version = *i.Version
	}
	model.ComponentInst = component.ComponentInst{
		Type:       i.Type,
		Version:    version,
		Properties: i.Properties,
	}
	return model
}

// PatchAppComponentInput 是更新应用组件的请求体。
type PatchAppComponentInput struct {
	// 修改组件名称
	Name *string `json:"name" binding:"omitempty,component_name"`
	// 组件属性
	Properties map[string]any `json:"properties"`
}

// ToModel 将请求体转换为应用组件更新数据。
func (i PatchAppComponentInput) ToModel() *appmodel.ComponentUpdateData {
	return &appmodel.ComponentUpdateData{
		Name:       i.Name,
		Properties: i.Properties,
	}
}

// AppComponentNameOutputObj 是包含组件名称的响应对象。
type AppComponentNameOutputObj struct {
	Name string `json:"name"`
}

// CreateAppComponentOutput 是创建应用组件的响应。
type CreateAppComponentOutput struct {
	Data *AppComponentNameOutputObj `json:"data"`
}
