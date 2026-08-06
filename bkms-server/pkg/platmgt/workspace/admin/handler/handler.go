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

// Package handler contains Gin handlers for workspace admin APIs.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	workspaceadmin "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/admin"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/admin/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ workspaceadmin.Handler = (*Handler)(nil)

// Handler handles Gin workspace admin API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// GetWorkspaceRoleStatus 查询指定用户在目标空间是否拥有指定角色。
//
//	@ID			GetWorkspaceRoleStatus
//	@Summary	查询指定用户在目标空间是否拥有指定角色
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		roleCode	query		string	true	"角色 Code"
//	@Param		username	query		string	true	"用户名"
//	@Success	200			{object}	serializer.GetRoleStatusResponse
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces/{workspaceID}/admins [get]
func (h *Handler) GetWorkspaceRoleStatus(c *gin.Context) {
	var path serializer.WorkspacePath
	var query serializer.RoleStatusQuery
	if err := ginutils.BindURIQuery(c, &path, &query); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	status, err := h.workspaceAdminService().
		GetRoleStatus(c.Request.Context(), path.WorkspaceID, query.RoleCode, query.Username)
	if err != nil {
		bkerrs.AbortWithErr(c, toAPIError(err, "get platform workspace role status"))
		return
	}

	ginutils.OK(c, serializer.GetRoleStatusResponse{
		Data: serializer.NewRoleStatusOutput(status.HasRole),
	})
}

// GrantWorkspaceAdmin 为当前用户授予目标空间管理员身份。
//
//	@ID			GrantWorkspaceAdmin
//	@Summary	为当前用户授予目标空间管理员身份
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string						true	"工作空间 ID"
//	@Param		body		body		serializer.GrantAdminInput	true	"管理员授权参数"
//	@Success	204			{object}	nil
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces/{workspaceID}/admins [post]
func (h *Handler) GrantWorkspaceAdmin(c *gin.Context) {
	var path serializer.WorkspacePath
	var input serializer.GrantAdminInput
	if err := ginutils.BindURIJSON(c, &path, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	_, err := h.workspaceAdminService().
		GrantAdmin(c.Request.Context(), path.WorkspaceID, auth.MustGetUser(c.Request.Context()).ID, input.Temporary())
	if err != nil {
		bkerrs.AbortWithErr(c, toAPIError(err, "grant platform workspace admin"))
		return
	}

	ginutils.NoContent(c)
}

// RevokeWorkspaceAdmin 退出目标空间管理员身份。
//
//	@ID			RevokeWorkspaceAdmin
//	@Summary	退出目标空间管理员身份
//	@Tags		platmgt
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	204			{object}	nil
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	403			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Failure	500			{object}	bkerrs.GinErrorOutput
//	@Router		/plat-mgt/workspaces/{workspaceID}/admins [delete]
func (h *Handler) RevokeWorkspaceAdmin(c *gin.Context) {
	var path serializer.WorkspacePath
	if err := ginutils.BindURI(c, &path); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	_, err := h.workspaceAdminService().
		RevokeAdmin(c.Request.Context(), path.WorkspaceID, auth.MustGetUser(c.Request.Context()).ID)
	if err != nil {
		bkerrs.AbortWithErr(c, toAPIError(err, "revoke platform workspace admin"))
		return
	}

	ginutils.NoContent(c)
}

func (h *Handler) workspaceAdminService() *workspaceadmin.Service {
	return workspaceadmin.NewService(
		h.registry.WorkspaceStore,
		h.registry.TempAdminRecordStore,
		perm.NewManager(),
	)
}

func toAPIError(err error, internalMessage string) error {
	switch {
	case errors.Is(err, bkmsworkspace.ErrWorkspaceNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "platform workspace not found")
	case errors.Is(err, workspaceadmin.ErrWorkspaceAdminAlreadyExists):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "workspace admin already exists")
	case errors.Is(err, workspaceadmin.ErrWorkspaceAdminNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "workspace admin not found")
	case errors.Is(err, workspaceadmin.ErrTempAdminAlreadyExists):
		return bkerrs.Wrap(err, bkerrs.ErrCodeAlreadyExists, "temporary admin already exists")
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, internalMessage)
	}
}
