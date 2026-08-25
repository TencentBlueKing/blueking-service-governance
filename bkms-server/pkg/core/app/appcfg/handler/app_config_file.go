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

// Package handler contains Gin handlers for app config file APIs.
package handler

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ appcfg.Handler = (*Handler)(nil)

// Handler handles Gin app config file API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// CreateAppConfigFile 创建一个应用配置文件。
//
//	@ID				CreateAppConfigFile
//	@Summary		创建一个应用配置文件
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string								true	"应用 ID"
//	@Param			body	body		slz.CreateAppConfigFileInput	true	"创建应用配置文件请求"
//	@Success		200		{object}	slz.CreateAppConfigFileOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files [post]
func (h *Handler) CreateAppConfigFile(c *gin.Context) {
	var uriInput slz.AppURIInput
	var input slz.CreateAppConfigFileInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	configKind := normalizeConfigKind(input.ConfigKind)
	baseID, err := h.validateBaseAppConfigFileID(ctx, app.ID, configKind, input.Type, input.BaseAppConfigFileID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "validate baseAppConfigFileID"))
		return
	}

	fileFormat := appcfg.FileFormat(input.FileFormat)
	sourceType := appcfg.ContentSourceType(input.ContentSourceType)
	bscpCfg, err := h.validateBSCPConfig(ctx, configKind, sourceType, input.BscpConfig, fileFormat)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "validate BSCP config"))
		return
	}

	envName := appcfg.EnvNameDefault
	if input.EnvName != nil {
		envName = *input.EnvName
	}

	creator := auth.MustGetUser(ctx).ID
	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	obj, err := cfgService.Create(
		ctx,
		appcfg.CreateCfgFileParams{
			AppID:               app.ID,
			EnvName:             envName,
			Name:                input.Name,
			Type:                appcfg.AppConfigFileType(input.Type),
			ContentSourceType:   sourceType,
			Format:              fileFormat,
			ConfigKind:          configKind,
			MountPath:           input.MountPath,
			IsUnifiedConfig:     true, // 新建文件始终为统一配置，独立配置需后续通过 env-config-policy 接口开启
			MountedEnvNames:     input.MountedEnvNames,
			BaseAppConfigFileID: baseID,
			BSCPConfig:          bscpCfg,
			Creator:             creator,
			Description:         input.Description,
		},
	)
	if err != nil {
		if errors.Is(err, appcfg.ErrInvalidConfigSpec) ||
			errors.Is(err, appcfg.ErrPlainConfigMountPathConflict) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "creating app config file"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "creating app config file"))
		return
	}

	h.addAppConfigFileAudit(ctx, app, obj.EnvName, audit.OperationTypeCreate, nil, buildAppConfigFileAuditData(obj))
	ginutils.OK(c, slz.CreateAppConfigFileOutput{
		Item: new(slz.AppConfigFileOutputObj).FromModel(*obj),
	})
}

