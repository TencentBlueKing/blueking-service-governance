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

// Package handler contains Gin handlers for envvars APIs.
package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/serializer"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// Handler handles Gin envvars API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// CreateScopedEnvVar 创建作用域级别的环境变量（ScopedEnvVar）。
//
//	@ID				CreateScopedEnvVar
//	@Summary		创建作用域级别的环境变量（ScopedEnvVar）
//	@Tags			envvars
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string								true	"工作空间 ID"
//	@Param			body		body		serializer.CreateScopedEnvVarInput	true	"创建作用域级别环境变量请求"
//	@Success		200			{object}	serializer.CreateScopedEnvVarOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/scoped-env-vars [post]
func (h *Handler) CreateScopedEnvVar(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var input serializer.CreateScopedEnvVarInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	scope, err := envvartypes.ParseScopedEnvVarScope(input.ScopeType, input.ScopeValue)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "validate scoped env var scope"))
		return
	}
	if err = h.validateScopedEnvVarScopeValue(ctx, ws.ID, scope); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	id, err := h.registry.ScopedEnvVarStore.Create(ctx, envvars.ScopedEnvVar{
		WorkspaceID: ws.ID,
		ScopeType:   scope.ScopeType,
		ScopeValue:  scope.ScopeValue,
		Key:         input.Key,
		Value:       input.Value,
		Description: input.Description,
		IsSensitive: input.IsSensitive,
	})
	if err != nil {
		if errors.Is(err, envvars.ErrScopedEnvVarKeyConflict) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "create scoped env var"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create scoped env var"))
		return
	}

	obj, err := h.registry.ScopedEnvVarStore.GetByID(ctx, ws.ID, id)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get created scoped env var"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeScopedEnvVars),
		audit.WithDataAfter(obj),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.JSON(c, http.StatusOK, serializer.CreateScopedEnvVarOutput{
		Data: new(serializer.ScopedEnvVarOutputObj).FromModel(*obj),
	})
}

// UpdateScopedEnvVar 更新作用域级别的环境变量（ScopedEnvVar）。
//
//	@ID				UpdateScopedEnvVar
//	@Summary		更新作用域级别的环境变量（ScopedEnvVar）
//	@Tags			envvars
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID		path		string								true	"工作空间 ID"
//	@Param			scopedEnvVarID	path		string								true	"Scoped EnvVar ID"
//	@Param			body			body		serializer.UpdateScopedEnvVarInput	true	"更新作用域级别环境变量请求"
//	@Success		200				{object}	serializer.UpdateScopedEnvVarOutput
//	@Failure		400				{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/scoped-env-vars/{scopedEnvVarID} [put]
func (h *Handler) UpdateScopedEnvVar(c *gin.Context) {
	var uriInput serializer.ScopedEnvVarURIInput
	var input serializer.UpdateScopedEnvVarInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	id, err := bson.ObjectIDFromHex(uriInput.ScopedEnvVarID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid scoped env var id"))
		return
	}

	oldObj, err := h.registry.ScopedEnvVarStore.GetByID(ctx, ws.ID, id)
	if err != nil {
		if errors.Is(err, envvars.ErrScopedEnvVarNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "scoped env var %s not found", uriInput.ScopedEnvVarID),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get scoped env var"))
		return
	}

	// Sensitive env vars cannot be downgraded to non-sensitive; they can only be
	// deleted and recreated.
	if oldObj.IsSensitive && (input.IsSensitive != nil && !*input.IsSensitive) {
		bkerrs.AbortWithErr(
			c,
			bkerrs.New(bkerrs.ErrCodeInvalidRequest, "sensitive env var cannot be changed to non-sensitive"),
		)
		return
	}

	updateData := envvars.ScopedEnvVarUpdateData{
		Key:         input.Key,
		Value:       input.Value,
		Description: input.Description,
		IsSensitive: input.IsSensitive,
	}
	if err = h.registry.ScopedEnvVarStore.UpdateByID(ctx, ws.ID, id, updateData); err != nil {
		if errors.Is(err, envvars.ErrScopedEnvVarKeyConflict) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "update scoped env var"))
			return
		}
		if errors.Is(err, envvars.ErrScopedEnvVarNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "scoped env var %s not found", uriInput.ScopedEnvVarID),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update scoped env var"))
		return
	}

	updatedObj, err := h.registry.ScopedEnvVarStore.GetByID(ctx, ws.ID, id)
	if err != nil {
		if errors.Is(err, envvars.ErrScopedEnvVarNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "scoped env var %s not found", uriInput.ScopedEnvVarID),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get scoped env var"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeScopedEnvVars),
		audit.WithDataBefore(oldObj),
		audit.WithDataAfter(updatedObj),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.UpdateScopedEnvVarOutput{
		Data: new(serializer.ScopedEnvVarOutputObj).FromModel(*updatedObj),
	})
}

