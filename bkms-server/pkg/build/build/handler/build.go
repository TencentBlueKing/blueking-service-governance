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

// Package handler contains Gin handlers for build APIs.
package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build"
	buildserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build/serializer"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

var _ build.Handler = (*Handler)(nil)

// Handler handles Gin build API requests.
type Handler struct {
	registry   *storereg.Registry
	persistMgr *customruntime.PersistManager
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	snapshotService := snapshot.NewService(
		registry.SnapshotStore,
		registry.BuildConfigStore,
		registry.AppStore,
	)
	return &Handler{
		registry: registry,
		persistMgr: customruntime.NewPersistManager(
			registry.CustomRuntimeImageStore,
			snapshotService,
		),
	}
}

// newBuildService 装配构建服务，并注入平台构建镜像引用校验器
func (h *Handler) newBuildService() (*build.Service, error) {
	imageReferenceValidator := workloadruntime.NewImageReferenceValidator(
		h.registry.RuntimeImageStore,
		h.registry.SnapshotStore,
	)
	return build.NewService(
		h.registry.BuildConfigStore,
		h.registry.BuildRecordStore,
		imageReferenceValidator,
		h.persistMgr,
	)
}

// UpdateBuildConfig 更新应用构建配置。
//
//	@ID				UpdateBuildConfig
//	@Summary		更新应用构建配置
//	@Tags			builds
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string										true	"应用 ID"
//	@Param			body	body		buildserializer.UpdateBuildConfigInput	true	"更新构建配置请求"
//	@Success		200		{object}	buildserializer.UpdateBuildConfigOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-configs [put]
func (h *Handler) UpdateBuildConfig(c *gin.Context) {
	var uriInput buildserializer.AppURIInput
	var input buildserializer.UpdateBuildConfigInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	dataBefore, err := h.registry.BuildConfigStore.Get(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get app %s build config", app.ID))
		return
	}

	cfg, err := buildConfigFromInput(uriInput.AppID, input, dataBefore)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 校验放在所有副作用之前，避免非法配置已登记代码库或落库
	if err = h.validatePlatformBuildImages(ctx, app.WorkspaceID, cfg); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 登记代码库放在落库之前，蓝盾侧失败时本次配置整体不生效
	if err = h.ensureCodeRepositoryRegistered(ctx, app.WorkspaceID, cfg); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.registry.BuildConfigStore.Update(ctx, cfg); err != nil {
		bkerrs.AbortWithErr(
			c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "update app %s build config", app.ID),
		)
		return
	}

	// 收尾读已生效的配置，必须在落库成功之后
	h.scheduleBuildConfigUpdateFollowUps(ctx, app, dataBefore, cfg)

	ginutils.OK(c, buildserializer.UpdateBuildConfigOutput{
		Data: new(buildserializer.BuildConfigOutputObj).FromModel(cfg),
	})
}

// validatePlatformBuildImages 校验平台构建的 builder / runner 镜像，镜像源故障报内部错误，其余报参数错误
func (h *Handler) validatePlatformBuildImages(
	ctx context.Context, workspaceID string, cfg *imagebuild.Config,
) error {
	imageReferenceValidator := workloadruntime.NewImageReferenceValidator(
		h.registry.RuntimeImageStore,
		h.registry.SnapshotStore,
	)
	err := build.ValidatePlatformBuildImages(ctx, imageReferenceValidator, h.persistMgr, cfg, workspaceID)
	if err == nil {
		return nil
	}

	errCode := bkerrs.ErrCodeInvalidArgument
	if build.IsImageRegistryFailure(err) {
		errCode = bkerrs.ErrCodeInternalServerError
	}
	return bkerrs.New(errCode, err.Error())
}

// ensureCodeRepositoryRegistered 把代码库来源的构建配置登记到蓝盾，登记幂等，其余来源直接跳过
func (h *Handler) ensureCodeRepositoryRegistered(
	ctx context.Context, workspaceID string, cfg *imagebuild.Config,
) error {
	if cfg == nil || cfg.SourceType != imagebuild.SourceTypeCodeRepository || cfg.CodeRepo == nil {
		return nil
	}
	if err := bkci.EnsureWorkspaceRepositories(ctx, workspaceID, []bkci.RepositoryInitSpec{{
		URL:   cfg.CodeRepo.RepoURL,
		Alias: cfg.CodeRepo.RepoAlias,
	}}); err != nil {
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "ensure bkci repositories")
	}
	return nil
}