// UpdateAppConfigFile 修改一个应用配置文件的基础属性。
//
//	@ID				UpdateAppConfigFile
//	@Summary		修改一个应用配置文件的基础属性
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string								true	"应用 ID"
//	@Param			id		path		string								true	"应用配置文件 ID"
//	@Param			body	body		slz.UpdateAppConfigFileInput	true	"更新应用配置文件基础属性请求"
//	@Success		200		{object}	slz.UpdateAppConfigFileOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files/{id} [put]
func (h *Handler) UpdateAppConfigFile(c *gin.Context) {
	var uriInput slz.AppConfigFileURIInput
	var input slz.UpdateAppConfigFileInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, acf, err := h.validateAndGetAppConfigFile(ctx, uriInput.AppID, uriInput.ID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	oldAcf := *acf

	if err = h.applyUpdateInputToConfigFile(ctx, app.ID, acf, input); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "apply update input"))
		return
	}

	operator := auth.MustGetUser(ctx).ID
	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	if err = cfgService.UpdateFile(
		ctx,
		acf,
		operator,
		appcfg.UpdateCfgFileOptions{
			OperationType:          appcfg.AppConfigFileVersionOperationTypeUpdate,
			Description:            input.Description,
			ExpectedCurrentVersion: input.CurrentVersion,
		},
	); err != nil {
		if errors.Is(err, appcfg.ErrAppConfigFileVersionConflict) {
			bkerrs.AbortWithErr(c, bkerrs.WrapAppConfigFileVersionConflict(err, app.ID, acf.ID.Hex()))
			return
		}
		if errors.Is(err, appcfg.ErrInvalidConfigSpec) ||
			errors.Is(err, appcfg.ErrPlainConfigMountPathConflict) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "create app config file version"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create app config file version"))
		return
	}

	h.addAppConfigFileAudit(
		ctx,
		app,
		acf.EnvName,
		audit.OperationTypeUpdate,
		buildAppConfigFileAuditData(&oldAcf),
		buildAppConfigFileAuditData(acf),
	)
	ginutils.OK(c, slz.UpdateAppConfigFileOutput{
		Item: new(slz.AppConfigFileOutputObj).FromModel(*acf),
	})
}

func (h *Handler) applyUpdateInputToConfigFile(
	ctx context.Context,
	appID string,
	acf *appcfg.AppConfigFile,
	input slz.UpdateAppConfigFileInput,
) error {
	if input.BaseAppConfigFileID != "" {
		baseID, err := h.validateBaseAppConfigFileID(
			ctx, appID, acf.GetConfigKind(), string(acf.Type), input.BaseAppConfigFileID,
		)
		if err != nil {
			return err
		}
		acf.BaseAppConfigFileID = baseID
	}
	if input.BscpConfig != nil {
		bscpCfg, err := h.validateBSCPConfig(
			ctx, acf.GetConfigKind(), acf.ContentSourceType, input.BscpConfig, acf.GetConfigFormat(),
		)
		if err != nil {
			return err
		}
		acf.BSCPConfig = bscpCfg
	}
	acf.Name = input.Name
	if input.MountPath != "" {
		if acf.GetConfigKind() == appcfg.ConfigKindPlain &&
			acf.DefaultAppConfigFileID != nil &&
			input.MountPath != acf.MountPath {
			return pkgerrors.Wrap(appcfg.ErrInvalidConfigSpec, "plain env instance cannot change mountPath")
		}
		acf.MountPath = input.MountPath
	}
	if input.FileFormat != "" {
		acf.Format = appcfg.FileFormat(input.FileFormat)
	}
	return nil
}

