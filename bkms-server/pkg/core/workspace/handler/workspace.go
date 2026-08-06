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

// Package handler contains Gin handlers for workspace APIs.
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/serializer"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/worker"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/task"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// Handler handles Gin workspace API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// GetUserStatistics 获取用户统计信息。
//
//	@ID				GetUserStatistics
//	@Summary		获取用户统计信息
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Success		200	{object}	serializer.GetUserStatisticsOutput
//	@Failure		400	{object}	bkerrs.GinErrorOutput
//	@Router			/user-statistics [get]
func (h *Handler) GetUserStatistics(c *gin.Context) {
	ctx := c.Request.Context()

	var totalAppCount, totalEnvCount int64

	// 获取所有工作空间列表
	workspaces, err := h.registry.WorkspaceStore.List(ctx, &workspace.ListOptions{})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list workspaces"))
		return
	}
	// 如果没有工作空间，则返回空统计数据，避免请求权限管理器
	if len(workspaces) == 0 {
		ginutils.OK(c, serializer.GetUserStatisticsOutput{
			Data: &serializer.UserStatisticsOutputObj{
				WorkspaceCount: 0,
				AppCount:       0,
				EnvCount:       0,
			},
		})
		return
	}

	// 转换成 ID 列表并做权限检查
	workspaceIDs := lo.Map(workspaces, func(w workspace.Workspace, _ int) string { return w.ID })
	hasPermWorkspaceIDs, err := perm.NewManager().FilterViewableWorkspaces(ctx, workspaceIDs)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "filter workspaces perm"))
		return
	}

	// 各个空间下的统计数据
	var statistics []*serializer.UserWorkspaceStatisticsOutputObj
	for _, ws := range workspaces {
		// 没有查看权限的工作空间，跳过
		if !hasPermWorkspaceIDs.Has(ws.ID) {
			continue
		}

		// 查询出工作空间下的应用数量
		apps, lErr := h.registry.AppStore.ListApps(ctx, &bkmsapp.ListOpts{WorkspaceID: ws.ID})
		if lErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(lErr, bkerrs.ErrCodeInternalServerError, "list apps"))
			return
		}
		workspaceAppCount := int64(len(apps))

		// 查询出工作空间下的环境数量
		envs, lErr := h.registry.EnvStore.ListStdEnvs(ctx, ws.ID)
		if lErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(lErr, bkerrs.ErrCodeInternalServerError, "list envs"))
			return
		}
		workspaceEnvCount := int64(len(envs))

		totalAppCount += workspaceAppCount
		totalEnvCount += workspaceEnvCount
		statistics = append(statistics, &serializer.UserWorkspaceStatisticsOutputObj{
			WorkspaceID: ws.ID,
			AppCount:    workspaceAppCount,
			EnvCount:    workspaceEnvCount,
		})
	}

	ginutils.OK(c, serializer.GetUserStatisticsOutput{
		Data: &serializer.UserStatisticsOutputObj{
			WorkspaceCount:      int64(len(statistics)),
			AppCount:            totalAppCount,
			EnvCount:            totalEnvCount,
			WorkspaceStatistics: statistics,
		},
	})
}

// ListWorkspaceRoleMemberGroups 获取工作空间下角色成员组列表。
//
//	@ID				ListWorkspaceRoleMemberGroups
//	@Summary		获取工作空间下角色成员组列表
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	serializer.ListWorkspaceRoleMemberGroupsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/role-member-groups [get]
func (h *Handler) ListWorkspaceRoleMemberGroups(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	client := perm.NewManager()
	var groups []*serializer.RoleMemberGroupOutputObj
	// 获取角色列表
	roles, err := client.ListRoles(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list roles"))
		return
	}
	for _, r := range roles {
		// 获取指定角色下的成员
		members, lErr := client.ListRoleMembers(ctx, r.ID)
		if lErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(lErr, bkerrs.ErrCodeInternalServerError, "list role members"))
			return
		}
		groups = append(groups, &serializer.RoleMemberGroupOutputObj{
			RoleID:      r.ID,
			RoleName:    r.Name,
			RoleCode:    r.RoleCode,
			UserGroupID: int64(r.UserGroupID),
			Members:     members,
		})
	}

	ginutils.OK(c, serializer.ListWorkspaceRoleMemberGroupsOutput{Data: groups})
}

