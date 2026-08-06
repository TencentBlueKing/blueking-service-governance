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
	"time"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

// --- BkCI URI 参数 ---

// BkCIWorkspaceURIInput 路径参数
type BkCIWorkspaceURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,min=1,max=27,workspace_id"`
}

// BkCIPipelineURIInput 路径参数
type BkCIPipelineURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,min=1,max=27,workspace_id"`
	PipelineID  string `uri:"pipelineID" binding:"required,min=1"`
}

// BkCIPipelineRepoRefOptionsInput 获取分支/Tag选项的请求体
type BkCIPipelineRepoRefOptionsInput struct {
	PropertyID string `json:"propertyID" binding:"required"`
	Search     string `json:"search"`
}

// --- BkCI Query 参数 ---

// BkCIGitProjectsQueryInput 获取 Git 项目列表的查询参数
type BkCIGitProjectsQueryInput struct {
	Keyword string `form:"keyword"`
}

// BkCIPipelinesQueryInput 获取流水线列表的查询参数
type BkCIPipelinesQueryInput struct {
	Keyword  string `form:"keyword"`
	Page     int64  `form:"page" binding:"required,min=1"`
	PageSize int64  `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// --- BkCI Output ---

// BkCIOAuthGitProjectOutput OAuth Git 项目输出
type BkCIOAuthGitProjectOutput struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"alias"`
	Url   string `json:"url"`
}

// FromModel 从领域模型填充输出字段
func (o *BkCIOAuthGitProjectOutput) FromModel(p bkci.GitProject) *BkCIOAuthGitProjectOutput {
	if o == nil {
		return nil
	}
	*o = BkCIOAuthGitProjectOutput{
		ID:    p.ID,
		Name:  p.Name,
		Alias: p.Alias,
		Url:   p.Url,
	}
	return o
}

// ListBkCIOAuthGitProjectsOutput 获取 OAuth Git 项目列表的响应
type ListBkCIOAuthGitProjectsOutput struct {
	Data []*BkCIOAuthGitProjectOutput `json:"data"`
}

// BkCIPipelineOutput 蓝盾流水线输出
type BkCIPipelineOutput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     int64  `json:"version,string"`
	Creator     string `json:"creator"`
	Updater     string `json:"updater"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// FromModel 从领域模型填充输出字段
func (o *BkCIPipelineOutput) FromModel(p bkci.Pipeline) *BkCIPipelineOutput {
	if o == nil {
		return nil
	}
	*o = BkCIPipelineOutput{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Version:     p.Version,
		Creator:     p.Creator,
		Updater:     p.Updater,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
	return o
}

// PaginatedBkCIPipelineOutput 分页流水线输出
type PaginatedBkCIPipelineOutput struct {
	Count   int64                 `json:"count,string"`
	Results []*BkCIPipelineOutput `json:"results"`
}

// ListBkCIPipelinesOutput 获取流水线列表的响应
type ListBkCIPipelinesOutput struct {
	Data *PaginatedBkCIPipelineOutput `json:"data"`
}

// BkCIPipelineDetailOutput 蓝盾流水线详情输出
type BkCIPipelineDetailOutput struct {
	ID          string                        `json:"id"`
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Version     int64                         `json:"version,string"`
	Creator     string                        `json:"creator"`
	Updater     string                        `json:"updater"`
	CreatedAt   string                        `json:"createdAt"`
	UpdatedAt   string                        `json:"updatedAt"`
	Variables   []*BkCIPipelineVariableOutput `json:"variables"`
}

// FromModel 从领域模型填充输出字段
func (o *BkCIPipelineDetailOutput) FromModel(p bkci.Pipeline) *BkCIPipelineDetailOutput {
	if o == nil {
		return nil
	}
	*o = BkCIPipelineDetailOutput{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Version:     p.Version,
		Creator:     p.Creator,
		Updater:     p.Updater,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
		Variables: lo.Map(p.Variables, func(v bkci.PipelineVariable, _ int) *BkCIPipelineVariableOutput {
			return new(BkCIPipelineVariableOutput).FromModel(v)
		}),
	}
	return o
}

// GetBkCIPipelineOutput 获取流水线详情的响应
type GetBkCIPipelineOutput struct {
	Data *BkCIPipelineDetailOutput `json:"data"`
}

// BkCIPipelineVariableOutput 蓝盾流水线变量输出
type BkCIPipelineVariableOutput struct {
	ID           string                              `json:"id"`
	Name         string                              `json:"name"`
	Description  string                              `json:"description"`
	Required     bool                                `json:"required"`
	ReadOnly     bool                                `json:"readOnly"`
	Constant     bool                                `json:"constant"`
	DefaultValue string                              `json:"defaultValue"`
	Type         string                              `json:"type"`
	Options      []*BkCIPipelineVariableOptionOutput `json:"options"`
}

// FromModel 从领域模型填充输出字段
func (o *BkCIPipelineVariableOutput) FromModel(v bkci.PipelineVariable) *BkCIPipelineVariableOutput {
	if o == nil {
		return nil
	}
	*o = BkCIPipelineVariableOutput{
		ID:           v.ID,
		Name:         v.Name,
		Description:  v.Description,
		Required:     v.Required,
		ReadOnly:     v.ReadOnly,
		Constant:     v.Constant,
		DefaultValue: v.DefaultValue,
		Type:         v.Type,
		Options: lo.Map(v.Options, func(opt bkci.PipelineVariableOption, _ int) *BkCIPipelineVariableOptionOutput {
			return &BkCIPipelineVariableOptionOutput{Key: opt.Key, Value: opt.Value}
		}),
	}
	return o
}

// BkCIPipelineVariableOptionOutput 蓝盾流水线变量选项输出
type BkCIPipelineVariableOptionOutput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetBkCIOAuthUrlOutput 获取 OAuth 授权 URL 的响应
type GetBkCIOAuthUrlOutput struct {
	Data string `json:"data"`
}

// GetBkCIPipelineVariablesOutput 获取流水线变量列表的响应
type GetBkCIPipelineVariablesOutput struct {
	Data []*BkCIPipelineVariableOutput `json:"data"`
}

// BkCIPipelineRepoRefOutput 蓝盾流水线分支/Tag字段输出
type BkCIPipelineRepoRefOutput struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Label        string `json:"label"`
	Type         string `json:"type"`
	Required     bool   `json:"required"`
	ReadOnly     bool   `json:"readOnly"`
	Constant     bool   `json:"constant"`
	DefaultValue string `json:"defaultValue"`
	Value        string `json:"value"`
}

// FromModel 从领域模型填充输出字段
func (o *BkCIPipelineRepoRefOutput) FromModel(p bkci.RepoRefProperty) *BkCIPipelineRepoRefOutput {
	if o == nil {
		return nil
	}
	*o = BkCIPipelineRepoRefOutput{
		ID:           p.ID,
		Name:         p.Name,
		Label:        p.Label,
		Type:         p.Type,
		Required:     p.Required,
		ReadOnly:     p.ReadOnly,
		Constant:     p.Constant,
		DefaultValue: p.DefaultValue,
		Value:        p.Value,
	}
	return o
}

// ListBkCIPipelineRepoRefsOutput 获取流水线分支/Tag字段列表的响应
type ListBkCIPipelineRepoRefsOutput struct {
	Data []*BkCIPipelineRepoRefOutput `json:"data"`
}

// ListBkCIPipelineRepoRefOptionsOutput 获取流水线分支/Tag选项列表的响应
type ListBkCIPipelineRepoRefOptionsOutput struct {
	Data []*BkCIPipelineVariableOptionOutput `json:"data"`
}
