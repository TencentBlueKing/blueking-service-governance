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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/serializer"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking/serializer"
)

// ListAlertStrategies 查询应用下的告警策略列表
//
//	@ID			ListAlertStrategies
//	@Summary	查询告警策略列表
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		appID		path		string	true	"应用 ID"
//	@Success	200			{object}	serializer.ListAlertStrategiesResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies [get]
func (h *Handler) ListAlertStrategies(c *gin.Context) {
	var uriInput serializer.AlertStrategyAppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := validateAppInWorkspace(ctx, h.registry, uriInput.WorkspaceID, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	rules, err := h.alertStrategyService().ListByApp(ctx, app.WorkspaceID, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list alert strategies"))
		return
	}

	results := lo.Map(rules, func(r alertstrategy.AlertStrategy, _ int) *serializer.AlertStrategyOutput {
		return new(serializer.AlertStrategyOutput).FromModel(r)
	})
	ginutils.OK(c, &serializer.ListAlertStrategiesResp{Data: &serializer.ListAlertStrategiesOutput{
		Count:   int64(len(results)),
		Results: results,
	}})
}

// GetAlertStrategy 获取告警策略详情
//
//	@ID			GetAlertStrategy
//	@Summary	获取告警策略详情
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		strategyID	path		string	true	"本地策略 ID"
//	@Success	200			{object}	serializer.GetAlertStrategyResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID} [get]
func (h *Handler) GetAlertStrategy(c *gin.Context) {
	var uriInput serializer.AlertStrategyURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := validateAppInWorkspace(
		ctx, h.registry, uriInput.WorkspaceID, uriInput.AppID, ginperm.TypeView,
	); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	strategyObjID, err := bson.ObjectIDFromHex(uriInput.StrategyID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "invalid strategy ID"))
		return
	}

	rule, err := getAlertStrategyInApp(ctx, h.registry, strategyObjID, uriInput.WorkspaceID, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapAlertStrategyLookupErr(err))
		return
	}

	ginutils.OK(c, &serializer.GetAlertStrategyResp{Data: new(serializer.AlertStrategyOutput).FromModel(*rule)})
}

// CreateAlertStrategy 创建告警策略
//
//	@ID			CreateAlertStrategy
//	@Summary	创建告警策略
//	@Tags		bkintegrations-bkmonitor
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						true	"工作空间 ID"
//	@Param		appID		path		string						true	"应用 ID"
//	@Param		body		body		serializer.CreateAlertStrategyBody	true	"请求体"
//	@Success	200			{object}	serializer.CreateAlertStrategyResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies [post]
func (h *Handler) CreateAlertStrategy(c *gin.Context) {
	var uriInput serializer.AlertStrategyAppURIInput
	var body serializer.CreateAlertStrategyBody
	if err := ginutils.BindURIJSON(c, &uriInput, &body); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := validateAppInWorkspace(ctx, h.registry, uriInput.WorkspaceID, uriInput.AppID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ws, err := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}

	svc := h.alertStrategyService()
	operator := auth.MustGetUser(ctx).ID
	createReq, parseErr := body.ToCreateReq(app.WorkspaceID, app.ID, app.Name, operator)
	if parseErr != nil {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "invalid env IDs"))
		return
	}
	rule, err := svc.CreateAndSync(ctx, ws, createReq)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create alert strategy"))
		return
	}

	go audit.AddOperationRecordAsync(ctx,
		audit.OperationTypeCreate, audit.ResourceTypeAlertStrategy, rule.ID.Hex(),
		audit.WithDataAfter(body), audit.WithWorkspaceID(app.WorkspaceID), audit.WithAppID(app.ID),
	)

	ginutils.OK(c, &serializer.CreateAlertStrategyResp{Data: new(serializer.AlertStrategyOutput).FromModel(*rule)})
}

