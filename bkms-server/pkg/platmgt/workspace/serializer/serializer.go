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

// Package serializer defines Gin input and output serializers for platform workspace APIs.
package serializer

import (
	"time"

	platmgtworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace"
)

// ListWorkspacesQuery is the query input for listing platform workspaces.
type ListWorkspacesQuery struct {
	// 搜索关键词，匹配空间 ID / 空间名称
	Keyword string `form:"keyword" binding:"max=64"`
	// 空间状态过滤
	State string `form:"state" binding:"omitempty,oneof=Ready Processing Disabled"`
	// 排序字段，仅支持白名单字段
	SortBy string `form:"sortBy" binding:"omitempty,oneof=id displayName updatedAt,required_with=SortOrder"`
	// 排序方向，需与排序字段配套使用
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc,required_with=SortBy"`
	// 页码，从 1 开始
	Page int64 `form:"page" binding:"required,gte=1"`
	// 每页数量，仅支持固定枚举值
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// ToListOptions converts query input into service list options.
func (q ListWorkspacesQuery) ToListOptions() platmgtworkspace.WorkspaceListOptions {
	return platmgtworkspace.WorkspaceListOptions{
		Keyword:   q.Keyword,
		State:     q.State,
		SortBy:    q.SortBy,
		SortOrder: q.SortOrder,
		Page:      q.Page,
		PageSize:  q.PageSize,
	}
}

// WorkspacePath is the URI input for a single platform workspace.
type WorkspacePath struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
}

// WorkspaceWithStatsOutput is one platform workspace with aggregated statistics.
type WorkspaceWithStatsOutput struct {
	// 工作空间 ID
	ID string `json:"id"`
	// 工作空间展示名称
	DisplayName string `json:"displayName"`
	// 工作空间描述
	Description string `json:"description"`
	// 工作空间状态
	State string `json:"state"`
	// 创建人
	Creator string `json:"creator"`
	// 更新人
	Updater string `json:"updater"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
	// 应用数量
	AppCount int `json:"appCount"`
	// 环境数量
	EnvCount int `json:"envCount"`
}

// NewWorkspaceWithStatsOutput builds a list output from model.
func NewWorkspaceWithStatsOutput(item platmgtworkspace.WorkspaceWithStats) *WorkspaceWithStatsOutput {
	return &WorkspaceWithStatsOutput{
		ID:          item.ID,
		DisplayName: item.DisplayName,
		Description: item.Description,
		State:       item.State,
		Creator:     item.Creator,
		Updater:     item.Updater,
		UpdatedAt:   item.UpdatedAt,
		AppCount:    item.AppCount,
		EnvCount:    item.EnvCount,
	}
}

// WorkspaceInfoOutput is the basic info payload for one platform workspace.
type WorkspaceInfoOutput struct {
	// 工作空间 ID
	ID string `json:"id"`
	// 工作空间展示名称
	DisplayName string `json:"displayName"`
	// 工作空间描述
	Description string `json:"description"`
	// 工作空间状态
	State string `json:"state"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新人
	Updater string `json:"updater"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewWorkspaceInfoOutput builds a basic info output from model.
func NewWorkspaceInfoOutput(item platmgtworkspace.WorkspaceInfo) *WorkspaceInfoOutput {
	return &WorkspaceInfoOutput{
		ID:          item.ID,
		DisplayName: item.DisplayName,
		Description: item.Description,
		State:       item.State,
		Creator:     item.Creator,
		CreatedAt:   item.CreatedAt,
		Updater:     item.Updater,
		UpdatedAt:   item.UpdatedAt,
	}
}

// ListWorkspacesResponse is the JSON response for listing platform workspaces.
type ListWorkspacesResponse struct {
	Data *PaginatedWorkspaceOutput `json:"data"`
}

// PaginatedWorkspaceOutput is the paginated platform workspace list payload.
type PaginatedWorkspaceOutput struct {
	// 总数
	Count int64 `json:"count,string"`
	// 当前页码
	Page int64 `json:"page"`
	// 每页数量
	PageSize int64 `json:"pageSize"`
	// 当前页结果
	Results []*WorkspaceWithStatsOutput `json:"results"`
	// 按当前筛选条件命中的工作空间状态统计，基于未分页的完整结果集计算，不受 page / pageSize 影响
	Statistics *WorkspaceStatsOutput `json:"statistics"`
}

// GetWorkspaceResponse is the JSON response for querying one platform workspace.
type GetWorkspaceResponse struct {
	Data *WorkspaceInfoOutput `json:"data"`
}

// WorkspaceStatsOutput is the aggregated workspace state statistics payload.
type WorkspaceStatsOutput struct {
	// Ready 状态工作空间数量
	ReadyCount int64 `json:"readyCount,string"`
	// Processing 状态工作空间数量
	ProcessingCount int64 `json:"processingCount,string"`
	// Disabled 状态工作空间数量
	DisabledCount int64 `json:"disabledCount,string"`
	// TotalCount 工作空间总数
	TotalCount int64 `json:"totalCount,string"`
}

// NewWorkspaceStatsOutput builds statistics output from model.
func NewWorkspaceStatsOutput(
	item platmgtworkspace.WorkspaceStateStatistics,
) *WorkspaceStatsOutput {
	return &WorkspaceStatsOutput{
		ReadyCount:      item.ReadyCount,
		ProcessingCount: item.ProcessingCount,
		DisabledCount:   item.DisabledCount,
		TotalCount:      item.TotalCount,
	}
}

// WorkspaceStatsResponse is the JSON response for workspace state statistics.
type WorkspaceStatsResponse struct {
	Data *WorkspaceStatsOutput `json:"data"`
}
