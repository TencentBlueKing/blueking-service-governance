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
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	httpresp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/http"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/export"
	importer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/import"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/serializer"
)

const (
	exportScopeAppDefined     = "appDefined"
	exportScopeEffectiveByEnv = "effectiveByEnv"
)

// ImportPublicScopedEnvVar 正式导入公共环境变量。
//
//	@ID				ImportPublicScopedEnvVar
//	@Summary		正式导入公共环境变量
//	@Description	解析并导入公共环境变量，导入语义与预览接口保持一致。
//	@Tags			envvars
//	@Accept			mpfd
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string							true	"工作空间 ID"
//	@Param			file		formData	file							true	"公共环境变量导入请求文件"
//	@Success		200			{object}	serializer.ImportEnvVarOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/scoped-env-vars/public-vars/import [post]
func (h *Handler) ImportPublicScopedEnvVar(c *gin.Context) {
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
	// 正式写入前复用 preview 的校验语义，确保用户在预览和导入阶段看到的
	// scope 规则与格式规则完全一致。
	if err := h.newImportService().ImportPublic(ctx, ws.ID, content); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "import public scoped env vars"))
		return
	}
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeScopedEnvVars),
		audit.WithDataAfter(preview.Summary),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.ImportEnvVarOutput{
		Data: new(serializer.EnvVarImportPreviewSummaryOutputObj).FromModel(preview.Summary),
	})
}

// ImportEnvScopedEnvVar 正式导入单环境环境变量。
//
//	@ID				ImportEnvScopedEnvVar
//	@Summary		正式导入单环境环境变量
//	@Description	解析并导入当前环境作用域的环境变量，导入语义与预览接口保持一致。
//	@Tags			envvars
//	@Accept			mpfd
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string							true	"环境 ID"
//	@Param			file	formData	file							true	"单环境环境变量导入请求文件"
//	@Success		200		{object}	serializer.ImportEnvVarOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/scoped-env-vars/import/{envID} [post]
func (h *Handler) ImportEnvScopedEnvVar(c *gin.Context) {
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
	// 先走 preview，避免坏文件在目标 env scope 里产生部分写入。
	if err := h.newImportService().ImportEnv(ctx, *environment, content); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "import env scoped env vars"))
		return
	}
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeEnv,
		environment.ID.Hex(),
		audit.WithAttribute(audit.AttributeScopedEnvVars),
		audit.WithDataAfter(preview.Summary),
		audit.WithWorkspaceID(environment.WorkspaceID),
		audit.WithEnvName(environment.Name),
	)

	ginutils.OK(c, serializer.ImportEnvVarOutput{
		Data: new(serializer.EnvVarImportPreviewSummaryOutputObj).FromModel(preview.Summary),
	})
}

// ImportAppDefinedEnvVar 正式导入应用直接定义的环境变量。
//
//	@ID				ImportAppDefinedEnvVar
//	@Summary		正式导入应用直接定义的环境变量
//	@Description	解析并导入应用直接定义的环境变量，导入语义与预览接口保持一致。
//	@Tags			envvars
//	@Accept			mpfd
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string							true	"应用 ID"
//	@Param			file	formData	file							true	"应用环境变量导入请求文件"
//	@Success		200		{object}	serializer.ImportEnvVarOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/env-vars/import [post]
func (h *Handler) ImportAppDefinedEnvVar(c *gin.Context) {
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
	// 应用直接定义变量的导入沿用同一套 preview 契约，这样不支持的 scope
	// 元数据会在修改 workload.envVars 之前就被拦住。
	if err := h.newImportService().ImportApp(ctx, app, content); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "import app defined env vars"))
		return
	}
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeEnvVars),
		audit.WithDataAfter(preview.Summary),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.ImportEnvVarOutput{
		Data: new(serializer.EnvVarImportPreviewSummaryOutputObj).FromModel(preview.Summary),
	})
}

// ExportPublicScopedEnvVars 下载公共环境变量。
//
//	@ID				ExportPublicScopedEnvVars
//	@Summary		下载公共环境变量
//	@Tags			envvars
//	@Produce		octet-stream
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{string}	string	"dotenv file"
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/scoped-env-vars/public-vars/export [get]
func (h *Handler) ExportPublicScopedEnvVars(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
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

	content, err := h.newExportService().ExportPublic(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "export public scoped env vars"))
		return
	}

	writeEnvFileAttachment(c, "public-scoped-env-vars.env", content)
}

