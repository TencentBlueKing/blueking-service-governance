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

// Package handler contains Gin handlers for workload image APIs.
package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	reginfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/promotion"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot/tagdeletion"
)

var _ image.Handler = (*Handler)(nil)

// Handler handles Gin build and image API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListAppImages 获取应用镜像列表（从本地快照查询，含已部署环境信息和晋级状态）。
//
//	@ID				ListAppImages
//	@Summary		获取应用镜像列表
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string	true	"应用 ID"
//	@Param			keyword		query		string	false	"搜索关键字"
//	@Param			page		query		int		true	"分页参数：页码，从 1 开始"
//	@Param			pageSize	query		int		true	"分页参数：每页数量，支持 5/10/20/50/100"
//	@Success		200			{object}	serializer.ListAppImagesOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Failure		500			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/images [get]
func (h *Handler) ListAppImages(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var queryInput serializer.ListAppImagesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	snapshotService := snapshot.NewService(h.registry.SnapshotStore, h.registry.BuildConfigStore, h.registry.AppStore)
	snapshots, total, status, err := snapshotService.ListAppSnapshots(
		ctx,
		uriInput.AppID,
		queryInput.Keyword,
		int(queryInput.Page),
		int(queryInput.PageSize),
	)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list app %s images", uriInput.AppID),
		)
		return
	}

	promotionMap, err := h.buildPromotionMap(ctx, uriInput.AppID, status)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "build promotion map for app %s", uriInput.AppID),
		)
		return
	}

	envTypeMap, err := h.registry.EnvStore.GetEnvTypeMap(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list environments"))
		return
	}
	productionEnvNames := productionEnvNames(envTypeMap)

	deployedEnvsMap, err := h.buildDeployedEnvsMap(ctx, app, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "build deployed envs map for app %s", uriInput.AppID),
		)
		return
	}

	repository := ""
	if status != nil {
		repository = status.RepoName
	}
	results := make([]*serializer.AppImageOutputObj, 0, len(snapshots))
	for _, snap := range snapshots {
		results = append(results, new(serializer.AppImageOutputObj).FromModel(
			snap,
			repository,
			promotionMap[snap.Tag],
			deployedEnvsMap[snap.Tag],
			envTypeMap,
		))
	}

	ginutils.OK(c, serializer.ListAppImagesOutput{
		Data: &serializer.PaginatedAppImagesOutputObjs{
			Count:              total,
			Results:            results,
			SnapshotStatus:     new(serializer.SnapshotStatusInfoOutputObj).FromModel(status),
			ProductionEnvNames: productionEnvNames,
		},
	})
}

// RefreshAppImages 手动刷新应用镜像快照。
//
//	@ID				RefreshAppImages
//	@Summary		手动刷新应用镜像快照
//	@Tags			images
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Success		200		{object}	serializer.RefreshAppImagesOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/images/refresh [post]
func (h *Handler) RefreshAppImages(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	snapshotService := snapshot.NewService(h.registry.SnapshotStore, h.registry.BuildConfigStore, h.registry.AppStore)
	result, err := snapshotService.RefreshAppSnapshots(ctx, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ginutils.OK(c, serializer.RefreshAppImagesOutput{
		Data: new(serializer.RefreshResultInfoOutputObj).FromModel(result),
	})
}

// PromoteAppImage 制品晋级（标记镜像为已晋级 / 生产就绪）。
//
//	@ID				PromoteAppImage
//	@Summary		制品晋级
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			tag		path		string	true	"镜像标签"
//	@Success		200		{object}	serializer.ImageEmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/images/{tag}/promote [patch]
func (h *Handler) PromoteAppImage(c *gin.Context) {
	var uriInput serializer.AppImageTagURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	snapshotService := snapshot.NewService(h.registry.SnapshotStore, h.registry.BuildConfigStore, h.registry.AppStore)
	info, err := snapshotService.ResolveRepoKeyForApp(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "resolve repo key for app %s", app.ID),
		)
		return
	}

	if err = h.registry.PromotionStore.Upsert(
		ctx, app.ID, info.RepoKey, uriInput.Tag, auth.MustGetUser(ctx).ID,
	); err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "promote image %s:%s", app.ID, uriInput.Tag),
		)
		return
	}

	ginutils.OK(c, serializer.ImageEmptyOutput{})
}

// ListAppImageUsages 返回指定镜像 Tag 当前仍可能命中的占用信息
// 该接口仅返回提示信息，不会影响删除接口是否允许执行。
//
//	@ID				ListAppImageUsages
//	@Summary		检查应用镜像使用情况
//	@Description	独立返回镜像 tag 在当前生效工作负载中的占用情况，供前端做删除前风险提示
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			tag		path		string	true	"镜像标签"
//	@Success		200		{object}	serializer.ListAppImageUsagesOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/images/{tag}/usages [get]
func (h *Handler) ListAppImageUsages(c *gin.Context) {
	var uriInput serializer.AppImageTagURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	svc := tagdeletion.NewService(
		h.registry.SnapshotStore,
		h.registry.PromotionStore,
		h.registry.BuildConfigStore,
		h.registry.AppStore,
		h.registry.EnvStore,
		h.registry.AppModelDeployRecordStore,
		h.registry.HelmDeployRecordStore,
	)
	usages, err := svc.ListImageUsages(ctx, uriInput.AppID, uriInput.Tag)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(
				err,
				bkerrs.ErrCodeInternalServerError,
				"check image usage %s for app %s",
				uriInput.Tag,
				uriInput.AppID,
			),
		)
		return
	}

	ginutils.OK(c, serializer.ListAppImageUsagesOutput{
		Data: new(serializer.ImageTagUsagesOutputObj).FromModels(usages),
	})
}

