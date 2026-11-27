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

// 本文件是旧 trpc / taf 接口的兼容层：入参、出参结构保持不变，内部一律按新模型处理，
// 不读写 Application.TrpcSpec、Workload.Type / TrpcConfig / TafConfig 等旧字段。

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/lifecycle"
)

// UpdateAppTrpcSpec 更新应用 Trpc 配置。
//
//	@ID				UpdateAppTrpcSpec
//	@Summary		更新应用 Trpc 配置
//	@Description	已废弃：启动配置已统一由 startup-config 修改，请改用 PUT /apps/{appID}/startup-config。不允许修改语言
//	@Tags			app
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string								true	"应用 ID"
//	@Param			body	body		serializer.UpdateAppModelSpecInput	true	"更新 Trpc 配置请求"
//	@Success		200		{object}	serializer.EmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		409		{object}	bkerrs.GinErrorOutput	"文件名和挂载目录已存在，details.code=APP_CONFIG_FILE_MOUNT_IDENTITY_CONFLICT"
//	@Deprecated
//	@Router			/apps/{appID}/trpc-spec [put]
func (h *Handler) UpdateAppTrpcSpec(c *gin.Context) {
	h.updateAppSpecCompat(c, appruntime.FrameworkTRPC)
}

// UpdateAppTafSpec 更新应用 Taf 配置。
//
//	@ID				UpdateAppTafSpec
//	@Summary		更新应用 Taf 配置
//	@Description	已废弃：启动配置已统一由 startup-config 修改，请改用 PUT /apps/{appID}/startup-config
//	@Tags			app
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string								true	"应用 ID"
//	@Param			body	body		serializer.UpdateAppModelSpecInput	true	"更新 Taf 配置请求"
//	@Success		200		{object}	serializer.EmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Failure		409		{object}	bkerrs.GinErrorOutput	"文件名和挂载目录已存在，details.code=APP_CONFIG_FILE_MOUNT_IDENTITY_CONFLICT"
//	@Deprecated
//	@Router			/apps/{appID}/taf-spec [put]
func (h *Handler) UpdateAppTafSpec(c *gin.Context) {
	h.updateAppSpecCompat(c, appruntime.FrameworkTAF)
}

// updateAppSpecCompat 处理旧 trpc-spec / taf-spec 接口：旧类型应用与 bkmsApp 应用都可修改，
// 框架须与接口一致，不允许修改语言。
//
// Deprecated: bkmsApp 迁移兼容，仅供旧 trpc-spec / taf-spec 接口使用。
func (h *Handler) updateAppSpecCompat(c *gin.Context, framework string) {
	var uriInput serializer.AppURIInput
	var input serializer.UpdateAppModelSpecInput
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

	spec := input.AppModelSpec
	if spec == nil ||
		(framework == appruntime.FrameworkTRPC && spec.TrpcSpec == nil) ||
		(framework == appruntime.FrameworkTAF && spec.TafSpec == nil) {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "appModelSpec.%sSpec is required", framework),
		)
		return
	}
	params := spec.ToSpecCompatParams(framework)

	oldModel, newModel, err := lifecycle.NewService(h.registry.DB).UpdateSpecCompat(ctx, app, params)
	if err != nil {
		if errors.Is(err, appcfg.ErrAppConfigFileDefAlreadyExists) {
			bkerrs.AbortWithErr(c, bkerrs.WrapAppConfigFileMountIdentityConflict(
				err, app.ID, params.FileName, params.FilePath,
			))
			return
		}
		code := bkerrs.ErrCodeInternalServerError
		if errors.Is(err, lifecycle.ErrInvalidParams) || errors.Is(err, appcfg.ErrInvalidConfigSpec) {
			code = bkerrs.ErrCodeInvalidRequest
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, code, "update %s spec", framework))
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

// createBKMSAppCompat 旧创建接口传 trpc / taf 时，按 bkmsApp 创建应用，响应结构不变。
//
// Deprecated: bkmsApp 迁移兼容，仅供旧创建接口使用。
func (h *Handler) createBKMSAppCompat(c *gin.Context, workspaceID string, input *serializer.CreateAppInput) {
	framework, language, params := input.ToBKMSAppCompat()
	app := &bkmsapp.Application{
		ID:          input.ID,
		WorkspaceID: workspaceID,
		Name:        input.Name,
		Type:        appruntime.AppTypeBKMSApp,
		Framework:   framework,
		Language:    language,
		DisplayName: input.Name,
		Creator:     auth.MustGetUser(c.Request.Context()).ID,
		CreatedAt:   time.Now(),
	}
	h.createBKMSApp(c, app, params, input.BuildConfig)
}

// frameworkSpecView 准备详情中 trpcSpec / tafSpec 的取值。
// 框架与语言取自应用；文件名、挂载目录与默认内容取自应用的框架配置文件。
// 应用没有框架配置文件或有多个时，文件相关字段留空，详情接口照常返回。
//
// Deprecated: bkmsApp 迁移兼容，仅供详情输出 trpcSpec / tafSpec。
func (h *Handler) frameworkSpecView(
	ctx context.Context,
	app *bkmsapp.Application,
) (serializer.FrameworkSpecView, error) {
	if !appruntime.IsAppModelType(app.Type) {
		return serializer.FrameworkSpecView{}, nil
	}
	stack := mods.Of(app).Stack
	view := serializer.FrameworkSpecView{Framework: stack.Framework, Language: stack.Language}
	defs, err := appcfg.ListDefsByApp(
		ctx,
		store.Of(h.registry.DB, appcfg.AppConfigFileDefMeta),
		app.ID,
		appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework),
	)
	if err != nil {
		return view, errors.Wrapf(err, "list framework defs for app %s", app.ID)
	}
	if len(defs) != 1 {
		return view, nil
	}
	file, err := appcfg.NewMountableFileProvider(h.registry.DB).GetFrameworkMountableFile(ctx, app.ID, "")
	if err != nil {
		return view, errors.Wrapf(err, "get framework config file for app %s", app.ID)
	}
	view.FileName, view.FilePath, view.FileContent = file.Name, file.MountDir, file.Content
	return view, nil
}
