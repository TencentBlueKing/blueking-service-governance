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
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// ListPlatformBuildImages 获取平台通用构建镜像列表
//
//	@ID				ListPlatformBuildImages
//	@Summary		获取平台通用构建镜像列表
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			type	query		string	true	"镜像类型：builder / runner"
//	@Param			keyword	query		string	false	"搜索关键字"
//	@Success		200		{object}	serializer.ListRuntimeImagesOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/platform-build-images [get]
func (h *Handler) ListPlatformBuildImages(c *gin.Context) {
	var queryInput serializer.ListRuntimeImagesQueryInput
	if err := ginutils.BindQuery(c, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	images, err := h.registry.RuntimeImageStore.List(
		c.Request.Context(), workloadruntime.ListOptions{
			Type:    workloadruntime.ImageType(queryInput.Type),
			Keyword: queryInput.Keyword,
		},
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list %s runtime images", queryInput.Type,
		))
		return
	}

	results := make([]*serializer.RuntimeImageOutputObj, 0, len(images))
	for _, img := range images {
		results = append(results, new(serializer.RuntimeImageOutputObj).FromModel(img))
	}
	ginutils.OK(c, serializer.ListRuntimeImagesOutput{
		Data: &serializer.RuntimeImagesOutputObjs{Results: results},
	})
}

// ListPlatformBuildImageTags 获取平台通用构建镜像可用 TAG 列表
//
//	@ID				ListPlatformBuildImageTags
//	@Summary		获取平台通用构建镜像可用 TAG 列表
//	@Description	从本地镜像快照读取指定平台通用构建镜像的 TAG；如果本地还没有快照，会异步触发一次初始化同步
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			imageID		path		string	true	"平台通用构建镜像记录 ID"
//	@Param			keyword		query		string	false	"搜索关键字"
//	@Param			page		query		int		true	"分页参数：页码，从 1 开始"
//	@Param			pageSize	query		int		true	"分页参数：每页数量，支持 5/10/20/50/100"
//	@Success		200			{object}	serializer.ListRuntimeImageTagsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Failure		500			{object}	bkerrs.GinErrorOutput
//	@Router			/platform-build-images/{imageID}/tags [get]
func (h *Handler) ListPlatformBuildImageTags(c *gin.Context) {
	var uriInput serializer.RuntimeImageURIInput
	var queryInput serializer.ListRuntimeImageTagsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	runtimeImage, err := h.registry.RuntimeImageStore.GetByID(ctx, uriInput.ImageID)
	if err != nil {
		code := lo.Ternary(
			errors.Is(err, workloadruntime.ErrRuntimeImageNotFound),
			bkerrs.ErrCodeNotFound,
			bkerrs.ErrCodeInternalServerError,
		)
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, code, "get runtime image %s", uriInput.ImageID))
		return
	}

	snapshotService := snapshot.NewService(h.registry.SnapshotStore, nil, nil)
	snapshots, total, status, err := snapshotService.ListRepositorySnapshots(
		ctx,
		runtimeImage.Name,
		queryInput.Keyword,
		int(queryInput.Page),
		int(queryInput.PageSize),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list runtime image tags for %s", runtimeImage.Name,
		))
		return
	}

	results := make([]*serializer.RuntimeImageTagOutputObj, 0, len(snapshots))
	for _, snap := range snapshots {
		results = append(results, new(serializer.RuntimeImageTagOutputObj).FromModel(snap))
	}

	ginutils.OK(c, serializer.ListRuntimeImageTagsOutput{
		Data: &serializer.PaginatedRuntimeImageTagOutputObjs{
			Count:          total,
			Results:        results,
			SnapshotStatus: new(serializer.SnapshotStatusInfoOutputObj).FromModel(status),
		},
	})
}
