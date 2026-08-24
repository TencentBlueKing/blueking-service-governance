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

// Package handler contains Gin handlers for app APIs.
package handler

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/martinlindhe/base36"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/overview"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// Handler handles Gin app API requests.
type Handler struct {
	registry   *storereg.Registry
	persistMgr *customruntime.PersistManager
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	snapshotService := snapshot.NewService(registry.SnapshotStore, registry.BuildConfigStore, registry.AppStore)
	return &Handler{
		registry: registry,
		persistMgr: customruntime.NewPersistManager(
			registry.CustomRuntimeImageStore,
			snapshotService,
		),
	}
}

// CreateApp 创建应用。
//
//	@ID			CreateApp
//	@Summary	创建应用
//	@Tags		app
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						true	"工作空间 ID"
//	@Param		body		body		serializer.CreateAppInput	true	"创建应用请求"
//	@Success	200			{object}	serializer.CreateAppOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps [post]
func (h *Handler) CreateApp(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var input serializer.CreateAppInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	// 先校验 workspace 存在，再校验创建应用权限，保证错误语义与其他 workspace 级接口一致。
	if _, err := ginperm.ValidateWorkspaceForAppCreate(ctx, h.registry, uriInput.WorkspaceID); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 检查应用是否已存在
	existingApp, err := h.registry.AppStore.GetAppByName(ctx, uriInput.WorkspaceID, input.Name)
	if existingApp != nil {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "app %s already exists", input.Name))
		return
	}
	if err != nil && !errors.Is(err, bkmsapp.ErrAppNotFound) {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "check app existence"))
		return
	}

	// 构建 Application 模型
	app := &bkmsapp.Application{
		ID:          input.ID,
		WorkspaceID: uriInput.WorkspaceID,
		Name:        input.Name,
		Type:        input.Type,
		DisplayName: input.Name,
		Creator:     auth.MustGetUser(ctx).ID,
		CreatedAt:   time.Now(),
	}

	// 设置类型特定字段
	if input.Type == bkmsapp.AppTypeTRPC {
		// TrpcSpec/AppModelSpec 由 struct-level validator 强制要求；
		// 这里保留 nil 防御，避免 validator 后续被改动时导致 panic。
		if input.AppModelSpec != nil && input.AppModelSpec.TrpcSpec != nil {
			app.TrpcSpec = &bkmsapp.TrpcSpec{Language: input.AppModelSpec.TrpcSpec.Language}
		}
	} else if bkmsapp.IsHelmBasedType(input.Type) {
		if len(input.HelmSpec.HelmSource.ValueFiles) == 0 {
			input.HelmSpec.HelmSource.ValueFiles = []string{"values.yaml"}
		}
		helmSpec := new(bkmsapp.HelmSpec)
		if err = mapstructure.Decode(input.HelmSpec, helmSpec); err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "parse helm spec"))
			return
		}
		if helmSpec.HelmSource != nil && helmSpec.HelmSource.HelmRepoConfig != nil {
			if err = helmSpec.HelmSource.HelmRepoConfig.SetUserPass(
				nil,
				input.HelmSpec.HelmSource.HelmRepoConfig.Username,
				input.HelmSpec.HelmSource.HelmRepoConfig.Password,
			); err != nil {
				bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "set user pass"))
				return
			}
		}
		app.HelmSpec = helmSpec
	}

	// 创建构建配置
	buildConfig, err := input.BuildConfig.ToModel(input.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "parse build config"))
		return
	}
	imageReferenceValidator := workloadruntime.NewImageReferenceValidator(
		h.registry.RuntimeImageStore,
		h.registry.SnapshotStore,
	)
	if err = build.ValidatePlatformBuildImages(
		ctx, imageReferenceValidator, h.persistMgr, buildConfig, app.WorkspaceID,
	); err != nil {
		// 镜像源鉴权失败或不可达时改镜像引用也没用，按内部错误上报；其余按参数问题
		errCode := bkerrs.ErrCodeInvalidArgument
		if build.IsImageRegistryFailure(err) {
			errCode = bkerrs.ErrCodeInternalServerError
		}
		bkerrs.AbortWithErr(c, bkerrs.New(errCode, err.Error()))
		return
	}
	if err = bkci.EnsureWorkspaceRepositories(
		ctx,
		uriInput.WorkspaceID,
		collectBKCIRepositoriesForCreateApp(input.Type, app.HelmSpec, buildConfig),
	); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "ensure bkci repositories"))
		return
	}
	if err = h.registry.BuildConfigStore.Create(ctx, buildConfig); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create build config"))
		return
	}

	// 根据应用类型创建特定资源（复用现有逻辑）
	if err = h.createAppByType(ctx, app, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 应用与构建配置都已落库后再异步 get_or_create 自定义镜像记录，尚无成功快照的镜像会触发一次刷新
	// 放在应用创建之后，避免创建失败时仍往工作空间写入候选镜像
	// 失败只打日志，不回滚已保存的数据；WithoutCancel 避免请求结束后 persist 被取消
	go func() {
		rCtx := context.WithoutCancel(ctx)
		if persistErr := h.persistMgr.PersistAfterSave(rCtx, app.WorkspaceID, buildConfig); persistErr != nil {
			log.Errorf(rCtx, "persist custom runtime images for workspace %s failed: %v", app.WorkspaceID, persistErr)
		}
	}()

	// 创建应用后，异步为该应用初始化默认监控告警策略，应用部署后真正下发到监控平台
	go func() {
		bgCtx := context.WithoutCancel(ctx)
		ws, wsErr := h.registry.WorkspaceStore.Get(bgCtx, app.WorkspaceID)
		if wsErr != nil {
			log.Errorf(bgCtx, "get workspace %s for default alert strategies failed: %v", app.WorkspaceID, wsErr)
			return
		}
		noticeGroupIDs, groupErr := usergroup.ResolveDefaultAlertNoticeGroupIDs(
			bgCtx,
			ws,
			usergroup.New(),
			perm.NewManager(),
			auth.MustGetUser(ctx).ID,
		)
		if groupErr != nil {
			log.Errorf(
				bgCtx,
				"resolve default alert user group for workspace %s failed: %v",
				app.WorkspaceID,
				groupErr,
			)
			return
		}
		initErr := alertstrategy.NewService(
			storereg.G().AlertStrategyStore,
			storereg.G().EnvStore,
			storereg.G().AppStore,
			storereg.G().ResourceSnapshotStore,
		).InitDefaultAlertStrategiesForApp(
			bgCtx, app.WorkspaceID, app.ID, app.Name, auth.MustGetUser(ctx).ID, noticeGroupIDs,
		)
		if initErr != nil {
			// 初始化失败不影响应用创建结果，仅记录错误日志，由后续重试/手动补偿处理。
			log.Errorf(bgCtx, "init default alert strategies for app %s failed: %v", app.ID, initErr)
		}
	}()

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithDataAfter(app),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.CreateAppOutput{
		Data: new(serializer.AppOutputObj).FromModel(app),
	})
}