// scheduleBuildConfigUpdateFollowUps 异步收尾：快照刷新、自定义镜像落库、审计记录，失败只打日志不回滚
//
// WithoutCancel 重建 context，避免请求返回后协程被连带取消
func (h *Handler) scheduleBuildConfigUpdateFollowUps(
	ctx context.Context,
	app *bkmsapp.Application,
	dataBefore, cfg *imagebuild.Config,
) {
	go func() {
		rCtx := context.WithoutCancel(ctx)
		refreshErr := h.triggerSnapshotRefreshOnConfigChange(rCtx, app.ID, dataBefore, cfg)
		if refreshErr != nil {
			log.Errorf(rCtx, "trigger snapshot refresh on config change for app %s failed: %v", app.ID, refreshErr)
		}
	}()

	// 尚无成功快照的自定义镜像会在这里触发一次刷新；失败不回滚已保存的构建配置
	go func() {
		rCtx := context.WithoutCancel(ctx)
		if persistErr := h.persistMgr.PersistAfterSave(rCtx, app.WorkspaceID, cfg); persistErr != nil {
			log.Errorf(rCtx, "persist custom runtime images for workspace %s failed: %v", app.WorkspaceID, persistErr)
		}
	}()

	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeBuildConfig),
		audit.WithDataBefore(dataBefore),
		audit.WithDataAfter(cfg),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)
}

// ListBuildRecords 获取应用构建记录列表。
//
//	@ID				ListBuildRecords
//	@Summary		获取应用构建记录列表
//	@Tags			builds
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string	true	"应用 ID"
//	@Param			keyword		query		string	false	"搜索关键字"
//	@Param			page		query		int		true	"分页参数：页码，从 1 开始"
//	@Param			pageSize	query		int		true	"分页参数：每页数量，支持 5/10/20/50/100"
//	@Success		200			{object}	buildserializer.ListBuildRecordsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Failure		404			{object}	bkerrs.GinErrorOutput
//	@Failure		500			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/builds [get]
func (h *Handler) ListBuildRecords(c *gin.Context) {
	var uriInput buildserializer.AppURIInput
	var queryInput buildserializer.ListBuildRecordsQueryInput
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

	records, total, err := h.registry.BuildRecordStore.List(
		ctx,
		app.ID,
		queryInput.Keyword,
		queryInput.Page,
		queryInput.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list app %s build records", app.ID),
		)
		return
	}

	results := make([]*buildserializer.BuildRecordOutputObj, 0, len(records))
	for _, record := range records {
		results = append(results, new(buildserializer.BuildRecordOutputObj).FromModel(record))
	}

	ginutils.OK(c, buildserializer.ListBuildRecordsOutput{
		Data: &buildserializer.PaginatedBuildRecordOutputObjs{
			Count:   total,
			Results: results,
		},
	})
}

// CreateBuild 开始构建应用。
//
//	@ID				CreateBuild
//	@Summary		开始构建应用
//	@Tags			builds
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string							true	"应用 ID"
//	@Param			body	body		buildserializer.CreateBuildInput	true	"创建构建请求"
//	@Success		200		{object}	buildserializer.CreateBuildOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/builds [post]
func (h *Handler) CreateBuild(c *gin.Context) {
	var uriInput buildserializer.AppURIInput
	var input buildserializer.CreateBuildInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	buildService, err := h.newBuildService()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init build service"))
		return
	}

	buildRecord, err := build.StartAndScheduleBuild(
		ctx,
		buildService,
		app,
		input.Branch,
		input.ImageTag,
		build.StartOptions{},
	)
	if err != nil {
		// StartAndScheduleBuild 内部会跑构建前置校验，镜像引用填错或缺凭证属参数问题，
		// 与保存构建配置时的错误码保持一致；其余失败仍按内部错误上报
		errCode := bkerrs.ErrCodeInternalServerError
		if build.IsImageReferenceInvalid(err) || pkgerrors.Is(err, build.ErrWorkspaceImageCredentialMissing) {
			errCode = bkerrs.ErrCodeInvalidArgument
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, errCode, "start and schedule build"))
		return
	}

	ginutils.OK(c, buildserializer.CreateBuildOutput{
		Data: new(buildserializer.BuildRecordOutputObj).FromModel(*buildRecord),
	})
}

