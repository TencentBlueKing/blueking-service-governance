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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/dashboard"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/dashboard/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// ListAppDashboards 获取应用绑定的仪表盘列表
//
//	@ID			ListAppDashboards
//	@Summary	获取应用绑定的仪表盘列表
//	@Tags		bkmonitor-dashboard
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.ListDashboardsResp
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bkmonitor/dashboards [get]
func (h *Handler) ListAppDashboards(c *gin.Context) {
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

	ws, err := h.workspaces().Get(ctx, workspace.ByWorkspaceID(app.WorkspaceID))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}

	items, err := h.service.List(ctx, ws, app.ID, auth.MustGetUser(ctx).ID)
	if err != nil {
		bkerrs.AbortWithErr(c, h.wrapError(err, "list app dashboards"))
		return
	}

	ginutils.OK(c, &serializer.ListDashboardsResp{Data: serializer.NewDashboardOutputs(items)})
}

// CreateAppDashboard 创建应用仪表盘绑定。
//
//	@ID			CreateAppDashboard
//	@Summary	创建应用仪表盘绑定
//	@Tags		bkmonitor-dashboard
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		body	body	serializer.AppDashboardCreateInput	true	"绑定请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	409		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bkmonitor/dashboards [post]
func (h *Handler) CreateAppDashboard(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var bodyInput serializer.AppDashboardCreateInput
	if err := ginutils.BindURIJSON(c, &uriInput, &bodyInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ws, err := h.workspaces().Get(ctx, workspace.ByWorkspaceID(app.WorkspaceID))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}

	if err = h.service.Create(ctx, ws, app.ID, bodyInput.UID, auth.MustGetUser(ctx).ID); err != nil {
		bkerrs.AbortWithErr(c, h.wrapError(err, "create app dashboard"))
		return
	}

	go audit.AddOperationRecordAsync(ctx,
		audit.OperationTypeCreate, audit.ResourceTypeDashboard, bodyInput.UID,
		audit.WithDataAfter(bodyInput), audit.WithWorkspaceID(ws.ID), audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// UpdateAppDashboard 更新应用仪表盘绑定。
//
//	@ID			UpdateAppDashboard
//	@Summary	更新应用仪表盘绑定
//	@Tags		bkmonitor-dashboard
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		uid		path		string							true	"仪表盘 uid"
//	@Param		body	body	serializer.AppDashboardUpdateInput	true	"更新请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bkmonitor/dashboards/{uid} [put]
func (h *Handler) UpdateAppDashboard(c *gin.Context) {
	var uriInput serializer.AppDashboardURIInput
	var bodyInput serializer.AppDashboardUpdateInput
	if err := ginutils.BindURIJSON(c, &uriInput, &bodyInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if bodyInput.UID == nil {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "uid must be provided"))
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ws, err := h.workspaces().Get(ctx, workspace.ByWorkspaceID(app.WorkspaceID))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}

	err = h.service.Update(ctx, ws, app.ID, uriInput.UID, auth.MustGetUser(ctx).ID, &dashboard.AppDashboardUpdateData{
		UID: bodyInput.UID,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, h.wrapError(err, "update app dashboard"))
		return
	}

	go audit.AddOperationRecordAsync(ctx,
		audit.OperationTypeUpdate, audit.ResourceTypeDashboard, uriInput.UID,
		audit.WithDataAfter(bodyInput), audit.WithWorkspaceID(ws.ID), audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeleteAppDashboard 删除应用仪表盘绑定。
//
//	@ID			DeleteAppDashboard
//	@Summary	删除应用仪表盘绑定
//	@Tags		bkmonitor-dashboard
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		uid		path		string	true	"仪表盘 uid"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bkmonitor/dashboards/{uid} [delete]
func (h *Handler) DeleteAppDashboard(c *gin.Context) {
	var uriInput serializer.AppDashboardURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ws, err := h.workspaces().Get(ctx, workspace.ByWorkspaceID(app.WorkspaceID))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}

	if err = h.service.Delete(ctx, uriInput.AppID, uriInput.UID); err != nil {
		bkerrs.AbortWithErr(c, h.wrapError(err, "delete app dashboard"))
		return
	}

	go audit.AddOperationRecordAsync(ctx,
		audit.OperationTypeDelete, audit.ResourceTypeDashboard, uriInput.UID,
		audit.WithWorkspaceID(ws.ID), audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}
