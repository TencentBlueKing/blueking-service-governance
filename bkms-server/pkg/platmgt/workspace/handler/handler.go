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

// Package handler contains Gin handlers for platform workspace APIs.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	platmgtworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ platmgtworkspace.Handler = (*GinHandler)(nil)

// GinHandler handles Gin platform workspace API requests.
type GinHandler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *GinHandler {
	return &GinHandler{registry: registry}
}

func (h *GinHandler) service() *platmgtworkspace.Service {
	return platmgtworkspace.NewService(
		h.registry.WorkspaceStore,
		h.registry.AppStore,
		h.registry.EnvStore,
	)
}

// ListPlatWorkspaces 查询平台空间列表
//
//	@ID			ListPlatWorkspaces
//	@Summary	查询平台空间列表
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		keyword		query		string	false	"搜索关键词，匹配空间 ID / 空间名称"
//	@Param		state		query		string	false	"空间状态过滤：Ready / Processing / Disabled"
//	@Param		sortBy		query		string	false	"排序字段：id / displayName / updatedAt"
//	@Param		sortOrder	query		string	false	"排序方向：asc / desc"
//	@Param		page		query		int		true	"页码，从 1 开始"
//	@Param		pageSize	query		int		true	"每页数量，支持 5/10/20/50/100"
//	@Success	200	{object}	serializer.ListWorkspacesResponse
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces [get]
func (h *GinHandler) ListPlatWorkspaces(c *gin.Context) {
	var query serializer.ListWorkspacesQuery
	if err := ginutils.BindQuery(c, &query); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	items, err := h.service().List(c.Request.Context(), query.ToListOptions())
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list platform workspaces"))
		return
	}

	data := lo.Map(
		items.Results,
		func(item platmgtworkspace.WorkspaceWithStats, _ int) *serializer.WorkspaceWithStatsOutput {
			return serializer.NewWorkspaceWithStatsOutput(item)
		},
	)
	ginutils.OK(c, serializer.ListWorkspacesResponse{
		Data: &serializer.PaginatedWorkspaceOutput{
			Count:      items.Count,
			Page:       items.Page,
			PageSize:   items.PageSize,
			Results:    data,
			Statistics: serializer.NewWorkspaceStatsOutput(items.Statistics),
		},
	})
}

// GetPlatWorkspaceStats 查询平台所有工作空间统计数据
//
//	@ID			GetPlatWorkspaceStats
//	@Summary	查询平台工作空间数据统计
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.WorkspaceStatsResponse
//	@Failure	403	{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces/statistics [get]
func (h *GinHandler) GetPlatWorkspaceStats(c *gin.Context) {
	// 查找状态统计数据
	stats, err := h.service().GetStateStatistics(c.Request.Context())
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get platform workspace state statistics"),
		)
		return
	}

	ginutils.OK(c, serializer.WorkspaceStatsResponse{
		Data: serializer.NewWorkspaceStatsOutput(*stats),
	})
}

// GetPlatWorkspace 查询平台空间详情。
//
// 该接口保留在 platmgt 路由组下，是为了支撑平台管理员在平台管理视角下
// 直接查看任意空间的基础信息。
//
// 与 core/workspace 下的 GetWorkspace 相比：
// 1. 这里依赖 platmgt 路由组的权限语义，而不是 workspace 侧的空间查看权限校验。
// 2. 这里只返回平台管理页所需的基础字段，不返回镜像仓库、蓝鲸系统等扩展信息。
//
//	@ID			GetPlatWorkspace
//	@Summary	查询平台空间详情
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.GetWorkspaceResponse
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces/{workspaceID} [get]
func (h *GinHandler) GetPlatWorkspace(c *gin.Context) {
	var path serializer.WorkspacePath
	if err := ginutils.BindURI(c, &path); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	item, err := h.service().Get(c.Request.Context(), path.WorkspaceID)
	if err != nil {
		if errors.Is(err, bkmsworkspace.ErrWorkspaceNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "platform workspace not found"))
		} else {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get platform workspace"))
		}
		return
	}

	ginutils.OK(c, serializer.GetWorkspaceResponse{
		Data: serializer.NewWorkspaceInfoOutput(*item),
	})
}