// GetAppIDAutoSuffix 获取创建应用时使用的自动 ID 后缀。
//
//	@ID			GetAppIDAutoSuffix
//	@Summary	获取创建应用时使用的自动 ID 后缀
//	@Tags		app
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.GetAppIDAutoSuffixOutput
//	@Router		/apps/auto-id-suffix [get]
func (h *Handler) GetAppIDAutoSuffix(c *gin.Context) {
	suffix := newAppIDSuffix()
	ginutils.OK(c, serializer.GetAppIDAutoSuffixOutput{Suffix: suffix})
}

// newAppIDSuffix 生成用于拼接应用 ID 的后缀字符串，通常来说是一个格式为 "-xxxxxx" 的字符串。
// 注意：该函数并不保证应用使用该后缀生成的应用 ID 绝对不重复（当前使用随机数），调用方需要处理
// 可能的重复情况。
//
// - return: "-12ac55", nil
//
// 后缀去除 "-" 后，长度不超过 6 个字符，可能包含数字和小写字母。
func newAppIDSuffix() string {
	// AppID 自动后缀采用长度为 6 字符的 base36 编码，其中 zzzzzz 对应的十进制数是 2176782335，
	// 生成 [1, 2176782335] 范围的随机数。
	val := rand.IntN(2_176_782_335) + 1              // nolint: gosec
	s := strings.ToLower(base36.Encode(uint64(val))) // nolint
	return "-" + s
}

