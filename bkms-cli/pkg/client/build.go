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

// Package client provides build related types
package client

// BuildOptions 构建选项参数
type BuildOptions struct {
	// Branch 代码分支
	Branch string
	// ImageTag 镜像 Tag
	ImageTag string
}

// BuildRecord 构建记录
type BuildRecord struct {
	// PipelineID 流水线 ID
	PipelineID string `json:"pipelineID" yaml:"pipelineID"`
	// BuildID 构建 ID
	BuildID string `json:"buildID" yaml:"buildID"`
	// Num 构建号
	Num string `json:"num" yaml:"num"`
	// Params 构建参数（包含代码库，分支等信息）
	Params map[string]string `json:"params" yaml:"params"`
	// Status 构建状态
	Status string `json:"status" yaml:"status"`
	// RepoURL 代码仓库
	RepoURL string `json:"repoURL" yaml:"repoURL"`
	// Revision 代码版本
	Revision string `json:"revision" yaml:"revision"`
	// CommitID 代码提交 ID
	CommitID string `json:"commitID" yaml:"commitID"`
	// Artifact 构建产物
	Artifact string `json:"artifact" yaml:"artifact"`
	// Operator 触发人
	Operator string `json:"operator" yaml:"operator"`
	// Extras 额外信息
	Extras map[string]string `json:"extras" yaml:"extras"`
	// StartedAt 构建开始时间
	StartedAt string `json:"startedAt" yaml:"startedAt"`
	// EndedAt 构建结束时间
	EndedAt string `json:"endedAt" yaml:"endedAt"`
}

// ListBuildRecordsRespData 获取构建记录列表返回数据
type ListBuildRecordsRespData struct {
	Data PaginatedBuildRecords `json:"data"`
}

// PaginatedBuildRecords 分页构建记录
type PaginatedBuildRecords struct {
	Count   string        `json:"count"`
	Results []BuildRecord `json:"results"`
}
