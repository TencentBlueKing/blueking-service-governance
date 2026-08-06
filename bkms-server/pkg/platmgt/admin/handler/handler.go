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

// Package handler contains Gin handlers for platform administrator APIs.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	admin "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/admin"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/admin/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

var _ admin.Handler = (*Handler)(nil)

// Handler handles Gin platform administrator API requests.
type Handler struct {
	store admin.Store
}

// New creates a Handler.
func New(store admin.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) platAdminService() *admin.Service {
	return admin.NewService(h.store)
}

// ListRoles 查询可分配的平台管理员角色列表
//
//	@ID			ListRoles
//	@Summary	查询可分配的平台管理员角色列表
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.ListRolesResponse
//	@Failure	403	{object}	bkerrs.GinErrorOutput
//	@Failure	500	{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/admins/roles [get]
func (h *Handler) ListRoles(c *gin.Context) {
	roles := admin.Roles()
	data := make([]*serializer.RoleOutput, 0, len(roles))
	for _, roleInfo := range roles {
		data = append(data, serializer.NewRoleOutput(roleInfo))
	}
	ginutils.OK(c, serializer.ListRolesResponse{Data: data})
}

// ListRoleBindings 查询平台管理员角色绑定列表
//
//	@ID			ListRoleBindings
//	@Summary	查询平台管理员角色绑定列表
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		keyword	query		string	false	"用户名关键字"
//	@Success	200				{object}	serializer.ListRoleBindingsResponse
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Failure	403				{object}	bkerrs.GinErrorOutput
//	@Failure	500				{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/admins [get]
func (h *Handler) ListRoleBindings(c *gin.Context) {
	var query serializer.ListRoleBindingsQuery
	if err := ginutils.BindQuery(c, &query); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	roleBindings, err := h.platAdminService().List(c.Request.Context(), query.Keyword)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list platform administrator role bindings"),
		)
		return
	}

	data := make([]*serializer.RoleBindingOutput, 0, len(roleBindings))
	for _, roleBinding := range roleBindings {
		data = append(data, serializer.NewRoleBindingOutput(roleBinding))
	}
	ginutils.OK(c, serializer.ListRoleBindingsResponse{Data: data})
}

// AssignRoles 批量授予平台管理员角色；已存在的平台管理员绑定会被自动跳过且不报错。
//
//	@ID			AssignRoles
//	@Summary	批量授予平台管理员角色（已存在则跳过）
//	@Tags		platmgt
//	@Accept		json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		input	body		serializer.AssignRolesInput	true	"平台管理员角色批量授权参数"
//	@Success	204
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	403			{object}	bkerrs.GinErrorOutput
//	@Failure	500			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/admins [post]
func (h *Handler) AssignRoles(c *gin.Context) {
	var input serializer.AssignRolesInput
	if err := ginutils.BindJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	err := h.platAdminService().AssignRoles(
		c.Request.Context(),
		input.Usernames,
		input.RoleCode,
		auth.MustGetUser(c.Request.Context()).ID,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, toAPIError(err, "assign platform administrator roles"))
		return
	}
	ginutils.NoContent(c)
}

// RevokeRole 撤销平台管理员角色
//
//	@ID			RevokeRole
//	@Summary	撤销平台管理员角色
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		username	path		string	true	"平台管理员用户名"
//	@Success	204			{object}	nil
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	403			{object}	bkerrs.GinErrorOutput
//	@Failure	500			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/admins/{username} [delete]
func (h *Handler) RevokeRole(c *gin.Context) {
	var path serializer.RoleBindingPath
	if err := ginutils.BindURI(c, &path); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err := h.platAdminService().RevokeRole(c.Request.Context(), path.Username); err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "revoke platform administrator role"),
		)
		return
	}

	ginutils.NoContent(c)
}

func toAPIError(err error, internalMessage string) error {
	switch {
	case errors.Is(err, admin.ErrPermissionDenied):
		return bkerrs.New(bkerrs.ErrCodeNoPermission, "platform administrator permission required")
	case errors.Is(err, admin.ErrInvalidRoleCode):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid platform role code")
	case errors.Is(err, admin.ErrRoleBindingAlreadyExists):
		return bkerrs.Wrap(err, bkerrs.ErrCodeAlreadyExists, "platform administrator role binding already exists")
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, internalMessage)
	}
}