// GetApp 查询单个应用详情。
//
//	@ID			GetApp
//	@Summary	查询单个应用详情
//	@Tags		app
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.GetAppOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID} [get]
func (h *Handler) GetApp(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	buildConfig, err := h.registry.BuildConfigStore.Get(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get build config"))
		return
	}

	var appModel *appmodel.AppModel
	var components []serializer.ComponentOutputObj
	if bkmsapp.IsAppModelType(app.Type) {
		appModel, err = h.registry.AppModelStore.GetAppModel(ctx, app.ID)
		if err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get app model"))
			return
		}

		// NOTE: 对于引用了空间组件的应用组件，后端在实现上仅存储了引用关系。
		// 为了前端能够显示引用组件的相关信息，需要额外处理将空间组件的信息填入应用组件后再返回
		components, err = h.convertAppComponentsForOutput(ctx, app.WorkspaceID, appModel.Components)
		if err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "convert components"))
			return
		}
	}

	outputObj := new(serializer.AppDetailOutputObj).FromModel(app, buildConfig, appModel, components)
	ginutils.OK(c, serializer.GetAppOutput{Data: outputObj})
}

// ListApps 查询 app 列表，只返回基本信息。
//
//	@ID			ListApps
//	@Summary	查询 app 列表
//	@Tags		app
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		appName		query		string	false	"应用名过滤"
//	@Success	200			{object}	serializer.ListAppsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps [get]
func (h *Handler) ListApps(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var queryInput serializer.ListAppsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	opts := &bkmsapp.ListOpts{
		WorkspaceID: uriInput.WorkspaceID,
		AppName:     queryInput.AppName,
	}
	apps, err := h.registry.AppStore.ListApps(ctx, opts)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list apps"))
		return
	}

	if len(apps) == 0 {
		// 没有 app 时及时返回, 避免空请求权限管理器
		ginutils.OK(c, serializer.ListAppsOutput{Data: []*serializer.AppInfoOutputObj{}})
		return
	}

	appIDs := make([]string, 0, len(apps))
	for _, app := range apps {
		appIDs = append(appIDs, app.ID)
	}

	hasPermApps, err := perm.NewManager().FilterViewableApps(ctx, uriInput.WorkspaceID, appIDs)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "filter apps perm"))
		return
	}

	filteredApps := lo.Filter(apps, func(app *bkmsapp.Application, _ int) bool {
		return hasPermApps.Has(app.ID)
	})
	if len(filteredApps) == 0 {
		ginutils.OK(c, serializer.ListAppsOutput{Data: []*serializer.AppInfoOutputObj{}})
		return
	}

	username := auth.MustGetUser(ctx).ID
	filteredAppIDs := lo.Map(filteredApps, func(app *bkmsapp.Application, _ int) string { return app.ID })
	latestOpTimes, err := h.registry.OperationRecordStore.GetLatestOperationTimesByAppsForUser(
		ctx, uriInput.WorkspaceID, filteredAppIDs, username,
	)
	if err != nil {
		log.Warnf(ctx, "get latest operation times by apps for user: %s: %v", username, err)
		latestOpTimes = nil
	}

	// 查询应用在各环境下的部署状态
	deployStatusService := deploystatus.NewDeployStatusService(
		h.registry.AppStore,
		h.registry.EnvStore,
		h.registry.BuildAutoDeployRecordStore,
		h.registry.AppModelDeployRecordStore,
		h.registry.HelmDeployRecordStore,
	)
	deployStatusMap, err := deployStatusService.ListForAppsInWorkspace(ctx, uriInput.WorkspaceID, filteredApps)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list apps deploy statuses"))
		return
	}

	output := make([]*serializer.AppInfoOutputObj, 0, len(filteredApps))
	for _, app := range filteredApps {
		output = append(output, new(serializer.AppInfoOutputObj).FromModel(
			app, deployStatusMap[app.ID], latestOpTimes[app.ID],
		))
	}

	ginutils.OK(c, serializer.ListAppsOutput{Data: output})
}

