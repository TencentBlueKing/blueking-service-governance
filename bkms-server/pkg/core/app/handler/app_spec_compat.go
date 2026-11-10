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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/lifecycle"
)

// UpdateAppTrpcSpec 更新应用 Trpc 配置。
//
//	@ID				UpdateAppTrpcSpec
//	@Summary		更新应用 Trpc 配置
//	@Description	已废弃，请改用 PUT /apps/{appID}/startup-config。不允许修改语言
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
//	@Description	已废弃，请改用 PUT /apps/{appID}/startup-config
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

// updateAppSpecCompat 按新模型处理旧 trpc-spec / taf-spec 接口：旧类型应用与 bkmsApp 应用都可修改，
// 框架须与接口一致，不允许修改语言，不读写旧字段。
//
// Deprecated: 仅供旧 spec 接口兼容，旧接口下线后删除。
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