// UpdateAppConfigFileEnvConfig 修改一个应用配置文件的环境配置模式。
//
//	@ID				UpdateAppConfigFileEnvConfig
//	@Summary		修改一个应用配置文件的环境配置模式
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string										true	"应用 ID"
//	@Param			id		path		string										true	"默认配置文件 ID（envName=__default__ 的记录）"
//	@Param			body	body		slz.UpdateAppConfigFileEnvConfigInput	true	"修改环境配置模式请求"
//	@Success		200		{object}	slz.UpdateAppConfigFileOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files/{id}/env-config-policy [put]
func (h *Handler) UpdateAppConfigFileEnvConfig(c *gin.Context) {
	var uriInput slz.AppConfigFileURIInput
	var input slz.UpdateAppConfigFileEnvConfigInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, acf, err := h.validateAndGetAppConfigFile(ctx, uriInput.AppID, uriInput.ID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	oldAcf := *acf

	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	err = cfgService.UpdateEnvConfig(ctx, acf, appcfg.UpdateEnvConfigParams{
		IsUnifiedConfig:        input.IsUnifiedConfig,
		MountedEnvNames:        input.MountedEnvNames,
		FallbackConfigEnv:      input.FallbackConfigEnv,
		Operator:               auth.MustGetUser(ctx).ID,
		Description:            input.Description,
		ExpectedCurrentVersion: input.CurrentVersion,
	})
	if err != nil {
		if errors.Is(err, appcfg.ErrAppConfigFileVersionConflict) {
			bkerrs.AbortWithErr(c, bkerrs.WrapAppConfigFileVersionConflict(err, app.ID, acf.ID.Hex()))
			return
		}
		if errors.Is(err, appcfg.ErrEnvConfigRequiresDefaultFile) ||
			errors.Is(err, appcfg.ErrInvalidConfigSpec) ||
			errors.Is(err, appcfg.ErrPlainConfigMountPathConflict) ||
			errors.Is(err, appcfg.ErrFallbackRequiresIndependentConfig) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "update env config"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update env config"))
		return
	}

	h.addAppConfigFileAudit(
		ctx,
		app,
		acf.EnvName,
		audit.OperationTypeUpdate,
		buildAppConfigFileAuditData(&oldAcf),
		buildAppConfigFileAuditData(acf),
	)
	ginutils.OK(c, slz.UpdateAppConfigFileOutput{
		Item: new(slz.AppConfigFileOutputObj).FromModel(*acf),
	})
}

// ListAppConfigFiles 查看一个应用所有的应用配置文件列表。
//
//	@ID				ListAppConfigFiles
//	@Summary		查看一个应用所有的应用配置文件列表
//	@Tags			app-config-files
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path		string	true	"应用 ID"
//	@Param			type		query		string	false	"按文件类型过滤，仅展示指定类型（normal/overlay）"
//	@Param			envName		query		string	false	"按环境名称过滤，可选。为空表示不过滤"
//	@Success		200			{object}	slz.ListAppConfigFilesOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files [get]
func (h *Handler) ListAppConfigFiles(c *gin.Context) {
	var uriInput slz.AppURIInput
	var queryInput slz.ListAppConfigFilesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	opts := []appcfg.AcfListOption{appcfg.AcfOrderBy(appcfg.ListOrderByName)}
	if val := lo.FromPtr(queryInput.Type); val != "" {
		opts = append(opts, appcfg.AcfFilterType(val))
	}
	if val := lo.FromPtr(queryInput.EnvName); val != "" {
		opts = append(opts, appcfg.AcfFilterEnvName(val))
	}
	appConfigFiles, err := h.registry.AppConfigFileStore.List(ctx, app.ID, opts...)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "listing app config files"))
		return
	}

	items := make([]*slz.AppConfigFileOutputObj, 0, len(appConfigFiles))
	for _, acf := range appConfigFiles {
		items = append(items, new(slz.AppConfigFileOutputObj).FromModel(acf))
	}
	ginutils.OK(c, slz.ListAppConfigFilesOutput{Items: items})
}

// DeleteAppConfigFile 通过 ID 删除应用的一个应用配置文件。
//
//	@ID				DeleteAppConfigFile
//	@Summary		通过 ID 删除应用的一个应用配置文件
//	@Tags			app-config-files
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			id		path		string	true	"应用配置文件 ID"
//	@Success		200		{object}	slz.AppConfigFileEmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files/{id} [delete]
func (h *Handler) DeleteAppConfigFile(c *gin.Context) {
	var uriInput slz.AppConfigFileURIInput
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

	id, err := uriInput.AppConfigFileObjectID()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parse app config file ID"))
		return
	}

	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	oldAcf, err := cfgService.DeleteFile(ctx, app.ID, id)
	if err != nil {
		if errors.Is(err, appcfg.ErrPlainEnvInstanceDeleteNotAllowed) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "deleting app config file"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "deleting app config file"))
		return
	}

	h.addAppConfigFileAudit(
		ctx,
		app,
		oldAcf.EnvName,
		audit.OperationTypeDelete,
		buildAppConfigFileAuditData(oldAcf),
		nil,
	)
	ginutils.OK(c, slz.AppConfigFileEmptyOutput{})
}

