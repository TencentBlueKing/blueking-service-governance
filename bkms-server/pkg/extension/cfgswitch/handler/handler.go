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

// Package handler 包含配置开关 API 的 Handler 实现。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ cfgswitch.Handler = (*Handler)(nil)

// Handler 处理配置开关 API 请求。
type Handler struct {
	registry *storereg.Registry
}

// New 创建 Handler。
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// GetCfgSwitches 获取空间的配置开关。
//
//	@ID				GetCfgSwitches
//	@Summary		获取空间的配置开关
//	@Description	开关由平台管理员在平台管理中设置，前端据此决定是否展示配置管理（BSCP）、七彩石等 tab；未设置时为 false
//	@Tags			cfgswitch
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	slz.CfgSwitchesResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/cfg-switches [get]
func (h *Handler) GetCfgSwitches(c *gin.Context) {
	var uri slz.WorkspaceURI
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	switches, err := model.ListOfWorkspace(ctx, h.registry.DB, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list cfg switches"))
		return
	}

	ginutils.OK(c, slz.CfgSwitchesResponse{Data: slz.NewCfgSwitchesOutput(switches)})
}
