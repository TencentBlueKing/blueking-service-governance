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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// CreateBinding 创建应用侧依赖服务绑定。
//
//	@ID			CreateServiceBinding
//	@Summary	创建依赖服务绑定
//	@Tags		depservice-binding
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string								true	"应用 ID"
//	@Param		serviceName	path		string								true	"依赖服务名，目前仅支持 redis"	Enums(redis)
//	@Param		body		body		serializer.CreateBindingInput	true	"请求体"
//	@Success	201			{object}	serializer.BindingOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/{serviceName}/bindings [post]
func (h *Handler) CreateBinding(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var jsonInput serializer.CreateBindingInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	envMap, err := serializer.ParseEnvInstanceMap(jsonInput.EnvInstanceMap)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "parse envInstanceMap"))
		return
	}
	if err = serializer.ValidateEnvVars(jsonInput.EnvVars); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "validate envVars"))
		return
	}

	binding, err := h.serviceManager().CreateServiceBinding(ctx, &depservice.CreateServiceBindingParams{
		Name:           jsonInput.Name,
		AppID:          app.ID,
		WorkspaceID:    app.WorkspaceID,
		ServiceName:    uriInput.ServiceName,
		EnvInstanceMap: envMap,
		EnvVars:        jsonInput.EnvVars,
		Description:    jsonInput.Description,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "create service binding"))
		return
	}

	ginutils.Created(c, serializer.BindingOutput{
		Data: new(serializer.BindingOutputObj).FromModel(binding),
	})
}

// ListBindings 列出应用下某依赖服务的绑定。
//
//	@ID			ListServiceBindings
//	@Summary	查询依赖服务绑定列表
//	@Tags		depservice-binding
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		serviceName	path		string	true	"依赖服务名，目前仅支持 redis"	Enums(redis)
//	@Success	200			{object}	serializer.ListBindingsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/{serviceName}/bindings [get]
func (h *Handler) ListBindings(c *gin.Context) {
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

	bindings, err := h.serviceManager().ListServiceBindings(ctx, app.ID, uriInput.ServiceName)
	if err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "list service bindings"))
		return
	}

	ginutils.OK(c, serializer.ListBindingsOutput{
		Data: lo.Map(bindings, func(b *model.ServiceBinding, _ int) *serializer.BindingOutputObj {
			return new(serializer.BindingOutputObj).FromModel(b)
		}),
	})
}

// GetBinding 查询依赖服务绑定详情。
//
//	@ID			GetServiceBinding
//	@Summary	查询依赖服务绑定详情
//	@Tags		depservice-binding
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		serviceName	path		string	true	"依赖服务名，目前仅支持 redis"	Enums(redis)
//	@Param		name		path		string	true	"绑定名称"
//	@Success	200			{object}	serializer.BindingOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/{serviceName}/bindings/{name} [get]
func (h *Handler) GetBinding(c *gin.Context) {
	var uriInput serializer.AppBindingNameURIInput
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

	binding, err := h.serviceManager().GetServiceBinding(ctx, app.ID, uriInput.ServiceName, uriInput.Name)
	if err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "get service binding"))
		return
	}

	ginutils.OK(c, serializer.BindingOutput{
		Data: new(serializer.BindingOutputObj).FromModel(binding),
	})
}

// UpdateBinding 全量更新绑定的环境映射与环境变量。
//
//	@ID			UpdateServiceBinding
//	@Summary	更新依赖服务绑定
//	@Tags		depservice-binding
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string								true	"应用 ID"
//	@Param		serviceName	path		string								true	"依赖服务名，目前仅支持 redis"	Enums(redis)
//	@Param		name		path		string								true	"绑定名称"
//	@Param		body		body		serializer.UpdateBindingInput	true	"请求体"
//	@Success	200			{object}	serializer.BindingOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/{serviceName}/bindings/{name} [put]
func (h *Handler) UpdateBinding(c *gin.Context) {
	var uriInput serializer.AppBindingNameURIInput
	var jsonInput serializer.UpdateBindingInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	envMap, err := serializer.ParseEnvInstanceMap(jsonInput.EnvInstanceMap)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "parse envInstanceMap"))
		return
	}
	if err = serializer.ValidateEnvVars(jsonInput.EnvVars); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "validate envVars"))
		return
	}

	binding, err := h.serviceManager().UpdateServiceBinding(ctx, app.ID, uriInput.ServiceName, uriInput.Name,
		&depservice.UpdateServiceBindingParams{
			EnvInstanceMap: envMap,
			EnvVars:        jsonInput.EnvVars,
			Description:    jsonInput.Description,
		},
	)
	if err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "update service binding"))
		return
	}

	ginutils.OK(c, serializer.BindingOutput{
		Data: new(serializer.BindingOutputObj).FromModel(binding),
	})
}

// DeleteBinding 删除依赖服务绑定。
//
//	@ID			DeleteServiceBinding
//	@Summary	删除依赖服务绑定
//	@Tags		depservice-binding
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		serviceName	path		string	true	"依赖服务名，目前仅支持 redis"	Enums(redis)
//	@Param		name		path		string	true	"绑定名称"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/deps/{serviceName}/bindings/{name} [delete]
func (h *Handler) DeleteBinding(c *gin.Context) {
	var uriInput serializer.AppBindingNameURIInput
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

	if err = h.serviceManager().DeleteServiceBinding(ctx, app.ID, uriInput.ServiceName, uriInput.Name); err != nil {
		bkerrs.AbortWithErr(c, mapManagerErr(err, "delete service binding"))
		return
	}

	ginutils.OK(c, serializer.EmptyOutput{})
}
