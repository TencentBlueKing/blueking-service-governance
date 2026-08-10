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

import "strings"

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

// PipelineTypeBuildTriggerPrefix 触发专用流水线的类型前缀。
//
// 触发专用流水线只监听工蜂 Git 事件并回调 bkms，自身不执行构建，且要求**应用级唯一**。
// bkci_pipelines 的唯一索引仍为 workspaceID + type，应用级唯一是通过把 appID 编码进 type
// 达成的——workspaceID + build-trigger-{appID} 等价于 workspaceID + appID + 触发专用类型，
// 因此无需变更任何索引。这沿用了 type 字段已有的语义：它本就不是纯枚举，用户自定义流水线的
// type 直接就是 pipelineID。
//
// 注意 isBuiltinPipelineType 目前按精确匹配判定，复合 type 会被误判为用户自定义流水线。
// 「触发专用流水线按应用下发」子需求落地时需将其改为前缀匹配，详见
// design_notes/build_trigger_contract.md
const PipelineTypeBuildTriggerPrefix = "build-trigger-"

// BuildTriggerPipelineType 拼装指定应用的触发专用流水线类型
func BuildTriggerPipelineType(appID string) PipelineType {
	return PipelineType(PipelineTypeBuildTriggerPrefix + appID)
}

// ParseBuildTriggerPipelineType 从触发专用流水线类型中解析出 appID，
// 类型前缀不匹配时返回空字符串与 false
func ParseBuildTriggerPipelineType(pipelineType string) (string, bool) {
	return strings.CutPrefix(pipelineType, PipelineTypeBuildTriggerPrefix)
}
