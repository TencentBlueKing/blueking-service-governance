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

// Package handler contains Gin handlers for workspace release strategies.
package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Handler handles workspace release strategy APIs.
type Handler struct {
	registry *storereg.Registry
}

// New creates a release strategy API handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListReleaseStrategies lists all release strategies in a workspace.
//
//	@ID			ListReleaseStrategies
//	@Summary	查询空间内全部发布策略
//	@Tags		release-strategies
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListReleaseStrategiesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/release-strategies [get]
func (h *Handler) ListReleaseStrategies(c *gin.Context) {
	var uri serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	strategies, err := h.strategies().List(
		ctx,
		releasestrategy.ByWorkspace(workspace.ID),
		releasestrategy.OldestFirst(),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, batchrelease.APIError(err, "list release strategies"))
		return
	}
	data := make([]*serializer.ReleaseStrategyOutputObj, 0, len(strategies))
	for _, s := range strategies {
		data = append(data, new(serializer.ReleaseStrategyOutputObj).FromModel(s))
	}
	ginutils.OK(c, serializer.ListReleaseStrategiesOutput{Data: data})
}

// GetReleaseStrategy returns a release strategy in a workspace.
//
//	@ID			GetReleaseStrategy
//	@Summary	查询发布策略详情
//	@Tags		release-strategies
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		strategyID	path		string	true	"发布策略 ID"
//	@Success	200			{object}	serializer.ReleaseStrategyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/release-strategies/{strategyID} [get]
func (h *Handler) GetReleaseStrategy(c *gin.Context) {
	var uri serializer.StrategyURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	strategy, err := releasestrategy.GetStrategy(ctx, h.strategies(), workspace.ID, uri.StrategyID)
	if err != nil {
		bkerrs.AbortWithErr(c, batchrelease.APIError(err, "get release strategy"))
		return
	}
	ginutils.OK(c, serializer.ReleaseStrategyOutput{
		Data: new(serializer.ReleaseStrategyOutputObj).FromModel(*strategy),
	})
}

// CreateReleaseStrategy creates a release strategy in a workspace.
//
//	@ID			CreateReleaseStrategy
//	@Summary	新建发布策略
//	@Tags		release-strategies
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		body		body		serializer.ReleaseStrategyInput	true	"发布策略"
//	@Success	200			{object}	serializer.ReleaseStrategyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/release-strategies [post]
func (h *Handler) CreateReleaseStrategy(c *gin.Context) {
	var uri serializer.WorkspaceURIInput
	var input serializer.ReleaseStrategyInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	created := input.ToModel(workspace.ID)
	username := auth.MustGetUser(ctx).ID
	created.Creator, created.Updater = username, username
	if err = h.strategies().Create(ctx, created); err != nil {
		bkerrs.AbortWithErr(c, batchrelease.APIError(err, "create release strategy"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeCreate, nil, created)
	ginutils.OK(c, serializer.ReleaseStrategyOutput{
		Data: new(serializer.ReleaseStrategyOutputObj).FromModel(*created),
	})
}

// UpdateReleaseStrategy replaces the editable content of a release strategy.
//
// 策略被进行中的分批发布使用时仍允许修改，已下发的 Rollout 不受影响
//
//	@ID			UpdateReleaseStrategy
//	@Summary	修改发布策略
//	@Tags		release-strategies
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		strategyID	path		string							true	"发布策略 ID"
//	@Param		body		body		serializer.ReleaseStrategyInput	true	"发布策略"
//	@Success	200			{object}	serializer.ReleaseStrategyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/release-strategies/{strategyID} [put]
func (h *Handler) UpdateReleaseStrategy(c *gin.Context) {
	var uri serializer.StrategyURIInput
	var input serializer.ReleaseStrategyInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	before, err := releasestrategy.GetStrategy(ctx, h.strategies(), workspace.ID, uri.StrategyID)
	if err != nil {
		bkerrs.AbortWithErr(c, batchrelease.APIError(err, "update release strategy"))
		return
	}
	updated := input.ToModel(workspace.ID)
	updated.ID = before.ID
	updated.Creator, updated.CreatedAt = before.Creator, before.CreatedAt
	updated.Updater = auth.MustGetUser(ctx).ID
	if err = releasestrategy.UpdateStrategy(ctx, h.strategies(), updated); err != nil {
		bkerrs.AbortWithErr(c, batchrelease.APIError(err, "update release strategy"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeUpdate, before, updated)
	ginutils.OK(c, serializer.ReleaseStrategyOutput{
		Data: new(serializer.ReleaseStrategyOutputObj).FromModel(*updated),
	})
}

// DeleteReleaseStrategy deletes a release strategy.
//
// 策略被进行中的分批发布使用时仍允许删除，已下发的 Rollout 与部署记录中的策略快照不受影响
//
//	@ID			DeleteReleaseStrategy
//	@Summary	删除发布策略
//	@Tags		release-strategies
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		strategyID	path		string	true	"发布策略 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/release-strategies/{strategyID} [delete]
func (h *Handler) DeleteReleaseStrategy(c *gin.Context) {
	var uri serializer.StrategyURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	deleted, err := releasestrategy.DeleteStrategy(ctx, h.strategies(), workspace.ID, uri.StrategyID)
	if err != nil {
		bkerrs.AbortWithErr(c, batchrelease.APIError(err, "delete release strategy"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeDelete, deleted, nil)
	ginutils.OK(c, serializer.EmptyOutput{})
}

func (h *Handler) strategies() *releasestrategy.Store {
	return store.Of(h.registry.DB, releasestrategy.Meta)
}

func recordAudit(
	ctx context.Context,
	workspaceID string,
	operation audit.OperationType,
	before, after *releasestrategy.ReleaseStrategy,
) {
	options := []audit.Option{
		audit.WithAttribute(audit.AttributeReleaseStrategy),
		audit.WithWorkspaceID(workspaceID),
	}
	if before != nil {
		options = append(options, audit.WithDataBefore(before))
	}
	if after != nil {
		options = append(options, audit.WithDataAfter(after))
	}
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		operation,
		audit.ResourceTypeWorkspace,
		workspaceID,
		options...,
	)
}
