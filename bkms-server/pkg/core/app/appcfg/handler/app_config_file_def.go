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
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// AppCfgFileDefHandler 基于新分层服务架构的 Gin API 处理器，委托给 AppCfgFileDefService（场景层）。
type AppCfgFileDefHandler struct {
	registry *storereg.Registry
}

// NewAppCfgFileDefHandler 创建基于分层服务的 handler。
func NewAppCfgFileDefHandler(registry *storereg.Registry) *AppCfgFileDefHandler {
	return &AppCfgFileDefHandler{registry: registry}
}

func (h *AppCfgFileDefHandler) newAppCfgFileDefService() *appcfg.AppCfgFileDefService {
	base := appcfg.NewBaseAppCfgFileService(
		h.registry.AppConfigFileDefStore,
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
	)
	return appcfg.NewAppCfgFileDefService(base, nil)
}

// AppConfigFileDefUpdate 修改配置文件的 def 信息（name、isUnifiedConfig、mountDir）。
//
//	@ID				AppConfigFileDefUpdate
//	@Summary		修改应用配置文件的逻辑定义信息
//	@Tags			app-config-files
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string									true	"应用 ID"
//	@Param			id		path		string									true	"应用配置文件 Def ID"
//	@Param			body	body		slz.AppConfigFileDefUpdateInput	true	"更新逻辑定义信息请求"
//	@Success		200		{object}	slz.AppConfigFileDefUpdateOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/app-config-file-defs/{id} [put]
func (h *AppCfgFileDefHandler) AppConfigFileDefUpdate(c *gin.Context) {
	var uriInput slz.AppConfigFileURIInput
	var input slz.AppConfigFileDefUpdateInput
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

	defID, err := bson.ObjectIDFromHex(uriInput.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parsing def ID"))
		return
	}
	def, err := h.registry.AppConfigFileDefStore.GetByID(ctx, defID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "loading def"))
		return
	}
	if def.AppID != app.ID {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeNotFound, "config file def not found"))
		return
	}

	operator := auth.MustGetUser(ctx).ID
	update := input.ToFileDefUpdate(operator)

	defService := h.newAppCfgFileDefService()
	if err = defService.UpdateFileDef(ctx, def, update); err != nil {
		if errors.Is(err, appcfg.ErrAppConfigFileVersionConflict) {
			bkerrs.AbortWithErr(c, bkerrs.WrapAppConfigFileVersionConflict(err, app.ID, defID.Hex()))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update config file def"))
		return
	}

	ginutils.OK(c, slz.AppConfigFileDefUpdateOutput{
		Item: new(slz.AppConfigFileDefOutputObj).FromDef(*def),
	})
}
