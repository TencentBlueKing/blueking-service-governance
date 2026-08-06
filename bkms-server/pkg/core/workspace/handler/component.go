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
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// CreateWorkspaceComponent 创建工作空间组件。
//
//	@ID			CreateWorkspaceComponent
//	@Summary	添加工作空间组件
//	@Tags		workspace-components
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string										true	"工作空间 ID"
//	@Param		body		body		serializer.CreateWorkspaceComponentInput	true	"添加工作空间组件请求"
//	@Success	200			{object}	serializer.CreateWorkspaceComponentOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/components [post]
func (h *Handler) CreateWorkspaceComponent(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var input serializer.CreateWorkspaceComponentInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	newComp := input.ToModel(uriInput.WorkspaceID)
	if _, err := h.registry.ComponentDefStore.Get(ctx, newComp.Type, newComp.Version); err != nil {
		if errors.Is(err, component.ErrComponentDefNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "component def(%s:%s) not found", newComp.Type, newComp.Version),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get component def"))
		return
	}

	if err := h.registry.WorkspaceCompsStore.Add(ctx, newComp); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "add workspace component"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeWorkspace,
		uriInput.WorkspaceID,
		audit.WithAttribute(audit.AttributeWorkspaceComponent),
		audit.WithDataAfter(newComp),
		audit.WithWorkspaceID(uriInput.WorkspaceID),
	)

	ginutils.OK(c, serializer.CreateWorkspaceComponentOutput{
		Data: &serializer.WorkspaceComponentNameOutputObj{Name: newComp.Name},
	})
}

// PatchWorkspaceComponent 更新工作空间组件。
//
//	@ID			PatchWorkspaceComponent
//	@Summary	更新工作空间组件
//	@Tags		workspace-components
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string										true	"工作空间 ID"
//	@Param		compName	path		string										true	"组件名称"
//	@Param		body		body		serializer.PatchWorkspaceComponentInput	true	"更新工作空间组件请求"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/components/{compName} [patch]
func (h *Handler) PatchWorkspaceComponent(c *gin.Context) {
	var uriInput serializer.WorkspaceComponentURIInput
	var input serializer.PatchWorkspaceComponentInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	oldComp, err := h.registry.WorkspaceCompsStore.GetByName(ctx, uriInput.WorkspaceID, uriInput.CompName)
	if err != nil {
		if errors.Is(err, workspace.ErrComponentNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(
					bkerrs.ErrCodeNotFound,
					"component(%s) not found in workspace(%s)",
					uriInput.CompName,
					uriInput.WorkspaceID,
				),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace component"))
		return
	}

	updateData := input.ToModel()
	if err = h.registry.WorkspaceCompsStore.Update(ctx, oldComp.ID, updateData); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update workspace component"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeWorkspace,
		uriInput.WorkspaceID,
		audit.WithAttribute(audit.AttributeWorkspaceComponent),
		audit.WithDataBefore(oldComp),
		audit.WithDataAfter(updateData),
		audit.WithWorkspaceID(uriInput.WorkspaceID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeleteWorkspaceComponent 删除工作空间组件。
//
//	@ID			DeleteWorkspaceComponent
//	@Summary	删除工作空间组件
//	@Tags		workspace-components
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		compName	path		string	true	"组件名称"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/components/{compName} [delete]
func (h *Handler) DeleteWorkspaceComponent(c *gin.Context) {
	var uriInput serializer.WorkspaceComponentURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	oldComp, err := h.registry.WorkspaceCompsStore.GetByName(ctx, uriInput.WorkspaceID, uriInput.CompName)
	if err != nil {
		if errors.Is(err, workspace.ErrComponentNotFound) {
			ginutils.OK(c, serializer.EmptyOutput{})
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace component"))
		return
	}

	// 删除前检查组件引用关系，若有其他应用引用该组件，则不允许删除
	compRefResolver := workspace.NewComponentRefResolver(h.registry.AppStore, h.registry.AppModelStore)
	compRefMap, err := compRefResolver.BuildRefMap(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "build component ref map"))
		return
	}
	if len(compRefMap[oldComp.Name]) > 0 {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"component(%s) is referenced by other apps, cannot be deleted",
			oldComp.Name,
		))
		return
	}

	if err = h.registry.WorkspaceCompsStore.Remove(ctx, oldComp.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "remove workspace component"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeWorkspace,
		uriInput.WorkspaceID,
		audit.WithAttribute(audit.AttributeWorkspaceComponent),
		audit.WithDataBefore(oldComp),
		audit.WithWorkspaceID(uriInput.WorkspaceID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// ListWorkspaceComponents 获取工作空间组件列表。
//
//	@ID			ListWorkspaceComponents
//	@Summary	获取工作空间组件列表
//	@Tags		workspace-components
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListWorkspaceComponentsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/components [get]
func (h *Handler) ListWorkspaceComponents(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	comps, err := h.registry.WorkspaceCompsStore.ListByWorkspace(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list workspace components"))
		return
	}

	compRefResolver := workspace.NewComponentRefResolver(h.registry.AppStore, h.registry.AppModelStore)
	compRefMap, err := compRefResolver.BuildRefMap(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "build component ref map"))
		return
	}

	output := make([]*serializer.WorkspaceComponentOutputObj, 0, len(comps))
	for _, comp := range comps {
		output = append(output, new(serializer.WorkspaceComponentOutputObj).FromModel(comp, compRefMap[comp.Name]))
	}
	ginutils.JSON(c, http.StatusOK, serializer.ListWorkspaceComponentsOutput{Data: output})
}