// DeleteScopedEnvVar 删除作用域级别的环境变量（ScopedEnvVar）。
//
//	@ID			DeleteScopedEnvVar
//	@Summary	删除作用域级别的环境变量（ScopedEnvVar）
//	@Tags		envvars
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID		path		string	true	"工作空间 ID"
//	@Param		scopedEnvVarID	path		string	true	"Scoped EnvVar ID"
//	@Success	200				{object}	serializer.EmptyOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/scoped-env-vars/{scopedEnvVarID} [delete]
func (h *Handler) DeleteScopedEnvVar(c *gin.Context) {
	var uriInput serializer.ScopedEnvVarURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	id, err := bson.ObjectIDFromHex(uriInput.ScopedEnvVarID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid scoped env var id"))
		return
	}

	obj, err := h.registry.ScopedEnvVarStore.GetByID(ctx, ws.ID, id)
	if err != nil {
		if errors.Is(err, envvars.ErrScopedEnvVarNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "scoped env var %s not found", uriInput.ScopedEnvVarID),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get scoped env var"))
		return
	}

	if err = h.registry.ScopedEnvVarStore.DeleteByID(ctx, ws.ID, id); err != nil {
		if errors.Is(err, envvars.ErrScopedEnvVarNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "scoped env var %s not found", uriInput.ScopedEnvVarID),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete scoped env var"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeScopedEnvVars),
		audit.WithDataBefore(obj),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// ListPublicScopedEnvVars 获取指定空间下的公共环境变量列表。
//
//	@ID				ListPublicScopedEnvVars
//	@Summary		获取指定空间下的公共环境变量列表
//	@Description	公开环境变量，指作用域为 workspace（工作空间）和 envType（环境类型）的作用域级别环境变量，不包含作用域为 env（单环境）。
//	@Tags			envvars
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	serializer.ListPublicScopedEnvVarsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/scoped-env-vars/public-vars [get]
func (h *Handler) ListPublicScopedEnvVars(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	scopedVars, err := h.registry.ScopedEnvVarStore.ListPublic(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list public scoped env vars"))
		return
	}

	data := lo.Map(scopedVars, func(item envvars.ScopedEnvVar, _ int) *serializer.ScopedEnvVarOutputObj {
		return new(serializer.ScopedEnvVarOutputObj).FromModel(item)
	})
	ginutils.OK(c, serializer.ListPublicScopedEnvVarsOutput{Data: data})
}

// ListDetailedEnvScopedEnvVars 获取指定环境下作用域为当前环境的环境变量详情。
//
//	@ID			ListDetailedEnvScopedEnvVars
//	@Summary	获取指定环境下作用域为当前环境的环境变量详情
//	@Tags		envvars
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string	true	"环境 ID"
//	@Success	200		{object}	serializer.ListDetailedEnvScopedEnvVarsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/scoped-env-vars/detailed-list/{envID} [get]
func (h *Handler) ListDetailedEnvScopedEnvVars(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	scopedVars, err := h.registry.ScopedEnvVarStore.List(
		ctx,
		env.WorkspaceID,
		envvars.WithScopes(envvartypes.ScopeEnv(env.Name)),
		envvars.WithOrdering("created"),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list env scoped env vars"))
		return
	}

	if len(scopedVars) == 0 {
		ginutils.OK(
			c,
			serializer.ListDetailedEnvScopedEnvVarsOutput{Data: []*serializer.ScopedEnvVarDetailedOutputObj{}},
		)
		return
	}

	keys := lo.Map(scopedVars, func(item envvars.ScopedEnvVar, _ int) string {
		return item.Key
	})
	reader := envvars.NewUnifiedEnvVarsReader(
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	)
	conflictedInfos, err := reader.BuildEnvConflictedInfoByKeys(ctx, keys, *env)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "build scoped env var conflicted info"),
		)
		return
	}

	data := lo.Map(scopedVars, func(item envvars.ScopedEnvVar, _ int) *serializer.ScopedEnvVarDetailedOutputObj {
		return new(serializer.ScopedEnvVarDetailedOutputObj).FromModel(item, conflictedInfos[item.Key])
	})
	ginutils.OK(c, serializer.ListDetailedEnvScopedEnvVarsOutput{Data: data})
}

