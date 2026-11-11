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

// Package handler contains Gin handlers for rainbow tconf APIs.
package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/rainbowtconf"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/rainbowtconf/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ rainbowtconf.Handler = (*Handler)(nil)

// Handler handles Gin rainbow tconf API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// GetRainbowTConf 获取应用的七彩石入口。
//
//	@ID				GetRainbowTConf
//	@Summary		获取应用的七彩石入口
//	@Description	未配置时 url 为空；应用所在空间未开启七彩石配置开关时不可用
//	@Tags			rainbowtconf
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string	true	"应用 ID"
//	@Success		200		{object}	serializer.RainbowTConfResponse
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/rainbow-tconf [get]
func (h *Handler) GetRainbowTConf(c *gin.Context) {
	var uri serializer.AppURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateApp(ctx, uri.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	entry, err := rainbowtconf.GetEntry(ctx, h.registry.DB, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get rainbow tconf entry"))
		return
	}

	// 使用 PureJSON 避免地址中的 & 被转义为 \u0026
	c.PureJSON(http.StatusOK, serializer.RainbowTConfResponse{Data: serializer.NewRainbowTConfOutput(entry)})
}

// PutRainbowTConf 设置应用的七彩石入口（首次配置或修改）。
//
//	@ID				PutRainbowTConf
//	@Summary		设置应用的七彩石入口
//	@Description	地址仅支持 https；应用所在空间未开启七彩石配置开关时不可用
//	@Tags			rainbowtconf
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID	path		string							true	"应用 ID"
//	@Param			body	body		serializer.PutRainbowTConfInput	true	"七彩石页面地址"
//	@Success		200		{object}	serializer.RainbowTConfResponse
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/rainbow-tconf [put]
func (h *Handler) PutRainbowTConf(c *gin.Context) {
	var uri serializer.AppURIInput
	var input serializer.PutRainbowTConfInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateApp(ctx, uri.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	entry, err := rainbowtconf.SaveURL(ctx, h.registry.DB, app.ID, input.URL, auth.MustGetUser(ctx).ID)
	if err != nil {
		if errors.Is(err, rainbowtconf.ErrInvalidURL) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid rainbow tconf url"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "save rainbow tconf entry"))
		return
	}

	// 使用 PureJSON 避免地址中的 & 被转义为 \u0026
	c.PureJSON(http.StatusOK, serializer.RainbowTConfResponse{Data: serializer.NewRainbowTConfOutput(entry)})
}

// validateApp 校验应用权限，并要求应用所在空间已开启七彩石配置开关。
// 本包所有接口都须经过它，开关关闭时读写接口一律不可用
//
// 涉及的模型：applications、workspace_cfg_switches
func (h *Handler) validateApp(ctx context.Context, appID string, permType perm.Type) (*bkmsapp.Application, error) {
	app, err := perm.ValidateAppByID(ctx, h.registry, appID, permType)
	if err != nil {
		return nil, err
	}
	if err = rainbowtconf.RequireEnabled(ctx, h.registry.DB, app.WorkspaceID); err != nil {
		if errors.Is(err, rainbowtconf.ErrNotEnabled) {
			return nil, bkerrs.Wrapf(
				err, bkerrs.ErrCodeNoPermission,
				"rainbow tconf is not enabled for workspace %s, contact the platform administrator", app.WorkspaceID,
			)
		}
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "check rainbow tconf cfg switch")
	}
	return app, nil
}
