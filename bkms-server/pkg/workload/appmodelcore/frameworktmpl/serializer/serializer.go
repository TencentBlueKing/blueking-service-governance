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

// Package serializer defines Gin serializers for workspace framework config templates.
package serializer

import (
	"time"

	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/frameworktmpl"
)

// WorkspaceURIInput binds a workspace-scoped path.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// TemplateURIInput binds a template path.
type TemplateURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 模板 ID，内置模板为 builtin-{framework}-{language}
	TemplateID string `uri:"templateID" binding:"required"`
}

// ListTemplatesQueryInput filters the template list.
type ListTemplatesQueryInput struct {
	// 框架：trpc、taf，不填返回全部
	Framework string `form:"framework"`
	// 语言：go、cpp，不填返回全部
	Language string `form:"language"`
}

// CreateTemplateInput is the body for creating a template.
type CreateTemplateInput struct {
	// 框架：trpc、taf
	Framework string `json:"framework" binding:"required"`
	// 语言：go、cpp。框架只支持一种语言时（如 TAF）可以不填
	Language string `json:"language"`
	// 模板名称，同一框架、同一语言下不能重复，不能为「平台默认」
	Name string `json:"name" binding:"required"`
	// 框架配置文件名
	FileName string `json:"fileName" binding:"required"`
	// 框架配置文件内容，最大 64KB
	Content string `json:"content" binding:"required"`
	// 是否设为创建应用时默认使用的模板
	IsDefault bool `json:"isDefault"`
}

// ToParams converts the input to create params.
func (input *CreateTemplateInput) ToParams(operator string) frameworktmpl.CreateParams {
	return frameworktmpl.CreateParams{
		Framework: input.Framework,
		Language:  input.Language,
		Definition: frameworktmpl.Definition{
			Name:     input.Name,
			FileName: input.FileName,
			Content:  input.Content,
		},
		IsDefault: input.IsDefault,
		Operator:  operator,
	}
}

// UpdateTemplateInput is the body for updating a template. Framework and language are immutable.
type UpdateTemplateInput struct {
	// 模板名称，同一框架、同一语言下不能重复，不能为「平台默认」
	Name string `json:"name" binding:"required"`
	// 框架配置文件名
	FileName string `json:"fileName" binding:"required"`
	// 框架配置文件内容，最大 64KB
	Content string `json:"content" binding:"required"`
	// 是否设为创建应用时默认使用的模板；不传不修改，取消默认后回落到平台默认模板
	IsDefault *bool `json:"isDefault"`
}

// ToParams converts the input to update params.
func (input *UpdateTemplateInput) ToParams(operator string) frameworktmpl.UpdateParams {
	return frameworktmpl.UpdateParams{
		Definition: frameworktmpl.Definition{
			Name:     input.Name,
			FileName: input.FileName,
			Content:  input.Content,
		},
		IsDefault: input.IsDefault,
		Operator:  operator,
	}
}

// TemplateOutputObj is one template.
type TemplateOutputObj struct {
	// 模板 ID，内置模板为 builtin-{framework}-{language}
	ID string `json:"id"`
	// 框架
	Framework string `json:"framework"`
	// 语言
	Language string `json:"language"`
	// 模板名称
	Name string `json:"name"`
	// 框架配置文件名
	FileName string `json:"fileName"`
	// 框架配置文件内容，列表接口不返回
	Content string `json:"content,omitempty"`
	// 是否为平台内置模板，内置模板只读
	Builtin bool `json:"builtin"`
	// 是否为所在框架、语言的默认模板
	IsDefault bool `json:"isDefault"`
	// 创建人，内置模板为空
	Creator string `json:"creator"`
	// 最近修改人，内置模板为空
	Updater string `json:"updater"`
	// 创建时间，内置模板为空
	CreatedAt *time.Time `json:"createdAt"`
	// 更新时间，内置模板为空
	UpdatedAt *time.Time `json:"updatedAt"`
}

// FromView fills the output from a template view.
func (output *TemplateOutputObj) FromView(view *frameworktmpl.View) *TemplateOutputObj {
	*output = TemplateOutputObj{
		ID:        view.ID,
		Framework: view.Framework,
		Language:  view.Language,
		Name:      view.Name,
		FileName:  view.FileName,
		Content:   view.Content,
		Builtin:   view.Builtin,
		IsDefault: view.IsDefault,
		Creator:   view.Creator,
		Updater:   view.Updater,
	}
	if !view.Builtin {
		output.CreatedAt = &view.CreatedAt
		output.UpdatedAt = &view.UpdatedAt
	}
	return output
}

// ListTemplatesOutput is the list response.
type ListTemplatesOutput struct {
	Data []*TemplateOutputObj `json:"data"`
}

// TemplateOutput is the single-template response.
type TemplateOutput struct {
	Data *TemplateOutputObj `json:"data"`
}

// EmptyOutput is an empty successful response.
type EmptyOutput struct{}
