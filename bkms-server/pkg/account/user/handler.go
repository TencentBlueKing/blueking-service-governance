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

package user

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/admin"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler handles current-user account requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a UserHandler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) platAdminService() *admin.Service {
	return admin.NewService(h.registry.PlatAdminStore)
}

// RoleInfo is the JSON representation of current user's platform role.
type RoleInfo struct {
	// 当前登录用户名
	Username string `json:"username"`
	// 当前用户的平台角色编码，没有平台角色时返回 null
	PlatRoleCode *admin.RoleCode `json:"platRoleCode"`
}

// GetRoleResponse is the JSON response for querying current user's platform role.
type GetRoleResponse struct {
	Data *RoleInfo `json:"data"`
}

// GetRole 查询当前用户的平台角色
//
//	@ID			GetRole
//	@Summary	查询当前用户的平台角色
//	@Tags		account
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	GetRoleResponse
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Failure	500	{object}	bkerrs.GinErrorOutput
//	@Router		/users/me/role [get]
func (h *Handler) GetRole(c *gin.Context) {
	ctx := c.Request.Context()
	username := auth.MustGetUser(ctx).ID

	roleCode, ok, err := h.platAdminService().GetRole(ctx, username)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "check platform administrator status"),
		)
		return
	}

	var platRoleCode *admin.RoleCode
	if ok {
		platRoleCode = &roleCode
	}

	ginutils.OK(c, GetRoleResponse{
		Data: &RoleInfo{
			Username:     username,
			PlatRoleCode: platRoleCode,
		},
	})
}
