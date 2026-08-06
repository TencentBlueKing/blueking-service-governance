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
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/serializer"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// ListAppEnvVars 获取应用部署到某个环境后可用的环境变量。
//
//	@ID				ListAppEnvVars
//	@Summary		获取应用部署到某个环境后可用的环境变量
//	@Tags			envvars
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string	true	"应用 ID"
//	@Param			envName		path		string	true	"环境名称"
//	@Success		200		{object}	serializer.ListAppEnvVarsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/envs/{envName}/env-variables [get]
func (h *Handler) ListAppEnvVars(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	var am *appmodel.AppModel
	am, err = h.registry.AppModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		if !errors.Is(err, appmodel.ErrAppModelNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get app model for app id=%s", app.ID),
			)
			return
		}
		am = nil
	}

	reader := envvars.NewUnifiedEnvVarsReader(
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	)
	envVars, err := envvars.BuildAppEnvVars(ctx, app, am, env, reader)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "build app env vars"))
		return
	}

	ginutils.OK(c, serializer.ListAppEnvVarsOutput{
		Data: lo.Map(envVars, func(item envvartypes.EnvVariableObj, _ int) *serializer.EnvVarOutputObj {
			return new(serializer.EnvVarOutputObj).FromModel(item)
		}),
	})
}

// ListEnvBgEnvVars 查询指定环境的背景环境变量列表。
//
//	@ID				ListEnvBgEnvVars
//	@Summary		查询指定环境的背景环境变量列表
//	@Description	背景环境变量指除了作用域为当前环境的其他环境变量，结果列表已按优先级对同 Key 变量去重。
//	@Tags			envvars
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string	true	"环境 ID"
//	@Success		200		{object}	serializer.ListEnvBgEnvVarsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/envs/{envID}/bg-env-vars [get]
func (h *Handler) ListEnvBgEnvVars(c *gin.Context) {
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
	bgVars, err := reader.ListEnvBgVars(ctx, *env)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list env background env vars"))
		return
	}

	ginutils.OK(c, serializer.ListEnvBgEnvVarsOutput{
		Data: lo.Map(
			bgVars.ToDeduplicatedList().Vars,
			func(item envvartypes.EnvVariableRichItem, _ int) *serializer.BgEnvVarOutputObj {
				return new(serializer.BgEnvVarOutputObj).FromModel(item)
			},
		),
	})
}

// ListAppBgEnvVars 查询应用在某个环境下的背景环境变量列表。
//
//	@ID				ListAppBgEnvVars
//	@Summary		查询应用在某个环境下的背景环境变量列表
//	@Description	背景环境变量指除了作用域为当前应用的其他环境变量，结果列表已按优先级对同 Key 变量去重。
//	@Tags			envvars
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string	true	"应用 ID"
//	@Param			envName		path		string	true	"环境名称"
//	@Success		200		{object}	serializer.ListAppBgEnvVarsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/envs/{envName}/bg-env-vars [get]
func (h *Handler) ListAppBgEnvVars(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
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

	env, err := h.registry.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, uriInput.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeNotFound, "get env by name"))
		return
	}

	reader := envvars.NewUnifiedEnvVarsReader(
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	)
	bgVars, err := reader.ListAppBgVars(ctx, *env, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list app background env vars"))
		return
	}

	ginutils.OK(c, serializer.ListAppBgEnvVarsOutput{
		Data: lo.Map(
			bgVars.ToDeduplicatedList().Vars,
			func(item envvartypes.EnvVariableRichItem, _ int) *serializer.BgEnvVarOutputObj {
				return new(serializer.BgEnvVarOutputObj).FromModel(item)
			},
		),
	})
}
