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

// Package handler contains Gin handlers for env APIs.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/serializer"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler handles Gin env API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) newDeployStatusService() *deploystatus.DeployStatusService {
	return deploystatus.NewDeployStatusService(
		h.registry.AppStore,
		h.registry.EnvStore,
		h.registry.BuildAutoDeployRecordStore,
		h.registry.AppModelDeployRecordStore,
		h.registry.HelmDeployRecordStore,
	)
}

// CreateEnv 创建部署环境。
//
//	@ID				CreateEnv
//	@Summary		创建部署环境
//	@Tags			env
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string							true	"工作空间 ID"
//	@Param			body		body		serializer.CreateEnvInput		true	"创建环境请求"
//	@Success		200			{object}	serializer.CreateEnvOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/envs [post]
func (h *Handler) CreateEnv(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var input serializer.CreateEnvInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	// 参数 & 权限校验
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = perm.NewManager().HasCreateEnvPerm(ctx, ws.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.WrapIAMNoPermission(err, ws.ID, "check app perm"))
		return
	}

	// 创建环境(包括内置环境变量)
	creator := auth.MustGetUser(ctx).ID
	svc := bkmsenv.NewEnvService(h.registry.EnvStore)

	envID, err := svc.Create(ctx, &envmodel.Environment{
		Name:        input.Name,
		WorkspaceID: ws.ID,
		Type:        input.Type,
		DisplayName: input.DisplayName,
		Description: input.Description,
		Cluster: envmodel.BizCluster{
			ProjectCode:  ws.BkSystems.BkBCSProjectCode,
			ClusterID:    input.Cluster.ClusterID,
			ClusterType:  input.Cluster.ClusterType,
			Namespace:    input.Cluster.Namespace,
			IsFederation: bkmsenv.IsFederationCluster(input.Cluster.ClusterID),
		},
		Creator: creator,
	})
	if err != nil {
		metrics.CreateEnvFailed(ws.ID, input.Name)
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidRequest, "create env %s", input.Name))
		return
	}

	// 目前存在两条路径往环境中写入 apm 相关环境变量:
	// - 在环境创建时，如果对应监控项目已经就绪, 则同步创建 ApmApp 并添加 apm 相关环境变量
	// - 在工作空间创建时的异步任务中，会定时检查监控项目状态。待其就绪时，会为空间下的所有环境创建 ApmApp 并添加 apm 相关环境变量
	if ws.BkSystems.BkMonitorProjectID != "" {
		apmSvc := bkmonitor.NewApmService(h.registry.ApmInstConfigStore, h.registry.ScopedEnvVarStore)
		bkMonitorProjectID := cast.ToInt64(ws.BkSystems.BkMonitorProjectID)

		if input.ApmID == nil || *input.ApmID == 0 {
			// apmID 为空，按默认逻辑创建同名 APM 并绑定
			if _, cErr := apmSvc.CreateAndBindToEnv(
				ctx, envID, input.Name, ws.BkSystems.BkBCSProjectCode,
				bkmonitor.CreateApmInstParams{
					WorkspaceID:  ws.ID,
					Username:     creator,
					BkmProjectID: bkMonitorProjectID,
				},
			); cErr != nil {
				bkerrs.AbortWithErr(c, bkerrs.Wrap(cErr, bkerrs.ErrCodeInternalError, "create and bind apm to env"))
				return
			}
			// APM 创建成功后，异步将 workspace 下 admin/sre 人员同步到新告警组（内部自带重试与延迟）
			go bkmonitor.NewUserGroupService(perm.NewManager(), h.registry.EnvStore).SyncMembersForEnvWithRetry(
				ctx, ws, input.Name, creator,
			)
		} else {
			// apmID 不为空，获取指定 APM 并绑定
			apm, cErr := apmSvc.Get(ctx, *input.ApmID, bkmonitor.CreateApmInstParams{
				WorkspaceID:  ws.ID,
				Username:     creator,
				BkmProjectID: bkMonitorProjectID,
			})
			if cErr != nil {
				bkerrs.AbortWithErr(c, bkerrs.Wrap(cErr, bkerrs.ErrCodeInternalError, "get or create apm"))
				return
			}
			if cErr = apmSvc.BindToEnv(ctx, apm, envID, input.Name); cErr != nil {
				bkerrs.AbortWithErr(c, bkerrs.Wrap(cErr, bkerrs.ErrCodeInternalError, "bind apm to env"))
				return
			}
		}
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeEnv,
		envID.Hex(),
		audit.WithDataAfter(input),
		audit.WithWorkspaceID(ws.ID),
		audit.WithEnvName(input.Name),
	)

	ginutils.OK(c, serializer.CreateEnvOutput{
		Data: &serializer.EnvIDOutput{ID: envID.Hex()},
	})
}