// GetWorkspace 获取指定工作空间。
//
//	@ID				GetWorkspace
//	@Summary		获取指定工作空间的详细信息，包括基本信息、镜像仓库、蓝鲸关联系统信息等
//	@Description	[bkms-cli 使用] 避免破坏性修改
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	serializer.GetWorkspaceOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID} [get]
func (h *Handler) GetWorkspace(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	imageRegistry, err := h.registry.ImageRegistryStore.GetByWorkspaceAndType(ctx, ws.ID, ws.ImageRegistryType)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get image registry"))
		return
	}

	ginutils.OK(c, serializer.GetWorkspaceOutput{
		Data: new(serializer.WorkspaceDetailOutputObj).FromModel(*ws, imageRegistry),
	})
}

// ListWorkspaces 获取工作空间列表。
//
//	@ID				ListWorkspaces
//	@Summary		获取工作空间列表
//	@Description	[bkms-cli 使用] 避免破坏性修改
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			keyword	query		string	false	"搜索关键词"
//	@Success		200		{object}	serializer.ListWorkspacesOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces [get]
func (h *Handler) ListWorkspaces(c *gin.Context) {
	var queryInput serializer.ListWorkspacesQueryInput
	if err := c.ShouldBindQuery(&queryInput); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "bind query"))
		return
	}

	ctx := c.Request.Context()

	listOpts := &workspace.ListOptions{}
	if queryInput.Keyword != nil {
		listOpts.Keyword = *queryInput.Keyword
	}
	// 从数据库获取工作空间列表（支持关键字搜索）
	workspaces, err := h.registry.WorkspaceStore.List(ctx, listOpts)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list workspaces"))
		return
	}

	// 没有 workspace 时避免请求权限管理器mgr
	if len(workspaces) == 0 {
		ginutils.OK(c, serializer.ListWorkspacesOutput{Data: []*serializer.WorkspaceInfoOutputObj{}})
		return
	}

	workspaceIDs := lo.Map(workspaces, func(ws workspace.Workspace, _ int) string {
		return ws.ID
	})

	allowedWorkspaceSet, err := perm.NewManager().FilterViewableWorkspaces(ctx, workspaceIDs)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, "check workspace perm"))
		return
	}

	// 转换为响应对象
	outputList := make([]*serializer.WorkspaceInfoOutputObj, 0, len(workspaces))
	for i := range workspaces {
		// 过滤掉用户没有权限的工作空间
		if !allowedWorkspaceSet.Has(workspaces[i].ID) {
			continue
		}
		outputList = append(outputList, new(serializer.WorkspaceInfoOutputObj).FromModel(workspaces[i]))
	}

	ginutils.OK(c, serializer.ListWorkspacesOutput{Data: outputList})
}

