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

package appcfgfiledef

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement"
)

// Handler 新版 def 视角的 API 处理器，同时支持 framework 和 plain 配置文件。
type Handler struct {
	registry *storereg.Registry
}

// NewHandler 创建 handler。
func NewHandler(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) newAppCfgFileDefService() *appcfg.AppCfgFileDefService {
	base := appcfg.NewBaseAppCfgFileService(
		h.registry.AppConfigFileDefStore,
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	return appcfg.NewAppCfgFileDefService(base)
}

// ListDefaultFilesWithDef 列出应用下所有配置文件定义及其默认文件信息，文件数量不多不分页。
//
//	@ID				ListDefaultFilesWithDef
//	@Summary		列出应用下所有配置文件定义及其默认文件信息
//	@Tags			app-config-file-defs
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Success		200		{object}	ListDefsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs/defaults [get]
func (h *Handler) ListDefaultFilesWithDef(c *gin.Context) {
	var uriInput AppURIInput
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

	defs, err := h.registry.AppConfigFileDefStore.ListByApp(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "listing defs"))
		return
	}

	// 批量查默认文件，按 defID 索引
	defIDs := make([]bson.ObjectID, 0, len(defs))
	for _, d := range defs {
		defIDs = append(defIDs, d.ID)
	}
	defaultFiles, err := h.registry.AppConfigFileStore.List(
		ctx, app.ID,
		appcfg.AcfFilterDefIDs(defIDs),
		appcfg.AcfFilterEnvName(appcfg.EnvNameDefault),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "listing default files"))
		return
	}
	fileByDefID := make(map[string]*appcfg.AppConfigFile, len(defaultFiles))
	for i := range defaultFiles {
		fileByDefID[defaultFiles[i].DefID.Hex()] = &defaultFiles[i]
	}

	items := make([]*DefDetailObj, 0, len(defs))
	for _, def := range defs {
		items = append(items, new(DefDetailObj).FromDefAndFile(def, fileByDefID[def.ID.Hex()]))
	}
	ginutils.OK(c, ListDefsOutput{Items: items})
}

// CreateAppConfigFileDef 创建配置文件定义 + 默认文件。
//
//	@ID				CreateAppConfigFileDef
//	@Summary		创建配置文件定义及默认文件
//	@Tags			app-config-file-defs
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string			true	"应用 ID"
//	@Param			body	body		CreateDefInput	true	"创建配置文件定义请求"
//	@Success		200		{object}	CreateDefOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs [post]
func (h *Handler) CreateAppConfigFileDef(c *gin.Context) {
	var uriInput AppURIInput
	var input CreateDefInput
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

	sourceType := appcfg.ContentSourceType(input.ContentSourceType)
	fileType := appcfg.AppConfigFileType(input.FileType)
	fileFormat := appcfg.FileFormat(input.FileFormat)

	baseFileID, err := h.validateBaseAppConfigFileID(ctx, app.ID, fileType, input.BaseAppConfigFileID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "validating base file"))
		return
	}
	bscpConfig, err := h.validateBSCPConfig(ctx, sourceType, input.BSCPConfig, fileFormat)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "validating bscp config"))
		return
	}

	creator := auth.MustGetUser(ctx).ID
	svc := h.newAppCfgFileDefService()

	result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
		AppID:               app.ID,
		EnvName:             appcfg.EnvNameDefault,
		Name:                input.Name,
		Type:                fileType,
		ContentSourceType:   sourceType,
		BaseAppConfigFileID: baseFileID,
		BSCPConfig:          bscpConfig,
		Format:              fileFormat,
		Content:             input.Content,
		MountDir:            input.MountDir,
		Creator:             creator,
		Description:         input.Description,
		ConfigKind:          appcfg.ConfigKind(input.ConfigKind),
	})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "creating def"))
		return
	}

	out := new(DefDetailObj).FromDefAndFile(
		*result.Def, &result.AppConfigFile,
	)
	ginutils.OK(c, CreateDefOutput{Item: out})
}

// GetAppCfgFileDetail 获取配置文件定义详情（含指定环境下生效的文件内容）。
//
//	@ID				GetAppConfigFileDefDetail
//	@Summary		获取配置文件定义详情
//	@Tags			app-config-file-defs
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			id		path		string	true	"配置文件定义 ID"
//	@Param			envName	query		string	false	"环境名称，为空返回默认文件"
//	@Success		200		{object}	GetDefDetailOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs/{id} [get]
func (h *Handler) GetAppCfgFileDetail(c *gin.Context) {
	var uriInput DefURIInput
	var query EnvNameQuery
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid query"))
		return
	}

	def, err := h.getAppCfgFileDef(c, uriInput)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	svc := h.newAppCfgFileDefService()
	result, err := svc.GetEnvFileDetail(ctx, def, query.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "loading env file detail"))
		return
	}

	ginutils.OK(c, GetDefDetailOutput{
		Item: new(DefDetailObj).FromEnvFileDetail(result),
	})
}