// DeleteApp 删除单个应用。
//
//	@ID			DeleteApp
//	@Summary	删除单个应用
//	@Tags		app
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Success	200				{object}	serializer.EmptyOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID} 	[delete]
func (h *Handler) DeleteApp(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeDelete)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	operator := auth.MustGetUser(ctx).ID

	// 检查是否存在活跃的部署
	if bkmsapp.IsAppModelType(app.Type) {
		hasActive, iErr := h.registry.AppModelDeployRecordStore.HasActiveDeployments(ctx, app.ID)
		if iErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(iErr, bkerrs.ErrCodeInternalServerError, "check active deployments"))
			return
		}
		if hasActive {
			bkerrs.AbortWithErr(c, bkerrs.New(
				bkerrs.ErrCodeInvalidRequest, "app has active deployments, please remove them first",
			))
			return
		}
	} else if bkmsapp.IsHelmBasedType(app.Type) {
		hasActive, iErr := h.registry.HelmDeployRecordStore.HasActiveDeployments(ctx, app.ID)
		if iErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(iErr, bkerrs.ErrCodeInternalServerError, "check active deployments"))
			return
		}
		if hasActive {
			bkerrs.AbortWithErr(c, bkerrs.New(
				bkerrs.ErrCodeInvalidRequest, "app has active deployments, please remove them first",
			))
			return
		}
	}

	// 清理应用关联资源
	if err = h.cleanupAppResources(ctx, app.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "cleanup app resources"))
		return
	}
	// 根据应用类型删除特定资源
	if bkmsapp.IsAppModelType(app.Type) {
		if err = h.deleteAppModelApp(ctx, app); err != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete appmodel app resources"),
			)
			return
		}
	} else if bkmsapp.IsHelmBasedType(app.Type) {
		if err = h.deleteHelmApp(ctx, app); err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete helm app resources"))
			return
		}
	}

	// 删除应用基础数据
	if err = h.registry.AppStore.DeleteAppByName(ctx, app.WorkspaceID, app.Name); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete app"))
		return
	}
	h.cleanupAlertStrategiesAsync(ctx, app, operator)

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithDataBefore(app),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// UpdateAppDisplayName 更新应用显示名称。
//
//	@ID			UpdateAppDisplayName
//	@Summary	更新应用显示名称
//	@Tags		app
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		body	body		serializer.UpdateAppDisplayNameInput	true	"更新显示名请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/display-name [put]
func (h *Handler) UpdateAppDisplayName(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.UpdateAppDisplayNameInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.registry.AppStore.UpdateDisplayName(ctx, app, input.DisplayName); err != nil {
		if errors.Is(err, bkmsapp.ErrAppNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app %s not found", uriInput.AppID))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update app display name"))
		return
	}

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeDisplayName),
		audit.WithDataBefore(app.DisplayName),
		audit.WithDataAfter(input.DisplayName),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// GetAppDeployStatuses 查询应用在各环境及各泳道上的部署状态。
//
//	@ID			GetAppDeployStatuses
//	@Summary	查询应用在各环境及各泳道上的部署状态
//	@Tags		app
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.GetAppDeployStatusesOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deploy-statuses [get]
func (h *Handler) GetAppDeployStatuses(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	deployStatusService := deploystatus.NewDeployStatusService(
		h.registry.AppStore,
		h.registry.EnvStore,
		h.registry.BuildAutoDeployRecordStore,
		h.registry.AppModelDeployRecordStore,
		h.registry.HelmDeployRecordStore,
	)
	deployStatusMap, err := deployStatusService.ListForAppsInWorkspace(
		ctx,
		app.WorkspaceID,
		[]*bkmsapp.Application{app},
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list app deploy statuses"))
		return
	}

	ginutils.OK(c, serializer.GetAppDeployStatusesOutput{
		Data: lo.Map(
			deployStatusMap[app.ID],
			func(row deploystatus.AppDeployStatus, _ int) *serializer.AppDeployedEnvOutputObj {
				return new(serializer.AppDeployedEnvOutputObj).FromModel(row)
			},
		),
	})
}