// UpdateAlertStrategy 更新告警策略
//
//	@ID			UpdateAlertStrategy
//	@Summary	更新告警策略
//	@Tags		bkintegrations-bkmonitor
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						true	"工作空间 ID"
//	@Param		appID		path		string						true	"应用 ID"
//	@Param		strategyID	path		string						true	"本地策略 ID"
//	@Param		body		body		serializer.UpdateAlertStrategyBody	true	"请求体"
//	@Success	200			{object}	serializer.GetAlertStrategyResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID} [put]
func (h *Handler) UpdateAlertStrategy(c *gin.Context) {
	var uriInput serializer.AlertStrategyURIInput
	var body serializer.UpdateAlertStrategyBody
	if err := ginutils.BindURIJSON(c, &uriInput, &body); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, ws, strategyObjID, err := h.loadStrategyContext(
		ctx,
		uriInput.WorkspaceID,
		uriInput.AppID,
		uriInput.StrategyID,
		ginperm.TypeEdit,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	operator := auth.MustGetUser(ctx).ID
	updateReq, parseErr := body.ToUpdateReq(operator)
	if parseErr != nil {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "invalid env IDs"))
		return
	}

	svc := h.alertStrategyService()
	changed, err := svc.UpdateAndSync(ctx, ws, strategyObjID, updateReq)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update alert strategy"))
		return
	}

	if changed {
		go audit.AddOperationRecordAsync(ctx,
			audit.OperationTypeUpdate, audit.ResourceTypeAlertStrategy, uriInput.StrategyID,
			audit.WithDataAfter(body), audit.WithWorkspaceID(ws.ID), audit.WithAppID(app.ID),
		)
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// DeleteAlertStrategy 删除告警策略
//
//	@ID			DeleteAlertStrategy
//	@Summary	删除告警策略
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		strategyID	path		string	true	"本地策略 ID"
//	@Success	200			{object}	slz.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID} [delete]
func (h *Handler) DeleteAlertStrategy(c *gin.Context) {
	var uriInput serializer.AlertStrategyURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, ws, strategyObjID, err := h.loadStrategyContext(
		ctx,
		uriInput.WorkspaceID,
		uriInput.AppID,
		uriInput.StrategyID,
		ginperm.TypeEdit,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.alertStrategyService().Delete(ctx, ws, strategyObjID, auth.MustGetUser(ctx).ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete alert strategy"))
		return
	}

	go audit.AddOperationRecordAsync(ctx,
		audit.OperationTypeDelete, audit.ResourceTypeAlertStrategy, uriInput.StrategyID,
		audit.WithWorkspaceID(ws.ID), audit.WithAppID(app.ID),
	)

	ginutils.OK(c, slz.EmptyOutput{})
}

// SyncAlertStrategy 手动触发告警策略同步到远端
//
//	@ID			SyncAlertStrategy
//	@Summary	同步告警策略到远端
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		strategyID	path		string	true	"本地策略 ID"
//	@Success	200			{object}	slz.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/sync [post]
func (h *Handler) SyncAlertStrategy(c *gin.Context) {
	var uriInput serializer.AlertStrategyURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	_, ws, strategyObjID, err := h.loadStrategyContext(
		ctx,
		uriInput.WorkspaceID,
		uriInput.AppID,
		uriInput.StrategyID,
		ginperm.TypeEdit,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	operator := auth.MustGetUser(ctx).ID
	if err = h.alertStrategyService().SyncToRemote(ctx, ws, strategyObjID, operator); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "sync alert strategy"))
		return
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// SwitchAlertStrategy 切换告警策略启停状态
//
//	@ID			SwitchAlertStrategy
//	@Summary	切换告警策略启停状态
//	@Tags		bkintegrations-bkmonitor
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						true	"工作空间 ID"
//	@Param		appID		path		string						true	"应用 ID"
//	@Param		strategyID	path		string						true	"本地策略 ID"
//	@Param		body		body		serializer.SwitchAlertStrategyBody	true	"请求体"
//	@Success	200			{object}	slz.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/apps/{appID}/bkmonitor/alert-strategies/{strategyID}/switch [post]
func (h *Handler) SwitchAlertStrategy(c *gin.Context) {
	var uriInput serializer.AlertStrategyURIInput
	var body serializer.SwitchAlertStrategyBody
	if err := ginutils.BindURIJSON(c, &uriInput, &body); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, ws, strategyObjID, err := h.loadStrategyContext(
		ctx,
		uriInput.WorkspaceID,
		uriInput.AppID,
		uriInput.StrategyID,
		ginperm.TypeEdit,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	operator := auth.MustGetUser(ctx).ID
	if err = h.alertStrategyService().SwitchEnabled(ctx, ws, strategyObjID, body.Enabled, operator); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "switch alert strategy"))
		return
	}

	go audit.AddOperationRecordAsync(ctx,
		audit.OperationTypeUpdate, audit.ResourceTypeAlertStrategy, uriInput.StrategyID,
		audit.WithDataAfter(body), audit.WithWorkspaceID(ws.ID), audit.WithAppID(app.ID),
	)

	ginutils.OK(c, slz.EmptyOutput{})
}