// UpdateAppCfgFileDef 更新配置文件定义信息（name / mountDir / isUnifiedConfig / mountedEnvNames）。
//
//	@ID				UpdateAppConfigFileDef
//	@Summary		更新配置文件定义信息
//	@Tags			app-config-file-defs
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string			true	"应用 ID"
//	@Param			id		path		string			true	"配置文件定义 ID"
//	@Param			body	body		UpdateDefInput	true	"更新配置文件定义请求"
//	@Success		200		{object}	UpdateDefOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs/{id} [put]
func (h *Handler) UpdateAppCfgFileDef(c *gin.Context) {
	var uriInput DefURIInput
	var input UpdateDefInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	def, err := h.getAppCfgFileDef(c, uriInput)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	// 需要编辑权限
	if _, err = perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	operator := auth.MustGetUser(ctx).ID
	update := input.ToFileDefUpdate(operator)

	svc := h.newAppCfgFileDefService()
	if err = svc.UpdateAppCfgFileDef(ctx, def, update); err != nil {
		if errors.Is(err, appcfg.ErrAppConfigFileVersionConflict) {
			bkerrs.AbortWithErr(c, bkerrs.WrapAppConfigFileVersionConflict(err, def.AppID, def.ID.Hex()))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "updating def"))
		return
	}

	ginutils.OK(c, UpdateDefOutput{
		Item: new(DefSummaryObj).FromDef(*def),
	})
}

// DeleteAppCfgFileDef 删除配置文件定义及其所有关联文件和版本记录。
//
//	@ID				DeleteAppConfigFileDef
//	@Summary		删除配置文件定义及关联数据
//	@Tags			app-config-file-defs
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			id		path		string	true	"配置文件定义 ID"
//	@Success		200		{object}	DeleteDefOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs/{id} [delete]
func (h *Handler) DeleteAppCfgFileDef(c *gin.Context) {
	var uriInput DefURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	def, err := h.getAppCfgFileDef(c, uriInput)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err = perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 查默认文件并通过 base service 删除（级联删 sibling / def）
	svc := h.newAppCfgFileDefService()
	defaultFile, fErr := h.registry.AppConfigFileStore.GetByDefIDAndEnv(ctx, def.ID, appcfg.EnvNameDefault)
	if fErr != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(fErr, bkerrs.ErrCodeInternalServerError, "loading default file for deletion"),
		)
		return
	}
	if _, err = svc.DeleteFile(ctx, def.AppID, defaultFile.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "deleting def"))
		return
	}

	ginutils.OK(c, DeleteDefOutput{})
}

// ListEnvInstances 列出配置文件定义下的所有环境实例。
//
//	@ID				ListAppConfigFileDefEnvInstances
//	@Summary		列出配置文件定义的环境实例
//	@Tags			app-config-file-defs
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			id		path		string	true	"配置文件定义 ID"
//	@Success		200		{object}	ListEnvInstancesOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs/{id}/env-instances [get]
func (h *Handler) ListEnvInstances(c *gin.Context) {
	var uriInput DefURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	def, err := h.getAppCfgFileDef(c, uriInput)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	files, err := h.registry.AppConfigFileStore.ListByDefID(ctx, def.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "listing env instances"))
		return
	}

	items := make([]*EnvInstanceObj, 0, len(files))
	for _, f := range files {
		items = append(items, new(EnvInstanceObj).FromModel(f))
	}
	ginutils.OK(c, ListEnvInstancesOutput{Items: items})
}