// GetAppDeployOverview 查询 trpc/taf 应用在全部已关联环境上的部署总览。
//
//	@ID			GetAppDeployOverview
//	@Summary	查询应用在各环境上的部署总览（仅 trpc/taf）
//	@Tags		app
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.GetAppDeployOverviewOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deploy-overview [get]
func (h *Handler) GetAppDeployOverview(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if !bkmsapp.IsAppModelType(app.Type) {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(
			bkerrs.ErrCodeInvalidArgument,
			"deploy overview is only supported for trpc/taf apps, got type %q",
			app.Type,
		))
		return
	}

	svc := overview.NewService(
		h.registry.EnvStore,
		h.registry.AppStore,
		h.registry.AppModelStore,
		h.registry.BuildAutoDeployRecordStore,
		h.registry.AppModelDeployRecordStore,
		h.registry.HelmDeployRecordStore,
		h.registry.GPAConfigStore,
	)
	result, err := svc.GetOverview(ctx, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "build app deploy overview"))
		return
	}

	ginutils.OK(c, new(serializer.GetAppDeployOverviewOutput).FromModel(result))
}

// createAppByType 根据应用类型创建特定资源
func (h *Handler) createAppByType(
	ctx context.Context,
	app *bkmsapp.Application,
	input *serializer.CreateAppInput,
) error {
	switch app.Type {
	case bkmsapp.AppTypeTRPC:
		if err := h.createTrpcApp(ctx, app, input.AppModelSpec); err != nil {
			return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create trpc app resources")
		}
	case bkmsapp.AppTypeTAF:
		if err := h.createTafApp(ctx, app, input.AppModelSpec); err != nil {
			return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create taf app resources")
		}
	// Helm，Agones 底层实现都是基于 Helm Chart
	case bkmsapp.AppTypeHelm, bkmsapp.AppTypeAgones:
		if err := h.createHelmApp(ctx, app); err != nil {
			return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create helm app resources")
		}
	default:
		return bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "unsupported app type: %s", app.Type)
	}
	return nil
}

// deleteAppModelApp 删除 AppModel 类型应用资源
// 由于不同 workloadType 的删除流程一致，因此不需要根据 workloadType 分发
func (h *Handler) deleteAppModelApp(ctx context.Context, app *bkmsapp.Application) error {
	// 1. 删除 AppModel
	if err := h.registry.AppModelStore.DeleteAppModel(ctx, app.ID); err != nil {
		return errors.Wrapf(err, "delete app(%s) model", app.Name)
	}

	// 2. 删除所有关联的配置文件（AppConfigFile）
	if _, err := h.registry.AppConfigFileStore.DeleteByApp(ctx, app.ID); err != nil {
		return errors.Wrapf(err, "delete app(%s) config files", app.Name)
	}

	// 3. 删除北极星配置
	if err := h.registry.PolarisConfigStore.DeleteByApp(ctx, app.ID); err != nil {
		return errors.Wrapf(err, "delete app(%s) polaris configs", app.Name)
	}

	// 删除应用规格配置
	if err := h.registry.AppSpecStore.DeleteByApp(ctx, app.ID); err != nil {
		return errors.Wrap(err, "delete app specs")
	}

	return nil
}

