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
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// ListUserGroups 查询工作空间监控空间下的告警组列表。
//
//	@ID			ListUserGroups
//	@Summary	查询告警组列表
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListUserGroupsResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkmonitor/user-groups [get]
func (h *Handler) ListUserGroups(c *gin.Context) {
	var uriInput serializer.UserGroupWorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	results, err := h.service.List(ctx, ws, auth.MustGetUser(ctx).ID)
	if err != nil {
		bkerrs.AbortWithErr(c, h.wrapServiceError(err, "list user groups"))
		return
	}

	ginutils.OK(c, &serializer.ListUserGroupsResp{Data: &serializer.ListUserGroupsOutput{
		Count:   int64(len(results)),
		Results: results,
	}})
}

// GetUserGroup 获取告警组详情。
//
//	@ID			GetUserGroup
//	@Summary	获取告警组详情
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		groupID		path		int		true	"告警组 ID"
//	@Success	200			{object}	serializer.GetUserGroupResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkmonitor/user-groups/{groupID} [get]
func (h *Handler) GetUserGroup(c *gin.Context) {
	var uriInput serializer.UserGroupURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	detail, err := h.service.Get(ctx, ws, uriInput.GroupID, auth.MustGetUser(ctx).ID)
	if err != nil {
		bkerrs.AbortWithErr(c, h.wrapServiceError(err, "get user group"))
		return
	}

	ginutils.OK(c, &serializer.GetUserGroupResp{Data: detail})
}

// CreateUserGroup 创建告警组。
//
//	@ID			CreateUserGroup
//	@Summary	创建告警组
//	@Tags		bkintegrations-bkmonitor
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string					true	"工作空间 ID"
//	@Param		body		body		serializer.SaveUserGroupBody	true	"请求体"
//	@Success	200			{object}	serializer.SaveUserGroupResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkmonitor/user-groups [post]
func (h *Handler) CreateUserGroup(c *gin.Context) {
	var uriInput serializer.UserGroupWorkspaceURIInput
	var body serializer.SaveUserGroupBody
	if err := ginutils.BindURIJSON(c, &uriInput, &body); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	params := serializer.NewSaveParams(body, auth.MustGetUser(ctx).ID)
	detail, err := h.service.Save(ctx, ws, &params)
	if err != nil {
		bkerrs.AbortWithErr(c, h.wrapServiceError(err, "create user group"))
		return
	}

	ginutils.OK(c, &serializer.SaveUserGroupResp{Data: detail})
}

// UpdateUserGroup 更新告警组。
//
//	@ID			UpdateUserGroup
//	@Summary	更新告警组
//	@Tags		bkintegrations-bkmonitor
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string					true	"工作空间 ID"
//	@Param		groupID		path		int						true	"告警组 ID"
//	@Param		body		body		serializer.SaveUserGroupBody	true	"请求体"
//	@Success	200			{object}	serializer.SaveUserGroupResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkmonitor/user-groups/{groupID} [put]
func (h *Handler) UpdateUserGroup(c *gin.Context) {
	var uriInput serializer.UserGroupURIInput
	var body serializer.SaveUserGroupBody
	if err := ginutils.BindURIJSON(c, &uriInput, &body); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	params := serializer.NewSaveParams(body, auth.MustGetUser(ctx).ID)
	params.ID = uriInput.GroupID
	detail, err := h.service.Save(ctx, ws, &params)
	if err != nil {
		bkerrs.AbortWithErr(c, h.wrapServiceError(err, "update user group"))
		return
	}

	ginutils.OK(c, &serializer.SaveUserGroupResp{Data: detail})
}

// DeleteUserGroup 删除告警组。
//
//	@ID			DeleteUserGroup
//	@Summary	删除告警组
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		groupID		path		int		true	"告警组 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkmonitor/user-groups/{groupID} [delete]
func (h *Handler) DeleteUserGroup(c *gin.Context) {
	var uriInput serializer.UserGroupURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.service.Delete(ctx, ws, uriInput.GroupID, auth.MustGetUser(ctx).ID); err != nil {
		bkerrs.AbortWithErr(c, h.wrapServiceError(err, "delete user group"))
		return
	}

	ginutils.OK(c, serializer.EmptyOutput{})
}