// DeleteAppImage 删除应用下的单个镜像 Tag。
// 该接口不做镜像占用校验；如需提示风险，请先调用 usages 接口。
//
//	@ID				DeleteAppImage
//	@Summary		删除应用镜像
//	@Description	直接删除远端镜像 tag，并清理本地快照与晋级记录；不依赖 usages 结果做拦截
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			tag		path		string	true	"镜像标签"
//	@Success		200		{object}	serializer.ImageEmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/images/{tag} [delete]
func (h *Handler) DeleteAppImage(c *gin.Context) {
	var uriInput serializer.AppImageTagURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	svc := tagdeletion.NewService(
		h.registry.SnapshotStore,
		h.registry.PromotionStore,
		h.registry.BuildConfigStore,
		h.registry.AppStore,
		h.registry.EnvStore,
		h.registry.AppModelDeployRecordStore,
		h.registry.HelmDeployRecordStore,
	)
	if err := svc.DeleteImageTag(ctx, uriInput.AppID, uriInput.Tag); err != nil {
		code := bkerrs.ErrCodeInternalServerError
		if reginfra.IsTagNotFound(err) {
			code = bkerrs.ErrCodeNotFound
		}
		wrappedErr := bkerrs.Wrapf(err, code, "delete image %s for app %s", uriInput.Tag, app.ID)
		if code == bkerrs.ErrCodeInternalServerError && errors.Is(err, tagdeletion.ErrImageRepoAuthRequired) {
			wrappedErr = wrappedErr.SetDetails(
				bkerrs.NewDetail(
					bkerrs.ErrDetailCodeImageRepositoryAuthRequired,
					"image repository authentication is required for deleting remote image tag",
					bkerrs.WithSystem(bkerrs.SystemName),
					bkerrs.WithModule(bkerrs.ModuleName),
				),
			)
		}
		bkerrs.AbortWithErr(c, wrappedErr)
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute("image"),
		audit.WithAppID(app.ID),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithDataBefore(map[string]any{
			"tag": uriInput.Tag,
		}),
	)

	ginutils.OK(c, serializer.ImageEmptyOutput{})
}

// ListImageTagDeployRecords 获取指定镜像 Tag 的部署记录列表。
//
//	@ID				ListImageTagDeployRecords
//	@Summary		获取指定镜像 Tag 的部署记录列表
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string	true	"应用 ID"
//	@Param			tag			path		string	true	"镜像标签"
//	@Param			page		query		int		true	"分页参数：页码，从 1 开始"
//	@Param			pageSize	query		int		true	"分页参数：每页数量，支持 5/10/20/50/100"
//	@Success		200			{object}	serializer.ListImageTagDeployRecordsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Failure		500			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/images/{tag}/deploy-records [get]
func (h *Handler) ListImageTagDeployRecords(c *gin.Context) {
	var uriInput serializer.AppImageTagURIInput
	var queryInput serializer.ListImageTagDeployRecordsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	var (
		total   int64
		results []*serializer.ImageTagDeployRecordOutputObj
	)
	if bkmsapp.IsAppModelType(app.Type) {
		records, count, listErr := h.registry.AppModelDeployRecordStore.ListByImageTag(
			ctx,
			uriInput.AppID,
			uriInput.Tag,
			queryInput.Page,
			queryInput.PageSize,
		)
		if listErr != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrapf(
					listErr,
					bkerrs.ErrCodeInternalServerError,
					"list app model deploy records for app %s tag %s",
					uriInput.AppID,
					uriInput.Tag,
				),
			)
			return
		}
		total = count
		results = make([]*serializer.ImageTagDeployRecordOutputObj, 0, len(records))
		for _, record := range records {
			results = append(results, new(serializer.ImageTagDeployRecordOutputObj).FromAppModelRecord(record))
		}
	} else {
		records, count, listErr := h.registry.HelmDeployRecordStore.ListByImageTag(
			ctx, uriInput.AppID, uriInput.Tag, queryInput.Page, queryInput.PageSize,
		)
		if listErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrapf(
				listErr, bkerrs.ErrCodeInternalServerError,
				"list helm deploy records for app %s tag %s", uriInput.AppID, uriInput.Tag,
			))
			return
		}
		total = count
		results = make([]*serializer.ImageTagDeployRecordOutputObj, 0, len(records))
		for _, record := range records {
			results = append(results, new(serializer.ImageTagDeployRecordOutputObj).FromHelmRecord(record))
		}
	}

	ginutils.OK(c, serializer.ListImageTagDeployRecordsOutput{
		Data: &serializer.PaginatedImageTagDeployRecordOutputObjs{
			Count:   total,
			Results: results,
		},
	})
}

