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

// Package handler contains Gin handlers for platform workspace cfg switch APIs.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/cfgswitch"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/cfgswitch/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ cfgswitch.Handler = (*Handler)(nil)

// Handler handles Gin platform workspace cfg switch API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListPlatCfgSwitches 查询空间的配置开关列表
//
//	@ID			ListPlatCfgSwitches
//	@Summary	查询空间的配置开关列表（配置管理 BSCP、七彩石），从未设置过的开关为关闭状态
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.PlatCfgSwitchListResponse
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces/{workspaceID}/cfg-switches [get]
func (h *Handler) ListPlatCfgSwitches(c *gin.Context) {
	var path serializer.WorkspacePath
	if err := ginutils.BindURI(c, &path); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	switches, err := h.manager().List(c.Request.Context(), path.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, toAPIError(err, "list platform workspace cfg switches"))
		return
	}

	ginutils.OK(c, serializer.PlatCfgSwitchListResponse{
		Data: lo.Map(switches, func(s model.WorkspaceCfgSwitch, _ int) *serializer.PlatCfgSwitchOutput {
			return serializer.NewPlatCfgSwitchOutput(&s)
		}),
	})
}

// UpdatePlatCfgSwitch 设置空间的一项配置开关
//
//	@ID				UpdatePlatCfgSwitch
//	@Summary		设置空间的一项配置开关
//	@Description	关闭配置管理（BSCP）后，该空间下应用的配置管理接口全部不可用，且之后的部署不再注入 BSCP 容器
//	@Tags			platmgt
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string								true	"工作空间 ID"
//	@Param			key			path		string								true	"开关标识：bscp / rainbowTConf"
//	@Param			body		body		serializer.UpdatePlatCfgSwitchInput	true	"开关取值"
//	@Success		200			{object}	serializer.PlatCfgSwitchResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/plat-mgt/workspaces/{workspaceID}/cfg-switches/{key} [put]
func (h *Handler) UpdatePlatCfgSwitch(c *gin.Context) {
	var path serializer.CfgSwitchPath
	var input serializer.UpdatePlatCfgSwitchInput
	if err := ginutils.BindURIJSON(c, &path, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	s, err := h.manager().Update(
		c.Request.Context(), path.WorkspaceID, model.SwitchKey(path.Key), lo.FromPtr(input.Enabled),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, toAPIError(err, "update platform workspace cfg switch"))
		return
	}

	ginutils.OK(c, serializer.PlatCfgSwitchResponse{Data: serializer.NewPlatCfgSwitchOutput(s)})
}

func (h *Handler) manager() *cfgswitch.Manager {
	return cfgswitch.NewManager(h.registry.DB)
}

// toAPIError 把 Manager 的错误映射为接口错误
func toAPIError(err error, internalMessage string) error {
	switch {
	case errors.Is(err, bkmsworkspace.ErrWorkspaceNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "platform workspace not found")
	case errors.Is(err, model.ErrUnknownSwitch):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "unknown cfg switch key")
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, internalMessage)
	}
}
