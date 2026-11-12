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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/lifecycle"
)

// CreateBKMSApp 创建 bkmsApp 应用。
//
//	@ID			CreateBKMSApp
//	@Summary	创建 bkmsApp 应用
//	@Tags		app
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		body		body		serializer.CreateBKMSAppInput	true	"创建 bkmsApp 应用请求"
//	@Success	200			{object}	serializer.CreateAppOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkmsapps [post]
func (h *Handler) CreateBKMSApp(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var input serializer.CreateBKMSAppInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if err := h.checkAppCreatable(ctx, uriInput.WorkspaceID, input.Name); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	app := &bkmsapp.Application{
		ID:          input.ID,
		WorkspaceID: uriInput.WorkspaceID,
		Name:        input.Name,
		Type:        appruntime.AppTypeBKMSApp,
		Framework:   input.Framework,
		Language:    input.Language,
		DisplayName: input.Name,
		Creator:     auth.MustGetUser(ctx).ID,
		CreatedAt:   time.Now(),
	}
	h.createBKMSApp(c, app, input.ToCreateParams(), input.BuildConfig)
}

// createBKMSApp 校验参数、写入构建配置，再创建 bkmsApp 应用的其余数据并返回结果。
func (h *Handler) createBKMSApp(
	c *gin.Context,
	app *bkmsapp.Application,
	params *lifecycle.CreateParams,
	buildConfigInput *serializer.BuildConfigInput,
) {
	ctx := c.Request.Context()
	// 写入构建配置前先校验，参数不合法时不留下任何数据
	if err := lifecycle.ValidateCreate(app, params); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "validate bkmsApp"))
		return
	}

	buildConfig, err := h.createBuildConfig(ctx, app, buildConfigInput)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err = lifecycle.NewService(h.registry.DB).Create(ctx, app, params); err != nil {
		code := bkerrs.ErrCodeInternalServerError
		if errors.Is(err, lifecycle.ErrInvalidParams) || errors.Is(err, appcfg.ErrInvalidConfigSpec) {
			code = bkerrs.ErrCodeInvalidRequest
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, code, "create bkmsApp"))
		return
	}

	h.onAppCreated(ctx, app, buildConfig)

	ginutils.OK(c, serializer.CreateAppOutput{
		Data: new(serializer.AppOutputObj).FromModel(app),
	})
}

// UpdateAppStartupConfig 修改 bkmsApp 应用的启动配置：启动命令、参数、脚本与框架扩展配置。
//
//	@ID			UpdateAppStartupConfig
//	@Summary	修改 bkmsApp 应用的启动配置
//	@Tags		app
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string									true	"应用 ID"
//	@Param		body	body		serializer.UpdateAppStartupConfigInput	true	"修改请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/startup-config [put]
func (h *Handler) UpdateAppStartupConfig(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.UpdateAppStartupConfigInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if app.Type != appruntime.AppTypeBKMSApp {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest,
			"app type %s does not support startup-config, use trpc-spec or taf-spec", app.Type))
		return
	}
	params, err := input.ToUpdateStartupConfigParams()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "parse startup config"))
		return
	}

	oldModel, newModel, err := lifecycle.NewService(h.registry.DB).UpdateStartupConfig(ctx, app, params)
	if err != nil {
		code := lo.Ternary(errors.Is(err, lifecycle.ErrInvalidParams),
			bkerrs.ErrCodeInvalidRequest, bkerrs.ErrCodeInternalServerError)
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, code, "update startup config"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeAppModel),
		audit.WithDataBefore(oldModel),
		audit.WithDataAfter(newModel),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// GetAppRuntimeDefinitions 查询框架目录与能力矩阵。
//
//	@ID			GetAppRuntimeDefinitions
//	@Summary	查询框架目录与能力矩阵
//	@Tags		app
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.GetAppRuntimeDefinitionsOutput
//	@Router		/app-runtime-definitions [get]
func (h *Handler) GetAppRuntimeDefinitions(c *gin.Context) {
	matrix, err := mods.Matrix()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "build runtime mod matrix"))
		return
	}
	ginutils.OK(c, serializer.GetAppRuntimeDefinitionsOutput{
		Data: new(serializer.AppRuntimeDefinitionsOutputObj).FromModel(appruntime.Catalog(), matrix),
	})
}

// GetAppRuntimeMods 查询应用的能力集合。
//
//	@ID			GetAppRuntimeMods
//	@Summary	查询应用的能力集合
//	@Tags		app
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.GetAppRuntimeModsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/runtime-mods [get]
func (h *Handler) GetAppRuntimeMods(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	app, err := ginperm.ValidateAppByID(c.Request.Context(), h.registry, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ginutils.OK(c, serializer.GetAppRuntimeModsOutput{Data: mods.Of(app).Public()})
}