// ListDetailedAppEnvVars 获取指定应用的环境变量详情。
//
//	@ID				ListDetailedAppEnvVars
//	@Summary		获取指定应用的环境变量详情
//	@Description	获取指定应用的环境变量详情，包含可能的 Key 冲突信息。
//	@Tags			envvars
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Success		200		{object}	serializer.ListDetailedAppEnvVarsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/env-vars/detailed-list [get]
func (h *Handler) ListDetailedAppEnvVars(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	am, err := h.registry.AppModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			ginutils.OK(c, serializer.ListDetailedAppEnvVarsOutput{
				Data: []*serializer.AppEnvVarDetailedOutputObj{},
			})
			return
		}
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get app model for app id=%s", app.ID),
		)
		return
	}

	if len(am.Workload.EnvVars) == 0 {
		ginutils.OK(c, serializer.ListDetailedAppEnvVarsOutput{Data: []*serializer.AppEnvVarDetailedOutputObj{}})
		return
	}

	reader := envvars.NewUnifiedEnvVarsReader(
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	)
	keys := lo.Map(am.Workload.EnvVars, func(item appmodel.Variable, _ int) string {
		return item.Key
	})

	conflictedInfos, err := reader.BuildAppConflictedInfoByKeys(ctx, keys, app.WorkspaceID, app)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "build app env var conflicted info"),
		)
		return
	}

	data := lo.Map(am.Workload.EnvVars, func(item appmodel.Variable, _ int) *serializer.AppEnvVarDetailedOutputObj {
		return new(serializer.AppEnvVarDetailedOutputObj).FromModel(item, conflictedInfos[item.Key])
	})
	ginutils.OK(c, serializer.ListDetailedAppEnvVarsOutput{Data: data})
}

// ListAppDefinedEnvVars 获取应用直接定义的环境变量列表。
//
//	@Summary		获取应用直接定义的环境变量列表
//	@Description	只返回 AppModel.workload.envVars 中直接定义的变量，不包含任何继承或合并后的变量。
//	@Tags			envvars
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Success		200		{object}	serializer.ListAppDefinedEnvVarsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/env-vars [get]
func (h *Handler) ListAppDefinedEnvVars(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err = ensureAppSupportsDefinedEnvVars(app); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	envVars, err := appmodel.NewAppEnvVarService(h.registry.AppModelStore).List(ctx, app.ID)
	if err != nil {
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			ginutils.OK(c, serializer.ListAppDefinedEnvVarsOutput{Data: []*serializer.AppDefinedEnvVarOutputObj{}})
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list app defined env vars"))
		return
	}

	data := lo.Map(envVars, func(item appmodel.Variable, _ int) *serializer.AppDefinedEnvVarOutputObj {
		return new(serializer.AppDefinedEnvVarOutputObj).FromModel(item)
	})
	ginutils.OK(c, serializer.ListAppDefinedEnvVarsOutput{Data: data})
}

// CreateAppDefinedEnvVar 创建应用直接定义的环境变量。
//
//	@Summary		创建应用直接定义的环境变量
//	@Description	key 在同一应用内必须唯一，重复时服务端会拒绝写入。
//	@Tags			envvars
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string									true	"应用 ID"
//	@Param			body	body		serializer.CreateAppDefinedEnvVarInput	true	"创建应用环境变量请求"
//	@Success		200		{object}	serializer.CreateAppDefinedEnvVarOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/env-vars [post]
func (h *Handler) CreateAppDefinedEnvVar(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.CreateAppDefinedEnvVarInput
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
	if err = ensureAppSupportsDefinedEnvVars(app); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	created, err := appmodel.NewAppEnvVarService(h.registry.AppModelStore).Create(ctx, app.ID, appmodel.Variable{
		Key:         input.Key,
		Value:       input.Value,
		Description: input.Description,
		IsSensitive: input.IsSensitive,
	})
	if err != nil {
		if errors.Is(err, appmodel.ErrEnvVarKeyExists) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeAlreadyExists, "create app defined env var"))
			return
		}
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app(%s) model not found", app.Name))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create app defined env var"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeEnvVars),
		audit.WithDataAfter(created),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.CreateAppDefinedEnvVarOutput{
		Data: new(serializer.AppDefinedEnvVarOutputObj).FromModel(*created),
	})
}