// ListWorkspacesOverview 获取工作空间概览列表（含应用信息，按最近操作时间排序）。
//
//	@ID				ListWorkspacesOverview
//	@Summary		获取工作空间概览列表
//	@Description	含应用信息，按最近操作时间排序
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			limit	query		int		true	"返回的工作空间数量上限"
//	@Success		200		{object}	serializer.ListWorkspacesOverviewOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces-overview [get]
func (h *Handler) ListWorkspacesOverview(c *gin.Context) {
	var queryInput serializer.ListWorkspacesOverviewQueryInput
	if err := c.ShouldBindQuery(&queryInput); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "bind query"))
		return
	}

	ctx := c.Request.Context()
	username := auth.MustGetUser(ctx).ID

	// 查询所有 workspace 并按当前用户操作时间排序
	workspaces, err := workspace.ListSortByOpTime(
		ctx, h.registry.WorkspaceStore, h.registry.OperationRecordStore, username,
	)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list workspaces sorted by user op"),
		)
		return
	}
	if len(workspaces) == 0 {
		ginutils.OK(c, serializer.ListWorkspacesOverviewOutput{Data: []*serializer.WorkspaceWithAppsOutputObj{}})
		return
	}

	// 权限过滤
	workspaceIDs := lo.Map(workspaces, func(ws workspace.WorkspaceWithOpTime, _ int) string { return ws.ID })
	allowedWorkspaceSet, err := perm.NewManager().FilterViewableWorkspaces(ctx, workspaceIDs)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, "check workspace perm"))
		return
	}

	outputList := make([]*serializer.WorkspaceWithAppsOutputObj, 0, len(workspaces))
	for i := range workspaces {
		if !allowedWorkspaceSet.Has(workspaces[i].ID) {
			continue
		}
		ws := &workspaces[i]
		obj := &serializer.WorkspaceWithAppsOutputObj{
			ID:          ws.ID,
			DisplayName: ws.DisplayName,
			Description: ws.Description,
			State:       string(ws.State),
			Creator:     ws.Creator,
			CreatedAt:   ws.CreatedAt,
			Updater:     ws.Updater,
			UpdatedAt:   ws.UpdatedAt,
		}
		if !ws.LastOperatedAt.IsZero() {
			t := ws.LastOperatedAt
			obj.LastOperatedAt = &t
		}
		outputList = append(outputList, obj)
	}

	// 截取前 limit 个工作空间
	limit := int(queryInput.Limit)
	if limit > len(outputList) {
		limit = len(outputList)
	}
	outputList = outputList[:limit]

	// 构造部署状态查询服务
	deployStatusService := deploystatus.NewDeployStatusService(
		h.registry.AppStore,
		h.registry.EnvStore,
		h.registry.BuildAutoDeployRecordStore,
		h.registry.AppModelDeployRecordStore,
		h.registry.HelmDeployRecordStore,
	)

	// 查询每个 workspace 下的 app
	for idx := range outputList {
		wsObj := outputList[idx]
		appDetails, appErr := bkmsapp.ListSortByOpTime(
			ctx, h.registry.AppStore, h.registry.OperationRecordStore,
			wsObj.ID, username,
		)
		if appErr != nil {
			continue
		}

		// 权限过滤
		appIDs := lo.Map(appDetails, func(a bkmsapp.AppWithDetails, _ int) string { return a.ID })
		hasPermApps, permErr := perm.NewManager().FilterViewableApps(ctx, wsObj.ID, appIDs)
		if permErr != nil {
			continue
		}

		filteredAppDetails := lo.Filter(appDetails, func(a bkmsapp.AppWithDetails, _ int) bool {
			return hasPermApps.Has(a.ID)
		})

		// 查询 app 在各环境下的部署状态，返回按照 appID 分组
		deployStatusMap, depErr := deployStatusService.ListForAppsInWorkspace(
			ctx,
			wsObj.ID,
			lo.Map(
				filteredAppDetails,
				func(a bkmsapp.AppWithDetails, _ int) *bkmsapp.Application { return a.Application },
			),
		)
		if depErr != nil {
			continue
		}

		appList := make([]*serializer.AppInfoOutputObj, 0, len(filteredAppDetails))
		for _, ad := range filteredAppDetails {
			language := ""
			if ad.TrpcSpec != nil {
				language = ad.TrpcSpec.Language
			}
			appInfo := &serializer.AppInfoOutputObj{
				ID:          ad.ID,
				WorkspaceID: ad.WorkspaceID,
				Name:        ad.Name,
				Type:        ad.Type,
				DisplayName: ad.DisplayName,
				Creator:     ad.Creator,
				Language:    language,
				DeployedEnvs: lo.Map(deployStatusMap[ad.ID], func(
					row deploystatus.AppDeployStatus, _ int,
				) *serializer.AppDeployedEnvOutputObj {
					return &serializer.AppDeployedEnvOutputObj{
						ID:              row.EnvID,
						Name:            row.EnvName,
						DisplayName:     row.EnvDisplayName,
						Type:            row.EnvType,
						Kind:            row.EnvKind,
						TrafficLaneName: row.TrafficLaneName,
						DeployStatus:    row.DeployStatus,
						ImageTag:        row.ImageTag,
					}
				}),
			}
			if !ad.LastOperatedAt.IsZero() {
				t := ad.LastOperatedAt
				appInfo.LastOperatedAt = &t
			}
			appList = append(appList, appInfo)
		}
		wsObj.Apps = appList
	}

	ginutils.OK(c, serializer.ListWorkspacesOverviewOutput{Data: outputList})
}

