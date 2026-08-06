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

// Package bkci 蓝盾项目、流水线、凭证等相关接入实现
package bkci

// PipelineType 流水线类型
type PipelineType string

const (
	// PipelineTypeDockerfile 基于 dockerfile 构建镜像的流水线
	PipelineTypeDockerfile PipelineType = "dockerfile"
	// PipelineTypeHelmGitBuild 基于 Git 源码构建 Helm Chart 的流水线
	PipelineTypeHelmGitBuild PipelineType = "helm-git-build"
)

// builtinPipelineTypes 内置的流水线类型
// 用户自定义的流水线，类型即 pipelineID：p-[a-z0-9]{32}
var builtinPipelineTypes = []PipelineType{
	PipelineTypeDockerfile,
	PipelineTypeHelmGitBuild,
}