// GetRecommendedImageTag 获取应用推荐的镜像 Tag。
//
//	@ID				GetRecommendedImageTag
//	@Summary		获取应用推荐的镜像 Tag
//	@Tags			builds
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			branch	query		string	false	"分支/Tag（仅 custom 类型使用）"
//	@Success		200		{object}	buildserializer.GetRecommendedImageTagOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		404		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/recommended-image-tag [get]
func (h *Handler) GetRecommendedImageTag(c *gin.Context) {
	var uriInput buildserializer.AppURIInput
	var queryInput buildserializer.GetRecommendedImageTagQueryInput
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

	cfg, err := h.registry.BuildConfigStore.Get(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeNotFound, "get app %s build config", app.ID))
		return
	}

	var imageTag string
	switch cfg.TagConfig.Type {
	case imagebuild.VersionTypeSemver:
		repoInfo, repoErr := imagebuild.ResolveImageRepoInfo(ctx, cfg, app.WorkspaceID, app.Name)
		if repoErr != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrapf(repoErr, bkerrs.ErrCodeInternalServerError, "resolve image repo info for app %s", app.ID),
			)
			return
		}
		client := registry.New(repoInfo.Username, repoInfo.Password, true)
		tags, listErr := client.ListAllTags(ctx, repoInfo.RepoName)
		if listErr != nil {
			bkerrs.AbortWithErr(
				c, bkerrs.Wrapf(listErr, bkerrs.ErrCodeInternalServerError, "list tags for %s", repoInfo.RepoName),
			)
			return
		}
		imageTag = imagebuild.GenerateRecommendedSemverImageTag(tags)
	case imagebuild.VersionTypeCustom:
		imageTag = imagebuild.GenerateRecommendedCustomImageTag(
			queryInput.Branch,
			time.Now(),
			cfg.TagConfig.CustomOpts,
		)
	default:
		log.Warnf(ctx, "app %s has no recommended image tag config", app.ID)
	}

	ginutils.OK(c, buildserializer.GetRecommendedImageTagOutput{Data: imageTag})
}

// triggerSnapshotRefreshOnConfigChange refreshes image snapshots only when a
// build config change points the app to a different image repository.
func (h *Handler) triggerSnapshotRefreshOnConfigChange(
	ctx context.Context,
	appID string,
	oldCfg, newCfg *imagebuild.Config,
) error {
	app, err := h.registry.AppStore.GetApp(ctx, appID)
	if err != nil {
		return pkgerrors.Wrapf(err, "get app %s", appID)
	}

	oldRepoKey, err := computeRepoKeyFromConfig(ctx, oldCfg, app)
	if err != nil {
		return pkgerrors.Wrapf(err, "compute old repoKey for app %s", appID)
	}
	newRepoKey, err := computeRepoKeyFromConfig(ctx, newCfg, app)
	if err != nil {
		return pkgerrors.Wrapf(err, "compute new repoKey for app %s", appID)
	}
	if oldRepoKey == newRepoKey {
		return nil
	}

	snapshotService := snapshot.NewService(h.registry.SnapshotStore, h.registry.BuildConfigStore, h.registry.AppStore)
	if _, err = snapshotService.RefreshAppSnapshots(context.WithoutCancel(ctx), appID); err != nil {
		return pkgerrors.Wrapf(err, "refresh snapshots on config change for app %s", appID)
	}
	return nil
}

// computeRepoKeyFromConfig resolves the effective image repository from a build
// config and turns it into the repo key used by snapshot storage.
func computeRepoKeyFromConfig(
	ctx context.Context,
	cfg *imagebuild.Config,
	app *bkmsapp.Application,
) (string, error) {
	if cfg == nil {
		return "", nil
	}
	if app == nil {
		return "", pkgerrors.New("app is nil")
	}

	info, err := imagebuild.ResolveImageRepoInfo(ctx, cfg, app.WorkspaceID, app.Name)
	if err != nil {
		return "", pkgerrors.Wrap(err, "resolve image repo info")
	}
	return snapshot.GenerateRepoKey(info.RepoName, info.Username, info.Password), nil
}
