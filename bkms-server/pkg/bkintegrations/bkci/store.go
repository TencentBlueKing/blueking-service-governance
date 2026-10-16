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

package bkci

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrProjectNotFound 蓝盾项目未找到
	ErrProjectNotFound = errors.New("bkci project not found")
	// ErrPipelineTemplateNotFound 流水线模板未找到
	ErrPipelineTemplateNotFound = errors.New("pipeline template not found")
	// ErrPipelineNotFound 流水线未找到
	ErrPipelineNotFound = errors.New("pipeline not found")
	// ErrRepositoryNotFound 代码仓库未找到
	ErrRepositoryNotFound = errors.New("repository not found")
)

type (
	// ProjectStore 蓝盾项目存储
	ProjectStore = store.GenericStore[Project]
	// PipelineTemplateStore 流水线模板存储
	PipelineTemplateStore = store.GenericStore[PipelineTemplate]
	// PipelineStore 流水线存储
	PipelineStore = store.GenericStore[Pipeline]
	// RepositoryStore 代码仓库存储
	RepositoryStore = store.GenericStore[Repository]
)

// ProjectMeta 蓝盾项目的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id
// - 唯一：code
// - 唯一：workspaceID
var ProjectMeta = store.Meta[Project]{
	Collection: "bkci_projects",
	NotFound:   ErrProjectNotFound,
}

// PipelineTemplateMeta 流水线模板的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id
// - 唯一：type
var PipelineTemplateMeta = store.Meta[PipelineTemplate]{
	Collection: "bkci_pipeline_templates",
	NotFound:   ErrPipelineTemplateNotFound,
}

// PipelineMeta 流水线的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id
// - 唯一：workspaceID + type
var PipelineMeta = store.Meta[Pipeline]{
	Collection: "bkci_pipelines",
	NotFound:   ErrPipelineNotFound,
}

// RepositoryMeta 代码仓库的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id + projectCode
// - 唯一：workspaceID + alias
var RepositoryMeta = store.Meta[Repository]{
	Collection: "bkci_repositories",
	NotFound:   ErrRepositoryNotFound,
}

// ByWorkspace 按工作空间 ID 匹配，适用于 Project、Pipeline 与 Repository
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByCode 按项目 Code 匹配 Project
func ByCode(code string) store.Opt {
	return store.Where("code", code)
}

// ByType 按类型匹配，适用于 PipelineTemplate（类型全局唯一）
func ByType(typ string) store.Opt {
	return store.Where("type", typ)
}

// ByWorkspaceAndType 按工作空间 ID 与流水线类型匹配 Pipeline
func ByWorkspaceAndType(workspaceID, pipelineType string) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), ByType(pipelineType))
}

// ByWorkspaceAndAlias 按工作空间 ID 与仓库别名匹配 Repository
func ByWorkspaceAndAlias(workspaceID, alias string) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.Where("alias", alias))
}

// UpdateBuiltinTemplateVersion 更新内置流水线已应用的模板版本、名称与描述。
// 注意：仅适用于内置流水线（每个 Workspace 每种内置类型只有一条 Pipeline 记录，且
// 名称/描述由模板决定）。禁止用于用户自定义流水线，否则会用本地陈旧的 name/description
// 覆盖蓝盾侧用户维护的最新值。
func UpdateBuiltinTemplateVersion(ctx context.Context, pipelineStore *PipelineStore, pipeline *Pipeline) error {
	err := pipelineStore.Update(
		ctx, pipeline, store.UpdateSelected("name", "description", "templateVersion"),
		ByWorkspaceAndType(pipeline.WorkspaceID, pipeline.Type),
	)
	return errors.Wrapf(err, "update template version of pipeline %s", pipeline.Type)
}