// CreateWorkspace 创建工作空间。
//
//	@ID				CreateWorkspace
//	@Summary		创建工作空间
//	@Description	1. 级联创建/绑定蓝盾、BCS、镜像仓库、监控、日志等\n2. 初始化用户权限\n3. 写入 DB workspace\n4. 创建默认环境
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			body	body		serializer.CreateWorkspaceInput	true	"创建工作空间请求"
//	@Success		200		{object}	serializer.CreateWorkspaceOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces [post]
func (h *Handler) CreateWorkspace(c *gin.Context) {
	var input serializer.CreateWorkspaceInput
	if err := ginutils.BindJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	if input.BkCCBizID <= 0 && input.BkCIProjectID == "" {
		// bkccID 与 bkCIProjectID 不能同时为空
		bkerrs.AbortWithErr(
			c,
			bkerrs.New(bkerrs.ErrCodeInvalidRequest, "bkCCBizID and bkCIProjectID cannot both be empty"),
		)
		return
	}
	if input.BkCCBizID > 0 && input.BkCIProjectID != "" {
		// 新建容器项目时传 bkCCBizID，绑定已有容器项目时传 bkCIProjectID，两者不能同时传入
		bkerrs.AbortWithErr(
			c,
			bkerrs.New(bkerrs.ErrCodeInvalidRequest, "bkCCBizID and bkCIProjectID cannot both be provided"),
		)
		return
	}

	bkSystem, err := workspace.EnsureBkSystems(ctx, input.ID, input.BkCIProjectID, input.BkCCBizID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "ensure blueking system"))
		return
	}

	// 如果用户未指定镜像仓库, 则类型就是使用内置镜像仓库
	imageRegistryType := bkmsreg.ImageRegistryTypeBuiltin
	if reg := input.ImageRegistry; reg != nil {
		if err = workspace.CreateExternalImageRegistry(
			ctx,
			input.ID,
			bkSystem.BkCIProjectID,
			h.registry.ImageRegistryStore,
			reg.Registry,
			reg.Username,
			reg.Password,
		); err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "ensure image registry"))
			return
		}
		imageRegistryType = bkmsreg.ImageRegistryTypeExternal
	}

	// 初始化工作空间管理员用户
	managers := lo.Uniq(append(input.Managers, auth.MustGetUser(ctx).ID))
	if err = workspace.InitWorkspaceUser(ctx, input.ID, input.DisplayName, managers, *bkSystem); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init workspace user"))
		return
	}

	// 构建工作空间对象
	ws := &workspace.Workspace{
		ID:                input.ID,
		DisplayName:       input.DisplayName,
		Description:       input.Description,
		ImageRegistryType: imageRegistryType,
		// 默认不启用工作空间，需由异步任务检查到相关依赖初始化完成后再启用
		State:     workspace.StateProcessing,
		BkSystems: *bkSystem,
		Creator:   auth.MustGetUser(ctx).ID,
	}
	if err = h.registry.WorkspaceStore.Create(ctx, ws); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create workspace in DB"))
		return
	}

	// 创建工作空间默认环境（production, development, test 等）
	imageRegistry, err := workspace.GetWorkspaceImageRegistry(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get image registry"))
		return
	}
	envSvc := bkmsenv.NewEnvService(h.registry.EnvStore)
	defaultEnvs := workspace.BuildDefaultEnvs(auth.MustGetUser(ctx).ID, ws.ID, bkSystem.BkBCSProjectCode)
	for _, env := range defaultEnvs {
		if _, envErr := envSvc.Create(ctx, &env); envErr != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrapf(envErr, bkerrs.ErrCodeInternalServerError, "create default env %s", env.Name),
			)
			return
		}
	}

	// 异步轮询对应的监控项目状态
	if _, err = worker.ApplyTask(
		ctx,
		config.G.RabbitMQ.GetURI(),
		config.G.RabbitMQ.Queue,
		task.PollingWorkspaceInitStatus,
		task.PollingWorkspaceInitStatusArgs{
			WorkspaceID: ws.ID,
		},
	); err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "apply polling workspace status task"),
		)
		return
	}

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithDataAfter(ws),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.JSON(c, http.StatusOK, serializer.CreateWorkspaceOutput{
		Data: new(serializer.WorkspaceDetailOutputObj).FromModel(*ws, imageRegistry),
	})
}