// CreateFeatureEnv 创建应用特性环境。
//
//	@ID				CreateFeatureEnv
//	@Summary		创建应用特性环境
//	@Tags			env
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string								true	"应用 ID"
//	@Param			body	body		serializer.CreateFeatureEnvInput		true	"创建特性环境请求"
//	@Success		200		{object}	serializer.CreateFeatureEnvOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/feat-envs [post]
func (h *Handler) CreateFeatureEnv(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.CreateFeatureEnvInput
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

	// Require source-env deploy permission; standard-kind / same-workspace checks
	// happen later in FeatureEnvService.Create.
	sourceEnv, err := ginperm.ValidateEnvByID(ctx, h.registry, input.SourceEnvID, ginperm.TypeDeploy)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	svc := bkmsenv.NewFeatureEnvService(
		h.registry.EnvStore,
		h.registry.FeatureEnvCounterStore,
		bkmsenv.NewFeatureEnvNamespaceInitializer(),
	)
	featureEnv, err := svc.Create(ctx, bkmsenv.CreateFeatureEnvInput{
		App:         app,
		SourceEnv:   sourceEnv,
		DisplayName: input.DisplayName,
		Creator:     auth.MustGetUser(ctx).ID,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "create feature environment"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeEnv,
		featureEnv.ID.Hex(),
		audit.WithDataAfter(featureEnv),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithEnvName(featureEnv.Name),
	)

	ginutils.OK(c, serializer.CreateFeatureEnvOutput{
		Data: new(serializer.EnvOutput).FromModel(*featureEnv),
	})
}

// ListEnvs 获取空间下的环境列表。
//
//	@ID				ListEnvs
//	@Summary		获取空间下的环境列表
//	@Description	[bkms-cli 使用] 避免破坏性修改
//	@Tags			env
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string					true	"工作空间 ID"
//	@Success		200			{object}	serializer.ListEnvsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/envs [get]
func (h *Handler) ListEnvs(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	if err := perm.NewManager().HasViewWorkspacePerm(ctx, uriInput.WorkspaceID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.WrapIAMNoPermission(err, uriInput.WorkspaceID, "check workspace perm"))
		return
	}

	svc := bkmsenv.NewEnvService(h.registry.EnvStore)
	envs, err := svc.ListStdEnvs(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "list envs"))
		return
	}

	ginutils.OK(c, serializer.ListEnvsOutput{
		Data: lo.Map(envs, func(env envmodel.Environment, _ int) *serializer.EnvOutput {
			return new(serializer.EnvOutput).FromModel(env)
		}),
	})
}

// ListAppEnvs 获取应用可用的环境列表。
//
//	@ID				ListAppEnvs
//	@Summary		获取应用可用环境列表，包含工作空间下的标准环境以及应用专用的特性环境。
//	@Tags			env
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string					true	"应用 ID"
//	@Success		200		{object}	serializer.ListEnvsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/envs [get]
func (h *Handler) ListAppEnvs(c *gin.Context) {
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

	envs, err := h.registry.EnvStore.ListAppEnvs(ctx, app.WorkspaceID, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "list app envs"))
		return
	}

	ginutils.OK(c, serializer.ListEnvsOutput{
		Data: lo.Map(envs, func(env envmodel.Environment, _ int) *serializer.EnvOutput {
			return new(serializer.EnvOutput).FromModel(env)
		}),
	})
}

// ListFeatureEnvs 获取应用的特性环境管理列表。
//
//	@ID			ListFeatureEnvs
//	@Summary		获取应用的所有特性环境
//	@Description	返回管理页展示所需的特性环境、来源环境、部署位置及创建信息。
//	@Tags			env
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			with_deploy_status	query		bool	false	"是否返回当前应用在每个特性环境下的部署状态"
//	@Success		200		{object}	serializer.ListFeatureEnvsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		500		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/feat-envs [get]
func (h *Handler) ListFeatureEnvs(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var queryInput serializer.ListFeatureEnvsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	featureEnvs, sourceEnvByFeatEnvName, err := bkmsenv.ListAppFeatEnvs(ctx, h.registry.EnvStore, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "list feature envs"))
		return
	}

	// nil 表示调用方未请求部署状态；非 nil map 表示已请求，序列化时会将无部署记录转换为 []。
	var deployStatusByFeatEnvName map[string][]deploystatus.AppDeployStatus
	if queryInput.WithDeployStatus && len(featureEnvs) > 0 {
		deployStatusByFeatEnvName, err = h.newDeployStatusService().ListFeatureEnvsForApp(ctx, app, featureEnvs)
		if err != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrap(err, bkerrs.ErrCodeInternalError, "list feature env deploy statuses"),
			)
			return
		}
	}

	ginutils.OK(c, serializer.NewListFeatureEnvsOutput(
		featureEnvs,
		sourceEnvByFeatEnvName,
		deployStatusByFeatEnvName,
	))
}

