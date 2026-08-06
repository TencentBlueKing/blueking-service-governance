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

// Package handler 包含 envvars 模块的 Gin HTTP 处理器。
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	parserpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/parser"
	previewsvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/preview"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/serializer"
)

func (h *Handler) newImportPreviewService() *previewsvc.ImportPreviewService {
	return previewsvc.NewPreviewService(
		h.registry.ScopedEnvVarStore,
		h.registry.AppModelStore,
	)
}

// PreviewPublicScopedEnvVar 预览导入公共环境变量
//
//	@ID				PreviewPublicScopedEnvVar
//	@Summary		预览导入公共环境变量
//	@Description	解析 `.env` 文本并返回公共环境变量导入预览结果，不会保存任何变更。
//	@Tags			envvars
//	@Accept			mpfd
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string							true	"工作空间 ID"
//	@Param			file		formData	file							true	"公共环境变量导入预览请求文件"
//	@Success		200			{object}	serializer.PreviewEnvVarOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/scoped-env-vars/public-vars/preview [post]
func (h *Handler) PreviewPublicScopedEnvVar(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	content, err := readUploadedEnvFileContent(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	preview, err := h.newImportPreviewService().PreviewPublic(ctx, ws.ID, content)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapPreviewError(err))
		return
	}

	ginutils.OK(c, serializer.PreviewEnvVarOutput{
		Data: new(serializer.EnvVarImportPreviewOutputObj).FromModel(preview),
	})
}

// PreviewEnvScopedEnvVar 预览导入单环境环境变量
//
//	@ID				PreviewEnvScopedEnvVar
//	@Summary		预览导入单环境环境变量
//	@Description	解析 `.env` 文本并返回当前环境的导入预览结果，不会保存任何变更。
//	@Tags			envvars
//	@Accept			mpfd
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string							true	"环境 ID"
//	@Param			file	formData	file							true	"单环境环境变量导入预览请求文件"
//	@Success		200		{object}	serializer.PreviewEnvVarOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/scoped-env-vars/preview/{envID} [post]
func (h *Handler) PreviewEnvScopedEnvVar(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	content, err := readUploadedEnvFileContent(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	environment, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	preview, err := h.newImportPreviewService().PreviewEnv(ctx, *environment, content)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapPreviewError(err))
		return
	}

	ginutils.OK(c, serializer.PreviewEnvVarOutput{
		Data: new(serializer.EnvVarImportPreviewOutputObj).FromModel(preview),
	})
}

// PreviewAppDefinedEnvVar 预览导入应用直接定义的环境变量
//
//	@ID				PreviewAppDefinedEnvVar
//	@Summary		预览导入应用直接定义的环境变量
//	@Description	解析 `.env` 文本并返回应用环境变量导入预览结果，不会保存任何变更。
//	@Tags			envvars
//	@Accept			mpfd
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string							true	"应用 ID"
//	@Param			file	formData	file							true	"应用环境变量导入预览请求文件"
//	@Success		200		{object}	serializer.PreviewEnvVarOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/env-vars/preview [post]
func (h *Handler) PreviewAppDefinedEnvVar(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	content, err := readUploadedEnvFileContent(c)
	if err != nil {
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

	preview, err := h.newImportPreviewService().PreviewApp(ctx, app, content)
	if err != nil {
		bkerrs.AbortWithErr(c, wrapPreviewError(err))
		return
	}

	ginutils.OK(c, serializer.PreviewEnvVarOutput{
		Data: new(serializer.EnvVarImportPreviewOutputObj).FromModel(preview),
	})
}

// wrapPreviewError 将预览服务返回的 error 包装为 bkerrs 错误。
// 内容校验错误（parser.ErrInvalidEnvFileContent）返回 HTTP 400，其他错误返回 HTTP 500。
func wrapPreviewError(err error) error {
	if errors.Is(err, parserpkg.ErrInvalidEnvFileContent) {
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "env file content validation failed")
	}
	return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "env var import preview")
}