// UpdateWorkspaceInfo 更新工作空间信息。
//
//	@ID				UpdateWorkspaceInfo
//	@Summary		更新工作空间信息
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string								true	"工作空间 ID"
//	@Param			body		body		serializer.UpdateWorkspaceInfoInput	true	"更新工作空间信息请求"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/info [put]
func (h *Handler) UpdateWorkspaceInfo(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var input serializer.UpdateWorkspaceInfoInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	oldWs := *ws
	// 更新字段
	ws.DisplayName = input.DisplayName
	ws.Description = input.Description
	ws.Updater = auth.MustGetUser(ctx).ID

	// 保存更新
	if err = h.registry.WorkspaceStore.Update(ctx, ws); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update workspace in DB"))
		return
	}

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithDataBefore(oldWs),
		audit.WithDataAfter(ws),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// AddWorkspaceUser 授予用户工作空间下角色身份。
//
//	@ID				AddWorkspaceUser
//	@Summary		授予用户工作空间下角色身份
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string							true	"工作空间 ID"
//	@Param			roleCode	path		string							true	"角色 Code"
//	@Param			body		body		serializer.AddWorkspaceUserInput	true	"添加用户请求"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/roles/{roleCode}/users [post]
func (h *Handler) AddWorkspaceUser(c *gin.Context) {
	var uriInput serializer.WorkspaceRoleURIInput
	var input serializer.AddWorkspaceUserInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	permMgr := perm.NewManager()

	role, err := permMgr.GetRole(ctx, ws.ID, uriInput.RoleCode)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get role"))
		return
	}

	if err = permMgr.AddRoleForUsers(ctx, role.ID, input.UserIDs); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "add role for users"))
		return
	}

	// 对于 admin/sre 角色的新增，交由 UserGroupService 基于 workspace 统一处理
	if uriInput.RoleCode == perm.RoleCodeAdmin || uriInput.RoleCode == perm.RoleCodeSre {
		go bkmonitor.NewUserGroupService(perm.NewManager(), h.registry.EnvStore).SyncMembersForWorkspace(
			ctx, ws, auth.MustGetUser(ctx).ID,
		)
	}

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeUserRole),
		audit.WithDataAfter(map[string]interface{}{
			"workspaceID": ws.ID,
			"roleCode":    uriInput.RoleCode,
			"userIDs":     input.UserIDs,
		}),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// RemoveWorkspaceUser 移除用户工作空间下角色身份。
//
//	@ID				RemoveWorkspaceUser
//	@Summary		移除用户工作空间下角色身份
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			userID		path		string	true	"用户 ID"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/users/{userID} [delete]
func (h *Handler) RemoveWorkspaceUser(c *gin.Context) {
	var uriInput serializer.WorkspaceUserURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	userRoleCode := ""
	permMgr := perm.NewManager()

	// 遍历所有角色, 如果用户有对应角色身份, 则删除用户在该角色下的身份
	for _, roleCode := range perm.RoleCodes() {
		role, err := permMgr.GetRole(ctx, ws.ID, roleCode)
		if err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get role"))
			return
		}

		members, err := permMgr.ListRoleMembers(ctx, role.ID)
		if err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list role members"))
			return
		}
		if !lo.Contains(members, uriInput.UserID) {
			continue
		}

		if err := permMgr.DeleteRoleForUsers(ctx, role.ID, []string{uriInput.UserID}); err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete role for users"))
			return
		}
		userRoleCode = roleCode
	}

	// 从各环境的蓝鲸监控告警组中移除该用户（仅处理 type=user）
	if userRoleCode == perm.RoleCodeAdmin || userRoleCode == perm.RoleCodeSre {
		go bkmonitor.NewUserGroupService(perm.NewManager(), h.registry.EnvStore).RemoveMemberForWorkspace(
			ctx, ws, uriInput.UserID, auth.MustGetUser(ctx).ID,
		)
	}

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeUserRole),
		audit.WithDataBefore(map[string]interface{}{
			"workspaceID": ws.ID,
			"userID":      uriInput.UserID,
		}),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// SetWorkspaceState 设置工作空间状态。