// GetEnv 获取环境详情。
//
//	@ID				GetEnv
//	@Summary		获取单个环境详情
//	@Tags			env
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string				true	"环境 ID"
//	@Success		200		{object}	serializer.GetEnvOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/envs/{envID} [get]
func (h *Handler) GetEnv(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := ginperm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	deployStatuses, err := h.newDeployStatusService().ListForEnvironment(ctx, env)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalError, "list for environment"))
		return
	}

	ginutils.OK(c, serializer.GetEnvOutput{
		Data: new(serializer.EnvDetailOutput).FromModel(*env, deployStatuses),
	})
}

// UpdateEnvBasicInfo 更新部署环境基本信息。
//
//	@ID				UpdateEnvBasicInfo
//	@Summary		更新部署环境基本信息
//	@Tags			env
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string								true	"环境 ID"
//	@Param			body	body		serializer.UpdateEnvBasicInfoInput	true	"更新环境基本信息请求"
//	@Success		200		{object}	serializer.EmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/envs/{envID}/basic-info [put]
func (h *Handler) UpdateEnvBasicInfo(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	var input serializer.UpdateEnvBasicInfoInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := ginperm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	updateData := &envmodel.EnvironmentUpdateData{}
	if input.DisplayName != nil {
		updateData.DisplayName = input.DisplayName
	}
	if input.Type != nil {
		updateData.Type = input.Type
	}

	svc := bkmsenv.NewEnvService(h.registry.EnvStore)
	if err = svc.Update(ctx, env.ID, updateData); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "update env basic info"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeEnv,
		env.ID.Hex(),
		audit.WithDataBefore(env),
		audit.WithDataAfter(updateData),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// UpdateEnvCluster 更新部署环境集群配置。
//
//	@ID				UpdateEnvCluster
//	@Summary		更新部署环境集群配置
//	@Tags			env
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string							true	"环境 ID"
//	@Param			body	body		serializer.UpdateEnvClusterInput	true	"更新环境集群配置请求"
//	@Success		200		{object}	serializer.EmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/envs/{envID}/cluster [put]
func (h *Handler) UpdateEnvCluster(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	var input serializer.UpdateEnvClusterInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := ginperm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	updateData := &envmodel.EnvironmentUpdateData{}
	if input.ClusterID != "" {
		isFederation := bkmsenv.IsFederationCluster(input.ClusterID)
		updateData.ClusterID = &input.ClusterID
		updateData.ClusterType = &input.ClusterType
		updateData.IsFederation = &isFederation
	}
	if input.Namespace != "" {
		updateData.Namespace = &input.Namespace
	}

	svc := bkmsenv.NewEnvService(h.registry.EnvStore)
	if err = svc.Update(ctx, env.ID, updateData); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "update env cluster"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeEnv,
		env.ID.Hex(),
		audit.WithDataBefore(env.Cluster),
		audit.WithDataAfter(updateData),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeleteEnv 删除环境。
//
//	@ID			DeleteEnv
//	@Summary	删除环境
//	@Tags		env
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string	true	"环境 ID"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID} [delete]
func (h *Handler) DeleteEnv(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := ginperm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, ginperm.TypeDelete)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	envID := env.ID

	if err = h.registry.ApmInstConfigStore.UnbindEnvFromAll(ctx, envID); err != nil {
		log.Warnf(ctx, "remove env %s from all APMs failed: %v", envID.Hex(), err)
	}

	svc := bkmsenv.NewEnvService(h.registry.EnvStore)
	if err = svc.Delete(ctx, envID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete environment"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeEnv,
		envID.Hex(),
		audit.WithDataBefore(env),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// ListEnvTrafficLanes 获取指定环境下的泳道列表。
//
//	@ID				ListEnvTrafficLanes
//	@Summary		获取指定环境下的泳道列表
//	@Tags			env
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string							true	"工作空间 ID"
//	@Param			envName		path		string							true	"环境名称"
//	@Success		200			{object}	serializer.ListEnvTrafficLanesOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/envs/{envName}/traffic-lanes [get]
func (h *Handler) ListEnvTrafficLanes(c *gin.Context) {
	var uriInput serializer.WorkspaceEnvNameURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	if err := perm.NewManager().HasViewWorkspacePerm(ctx, uriInput.WorkspaceID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.WrapIAMNoPermission(err, uriInput.WorkspaceID, "check app perm"))
		return
	}

	lanes, err := trafficmanager.New().ListTrafficLanes(ctx, uriInput.WorkspaceID, uriInput.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError,
			"list workspace %s env %s traffic lanes", uriInput.WorkspaceID, uriInput.EnvName,
		))
		return
	}

	ginutils.OK(c, serializer.ListEnvTrafficLanesOutput{
		Data: lo.Map(lanes, func(lane *trafficmanager.TrafficLane, _ int) *serializer.TrafficLaneOutput {
			return new(serializer.TrafficLaneOutput).FromModel(lane)
		}),
	},
	)
}