// GetAppConfigFileDetails 查看一个应用配置文件详情。
//
//	@ID				GetAppConfigFileDetails
//	@Summary		查看一个应用配置文件详情
//	@Tags			app-config-files
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			id		path		string	true	"应用配置文件 ID"
//	@Success		200		{object}	slz.GetAppConfigFileDetailsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files/{id}/details [get]
func (h *Handler) GetAppConfigFileDetails(c *gin.Context) {
	var uriInput slz.AppConfigFileURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	_, acf, err := h.validateAndGetAppConfigFile(ctx, uriInput.AppID, uriInput.ID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	provider, err := appcfg.NewBaseContentProvider(h.registry.AppConfigFileStore, acf)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "creating values file editor"))
		return
	}
	info, err := provider.GetInfo(ctx)
	if err != nil && !pkgerrors.Is(err, appcfg.ErrBaseContentEmpty) {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "getting base content info"))
		return
	}

	var baseContentInfo *slz.BaseContentInfoOutputObj
	if info != nil {
		baseContentInfo = &slz.BaseContentInfoOutputObj{
			HolderID:                info.HolderID.Hex(),
			HolderName:              info.HolderName,
			HolderContentSourceType: info.HolderContentSourceType,
			Content:                 info.Content,
			IsFromAnotherFile:       info.IsFromAnotherFile,
		}
	}

	editor, err := appcfg.NewAppConfigFileEditor(h.registry.AppConfigFileStore, acf)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "creating values file editor"))
		return
	}

	ginutils.OK(c, slz.GetAppConfigFileDetailsOutput{
		EditableContentField: string(editor.GetEditableContentField()),
		Content:              acf.Content,
		OverlayContent:       acf.OverlayContent,
		BaseContentInfo:      baseContentInfo,
		CurrentVersion:       acf.CurrentVersion,
		Updater:              acf.Updater,
		UpdatedAt:            acf.UpdatedAt.Format(time.RFC3339),
	})
}

// UpdateAppConfigFileContent 修改一个应用配置文件的 Content。
//
//	@ID				UpdateAppConfigFileContent
//	@Summary		修改一个应用配置文件的 Content
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string										true	"应用 ID"
//	@Param			id		path		string										true	"默认配置文件或环境实例文件 ID"
//	@Param			body	body		slz.UpdateAppConfigFileContentInput	true	"修改应用配置文件 Content 请求"
//	@Success		200		{object}	slz.UpdateAppConfigFileContentOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files/{id}/content [put]
func (h *Handler) UpdateAppConfigFileContent(c *gin.Context) {
	var uriInput slz.AppConfigFileURIInput
	var input slz.UpdateAppConfigFileContentInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	output, err := h.updateContentOrOverlay(
		c.Request.Context(),
		uriInput.AppID,
		uriInput.ID,
		input.Content,
		appcfg.AppConfigFileTypeNormal,
		input.Description,
		input.CurrentVersion,
		input.EnvName,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ginutils.OK(c, *output)
}

// UpdateAppConfigFileOverlayContent 修改一个应用配置文件的 overlayContent。
//
//	@ID				UpdateAppConfigFileOverlayContent
//	@Summary		修改一个应用配置文件的 overlayContent
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string												true	"应用 ID"
//	@Param			id		path		string												true	"overlay 类型配置文件 ID"
//	@Param			body	body		slz.UpdateAppConfigFileOverlayContentInput	true	"overlayContent 请求"
//	@Success		200		{object}	slz.UpdateAppConfigFileContentOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files/{id}/overlay-content [put]
func (h *Handler) UpdateAppConfigFileOverlayContent(c *gin.Context) {
	var uriInput slz.AppConfigFileURIInput
	var input slz.UpdateAppConfigFileOverlayContentInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	output, err := h.updateContentOrOverlay(
		c.Request.Context(),
		uriInput.AppID,
		uriInput.ID,
		input.OverlayContent,
		appcfg.AppConfigFileTypeOverlay,
		input.Description,
		input.CurrentVersion,
		"",
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ginutils.OK(c, *output)
}

// PreviewOverlayMerge 预览覆盖内容与基础配置文件合并的结果，不会保存任何变更。
//
//	@ID				PreviewOverlayMerge
//	@Summary		预览覆盖内容与基础配置文件合并的结果
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string							true	"应用 ID"
//	@Param			id		path		string							true	"基础应用配置文件 ID"
//	@Param			body	body		slz.PreviewOverlayMergeInput	true	"预览 overlay 合并请求"
//	@Success		200		{object}	slz.PreviewOverlayMergeOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-files/{id}/preview-overlay-merge [post]
func (h *Handler) PreviewOverlayMerge(c *gin.Context) {
	var uriInput slz.AppConfigFileURIInput
	var input slz.PreviewOverlayMergeInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	baseID, err := uriInput.AppConfigFileObjectID()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parse app config file ID"))
		return
	}
	baseFile, err := h.registry.AppConfigFileStore.GetByID(ctx, baseID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get base app config file"))
		return
	}
	if baseFile.GetConfigKind() == appcfg.ConfigKindPlain {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			pkgerrors.Wrap(appcfg.ErrInvalidConfigSpec, "plain config does not support overlay content"),
			bkerrs.ErrCodeInvalidArgument,
			"preview overlay merge",
		))
		return
	}

	virtualConfig := &appcfg.AppConfigFile{
		AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
			Type:                appcfg.AppConfigFileTypeOverlay,
			BaseAppConfigFileID: &baseID,
			ConfigKind:          baseFile.GetConfigKind(),
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				OverlayContent:    &input.OverlayContent,
				Format:            baseFile.GetConfigFormat(),
			},
		},
	}
	if validateErr := validateFrameworkFileContent(input.OverlayContent); validateErr != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(validateErr, bkerrs.ErrCodeInvalidArgument, "validate overlayContent"))
		return
	}

	editor, err := appcfg.NewAppConfigFileEditor(h.registry.AppConfigFileStore, virtualConfig)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create editor"))
		return
	}
	compiledContent, err := editor.GetCompiledContent(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "compile content"))
		return
	}

	ginutils.OK(c, slz.PreviewOverlayMergeOutput{Data: compiledContent})
}