// ExportEnvScopedEnvVars 下载单环境环境变量。
//
//	@ID				ExportEnvScopedEnvVars
//	@Summary		下载单环境环境变量
//	@Tags			envvars
//	@Produce		octet-stream
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string	true	"环境 ID"
//	@Success		200		{string}	string	"dotenv file"
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/scoped-env-vars/export/{envID} [get]
func (h *Handler) ExportEnvScopedEnvVars(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	environment, err := perm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	content, err := h.newExportService().ExportEnv(ctx, *environment)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "export env scoped env vars"))
		return
	}

	writeEnvFileAttachment(c, buildFilename("env", environment.Name, "scoped-env-vars.env"), content)
}

// ExportAppEnvVars 下载应用环境变量。
//
//	@ID				ExportAppEnvVars
//	@Summary		下载应用环境变量
//	@Description	支持导出应用直接定义的环境变量，或按环境导出最终生效的全部环境变量。
//	@Tags			envvars
//	@Produce		octet-stream
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string								true	"应用 ID"
//	@Param			scope	query		string								true	"导出范围：appDefined 或 effectiveByEnv"
//	@Param			envName	query		string								false	"环境名称；scope=effectiveByEnv 时必填"
//	@Success		200		{string}	string								"dotenv file"
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/env-vars/export [get]
func (h *Handler) ExportAppEnvVars(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var queryInput serializer.AppEnvVarsExportQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	switch queryInput.Scope {
	case exportScopeAppDefined:
		app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
		if err != nil {
			bkerrs.AbortWithErr(c, err)
			return
		}
		if err = ensureAppSupportsDefinedEnvVars(app); err != nil {
			bkerrs.AbortWithErr(c, err)
			return
		}

		content, err := h.newExportService().ExportAppDefined(ctx, app.ID)
		if err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "export app defined env vars"))
			return
		}
		writeEnvFileAttachment(c, buildFilename("app", app.Name, "env-vars.env"), content)
	case exportScopeEffectiveByEnv:
		// 最终生效变量的导出必须带环境名，因为结果依赖该环境下
		// workspace / envType / env / app 的层叠覆盖关系。
		if strings.TrimSpace(queryInput.EnvName) == "" {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(
					bkerrs.ErrCodeInvalidRequest,
					"envName is required when scope=%s",
					exportScopeEffectiveByEnv,
				),
			)
			return
		}
		app, environment, err := perm.ValidateAppEnvByName(
			ctx,
			h.registry,
			uriInput.AppID,
			queryInput.EnvName,
			perm.TypeEdit,
		)
		if err != nil {
			bkerrs.AbortWithErr(c, err)
			return
		}
		if err = ensureAppSupportsDefinedEnvVars(app); err != nil {
			bkerrs.AbortWithErr(c, err)
			return
		}

		content, err := h.newExportService().ExportEffectiveAppEnv(ctx, app, environment)
		if err != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "export effective app env vars"),
			)
			return
		}
		writeEnvFileAttachment(c, buildFilename("app", app.Name, environment.Name, "effective-env-vars.env"), content)
	default:
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "unsupported export scope"))
	}
}

func writeEnvFileAttachment(c *gin.Context, filename, content string) {
	c.Header("Content-Disposition", httpresp.BuildAttachmentDisposition(filename))
	c.Data(http.StatusOK, httpresp.AttachmentContentType, []byte(content))
}

func buildFilename(parts ...string) string {
	sanitized := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		sanitized = append(sanitized, trimmed)
	}
	return strings.Join(sanitized, "-")
}

func (h *Handler) newImportService() *importer.Service {
	return importer.NewService(h.registry.ScopedEnvVarStore, h.registry.AppModelStore)
}

func (h *Handler) newExportService() *export.Service {
	return export.NewService(
		h.registry.ScopedEnvVarStore,
		h.registry.AppModelStore,
		envvars.NewUnifiedEnvVarsReader(
			h.registry.ScopedEnvVarStore,
			h.registry.AppDepsVarReader,
			h.registry.PolarisVarReader,
		),
	)
}
