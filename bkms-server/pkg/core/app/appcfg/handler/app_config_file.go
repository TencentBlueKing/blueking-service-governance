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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement"
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

	baseID, err := h.validateBaseAppConfigFileID(ctx, app.ID, input.Type, input.BaseAppConfigFileID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "validate baseAppConfigFileID"))
		return
	}

	fileFormat := appcfg.FileFormat(input.FileFormat)
	sourceType := appcfg.ContentSourceType(input.ContentSourceType)
	bscpCfg, err := h.validateBSCPConfig(ctx, sourceType, input.BscpConfig, fileFormat)
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
			BaseAppConfigFileID: baseID,
			BSCPConfig:          bscpCfg,
			Creator:             creator,
			Description:         input.Description,
		},
	)
	if err != nil {
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

	if input.BaseAppConfigFileID != "" {
		baseID, vErr := h.validateBaseAppConfigFileID(ctx, app.ID, string(acf.Type), input.BaseAppConfigFileID)
		if vErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(vErr, bkerrs.ErrCodeInvalidArgument, "validate baseAppConfigFileID"))
			return
		}
		acf.BaseAppConfigFileID = baseID
	}

	if input.BscpConfig != nil {
		bscpCfg, vErr := h.validateBSCPConfig(ctx, acf.ContentSourceType, input.BscpConfig, acf.GetConfigFormat())
		if vErr != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(vErr, bkerrs.ErrCodeInvalidArgument, "validate BSCP config"))
			return
		}
		acf.BSCPConfig = bscpCfg
	}

	acf.Name = input.Name
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
//	@Param			id		path		string										true	"应用配置文件 ID"
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
//	@Param			id		path		string												true	"应用配置文件 ID"
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

	virtualConfig := &appcfg.AppConfigFile{
		AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
			Type:                appcfg.AppConfigFileTypeOverlay,
			ContentSourceType:   appcfg.ContentSourceTypeLocal,
			BaseAppConfigFileID: &baseID,
			OverlayContent:      &input.OverlayContent,
			Format:              baseFile.GetConfigFormat(),
		},
	}
	if validateErr := validateFileContent(input.OverlayContent, baseFile.GetConfigFormat()); validateErr != nil {
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
	ctx context.Context,
	appID string,
	id string,
	content string,
	fileType appcfg.AppConfigFileType,
	description string,
	expectedCurrentVersion *int64,
) (*slz.UpdateAppConfigFileContentOutput, error) {
	if fileType != appcfg.AppConfigFileTypeNormal && fileType != appcfg.AppConfigFileTypeOverlay {
		return nil, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "invalid values file type: %s", fileType)
	}
	app, acf, err := h.validateAndGetAppConfigFile(ctx, appID, id, perm.TypeEdit)
	if err != nil {
		return nil, err
	}
	oldAcf := *acf

	if validateErr := validateFileContent(content, acf.GetConfigFormat()); validateErr != nil {
		return nil, bkerrs.Wrap(validateErr, bkerrs.ErrCodeInvalidArgument, "validate file content")
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

	arranger := arrangement.NewAppArranger(h.registry.AppStore)
	validatedRet, err := arranger.ValidateFileContent(ctx, app, []byte(compiledContent), acf.GetConfigFormat())
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "validating file content")
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
			Description:            description,
			ExpectedCurrentVersion: expectedCurrentVersion,
		},
	); err != nil {
		if errors.Is(err, appcfg.ErrAppConfigFileVersionConflict) {
			return nil, bkerrs.WrapAppConfigFileVersionConflict(err, app.ID, acf.ID.Hex())
		}
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create app config file version")
	}
	h.addAppConfigFileAudit(
		ctx,
		app,
		acf.EnvName,
		audit.OperationTypeUpdate,
		buildAppConfigFileAuditData(&oldAcf),
		buildAppConfigFileAuditData(acf),
	)

	return &slz.UpdateAppConfigFileContentOutput{
		CompiledContent: compiledContent,
		ArrgData: &slz.ValidateArrgValuesYAMLOutputObj{
			WorkloadImage: &slz.ArrgResultItemOutputObj{
				Status:        string(validatedRet.WorkloadImage.Status),
				SkippedReason: validatedRet.WorkloadImage.SkippedReason,
			},
			IngressDomain: &slz.ArrgResultItemOutputObj{
				Status:        string(validatedRet.IngressDomain.Status),
				SkippedReason: validatedRet.IngressDomain.SkippedReason,
			},
		},
	}, nil
}