// cleanupAppResources 清理应用关联的所有资源
// 注意：此函数不删除应用本身，仅清理关联资源
func (h *Handler) cleanupAppResources(ctx context.Context, appID string) error {
	// TODO 确认是否需要删除蓝盾侧数据
	// 删除构建配置
	if err := h.registry.BuildConfigStore.Delete(ctx, appID); err != nil {
		return errors.Wrap(err, "delete build config")
	}

	// 删除镜像晋级记录
	if err := h.registry.PromotionStore.DeleteByApp(ctx, appID); err != nil {
		return errors.Wrap(err, "delete promotions")
	}

	return nil
}

// cleanupAlertStrategiesAsync 异步清理已删除应用关联的全部告警策略。
func (h *Handler) cleanupAlertStrategiesAsync(
	ctx context.Context,
	app *bkmsapp.Application,
	operator string,
) {
	go func() {
		bgCtx := context.WithoutCancel(ctx)
		ws, err := h.registry.WorkspaceStore.Get(bgCtx, app.WorkspaceID)
		if err != nil {
			log.Errorf(bgCtx, "get workspace %s for app %s alert cleanup failed: %v", app.WorkspaceID, app.ID, err)
			return
		}

		err = alertstrategy.NewService(
			h.registry.AlertStrategyStore,
			h.registry.EnvStore,
			h.registry.AppStore,
			h.registry.ResourceSnapshotStore,
		).DeleteByApp(bgCtx, ws, app.WorkspaceID, app.ID, operator)
		if err != nil {
			log.Errorf(bgCtx, "cleanup alert strategies for deleted app %s failed: %v", app.ID, err)
		}
	}()
}

func (h *Handler) convertAppComponentsForOutput(
	ctx context.Context,
	workspaceID string,
	appComponents []*component.Component,
) ([]serializer.ComponentOutputObj, error) {
	var workspaceComponentsMap map[string]*workspace.Component
	if hasWorkspaceComponentRef(appComponents) {
		workspaceComponents, err := h.registry.WorkspaceCompsStore.ListByWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		workspaceComponentsMap = make(map[string]*workspace.Component, len(workspaceComponents))
		for _, workspaceComp := range workspaceComponents {
			workspaceComponentsMap[workspaceComp.Name] = workspaceComp
		}
	}

	output := make([]serializer.ComponentOutputObj, 0, len(appComponents))
	for _, appComponent := range appComponents {
		output = append(output, convertAppComponentForOutput(ctx, appComponent, workspaceComponentsMap))
	}
	return output, nil
}

func hasWorkspaceComponentRef(appComponents []*component.Component) bool {
	for _, appComponent := range appComponents {
		if appComponent != nil && appComponent.RefWorkspaceCompName != "" {
			return true
		}
	}
	return false
}

func convertAppComponentForOutput(
	ctx context.Context,
	appComponent *component.Component,
	workspaceComponentsMap map[string]*workspace.Component,
) serializer.ComponentOutputObj {
	if appComponent == nil {
		return serializer.ComponentOutputObj{}
	}
	if appComponent.RefWorkspaceCompName != "" {
		// convert referenced component for output
		workspaceComponent, ok := workspaceComponentsMap[appComponent.RefWorkspaceCompName]
		if !ok {
			// 通常情况下应该存在对应的空间组件，这里为了避免异常情况下影响接口返回，增加托底逻辑
			log.Warnf(ctx, "workspace component %s not found", appComponent.RefWorkspaceCompName)
			workspaceComponent = &workspace.Component{}
		}

		return serializer.ComponentOutputObj{
			Name:                 appComponent.Name,
			RefWorkspaceCompName: appComponent.RefWorkspaceCompName,
			Type:                 workspaceComponent.Type,
			Version:              workspaceComponent.Version,
			Properties:           workspaceComponent.Properties,
			ScopeType:            string(workspaceComponent.ScopeType),
			ScopeEnvNames:        workspaceComponent.ScopeEnvNames,
		}
	}
	// convert instantiated component for output
	return serializer.ComponentOutputObj{
		Name:       appComponent.Name,
		Type:       appComponent.Type,
		Version:    appComponent.Version,
		Properties: appComponent.Properties,
		// 组件实例默认全局生效
		ScopeType: string(component.ScopeTypeGlobal),
	}
}