// UpdateAppDefinedEnvVar 更新应用直接定义的环境变量。
//
//	@Summary		更新应用直接定义的环境变量
//	@Description	key 表示当前变量 key，updatedKey 表示更新后的变量 key，因此该接口支持“重命名 key”。
//	@Tags			envvars
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string									true	"应用 ID"
//	@Param			key		path		string									true	"旧环境变量 Key"
//	@Param			body	body		serializer.UpdateAppDefinedEnvVarInput	true	"更新应用环境变量请求"
//	@Success		200		{object}	serializer.UpdateAppDefinedEnvVarOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/env-vars/{key} [put]
func (h *Handler) UpdateAppDefinedEnvVar(c *gin.Context) {
	var uriInput serializer.AppEnvVarURIInput
	var input serializer.UpdateAppDefinedEnvVarInput
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
	if err = ensureAppSupportsDefinedEnvVars(app); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	oldEnvVar, updatedEnvVar, err := appmodel.NewAppEnvVarService(h.registry.AppModelStore).Update(
		ctx,
		app.ID,
		uriInput.Key,
		appmodel.AppEnvVarUpdateData{
			Key:         input.UpdatedKey,
			Value:       input.Value,
			Description: input.Description,
			IsSensitive: input.IsSensitive,
		},
	)
	if err != nil {
		if errors.Is(err, appmodel.ErrEnvVarNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app env var %s not found", uriInput.Key))
			return
		}
		if errors.Is(err, appmodel.ErrEnvVarKeyExists) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeAlreadyExists, "update app defined env var"))
			return
		}
		if errors.Is(err, appmodel.ErrEnvVarSensitivityImmutable) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "update app defined env var"))
			return
		}
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app(%s) model not found", app.Name))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update app defined env var"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeEnvVars),
		audit.WithDataBefore(oldEnvVar),
		audit.WithDataAfter(updatedEnvVar),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.UpdateAppDefinedEnvVarOutput{
		Data: new(serializer.AppDefinedEnvVarOutputObj).FromModel(*updatedEnvVar),
	})
}

// DeleteAppDefinedEnvVar 删除应用直接定义的环境变量。
//
//	@Summary		删除应用直接定义的环境变量
//	@Description	按应用内 key 删除，不再通过整份 AppModelSpec 回写。
//	@Tags			envvars
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			key		path		string	true	"环境变量 Key"
//	@Success		200		{object}	serializer.EmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/env-vars/{key} [delete]
func (h *Handler) DeleteAppDefinedEnvVar(c *gin.Context) {
	var uriInput serializer.AppEnvVarURIInput
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
	if err = ensureAppSupportsDefinedEnvVars(app); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	deletedEnvVar, err := appmodel.NewAppEnvVarService(h.registry.AppModelStore).Delete(ctx, app.ID, uriInput.Key)
	if err != nil {
		if errors.Is(err, appmodel.ErrEnvVarNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app env var %s not found", uriInput.Key))
			return
		}
		if errors.Is(err, appmodel.ErrAppModelNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app(%s) model not found", app.Name))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete app defined env var"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeEnvVars),
		audit.WithDataBefore(deletedEnvVar),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// ListEnvAvailableEnvVars 查询指定环境下所有可用的环境变量列表。
//
//	@ID				ListEnvAvailableEnvVars
//	@Summary		查询指定环境下所有可用的环境变量列表
//	@Description	包含所有内置、公共及当前环境下直接配置的全部变量，结果列表已按优先级去重。这些变量将在创建应用等应用缺席的场景中被使用。
//	@Tags			envvars
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string	true	"环境 ID"
//	@Success		200		{object}	serializer.ListEnvAvailableEnvVarsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/envs/{envID}/available-env-vars [get]
func (h *Handler) ListEnvAvailableEnvVars(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	reader := envvars.NewUnifiedEnvVarsReader(
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	)
	envVars, err := reader.ListEnvVars(ctx, *env)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list available env vars"))
		return
	}

	data := lo.Map(envVars, func(item envvartypes.EnvVariableObj, _ int) *serializer.EnvVarOutputObj {
		return new(serializer.EnvVarOutputObj).FromModel(item)
	})
	ginutils.OK(c, serializer.ListEnvAvailableEnvVarsOutput{Data: data})
}

// validateScopedEnvVarScopeValue validates scopeValue that depends on persistent workspace data.
// It is intentionally kept outside go-playground/validator because env scope
// validation needs registry/database access to ensure the named environment
// exists in the current workspace.
func (h *Handler) validateScopedEnvVarScopeValue(
	ctx context.Context,
	workspaceID string,
	scope envvartypes.ScopedEnvVarScope,
) error {
	if scope.ScopeType != envvartypes.ScopeTypeEnv {
		return nil
	}

	if _, err := h.registry.EnvStore.GetStdEnvByName(ctx, workspaceID, scope.ScopeValue); err != nil {
		if errors.Is(err, envmodel.ErrEnvNotFound) {
			return bkerrs.Errorf(bkerrs.ErrCodeNotFound, "environment %s not found", scope.ScopeValue)
		}
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get standard environment by name")
	}
	return nil
}

func ensureAppSupportsDefinedEnvVars(app *bkmsapp.Application) error {
	if bkmsapp.IsAppModelType(app.Type) {
		return nil
	}
	return bkerrs.Errorf(
		bkerrs.ErrCodeInvalidRequest,
		"app type %s does not support app-defined env vars",
		app.Type,
	)
}