// updateContentOrOverlay validates, compiles, and persists file content changes.
//
// - appID/id: identify the target app config file to load and permission-check.
// - fileType: selects whether the incoming payload should update `content` or `overlayContent`.
// - description/expectedCurrentVersion: participate in version creation when persisting changes.
func (h *Handler) updateContentOrOverlay(
	ctx context.Context, appID, id, content string, fileType appcfg.AppConfigFileType,
	description string, expectedCurrentVersion *int64, envName string,
) (*slz.UpdateAppConfigFileContentOutput, error) {
	if fileType != appcfg.AppConfigFileTypeNormal && fileType != appcfg.AppConfigFileTypeOverlay {
		return nil, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "invalid values file type: %s", fileType)
	}
	app, acf, err := h.validateAndGetAppConfigFile(ctx, appID, id, perm.TypeEdit)
	if err != nil {
		return nil, err
	}
	if acf.GetConfigKind() == appcfg.ConfigKindPlain && fileType == appcfg.AppConfigFileTypeOverlay {
		return nil, bkerrs.Wrap(
			pkgerrors.Wrap(appcfg.ErrInvalidConfigSpec, "plain config does not support overlay content"),
			bkerrs.ErrCodeInvalidArgument, "setting content")
	}
	cfgService := appcfg.NewAppConfigFileService(h.registry.AppConfigFileStore, h.registry.AppConfigFileVersionStore)
	operator := auth.MustGetUser(ctx).ID
	// plain 独立配置模式下，指定 envName 时：无独立实例则 copy-on-write 创建；
	// 已有独立实例则改写该实例，避免把内容写到默认记录上。
	if envName != "" && acf.GetConfigKind() == appcfg.ConfigKindPlain &&
		acf.EnvName == appcfg.EnvNameDefault && acf.HasIndependentEnvConfig() {
		envAcf, created, lazyErr := h.lazyCreatePlainEnvInstance(
			ctx, cfgService, acf, envName, content, operator, description)
		if lazyErr != nil {
			return nil, lazyErr
		}
		if created {
			h.addAppConfigFileAudit(ctx, app, envAcf.EnvName, audit.OperationTypeCreate,
				nil, buildAppConfigFileAuditData(envAcf))
			return &slz.UpdateAppConfigFileContentOutput{CompiledContent: content}, nil
		}
		if envAcf != nil {
			acf = envAcf
		}
	}
	oldAcf := *acf
	if acf.GetConfigKind() == appcfg.ConfigKindFramework {
		if validateErr := validateFrameworkFileContent(content); validateErr != nil {
			return nil, bkerrs.Wrap(validateErr, bkerrs.ErrCodeInvalidArgument, "validate file content")
		}
	}
	editor, err := appcfg.NewAppConfigFileEditor(h.registry.AppConfigFileStore, acf)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "creating values file editor")
	}
	if fileType == appcfg.AppConfigFileTypeNormal {
		err = editor.SetContent(content)
	} else {
		err = editor.SetOverlayContent(content)
	}
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "setting content")
	}
	compiledContent, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "compiling new content")
	}
	validatedRet, vErr := h.validateFrameworkArrangement(ctx, app, acf, compiledContent)
	if vErr != nil {
		return nil, vErr
	}
	if err = cfgService.UpdateFile(ctx, acf, operator, appcfg.UpdateCfgFileOptions{
		OperationType:          appcfg.AppConfigFileVersionOperationTypeUpdate,
		Description:            description,
		ExpectedCurrentVersion: expectedCurrentVersion,
	}); err != nil {
		if errors.Is(err, appcfg.ErrAppConfigFileVersionConflict) {
			return nil, bkerrs.WrapAppConfigFileVersionConflict(err, app.ID, acf.ID.Hex())
		}
		if errors.Is(err, appcfg.ErrInvalidConfigSpec) ||
			errors.Is(err, appcfg.ErrPlainConfigMountPathConflict) {
			return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "create app config file version")
		}
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create app config file version")
	}
	h.addAppConfigFileAudit(ctx, app, acf.EnvName, audit.OperationTypeUpdate,
		buildAppConfigFileAuditData(&oldAcf), buildAppConfigFileAuditData(acf))
	return &slz.UpdateAppConfigFileContentOutput{CompiledContent: compiledContent, ArrgData: validatedRet}, nil
}

