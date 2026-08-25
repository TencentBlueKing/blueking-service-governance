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

package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// ListCustomBuildImages 获取工作空间自定义构建镜像候选列表
//
//	@ID				ListCustomBuildImages
//	@Summary		获取工作空间自定义构建镜像候选列表
//	@Description	候选仅以工作空间已落库的自定义镜像记录为准，不过滤快照同步状态，也不校验镜像在仓库中是否仍然存在；候选数量预期在百条以内，因此不分页
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			type		query		string	true	"镜像类型：builder / runner"
//	@Param			keyword		query		string	false	"搜索关键字"
//	@Success		200			{object}	serializer.ListCustomRuntimeImagesOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/custom-build-images [get]
func (h *Handler) ListCustomBuildImages(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var queryInput serializer.ListCustomRuntimeImagesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 只查落库记录：不校验镜像源是否绑定，也不读快照
	images, err := h.registry.CustomRuntimeImageStore.List(
		ctx,
		uriInput.WorkspaceID,
		customruntime.ListOptions{
			Type:    customruntime.ImageType(queryInput.Type),
			Keyword: queryInput.Keyword,
		},
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list workspace %s custom build images", uriInput.WorkspaceID,
		))
		return
	}

	results := make([]*serializer.CustomRuntimeImageOutputObj, 0, len(images))
	for _, img := range images {
		results = append(results, new(serializer.CustomRuntimeImageOutputObj).FromModel(img))
	}
	ginutils.OK(c, serializer.ListCustomRuntimeImagesOutput{
		Data: &serializer.CustomRuntimeImagesOutputObjs{Results: results},
	})
}

// ListCustomBuildImageTags 获取工作空间自定义构建镜像可用 TAG 列表
//
//	@ID				ListCustomBuildImageTags
//	@Summary		获取工作空间自定义构建镜像可用 TAG 列表
//	@Description	镜像以完整名称传入而非记录 ID，因为用户手动输入、尚未落库的镜像没有记录 ID。
//	@Description	已落库镜像读本地快照、手动输入镜像用工作空间凭证实时拉取，两条来源的出入参、
//	@Description	分页与总数口径完全一致，调用方无需按来源分支处理，也不传递来源标识
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			name		query		string	true	"镜像完整仓库名称，含仓库前缀且不带 tag"
//	@Param			keyword		query		string	false	"搜索关键字"
//	@Param			page		query		int		true	"分页参数：页码，从 1 开始"
//	@Param			pageSize	query		int		true	"分页参数：每页数量，支持 5/10/20/50/100"
//	@Success		200			{object}	serializer.ListCustomRuntimeImageTagsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/custom-build-images/tags [get]
func (h *Handler) ListCustomBuildImageTags(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var queryInput serializer.ListCustomRuntimeImageTagsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 与候选列表同一档查看权限；name 走 query 而非路径 ID，因为手动输入镜像尚未落库
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	queried, err := h.newTagQueryManager().ListTags(
		ctx,
		uriInput.WorkspaceID,
		queryInput.Name,
		queryInput.Keyword,
		int(queryInput.Page),
		int(queryInput.PageSize),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, customTagErrCode(err), "list tags of custom build image %s", queryInput.Name,
		))
		return
	}

	results := make([]*serializer.CustomRuntimeImageTagOutputObj, 0, len(queried.Tags))
	for _, snap := range queried.Tags {
		results = append(results, new(serializer.CustomRuntimeImageTagOutputObj).FromModel(snap))
	}

	ginutils.OK(c, serializer.ListCustomRuntimeImageTagsOutput{
		Data: &serializer.PaginatedCustomRuntimeImageTagOutputObjs{
			Count:          queried.Total,
			Results:        results,
			SnapshotStatus: new(serializer.SnapshotStatusInfoOutputObj).FromModel(queried.Status),
		},
	})
}

// RefreshCustomBuildImageTags 手动刷新工作空间自定义构建镜像的 TAG 快照
//
//	@ID				RefreshCustomBuildImageTags
//	@Summary		手动刷新工作空间自定义构建镜像的 TAG 快照
//	@Description	同步等待上限为 10 秒。刷新中与刷新失败均为正常响应，通过 data.status 的 refreshing / failed 表达，不作为错误抛出
//	@Tags			images
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string											true	"工作空间 ID"
//	@Param			body		body		serializer.RefreshCustomRuntimeImageTagsInput	true	"刷新参数"
//	@Success		200			{object}	serializer.RefreshCustomRuntimeImageTagsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/custom-build-images/tags/refresh [post]
func (h *Handler) RefreshCustomBuildImageTags(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var jsonInput serializer.RefreshCustomRuntimeImageTagsInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 刷新会改快照，权限档位用 TypeEdit
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	startedAt := time.Now()
	metricStatus := metrics.StatusOK
	defer func() {
		metrics.ImageSnapshotRefreshFinished(metricStatus, startedAt)
	}()

	result, err := h.newTagQueryManager().RefreshTags(ctx, uriInput.WorkspaceID, jsonInput.Name)
	if err != nil {
		// 超时单独上报，便于与仓库拒绝、平台故障区分开观测
		metricStatus = metrics.StatusErr
		if errors.Is(err, customruntime.ErrRegistryAccessTimeout) {
			metricStatus = metrics.StatusTimeout
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, customTagErrCode(err), "refresh tags of custom build image %s", jsonInput.Name,
		))
		return
	}
	if result.Status == snapshot.RefreshResultFailed {
		metricStatus = metrics.StatusErr
	}

	ginutils.OK(c, serializer.RefreshCustomRuntimeImageTagsOutput{
		Data: new(serializer.RefreshResultInfoOutputObj).FromModel(result),
	})
}

// newTagQueryManager 组装自定义镜像 tag 查询依赖。
//
// 工作空间口径的 repoKey 只依赖镜像源账密，不读构建配置与应用，故快照服务的另两个 store 传 nil
func (h *Handler) newTagQueryManager() *customruntime.TagQueryManager {
	return customruntime.NewTagQueryManager(
		h.registry.CustomRuntimeImageStore,
		snapshot.NewService(h.registry.SnapshotStore, nil, nil),
	)
}

// customTagErrCode 把自定义镜像 tag 链路的错误映射为对外错误码。
//
// 全部复用既有 ErrCode，不新增常量；bkerrs 没有 deadline 语义的错误码，
// 超时只能归入服务端错误，可识别性由错误信息中的超时字样承担
func customTagErrCode(err error) bkerrs.ErrCode {
	switch {
	case errors.Is(err, customruntime.ErrImageNameNotFound),
		errors.Is(err, customruntime.ErrCustomRuntimeImageNotFound):
		return bkerrs.ErrCodeNotFound
	case errors.Is(err, customruntime.ErrRegistryAccessDenied),
		errors.Is(err, customruntime.ErrWorkspaceRegistryUnbound),
		errors.Is(err, customruntime.ErrImageNotInWorkspaceRegistry):
		return bkerrs.ErrCodeInvalidRequest
	default:
		return bkerrs.ErrCodeInternalServerError
	}
}