// UpdateContent 更新配置内容（lazy-create：独立模式下首次编辑自动创建环境实例）。
//
//	@ID				UpdateAppConfigFileDefContent
//	@Summary		更新配置文件内容
//	@Tags			app-config-file-defs
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string				true	"应用 ID"
//	@Param			id		path		string				true	"配置文件定义 ID"
//	@Param			envName	query		string				false	"环境名称，为空更新默认文件"
//	@Param			body	body		UpdateContentInput	true	"更新配置内容请求"
//	@Success		200		{object}	UpdateContentOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs/{id}/content [put]
func (h *Handler) UpdateContent(c *gin.Context) {
	var uriInput DefURIInput
	var query EnvNameQuery
	var input UpdateContentInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid query"))
		return
	}

	def, err := h.getAppCfgFileDef(c, uriInput)
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

	operator := auth.MustGetUser(ctx).ID
	svc := h.newAppCfgFileDefService()
	result, err := svc.UpsertEnvContent(ctx, def, appcfg.UpsertEnvContentParams{
		EnvName:                query.EnvName,
		Content:                input.Content,
		Operator:               operator,
		Description:            input.Description,
		ExpectedCurrentVersion: input.CurrentVersion,
		ValidateCompiledContent: func(targetFile *appcfg.AppConfigFile, compiledContent string) error {
			if def.ConfigKind != appcfg.ConfigKindFramework {
				return nil
			}
			arranger := arrangement.NewAppArranger(h.registry.AppStore)
			_, err := arranger.ValidateFileContent(ctx, app, []byte(compiledContent), targetFile.GetConfigFormat())
			if err != nil {
				return errors.Wrap(appcfg.ErrInvalidConfigSpec, err.Error())
			}
			return nil
		},
	})
	if err != nil {
		code := bkerrs.ErrCodeInternalServerError
		if errors.Is(err, appcfg.ErrInvalidConfigSpec) {
			code = bkerrs.ErrCodeInvalidArgument
		}
		if errors.Is(err, appcfg.ErrAppConfigFileVersionConflict) {
			bkerrs.AbortWithErr(c, bkerrs.WrapAppConfigFileVersionConflict(err, def.AppID, result.File.ID.Hex()))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, code, "upserting content"))
		return
	}

	ginutils.OK(c, UpdateContentOutput{
		FileID:         result.File.ID.Hex(),
		CurrentVersion: result.File.CurrentVersion,
		Content:        result.CompiledContent,
	})
}

// ResetEnvToDefault 恢复指定环境配置为默认值（删除该环境的独立实例）。
//
//	@ID				ResetAppConfigFileDefEnvToDefault
//	@Summary		恢复环境配置为默认值
//	@Tags			app-config-file-defs
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			id		path		string	true	"配置文件定义 ID"
//	@Param			envName	path		string	true	"环境名称"
//	@Success		200		{object}	ResetEnvOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs/{id}/envs/{envName} [delete]
func (h *Handler) ResetEnvToDefault(c *gin.Context) {
	var uriInput DefEnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if uriInput.EnvName == appcfg.EnvNameDefault {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidArgument, "default env cannot be reset"))
		return
	}

	def, err := h.getAppCfgFileDef(c, uriInput.DefURIInput)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err = perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	svc := h.newAppCfgFileDefService()
	if err = svc.ResetEnvInstanceToDefault(ctx, def, uriInput.EnvName); err != nil {
		if errors.Is(err, appcfg.ErrResetToDefaultRequiresIndependentConfig) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "reset requires independent mode"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "resetting env to default"))
		return
	}

	ginutils.OK(c, ResetEnvOutput{})
}

// GetMountPreview 获取指定环境下最终挂载到容器的所有配置文件预览。
//
//	@ID				GetMountPreview
//	@Summary		获取配置文件挂载预览
//	@Tags			app-config-file-defs
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			envName	query		string	false	"环境名称"
//	@Success		200		{object}	MountPreviewOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/mount-preview [get]
func (h *Handler) GetMountPreview(c *gin.Context) {
	var uriInput AppURIInput
	var query EnvNameQuery
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid query"))
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	svc := h.newAppCfgFileDefService()
	items, err := svc.GetMountPreview(ctx, app.ID, query.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "getting mount preview"))
		return
	}

	output := make([]*MountPreviewItemObj, 0, len(items))
	for _, item := range items {
		output = append(output, &MountPreviewItemObj{
			DefID:         item.Def.ID.Hex(),
			Name:          item.Def.Name,
			ConfigKind:    string(item.Def.ConfigKind),
			MountDir:      item.Def.MountDir,
			ContentSource: string(item.ContentSource),
			HasEnvFile:    item.HasEnvFile,
		})
	}
	ginutils.OK(c, MountPreviewOutput{Items: output})
}

func (h *Handler) getAppCfgFileDef(ctx *gin.Context, uri DefURIInput) (*appcfg.AppConfigFileDef, error) {
	app, err := perm.ValidateAppByID(ctx.Request.Context(), h.registry, uri.AppID, perm.TypeView)
	if err != nil {
		return nil, err
	}
	defID, err := bson.ObjectIDFromHex(uri.ID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parsing def ID")
	}
	def, err := h.registry.AppConfigFileDefStore.GetByID(ctx.Request.Context(), defID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "config file def not found")
	}
	if def.AppID != app.ID {
		return nil, bkerrs.New(bkerrs.ErrCodeNotFound, "config file def not found")
	}
	return def, nil
}