// lazyCreatePlainEnvInstance 在 plain 独立配置模式下解析目标环境的写入对象。
// 已有独立实例时返回该实例且 created=false，调用方应继续走更新流程；
// 尚无实例时以请求内容创建新记录并返回 created=true。
func (h *Handler) lazyCreatePlainEnvInstance(
	ctx context.Context,
	cfgService *appcfg.AppConfigFileService,
	defaultFile *appcfg.AppConfigFile,
	envName string,
	content string,
	operator string,
	description string,
) (*appcfg.AppConfigFile, bool, error) {
	if !defaultFile.IsMountedToEnv(envName) {
		return nil, false, bkerrs.Wrap(
			pkgerrors.Wrap(appcfg.ErrInvalidConfigSpec, "plain env instance envName is not in mountedEnvNames"),
			bkerrs.ErrCodeInvalidArgument,
			"lazy create plain env instance",
		)
	}
	existing, err := cfgService.FindPlainEnvInstance(ctx, *defaultFile, envName)
	if err != nil {
		return nil, false, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "find plain env instance")
	}
	if existing != nil {
		return existing, false, nil
	}
	envAcf, err := cfgService.CreatePlainEnvInstance(
		ctx, *defaultFile, envName, &content, operator, description,
	)
	if err != nil {
		if errors.Is(err, appcfg.ErrInvalidConfigSpec) {
			return nil, false, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "lazy create plain env instance")
		}
		return nil, false, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "lazy create plain env instance")
	}
	return envAcf, true, nil
}
