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

// Package handler contains Gin handlers for component definition APIs.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler handles Gin component definition API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListComponentDefs 获取组件定义列表。
//
//	@ID			ListComponentDefs
//	@Summary	获取组件定义
//	@Tags		component-defs
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		scopeWorkspaceID		query		string	false	"按可使用该组件定义的工作空间 ID 过滤"
//	@Param		managedByWorkspaceID	query		string	false	"按可管理该组件定义的工作空间 ID 过滤"
//	@Param		keyword					query		string	false	"搜索关键词"
//	@Success	200						{object}	serializer.ListComponentDefsOutput
//	@Failure	400						{object}	bkerrs.GinErrorOutput
//	@Router		/component-defs [get]
func (h *Handler) ListComponentDefs(c *gin.Context) {
	var queryInput serializer.ListComponentDefsQueryInput
	if err := ginutils.BindQuery(c, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	compDefs, err := h.registry.ComponentDefStore.List(ctx, queryInput.ToModel())
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list component defs"))
		return
	}

	output := make([]*serializer.ComponentDefOutputObj, 0, len(compDefs))
	for _, compDef := range compDefs {
		output = append(output, new(serializer.ComponentDefOutputObj).FromModel(compDef))
	}
	ginutils.OK(c, serializer.ListComponentDefsOutput{Data: output})
}

// CreateComponentDef 创建组件定义。
//
//	@ID			CreateComponentDef
//	@Summary	创建组件定义
//	@Tags		component-defs
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		body	body		serializer.CreateComponentDefInput	true	"创建组件定义请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/component-defs [post]
func (h *Handler) CreateComponentDef(c *gin.Context) {
	var input serializer.CreateComponentDefInput
	if err := ginutils.BindJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// Create 请求包含完整字段，patchers 和 specs 的跨字段约束已由 binding 校验。
	compDef := input.ToModel(auth.MustGetUser(c.Request.Context()).ID)

	ctx := c.Request.Context()
	// name + version 全局唯一，由 store.Create 的唯一索引保证，避免并发下先查后写被覆盖。
	if err := h.registry.ComponentDefStore.Create(ctx, compDef); err != nil {
		if errors.Is(err, component.ErrComponentDefAlreadyExists) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeAlreadyExists,
				"component def(%s:%s) already exists",
				compDef.Name,
				compDef.Version,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create component def"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeComponentDef,
		compDef.Key(),
		audit.WithDataAfter(compDef),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// PatchComponentDef 更新组件定义。
//
//	@ID			PatchComponentDef
//	@Summary	更新组件定义
//	@Tags		component-defs
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		compDefName	path		string								true	"组件定义名称"
//	@Param		body		body		serializer.PatchComponentDefInput	true	"更新组件定义请求"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/component-defs/{compDefName} [patch]
func (h *Handler) PatchComponentDef(c *gin.Context) {
	var uriInput serializer.ComponentDefURIInput
	var input serializer.PatchComponentDefInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	existingCompDef, err := h.registry.ComponentDefStore.Get(
		ctx,
		uriInput.CompDefName,
		component.DefaultComponentDefVersion,
	)
	if err != nil {
		if errors.Is(err, component.ErrComponentDefNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "component def(%s) not found", uriInput.CompDefName),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get component def"))
		return
	}
	if existingCompDef.IsBuiltin {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"builtin component def(%s) cannot be updated",
			uriInput.CompDefName,
		))
		return
	}

	oldCompDef := *existingCompDef
	nextCompDef := input.ToModel(existingCompDef, auth.MustGetUser(ctx).ID)
	// Patch binding 只能校验本次传入的增量字段，无法校验与存量数据合并后的跨字段约束。
	// 因此需要在 ToModel 合并后校验完整模型，例如 patchers 和 specs 不能同时为空。
	if err = component.ValidateComponentDef(nextCompDef); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "validate component def"))
		return
	}

	if err = h.registry.ComponentDefStore.Upsert(ctx, nextCompDef); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update component def"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeComponentDef,
		nextCompDef.Key(),
		audit.WithDataBefore(oldCompDef),
		audit.WithDataAfter(nextCompDef),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeleteComponentDef 删除组件定义。
//
//	@ID			DeleteComponentDef
//	@Summary	删除组件定义
//	@Tags		component-defs
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		compDefName	path	string	true	"组件定义名称"
//	@Success	200	{object}	serializer.EmptyOutput
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/component-defs/{compDefName} [delete]
func (h *Handler) DeleteComponentDef(c *gin.Context) {
	var uriInput serializer.ComponentDefURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	existingCompDef, err := h.registry.ComponentDefStore.Get(
		ctx,
		uriInput.CompDefName,
		component.DefaultComponentDefVersion,
	)
	if err != nil {
		if errors.Is(err, component.ErrComponentDefNotFound) {
			ginutils.OK(c, serializer.EmptyOutput{})
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get component def"))
		return
	}
	if existingCompDef.IsBuiltin {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"builtin component def(%s) cannot be deleted",
			uriInput.CompDefName,
		))
		return
	}
	if existingCompDef.AppCompInstanceCount > 0 || existingCompDef.WorkspaceCompInstanceCount > 0 {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"component def(%s) is still referenced by %d app component(s) and %d workspace component(s), cannot be deleted",
			uriInput.CompDefName,
			existingCompDef.AppCompInstanceCount,
			existingCompDef.WorkspaceCompInstanceCount,
		))
		return
	}

	if _, err = h.registry.ComponentDefStore.Delete(
		ctx, uriInput.CompDefName, component.DefaultComponentDefVersion,
	); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete component def"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeComponentDef,
		existingCompDef.Key(),
		audit.WithDataBefore(existingCompDef),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// ListBuiltinVars 获取组件输出模板系统变量列表。
//
//	@Summary	获取组件输出模板系统变量列表
//	@Tags		component-defs
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.ListBuiltinVarsOutput
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/component-defs/builtin-vars [get]
func (h *Handler) ListBuiltinVars(c *gin.Context) {
	ginutils.OK(
		c,
		new(serializer.ListBuiltinVarsOutput).FromModels(component.BuiltinVars),
	)
}
