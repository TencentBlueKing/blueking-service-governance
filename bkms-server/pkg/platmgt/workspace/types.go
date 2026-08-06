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

package workspace

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
)

// WorkspaceWithStats is a workspace with aggregated statistics for platform workspace listing.
type WorkspaceWithStats struct {
	// ID is the unique identifier of the workspace.
	ID string
	// DisplayName is the display name shown for the workspace.
	DisplayName string
	// Description is the human-readable description of the workspace.
	Description string
	// State is the current lifecycle state of the workspace.
	State string
	// Creator is the user who created the workspace.
	Creator string
	// Updater is the user who last updated the workspace.
	Updater string
	// UpdatedAt is when the workspace was last updated.
	UpdatedAt time.Time
	// AppCount is the number of applications in the workspace.
	AppCount int
	// EnvCount is the number of environments in the workspace.
	EnvCount int
}

// WorkspaceInfo is the basic info model for a platform workspace.
type WorkspaceInfo struct {
	// ID is the unique identifier of the workspace.
	ID string
	// DisplayName is the display name shown for the workspace.
	DisplayName string
	// Description is the human-readable description of the workspace.
	Description string
	// State is the current lifecycle state of the workspace.
	State string
	// Creator is the user who created the workspace.
	Creator string
	// CreatedAt is when the workspace was created.
	CreatedAt time.Time
	// Updater is the user who last updated the workspace.
	Updater string
	// UpdatedAt is when the workspace was last updated.
	UpdatedAt time.Time
}

// WorkspaceListOptions is the query model for platform workspace listing.
type WorkspaceListOptions struct {
	// Keyword fuzzy matches workspace ID and display name.
	Keyword string
	// State filters workspaces by lifecycle state.
	State string
	// SortBy controls which white-listed field is used for sorting.
	SortBy string
	// SortOrder controls sorting direction and supports asc or desc.
	SortOrder string
	// Page is the current page number, starting from 1.
	Page int64
	// PageSize is the number of items returned per page.
	PageSize int64
}

// ToWorkspaceStoreListOptions converts list options to workspace store list options.
func (o WorkspaceListOptions) ToWorkspaceStoreListOptions() *bkmsworkspace.ListOptions {
	opts := &bkmsworkspace.ListOptions{Keyword: o.Keyword}
	if o.State != "" {
		state := bkmsworkspace.State(o.State)
		opts.State = &state
	}
	// 构建排序字段
	order := int64(-1)
	if o.SortOrder == "asc" {
		order = 1
	}
	switch o.SortBy {
	case "id":
		opts.Sort = bson.D{
			{Key: "id", Value: order},
		}
	case "displayName", "updatedAt":
		opts.Sort = bson.D{
			{Key: o.SortBy, Value: order},
			{Key: "id", Value: 1},
		}
	default:
		// 未指定或非白名单字段时回退到 store 默认排序，避免未来上层校验缺失时透传非法字段。
		return opts
	}
	return opts
}

// ToWorkspaceStoreStatisticsOptions 将列表查询参数转换为顶部状态统计卡片使用的过滤条件。
// 统计跟随关键字（Keyword）过滤，但与当前选中的状态标签页相互独立。
func (o WorkspaceListOptions) ToWorkspaceStoreStatisticsOptions() *bkmsworkspace.ListOptions {
	return &bkmsworkspace.ListOptions{Keyword: o.Keyword}
}

// ToWorkspaceStoreListPageOptions converts list options to workspace store page options.
func (o WorkspaceListOptions) ToWorkspaceStoreListPageOptions() *bkmsworkspace.ListPageOptions {
	return &bkmsworkspace.ListPageOptions{
		ListOptions: *o.ToWorkspaceStoreListOptions(),
		Page:        o.Page,
		PageSize:    o.PageSize,
	}
}

// WorkspaceStateStatistics is the aggregated workspace state statistics for matched workspaces.
type WorkspaceStateStatistics struct {
	// ReadyCount is the number of workspaces in Ready state.
	ReadyCount int64
	// ProcessingCount is the number of workspaces in Processing state.
	ProcessingCount int64
	// DisabledCount is the number of workspaces in Disabled state.
	DisabledCount int64
	// TotalCount is the total number of workspaces across all states.
	TotalCount int64
}

// WorkspaceListResult is the paginated platform workspace listing result.
type WorkspaceListResult struct {
	// Count is the total number of matched workspaces.
	Count int64
	// Page is the current page number.
	Page int64
	// PageSize is the number of items returned per page.
	PageSize int64
	// Results is the current page of workspace items with stats.
	Results []WorkspaceWithStats
	// Statistics is the state distribution for workspaces matched by the keyword
	// filter before pagination, independent of the selected state tab.
	Statistics WorkspaceStateStatistics
}
