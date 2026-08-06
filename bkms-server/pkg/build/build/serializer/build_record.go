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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelinevar"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
)

// ListBuildRecordsQueryInput is the query input for listing build records.
type ListBuildRecordsQueryInput struct {
	// 搜索关键字
	Keyword string `form:"keyword"`
	// 页码，从 1 开始
	Page int64 `form:"page" binding:"required,gte=1"`
	// 每页数量，支持 5/10/20/50/100
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// CreateBuildInput is the JSON body for creating a build.
type CreateBuildInput struct {
	// 构建分支或标签
	Branch string `json:"branch"`
	// 本次构建使用的镜像 Tag
	ImageTag string `json:"imageTag"`
}

// BuildRecordOutputObj is the JSON representation of one build record.
type BuildRecordOutputObj struct {
	// 蓝盾流水线 ID
	PipelineID string `json:"pipelineID"`
	// 蓝盾构建 ID
	BuildID string `json:"buildID"`
	// 构建序号
	Num int64 `json:"num,string"`
	// 构建参数
	Params map[string]string `json:"params"`
	// 构建状态
	Status string `json:"status"`
	// 代码仓库地址
	RepoURL string `json:"repoURL"`
	// 代码版本
	Revision string `json:"revision"`
	// 提交哈希
	CommitID string `json:"commitID"`
	// 产物地址
	Artifact string `json:"artifact"`
	// 操作人
	Operator string `json:"operator"`
	// 额外元数据
	Extras map[string]string `json:"extras"`
	// 开始时间
	StartedAt time.Time `json:"startedAt"`
	// 结束时间
	EndedAt time.Time `json:"endedAt"`
}

// FromModel fills output fields from a build record model.
func (o *BuildRecordOutputObj) FromModel(r imagebuild.Record) *BuildRecordOutputObj {
	*o = BuildRecordOutputObj{
		PipelineID: r.PipelineID,
		BuildID:    r.BuildID,
		Num:        r.Num,
		Params:     r.Params,
		Status:     string(r.Status),
		RepoURL:    r.Params[pipelineparam.RepoURL],
		Revision:   r.Params[pipelineparam.RepoRevision],
		CommitID:   r.Extras[pipelinevar.GitRepoHeadCommitID],
		Artifact:   r.Artifact,
		Operator:   r.Operator,
		Extras:     r.Extras,
		StartedAt:  r.StartedAt,
		EndedAt:    r.EndedAt,
	}
	return o
}

// PaginatedBuildRecordOutputObjs is the paginated build record payload.
type PaginatedBuildRecordOutputObjs struct {
	// 记录总数
	Count int64 `json:"count,string"`
	// 当前页结果
	Results []*BuildRecordOutputObj `json:"results"`
}

// ListBuildRecordsOutput is the JSON response for listing build records.
type ListBuildRecordsOutput struct {
	// 构建记录分页结果
	Data *PaginatedBuildRecordOutputObjs `json:"data"`
}

// CreateBuildOutput is the JSON response for creating a build.
type CreateBuildOutput struct {
	// 新创建的构建记录
	Data *BuildRecordOutputObj `json:"data"`
}
