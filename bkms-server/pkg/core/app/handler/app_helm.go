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
	"context"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// UpdateHelmSpec 更新应用 Helm Chart 配置。
//
//	@ID			UpdateHelmSpec
//	@Summary	更新应用 Helm Chart 配置
//	@Tags		app
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path	string							true	"应用 ID"
//	@Param		body	body	serializer.UpdateHelmSpecInput	true	"更新 Helm 配置请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/helm-spec [put]
func (h *Handler) UpdateHelmSpec(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.UpdateHelmSpecInput
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

	helmSource := new(bkmsapp.HelmSource)
	if err = mapstructure.Decode(input.HelmSpec.HelmSource, helmSource); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "decode helm source"))
		return
	}

	// 设置 HelmRepoConfig 的 username 和 password 字段
	// 请求中的 Username 和 Password 字段不是必填的，在请求不提供时，应当使用旧值
	// SetUserPass 内部会自动校验 Username 和 Password 的合法性
	var existingHelmRepo *bkmsapp.HelmRepoConfig
	if app.HelmSpec != nil && app.HelmSpec.HelmSource != nil {
		existingHelmRepo = app.HelmSpec.HelmSource.HelmRepoConfig
	}
	if helmSource.HelmRepoConfig != nil && input.HelmSpec.HelmSource.HelmRepoConfig != nil {
		if err = helmSource.HelmRepoConfig.SetUserPass(
			existingHelmRepo,
			input.HelmSpec.HelmSource.HelmRepoConfig.Username,
			input.HelmSpec.HelmSource.HelmRepoConfig.Password,
		); err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "set user pass"))
			return
		}
	}

	if err = h.registry.AppStore.UpdateHelmSource(ctx, app, helmSource); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update helm spec"))
		return
	}

	var helmSourceBefore *bkmsapp.HelmSource
	if app.HelmSpec != nil {
		helmSourceBefore = app.HelmSpec.HelmSource
	}

	// 添加操作审计
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeHelmSpec),
		audit.WithDataBefore(helmSourceBefore),
		audit.WithDataAfter(helmSource),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// createHelmApp 创建 Helm 类型应用, 包括创建 Values 文件
func (h *Handler) createHelmApp(ctx context.Context, app *bkmsapp.Application) error {
	// 创建 Helm 应用默认的 Values 文件
	// 新应用默认拥有一个名为 default 的本地普通 values 文件
	cfgService := appcfg.NewAppConfigFileService(
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	if _, err := cfgService.Create(
		ctx,
		appcfg.CreateCfgFileParams{
			AppID:             app.ID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              appcfg.DefaultAppConfigFileName,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatYAML,
			Creator:           appcfg.CfgSystemUser,
			Description:       appcfg.CfgSystemVersionDescription,
		},
	); err != nil {
		return errors.Wrap(err, "create default values file")
	}

	if err := h.registry.AppStore.CreateApp(ctx, app); err != nil {
		return errors.Wrap(err, "create app")
	}
	return nil
}

// deleteHelmApp 删除 Helm 类型应用
func (h *Handler) deleteHelmApp(ctx context.Context, app *bkmsapp.Application) error {
	// 删除所有关联的 Values 文件
	if _, err := h.registry.AppConfigFileStore.DeleteByApp(ctx, app.ID); err != nil {
		return errors.Wrap(err, "delete values files")
	}
	return nil
}