//
//	@ID				SetWorkspaceState
//	@Summary		设置工作空间状态
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string							true	"工作空间 ID"
//	@Param			body		body		serializer.SetWorkspaceStateInput	true	"设置工作空间状态请求"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/state [patch]
func (h *Handler) SetWorkspaceState(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var input serializer.SetWorkspaceStateInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	rawWs := *ws

	// 更新字段
	ws.State = workspace.State(input.State)
	ws.Updater = auth.MustGetUser(ctx).ID

	// 停用工作空间时，检查是否存在活跃的部署
	if workspace.State(input.State) == workspace.StateDisabled {
		hasActive, checkErr := h.hasActiveDeploymentsInWorkspace(ctx, ws.ID)
		if checkErr != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrap(checkErr, bkerrs.ErrCodeInternalServerError, "check active deployments"),
			)
			return
		}
		if hasActive {
			bkerrs.AbortWithErr(c,
				bkerrs.New(bkerrs.ErrCodeInvalidRequest,
					"workspace has active deployments, please remove them first"))
			return
		}
	}

	// 保存更新
	if err = h.registry.WorkspaceStore.Update(ctx, ws); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update workspace in DB"))
		return
	}

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithDataBefore(rawWs),
		audit.WithDataAfter(ws),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeleteWorkspace 删除工作空间。
//
//	@ID				DeleteWorkspace
//	@Summary		删除工作空间
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID} [delete]
func (h *Handler) DeleteWorkspace(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeDelete)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	// FIXME 检查工作空间是否有关联的环境, 如果有, 则提示错误阻止用户删除

	// 保留创建的镜像仓库/蓝盾项目等, 不做删除

	// 删除工作空间下所有空间组件
	if err = h.registry.WorkspaceCompsStore.DeleteByWorkspace(ctx, ws.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete workspace components"))
		return
	}

	// 删除工作空间下所有角色
	if err = perm.NewManager().DeleteAllRolesByWorkspaceID(ctx, ws.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete all roles by workspace ID"))
		return
	}

	// 删除工作空间
	if err = h.registry.WorkspaceStore.Delete(ctx, ws.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete workspace in DB"))
		return
	}

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithDataBefore(ws),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// hasActiveDeploymentsInWorkspace checks whether any app in the workspace has
// an active (non-uninstalled) deployment. It reuses the per-app
// HasActiveDeployments method from each deploy record store, matching the same
// pattern used when deleting an app.
func (h *Handler) hasActiveDeploymentsInWorkspace(ctx context.Context, workspaceID string) (bool, error) {
	apps, err := h.registry.AppStore.ListApps(ctx, &bkmsapp.ListOpts{WorkspaceID: workspaceID})
	if err != nil {
		return false, err
	}
	for _, app := range apps {
		if bkmsapp.IsAppModelType(app.Type) {
			hasActive, err := h.registry.AppModelDeployRecordStore.HasActiveDeployments(ctx, app.ID)
			if err != nil {
				return false, err
			}
			if hasActive {
				return true, nil
			}
		} else if bkmsapp.IsHelmBasedType(app.Type) {
			hasActive, err := h.registry.HelmDeployRecordStore.HasActiveDeployments(ctx, app.ID)
			if err != nil {
				return false, err
			}
			if hasActive {
				return true, nil
			}
		}
	}
	return false, nil
}