// ListDeployableImageTags 获取应用在指定环境下可部署的镜像 TAG 列表。
//
//	@ID				ListDeployableImageTags
//	@Summary		获取应用在指定环境下可部署的镜像 TAG 列表
//	@Tags			images
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string	true	"应用 ID"
//	@Param			envName		path		string	true	"环境名称"
//	@Param			keyword		query		string	false	"搜索关键字（按 TAG 名称模糊搜索）"
//	@Param			page		query		int		true	"分页参数：页码，从 1 开始"
//	@Param			pageSize	query		int		true	"分页参数：每页数量，支持 5/10/20/50/100"
//	@Success		200			{object}	serializer.ListDeployableImageTagsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Failure		500			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/envs/{envName}/deployable-image-tags [get]
func (h *Handler) ListDeployableImageTags(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.ListDeployableImageTagsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	snapshotService := snapshot.NewService(h.registry.SnapshotStore, h.registry.BuildConfigStore, h.registry.AppStore)
	repoInfo, err := snapshotService.ResolveRepoKeyForApp(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "resolve repo key for app %s", app.ID),
		)
		return
	}

	page := int(queryInput.Page)
	pageSize := int(queryInput.PageSize)

	var (
		snapshots []snapshot.Image
		total     int64
	)
	if bkmsenv.IsProductionType(bkmsenv.Type(env.Type)) {
		promotedTags, listErr := h.registry.PromotionStore.ListTagsByAppAndRepoKey(ctx, app.ID, repoInfo.RepoKey)
		if listErr != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrapf(listErr, bkerrs.ErrCodeInternalServerError, "list promoted tags for app %s", app.ID),
			)
			return
		}
		snapshots, total, err = h.registry.SnapshotStore.ListByRepoKeyAndTags(
			ctx,
			repoInfo.RepoKey,
			promotedTags,
			queryInput.Keyword,
			page,
			pageSize,
		)
	} else {
		snapshots, total, err = h.registry.SnapshotStore.ListByRepoKey(
			ctx, repoInfo.RepoKey, queryInput.Keyword, page, pageSize,
		)
	}
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list deployable image tags for app %s env %s", app.ID, env.Name,
		))
		return
	}

	results := make([]*serializer.DeployableImageTagOutputObj, 0, len(snapshots))
	for _, snap := range snapshots {
		results = append(results, new(serializer.DeployableImageTagOutputObj).FromModel(snap))
	}

	ginutils.OK(c, serializer.ListDeployableImageTagsOutput{
		Data: &serializer.PaginatedDeployableImageTagOutputObjs{
			Count:   total,
			Results: results,
		},
	})
}

// buildPromotionMap loads promotion records for the current snapshot repo and
// indexes them by tag so response assembly can enrich each image in O(1).
func (h *Handler) buildPromotionMap(
	ctx context.Context,
	appID string,
	status *snapshot.RepoSnapshotStatus,
) (map[string]*promotion.Image, error) {
	if status == nil {
		return nil, nil
	}

	promotions, err := h.registry.PromotionStore.ListByAppAndRepoKey(ctx, appID, status.RepoKey)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*promotion.Image, len(promotions))
	for i := range promotions {
		result[promotions[i].Tag] = &promotions[i]
	}
	return result, nil
}

// buildDeployedEnvsMap groups deploy records by image tag. Environment
// existence and type enrichment are applied later when serializer output is
// assembled from the workspace env map.
func (h *Handler) buildDeployedEnvsMap(
	ctx context.Context,
	app *bkmsapp.Application,
	appID string,
) (map[string][]deploytypes.ImageTagEnvPair, error) {
	var (
		pairs []deploytypes.ImageTagEnvPair
		err   error
	)
	if bkmsapp.IsAppModelType(app.Type) {
		pairs, err = h.registry.AppModelDeployRecordStore.ListImageTagDeployedEnvs(ctx, appID)
	} else {
		pairs, err = h.registry.HelmDeployRecordStore.ListImageTagDeployedEnvs(ctx, appID)
	}
	if err != nil {
		return nil, err
	}

	result := make(map[string][]deploytypes.ImageTagEnvPair, len(pairs))
	for _, pair := range pairs {
		result[pair.ImageTag] = append(result[pair.ImageTag], pair)
	}
	return result, nil
}

// productionEnvNames extracts production environment names from the workspace
// env type map for frontend deploy guidance.
func productionEnvNames(envTypeMap map[string]string) []string {
	results := make([]string, 0, len(envTypeMap))
	for envName, envType := range envTypeMap {
		if bkmsenv.IsProductionType(bkmsenv.Type(envType)) {
			results = append(results, envName)
		}
	}
	return results
}
