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
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// CreateAppComponent 添加应用组件。
//
//	@ID			CreateAppComponent
//	@Summary	添加应用组件
//	@Tags		app-components
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		body	body		serializer.CreateAppComponentInput	true	"添加应用组件请求"
//	@Success	200		{object}	serializer.CreateAppComponentOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/components [post]
func (h *Handler) CreateAppComponent(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.CreateAppComponentInput
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

	newComp := input.ToModel()
	if newComp.RefWorkspaceCompName != "" {
		if _, err = h.registry.WorkspaceCompsStore.GetByName(ctx, app.WorkspaceID, newComp.RefWorkspaceCompName); err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace component"))
			return
		}
	} else {
		def, iErr := h.registry.ComponentDefStore.Get(ctx, newComp.Type, newComp.Version)
		if iErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(iErr, bkerrs.ErrCodeInternalServerError, "get component def"))
			return
		}
		newComp.Type = def.Name
		newComp.Version = def.Version
	}

	if err = h.registry.AppModelStore.AddComponent(ctx, app.ID, newComp); err != nil {
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app(%s) model not found", app.Name))
			return
		}
		if errors.Is(err, appmodel.ErrComponentNameExists) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeInvalidRequest, "component name(%s) already exists in app(%s)", newComp.Name, app.Name,
			))
			return
		}
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "add component to app(%s) model", app.Name),
		)
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeAppComponents),
		audit.WithDataAfter(newComp),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.CreateAppComponentOutput{
		Data: &serializer.AppComponentNameOutputObj{Name: newComp.Name},
	})
}

// PatchAppComponent 更新应用组件。
//
//	@ID			PatchAppComponent
//	@Summary	更新应用组件
//	@Tags		app-components
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string								true	"应用 ID"
//	@Param		compName	path		string								true	"组件名称"
//	@Param		body		body		serializer.PatchAppComponentInput	true	"更新应用组件请求"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/components/{compName} [patch]
func (h *Handler) PatchAppComponent(c *gin.Context) {
	var uriInput serializer.AppComponentURIInput
	var input serializer.PatchAppComponentInput
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

	appModel, err := h.registry.AppModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app(%s) model not found", app.Name))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get app(%s) model", app.Name))
		return
	}

	var targetComp *component.Component
	for _, comp := range appModel.Components {
		if comp.Name == uriInput.CompName {
			targetComp = comp
			break
		}
	}
	if targetComp == nil {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(
			bkerrs.ErrCodeNotFound, "component(%s) not found in app(%s)", uriInput.CompName, app.Name,
		))
		return
	}
	if targetComp.RefWorkspaceCompName != "" {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"component(%s) is a reference component, cannot update",
			uriInput.CompName,
		))
		return
	}

	updateData := input.ToModel()
	if err = h.registry.AppModelStore.UpdateComponent(ctx, app.ID, uriInput.CompName, updateData); err != nil {
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app(%s) model not found", app.Name))
			return
		}
		if errors.Is(err, appmodel.ErrComponentNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound, "component(%s) not found in app(%s)", uriInput.CompName, app.Name,
			))
			return
		}
		if errors.Is(err, appmodel.ErrComponentNameExists) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeInvalidRequest, "component name already exists in app(%s)", app.Name,
			))
			return
		}
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "update component in app(%s) model", app.Name),
		)
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeAppComponents),
		audit.WithDataBefore(targetComp),
		audit.WithDataAfter(updateData),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeleteAppComponent 删除应用组件。
//
//	@ID			DeleteAppComponent
//	@Summary	删除应用组件
//	@Tags		app-components
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		compName	path		string	true	"组件名称"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/components/{compName} [delete]
func (h *Handler) DeleteAppComponent(c *gin.Context) {
	var uriInput serializer.AppComponentURIInput
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

	if err = h.registry.AppModelStore.RemoveComponent(ctx, app.ID, uriInput.CompName); err != nil {
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app(%s) model not found", app.Name))
			return
		}
		if errors.Is(err, appmodel.ErrComponentNotFound) {
			ginutils.OK(c, serializer.EmptyOutput{})
			return
		}
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "remove component from app(%s) model", app.Name),
		)
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeAppComponents),
		audit.WithDataBefore(uriInput.CompName),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}
