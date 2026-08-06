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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// ListAppConfigFileVersions 查询应用配置文件版本列表。
//
//	@ID				ListAppConfigFileVersions
//	@Summary		查询应用配置文件版本列表
//	@Tags			app-config-files
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID				path		string	true	"应用 ID"
//	@Param			appConfigFileID		query		string	false	"应用配置文件 ID"
//	@Param			envName				query		string	false	"环境名"
//	@Param			name				query		string	false	"文件名"
//	@Param			version				query		int		false	"版本号"
//	@Param			creator				query		string	false	"创建人"
//	@Param			description			query		string	false	"版本描述"
//	@Param			page				query		int		true	"页码，从 1 开始"
//	@Param			pageSize			query		int		true	"每页数量，仅支持 5/10/20/50/100"
//	@Success		200					{object}	serializer.ListAppConfigFileVersionsOutput
//	@Failure		400					{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file/versions [get]
func (h *Handler) ListAppConfigFileVersions(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var queryInput serializer.ListAppConfigFileVersionsQueryInput
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

	opts, err := queryInput.ToVersionListOptions(app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parse appConfigFileID"))
		return
	}

	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	versions, total, err := cfgService.ListVersions(ctx, opts)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list app config file versions"))
		return
	}

	items := make([]*serializer.AppConfigFileVersionOutputObj, 0, len(versions))
	for _, version := range versions {
		items = append(items, new(serializer.AppConfigFileVersionOutputObj).FromModel(version))
	}
	ginutils.OK(c, serializer.ListAppConfigFileVersionsOutput{
		Data: &serializer.PaginatedAppConfigFileVersionOutputObjs{
			Count:   total,
			Results: items,
		},
	})
}

// GetAppConfigFileVersion 查询应用配置文件某个版本详情。
//
//	@ID				GetAppConfigFileVersion
//	@Summary		查询应用配置文件某个版本详情
//	@Tags			app-config-files
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			id		path		string	true	"版本记录 ID"
//	@Success		200		{object}	serializer.GetAppConfigFileVersionOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file/versions/{id} [get]
func (h *Handler) GetAppConfigFileVersion(c *gin.Context) {
	var uriInput serializer.AppConfigFileVersionURIInput
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

	id, err := uriInput.VersionObjectID()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parse version ID"))
		return
	}

	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	version, err := cfgService.GetVersionByAppAndID(ctx, app.ID, id)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get app config file version"))
		return
	}
	ginutils.OK(c, serializer.GetAppConfigFileVersionOutput{
		Data: new(serializer.AppConfigFileVersionOutputObj).FromModel(*version),
	})
}

// CompareAppConfigFileVersions 对比两个应用配置文件版本。
//
//	@ID				CompareAppConfigFileVersions
//	@Summary		对比两个应用配置文件版本
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string											true	"应用 ID"
//	@Param			body	body		serializer.CompareAppConfigFileVersionsInput	true	"对比应用配置文件版本请求"
//	@Success		200		{object}	serializer.CompareAppConfigFileVersionsOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file/versions/compare [post]
func (h *Handler) CompareAppConfigFileVersions(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.CompareAppConfigFileVersionsInput
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

	previousID, currentID, err := input.VersionIDs()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parse compare version IDs"))
		return
	}

	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	previous, current, err := cfgService.CompareVersions(ctx, app.ID, previousID, currentID)
	if err != nil {
		if errors.Is(err, appcfg.ErrAppCfgFileVersionNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeNotFound, err.Error()))
			return
		}
		if errors.Is(err, appcfg.ErrComparedVersionsBelongToDifferentFiles) {
			bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidArgument, err.Error()))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "compare app config file versions"))
		return
	}
	ginutils.OK(c, serializer.CompareAppConfigFileVersionsOutput{
		Previous: new(serializer.AppConfigFileVersionOutputObj).FromModel(*previous),
		Current:  new(serializer.AppConfigFileVersionOutputObj).FromModel(*current),
	})
}

// RollbackAppConfigFileVersion 回滚到指定应用配置文件版本。
//
//	@ID				RollbackAppConfigFileVersion
//	@Summary		回滚到指定应用配置文件版本
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string											true	"应用 ID"
//	@Param			id		path		string											true	"版本记录 ID"
//	@Param			body	body		serializer.RollbackAppConfigFileVersionInput	true	"回滚应用配置文件版本请求"
//	@Success		200		{object}	serializer.RollbackAppConfigFileVersionOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file/versions/{id}/rollback [post]
func (h *Handler) RollbackAppConfigFileVersion(c *gin.Context) {
	var uriInput serializer.AppConfigFileVersionURIInput
	var input serializer.RollbackAppConfigFileVersionInput
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

	id, err := uriInput.VersionObjectID()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parse version ID"))
		return
	}

	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	targetVersion, err := cfgService.GetVersionByAppAndID(ctx, app.ID, id)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get rollback target version"))
		return
	}
	acf, err := h.registry.AppConfigFileStore.GetByID(ctx, targetVersion.AppConfigFileID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get app config file"))
		return
	}
	oldAcf := *acf

	operator := auth.MustGetUser(ctx).ID
	rollbackDescription := ""
	if input.Description != nil {
		rollbackDescription = *input.Description
	}
	if acf, _, err = cfgService.Rollback(
		ctx,
		app.ID,
		id,
		operator,
		rollbackDescription,
		input.CurrentVersion,
	); err != nil {
		if errors.Is(err, appcfg.ErrAppConfigFileVersionConflict) {
			bkerrs.AbortWithErr(c, bkerrs.WrapAppConfigFileVersionConflict(err, app.ID, id.Hex()))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create rollback version"))
		return
	}

	h.addAppConfigFileAudit(
		ctx,
		app,
		acf.EnvName,
		audit.OperationTypeRollback,
		buildAppConfigFileAuditData(&oldAcf),
		buildAppConfigFileAuditData(acf),
	)
	ginutils.OK(c, serializer.RollbackAppConfigFileVersionOutput{
		Data: new(serializer.AppConfigFileOutputObj).FromModel(*acf),
	})
}

// DeleteAppConfigFileVersion 删除应用配置文件历史版本。
//
//	@ID				DeleteAppConfigFileVersion
//	@Summary		删除应用配置文件历史版本
//	@Tags			app-config-files
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Param			id		path		string	true	"版本记录 ID"
//	@Success		200		{object}	serializer.AppConfigFileEmptyOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file/versions/{id} [delete]
func (h *Handler) DeleteAppConfigFileVersion(c *gin.Context) {
	var uriInput serializer.AppConfigFileVersionURIInput
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

	id, err := uriInput.VersionObjectID()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parse version ID"))
		return
	}

	deleter := auth.MustGetUser(ctx).ID
	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	version, _, err := cfgService.DeleteVersion(ctx, app.ID, id, deleter)
	if err != nil {
		if errors.Is(err, appcfg.ErrAppCfgFileVersionNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get app config file version"))
			return
		}
		if errors.Is(err, appcfg.ErrUsingVersionCannotBeDeleted) {
			bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidArgument, err.Error()))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete app config file version"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithDataBefore(new(serializer.AppConfigFileVersionOutputObj).FromModel(*version)),
	)
	ginutils.OK(c, serializer.AppConfigFileEmptyOutput{})
}
