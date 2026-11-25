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
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/customrole"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// builtinRoleNames 内置角色的展示名称
var builtinRoleNames = map[string]string{
	perm.RoleCodeAdmin:     "管理员",
	perm.RoleCodeDeveloper: "开发者",
	perm.RoleCodeSre:       "SRE",
	perm.RoleCodeOperator:  "运营者",
}

// ListWorkspaceRoles 获取工作空间的角色列表。
//
//	@ID				ListWorkspaceRoles
//	@Summary		获取工作空间的内置角色和自定义角色，包含权限、生效范围与成员数量
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	serializer.ListWorkspaceRolesOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/roles [get]
func (h *Handler) ListWorkspaceRoles(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
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

	permMgr := perm.NewManager()
	roles, err := permMgr.ListRoles(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list roles"))
		return
	}
	names, err := h.loadRoleResourceNames(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	membersByRole, err := listMembersByRole(ctx, permMgr, roles)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	outputs := make([]*serializer.RoleOutputObj, 0, len(roles))
	for _, r := range roles {
		outputs = append(outputs, toRoleOutput(r, len(membersByRole[r.ID]), names))
	}
	ginutils.OK(c, serializer.ListWorkspaceRolesOutput{
		Data:        outputs,
		Permissions: lo.Without(role.AllPermissions, role.Perm.SpaceCreate),
	})
}

// CreateWorkspaceRole 创建自定义角色。
//
//	@ID				CreateWorkspaceRole
//	@Summary		创建空白自定义角色，或克隆已有角色（复制权限与范围，不复制成员），仅空间管理员可操作
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string								true	"工作空间 ID"
//	@Param			body		body		serializer.CreateWorkspaceRoleInput	true	"创建角色请求"
//	@Success		200			{object}	serializer.WorkspaceRoleOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/roles [post]
func (h *Handler) CreateWorkspaceRole(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var input serializer.CreateWorkspaceRoleInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := h.validateWorkspaceAdmin(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	permMgr := perm.NewManager()
	name := strings.TrimSpace(input.Name)
	perms, appScope, envScope := input.Permissions, input.AppScope.ToModel(), input.EnvScope.ToModel()
	if input.CloneFrom != "" {
		source, gErr := getRoleByKey(ctx, permMgr, ws.ID, input.CloneFrom)
		if gErr != nil {
			bkerrs.AbortWithErr(c, gErr)
			return
		}
		perms, appScope, envScope = customrole.Definition(source)
		if name == "" {
			name = roleDisplayName(source) + " 副本"
		}
	}
	if name == "" {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidArgument, "role name is required"))
		return
	}
	if err = checkRoleNameAvailable(ctx, permMgr, ws.ID, name, ""); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	data, err := h.buildCustomRoleData(ctx, ws, perms, appScope, envScope)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	r, err := permMgr.CreateCustomRole(ctx, name, input.Description, data)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create custom role"))
		return
	}
	h.addRoleAudit(ctx, ws.ID, audit.OperationTypeCreate, nil, r)

	ginutils.OK(c, serializer.WorkspaceRoleOutput{Data: toRoleOutput(r, 0, roleResourceNamesFromData(data))})
}

// UpdateWorkspaceRole 修改自定义角色。
//
//	@ID				UpdateWorkspaceRole
//	@Summary		修改自定义角色的名称、权限和生效范围，变更立即影响角色的全部成员，仅空间管理员可操作
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string								true	"工作空间 ID"
//	@Param			roleCode	path		string								true	"自定义角色 ID"
//	@Param			body		body		serializer.UpdateWorkspaceRoleInput	true	"修改角色请求"
//	@Success		200			{object}	serializer.WorkspaceRoleOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/roles/{roleCode} [put]
func (h *Handler) UpdateWorkspaceRole(c *gin.Context) {
	var uriInput serializer.WorkspaceRoleKeyURIInput
	var input serializer.UpdateWorkspaceRoleInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidArgument, "role name is required"))
		return
	}

	ctx := c.Request.Context()
	ws, err := h.validateWorkspaceAdmin(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	permMgr := perm.NewManager()
	before, err := getCustomRole(ctx, permMgr, ws.ID, uriInput.RoleCode)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	beforeCopy := *before
	if err = checkRoleNameAvailable(ctx, permMgr, ws.ID, name, before.ID); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	data, err := h.buildCustomRoleData(
		ctx, ws, input.Permissions, input.AppScope.ToModel(), input.EnvScope.ToModel(),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	r, err := permMgr.UpdateCustomRole(ctx, before.ID, name, input.Description, data)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update custom role"))
		return
	}
	h.addRoleAudit(ctx, ws.ID, audit.OperationTypeUpdate, &beforeCopy, r)

	members, err := permMgr.ListRoleMembers(ctx, r.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list role members"))
		return
	}
	ginutils.OK(
		c,
		serializer.WorkspaceRoleOutput{Data: toRoleOutput(r, len(members), roleResourceNamesFromData(data))},
	)
}

// DeleteWorkspaceRole 删除自定义角色。
//
//	@ID				DeleteWorkspaceRole
//	@Summary		删除自定义角色，成员失去该角色的全部权限，其他角色不受影响，仅空间管理员可操作
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			roleCode	path		string	true	"自定义角色 ID"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/roles/{roleCode} [delete]
func (h *Handler) DeleteWorkspaceRole(c *gin.Context) {
	var uriInput serializer.WorkspaceRoleKeyURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := h.validateWorkspaceAdmin(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	permMgr := perm.NewManager()
	r, err := getCustomRole(ctx, permMgr, ws.ID, uriInput.RoleCode)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err = permMgr.DeleteCustomRole(ctx, r.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete custom role"))
		return
	}
	h.addRoleAudit(ctx, ws.ID, audit.OperationTypeDelete, r, nil)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// ListWorkspaceRoleUsers 获取角色成员。
//
//	@ID				ListWorkspaceRoleUsers
//	@Summary		获取工作空间下某个角色的成员
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			roleCode	path		string	true	"内置角色为角色 Code，自定义角色为角色 ID"
//	@Success		200			{object}	serializer.ListWorkspaceRoleUsersOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/roles/{roleCode}/users [get]
func (h *Handler) ListWorkspaceRoleUsers(c *gin.Context) {
	var uriInput serializer.WorkspaceRoleKeyURIInput
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
	permMgr := perm.NewManager()
	r, err := getRoleByKey(ctx, permMgr, ws.ID, uriInput.RoleCode)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	members, err := permMgr.ListRoleMembers(ctx, r.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list role members"))
		return
	}
	ginutils.OK(c, serializer.ListWorkspaceRoleUsersOutput{Data: lo.Ternary(members == nil, []string{}, members)})
}

// RemoveWorkspaceRoleUser 将用户移出某个角色。
//
//	@ID				RemoveWorkspaceRoleUser
//	@Summary		将用户移出某个角色，只离开该角色，其他角色不受影响，仅空间管理员可操作
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			roleCode	path		string	true	"内置角色为角色 Code，自定义角色为角色 ID"
//	@Param			userID		path		string	true	"用户 ID"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/roles/{roleCode}/users/{userID} [delete]
func (h *Handler) RemoveWorkspaceRoleUser(c *gin.Context) {
	var uriInput serializer.WorkspaceRoleUserURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := h.validateWorkspaceAdmin(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	permMgr := perm.NewManager()
	r, err := getRoleByKey(ctx, permMgr, ws.ID, uriInput.RoleCode)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	members, err := permMgr.ListRoleMembers(ctx, r.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list role members"))
		return
	}
	if !lo.Contains(members, uriInput.UserID) {
		ginutils.OK(c, serializer.EmptyOutput{})
		return
	}
	if r.RoleCode == perm.RoleCodeAdmin && len(members) == 1 {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "cannot remove the last workspace admin"))
		return
	}
	if err = permMgr.DeleteRoleForUsers(ctx, r.ID, []string{uriInput.UserID}); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete role for users"))
		return
	}
	h.syncMonitorMemberRemoval(ctx, permMgr, ws, r.RoleCode, uriInput.UserID, auth.MustGetUser(ctx).ID)

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeUserRole),
		audit.WithDataBefore(map[string]any{"workspaceID": ws.ID, "roleID": r.ID, "userID": uriInput.UserID}),
		audit.WithWorkspaceID(ws.ID),
	)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// AddWorkspaceUser 授予用户工作空间下角色身份。
//
//	@ID				AddWorkspaceUser
//	@Summary		授予用户工作空间下角色身份
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string							true	"工作空间 ID"
//	@Param			roleCode	path		string							true	"内置角色为角色 Code，自定义角色为角色 ID"
//	@Param			body		body		serializer.AddWorkspaceUserInput	true	"添加用户请求"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/roles/{roleCode}/users [post]
func (h *Handler) AddWorkspaceUser(c *gin.Context) {
	var uriInput serializer.WorkspaceRoleKeyURIInput
	var input serializer.AddWorkspaceUserInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := h.validateWorkspaceAdmin(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	permMgr := perm.NewManager()
	r, err := getRoleByKey(ctx, permMgr, ws.ID, uriInput.RoleCode)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = permMgr.AddRoleForUsers(ctx, r.ID, input.UserIDs); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "add role for users"))
		return
	}

	// 对于 admin/sre 角色的新增，交由 UserGroupService 基于 workspace 统一处理
	if r.RoleCode == perm.RoleCodeAdmin || r.RoleCode == perm.RoleCodeSre {
		operator := auth.MustGetUser(ctx).ID
		go bkmonitor.NewUserGroupService(h.registry.DB).SyncMembersForWorkspace(
			context.WithoutCancel(ctx), ws, operator,
		)
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeCreate,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeUserRole),
		audit.WithDataAfter(map[string]any{
			"workspaceID": ws.ID,
			"roleCode":    r.RoleCode,
			"roleID":      r.ID,
			"userIDs":     input.UserIDs,
		}),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// RemoveWorkspaceUser 移除用户工作空间下角色身份。
//
//	@ID				RemoveWorkspaceUser
//	@Summary		移除用户工作空间下角色身份
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			userID		path		string	true	"用户 ID"
//	@Success		200			{object}	serializer.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/users/{userID} [delete]
func (h *Handler) RemoveWorkspaceUser(c *gin.Context) {
	var uriInput serializer.WorkspaceUserURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := h.validateWorkspaceAdmin(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	permMgr := perm.NewManager()
	roles, err := permMgr.ListRoles(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list roles"))
		return
	}
	membersByRole, err := listMembersByRole(ctx, permMgr, roles)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 先校验不能移除最后一名空间管理员，避免先删除其他角色后再报错留下半清理状态
	for _, r := range roles {
		members := membersByRole[r.ID]
		if r.RoleCode == perm.RoleCodeAdmin && len(members) == 1 && members[0] == uriInput.UserID {
			bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "cannot remove the last workspace admin"))
			return
		}
	}

	removedMonitorRole := false
	for _, r := range roles {
		if !lo.Contains(membersByRole[r.ID], uriInput.UserID) {
			continue
		}
		if err = permMgr.DeleteRoleForUsers(ctx, r.ID, []string{uriInput.UserID}); err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete role for users"))
			return
		}
		removedMonitorRole = removedMonitorRole ||
			r.RoleCode == perm.RoleCodeAdmin || r.RoleCode == perm.RoleCodeSre
	}

	// 从各环境的蓝鲸监控告警组中移除该用户（仅处理 type=user）
	if removedMonitorRole {
		operator := auth.MustGetUser(ctx).ID
		go bkmonitor.NewUserGroupService(h.registry.DB).RemoveMemberForWorkspace(
			context.WithoutCancel(ctx), ws, uriInput.UserID, operator,
		)
	}

	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeDelete,
		audit.ResourceTypeWorkspace,
		ws.ID,
		audit.WithAttribute(audit.AttributeUserRole),
		audit.WithDataBefore(map[string]any{
			"workspaceID": ws.ID,
			"userID":      uriInput.UserID,
		}),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// ListWorkspaceUsers 按成员查看空间的角色。
//
//	@ID				ListWorkspaceUsers
//	@Summary		按成员列出空间内的角色、合并后的生效范围与是否可部署，可按角色筛选
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			roleCode	query		string	false	"只返回该角色的成员，内置角色为角色 Code，自定义角色为角色 ID"
//	@Success		200			{object}	serializer.ListWorkspaceUsersOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/users [get]
func (h *Handler) ListWorkspaceUsers(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var queryInput serializer.ListWorkspaceUsersQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	permMgr := perm.NewManager()
	if queryInput.RoleCode != "" {
		if _, err = getRoleByKey(ctx, permMgr, ws.ID, queryInput.RoleCode); err != nil {
			bkerrs.AbortWithErr(c, err)
			return
		}
	}
	rolesByUser, err := listRolesByUser(ctx, permMgr, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	names, err := h.loadRoleResourceNames(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	outputs := make([]*serializer.WorkspaceUserOutputObj, 0, len(rolesByUser))
	for _, username := range sortedKeys(rolesByUser) {
		roles := rolesByUser[username]
		if queryInput.RoleCode != "" && !lo.ContainsBy(roles, func(r *role.Role) bool {
			return r.ID == queryInput.RoleCode || (!r.IsCustom() && r.RoleCode == queryInput.RoleCode)
		}) {
			continue
		}
		merged := customrole.MergeScopes(roles)
		outputs = append(outputs, &serializer.WorkspaceUserOutputObj{
			Username: username,
			Roles:    toUserRoleOutputs(roles),
			AppScope: &serializer.MergedAppScopeOutputObj{
				All:  merged.AllApps,
				Apps: names.appOutputs(merged.AppIDs),
			},
			EnvScope: &serializer.MergedEnvScopeOutputObj{
				All:      merged.AllEnvs,
				EnvTypes: lo.Ternary(merged.EnvTypes == nil, []string{}, merged.EnvTypes),
				Envs:     names.envOutputs(merged.EnvIDs),
			},
			CanDeploy: lo.ContainsBy(roles, func(r *role.Role) bool {
				perms, _, _ := customrole.Definition(r)
				return lo.Contains(perms, role.Perm.EnvDeploy)
			}),
		})
	}
	ginutils.OK(c, serializer.ListWorkspaceUsersOutput{Data: outputs})
}

// ListEnvDeployers 列出各标准环境的可部署人员。
//
//	@ID				ListEnvDeployers
//	@Summary		列出空间内各标准环境能部署的人，以及每人来自哪些角色、能部署哪些应用
//	@Tags			workspace
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	serializer.ListEnvDeployersOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/env-deployers [get]
func (h *Handler) ListEnvDeployers(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
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
	rolesByUser, err := listRolesByUser(ctx, perm.NewManager(), ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	envs, err := store.Of(h.registry.DB, envmodel.EnvMeta).List(ctx, envmodel.StdEnvsIn(ws.ID), envmodel.NewestFirst())
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list envs"))
		return
	}
	names, err := h.loadRoleResourceNames(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	usernames := sortedKeys(rolesByUser)
	outputs := make([]*serializer.EnvDeployersOutputObj, 0, len(envs))
	for i := range envs {
		env := &envs[i]
		deployers := []*serializer.EnvDeployerOutputObj{}
		for _, username := range usernames {
			roles := lo.Filter(rolesByUser[username], func(r *role.Role, _ int) bool {
				return customrole.CanDeploy(r, env)
			})
			if len(roles) == 0 {
				continue
			}
			merged := customrole.MergeScopes(roles)
			deployers = append(deployers, &serializer.EnvDeployerOutputObj{
				Username: username,
				Roles:    toUserRoleOutputs(roles),
				AllApps:  merged.AllApps,
				Apps:     names.appOutputs(merged.AppIDs),
			})
		}
		outputs = append(outputs, &serializer.EnvDeployersOutputObj{
			EnvID: env.ID.Hex(), EnvName: env.Name, Deployers: deployers,
		})
	}
	ginutils.OK(c, serializer.ListEnvDeployersOutput{Data: outputs})
}

// listMembersByRole 并发拉取各角色的成员列表，返回 roleID → 成员用户名列表
func listMembersByRole(
	ctx context.Context, permMgr perm.Manager, roles []*role.Role,
) (map[string][]string, error) {
	membersList := make([][]string, len(roles))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(workspaceOverviewMaxConcurrency)
	for i, r := range roles {
		g.Go(func() error {
			members, err := permMgr.ListRoleMembers(gCtx, r.ID)
			if err != nil {
				return bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list role %s members", r.ID)
			}
			membersList[i] = members
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	membersByRole := make(map[string][]string, len(roles))
	for i, r := range roles {
		membersByRole[r.ID] = membersList[i]
	}
	return membersByRole, nil
}

// listRolesByUser 列出空间内的角色及成员，按用户名聚合
func listRolesByUser(ctx context.Context, permMgr perm.Manager, workspaceID string) (map[string][]*role.Role, error) {
	roles, err := permMgr.ListRoles(ctx, workspaceID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list roles")
	}
	membersByRole, err := listMembersByRole(ctx, permMgr, roles)
	if err != nil {
		return nil, err
	}
	rolesByUser := make(map[string][]*role.Role)
	for _, r := range roles {
		for _, username := range lo.Uniq(membersByRole[r.ID]) {
			rolesByUser[username] = append(rolesByUser[username], r)
		}
	}
	return rolesByUser, nil
}

// sortedKeys 返回按字典序排列的 key
func sortedKeys[V any](m map[string]V) []string {
	keys := lo.Keys(m)
	slices.Sort(keys)
	return keys
}

// toUserRoleOutputs 转换角色列表为成员视角中的角色信息
func toUserRoleOutputs(roles []*role.Role) []*serializer.UserRoleOutputObj {
	return lo.Map(roles, func(r *role.Role, _ int) *serializer.UserRoleOutputObj { return toUserRoleOutput(r) })
}

// toUserRoleOutput 转换角色为成员视角中的角色信息
func toUserRoleOutput(r *role.Role) *serializer.UserRoleOutputObj {
	return &serializer.UserRoleOutputObj{
		ID: r.ID, RoleCode: r.RoleCode, Name: roleDisplayName(r), Builtin: !r.IsCustom(),
	}
}

// validateWorkspaceAdmin 校验当前用户是空间管理员（属于管理员角色）。
//
// 角色与成员管理只允许空间管理员操作；自定义角色即使被授予编辑空间，也不能管理角色和成员，避免借此给自己提权。
func (h *Handler) validateWorkspaceAdmin(ctx context.Context, workspaceID string) (*workspace.Workspace, error) {
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, workspaceID, ginperm.TypeEdit)
	if err != nil {
		return nil, err
	}
	permMgr := perm.NewManager()
	adminRole, err := permMgr.GetRole(ctx, ws.ID, perm.RoleCodeAdmin)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get admin role")
	}
	admins, err := permMgr.ListRoleMembers(ctx, adminRole.ID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list admin role members")
	}
	if !lo.Contains(admins, auth.MustGetUser(ctx).ID) {
		return nil, bkerrs.New(bkerrs.ErrCodeNoPermission, "only workspace admins can manage roles and members")
	}
	return ws, nil
}

// getRoleByKey 按角色 Code（内置角色）或角色 ID（自定义角色）获取工作空间下的角色
func getRoleByKey(ctx context.Context, permMgr perm.Manager, workspaceID, key string) (*role.Role, error) {
	var (
		r   *role.Role
		err error
	)
	if lo.Contains(perm.RoleCodes(), key) {
		r, err = permMgr.GetRole(ctx, workspaceID, key)
	} else {
		r, err = permMgr.GetRoleByID(ctx, workspaceID, key)
	}
	if err != nil {
		if errors.Is(err, role.ErrRoleNotFound) {
			return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeNotFound, "role %s not found", key)
		}
		return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get role %s", key)
	}
	return r, nil
}

// getCustomRole 获取工作空间下的自定义角色，内置角色返回错误
func getCustomRole(ctx context.Context, permMgr perm.Manager, workspaceID, key string) (*role.Role, error) {
	r, err := getRoleByKey(ctx, permMgr, workspaceID, key)
	if err != nil {
		return nil, err
	}
	if !r.IsCustom() {
		return nil, bkerrs.New(
			bkerrs.ErrCodeInvalidRequest, "built-in roles can not be modified, clone it as a custom role instead",
		)
	}
	return r, nil
}

// checkRoleNameAvailable 校验角色名称在空间内未被其他角色使用
func checkRoleNameAvailable(
	ctx context.Context, permMgr perm.Manager, workspaceID, name, excludeRoleID string,
) error {
	roles, err := permMgr.ListRoles(ctx, workspaceID)
	if err != nil {
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list roles")
	}
	for _, r := range roles {
		if r.ID != excludeRoleID && roleDisplayName(r) == name {
			return bkerrs.Errorf(bkerrs.ErrCodeAlreadyExists, "role name %s already exists", name)
		}
	}
	return nil
}

// buildCustomRoleData 校验角色定义并生成授权所需的数据
//
// 读取的模型：app、env。
func (h *Handler) buildCustomRoleData(
	ctx context.Context, ws *workspace.Workspace, perms []string, appScope role.AppScope, envScope role.EnvScope,
) (bkiam.CustomRoleData, error) {
	data, err := customrole.BuildData(ctx, h.registry.DB, ws, perms, appScope, envScope)
	if err != nil {
		if errors.Is(err, role.ErrInvalidCustomRole) {
			return data, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid role definition")
		}
		return data, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "build role data")
	}
	return data, nil
}

// roleResourceNames 空间内应用与环境的展示信息，用于输出角色范围
type roleResourceNames struct {
	apps map[string]string
	envs map[string]*serializer.RoleResourceOutputObj
}

// loadRoleResourceNames 加载空间内应用与标准环境的展示信息
//
// 读取的模型：app、env。
func (h *Handler) loadRoleResourceNames(ctx context.Context, workspaceID string) (*roleResourceNames, error) {
	apps, err := store.Of(h.registry.DB, bkmsapp.AppMeta).List(ctx, bkmsapp.InWorkspace(workspaceID))
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list apps")
	}
	envs, err := store.Of(h.registry.DB, envmodel.EnvMeta).List(ctx, envmodel.StdEnvsIn(workspaceID))
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list envs")
	}
	return &roleResourceNames{
		apps: lo.SliceToMap(apps, func(app bkmsapp.Application) (string, string) {
			return app.ID, lo.CoalesceOrEmpty(app.DisplayName, app.Name)
		}),
		envs: lo.SliceToMap(envs, func(env envmodel.Environment) (string, *serializer.RoleResourceOutputObj) {
			return env.ID.Hex(), &serializer.RoleResourceOutputObj{
				ID: env.ID.Hex(), Name: lo.CoalesceOrEmpty(env.DisplayName, env.Name), Type: env.Type,
			}
		}),
	}, nil
}

// roleResourceNamesFromData 从已校验加载的授权数据直接构造资源展示信息，避免重复查库
func roleResourceNamesFromData(data bkiam.CustomRoleData) *roleResourceNames {
	return &roleResourceNames{
		apps: lo.SliceToMap(data.Apps, func(app bkiam.ResourceRef) (string, string) {
			return app.ID, app.Name
		}),
		envs: lo.SliceToMap(data.Envs, func(env bkiam.EnvResourceRef) (string, *serializer.RoleResourceOutputObj) {
			return env.ID, &serializer.RoleResourceOutputObj{ID: env.ID, Name: env.Name, Type: env.Type}
		}),
	}
}

// appOutputs 返回存在的应用的展示信息，已删除的应用被忽略
func (n *roleResourceNames) appOutputs(appIDs []string) []*serializer.RoleResourceOutputObj {
	return lo.FilterMap(appIDs, func(id string, _ int) (*serializer.RoleResourceOutputObj, bool) {
		name, ok := n.apps[id]
		return &serializer.RoleResourceOutputObj{ID: id, Name: name}, ok
	})
}

// envOutputs 返回存在的标准环境的展示信息，已删除的环境被忽略
func (n *roleResourceNames) envOutputs(envIDs []string) []*serializer.RoleResourceOutputObj {
	return lo.FilterMap(envIDs, func(id string, _ int) (*serializer.RoleResourceOutputObj, bool) {
		env, ok := n.envs[id]
		return env, ok
	})
}

// roleDisplayName 返回角色的展示名称
func roleDisplayName(r *role.Role) string {
	if r.IsCustom() {
		return r.DisplayName
	}
	return lo.CoalesceOrEmpty(builtinRoleNames[r.RoleCode], r.Name)
}

// toRoleOutput 转换角色为输出对象，已删除的应用和环境不出现在范围中
func toRoleOutput(r *role.Role, memberCount int, names *roleResourceNames) *serializer.RoleOutputObj {
	perms, appScope, envScope := customrole.Definition(r)
	return &serializer.RoleOutputObj{
		ID:          r.ID,
		RoleCode:    r.RoleCode,
		Name:        roleDisplayName(r),
		Description: r.Description,
		Builtin:     !r.IsCustom(),
		Permissions: lo.Ternary(perms == nil, []string{}, perms),
		AppScope: &serializer.RoleAppScopeOutputObj{
			Mode: appScope.Mode,
			Apps: names.appOutputs(appScope.AppIDs),
		},
		EnvScope: &serializer.RoleEnvScopeOutputObj{
			Mode:     envScope.Mode,
			EnvTypes: lo.Ternary(envScope.EnvTypes == nil, []string{}, envScope.EnvTypes),
			Envs:     names.envOutputs(envScope.EnvIDs),
		},
		MemberCount: memberCount,
	}
}

// addRoleAudit 记录角色定义变更的操作审计
func (h *Handler) addRoleAudit(
	ctx context.Context,
	workspaceID string,
	op audit.OperationType,
	before, after *role.Role,
) {
	toData := func(r *role.Role) any {
		if r == nil {
			return nil
		}
		return map[string]any{
			"roleID": r.ID, "name": r.DisplayName, "permissions": r.Permissions,
			"appScope": r.AppScope, "envScope": r.EnvScope,
		}
	}
	go audit.AddOperationRecordAsync(
		ctx,
		op,
		audit.ResourceTypeWorkspace,
		workspaceID,
		audit.WithAttribute(audit.AttributeUserRole),
		audit.WithDataBefore(toData(before)),
		audit.WithDataAfter(toData(after)),
		audit.WithWorkspaceID(workspaceID),
	)
}

// syncMonitorMemberRemoval 管理员、SRE 的成员同步在蓝鲸监控告警组中；移出单个角色时，仅当用户不再属于另一告警角色时一并移除
func (h *Handler) syncMonitorMemberRemoval(
	ctx context.Context, permMgr perm.Manager, ws *workspace.Workspace, removedRoleCode, userID, operator string,
) {
	if removedRoleCode != perm.RoleCodeAdmin && removedRoleCode != perm.RoleCodeSre {
		return
	}
	otherCode := lo.Ternary(removedRoleCode == perm.RoleCodeAdmin, perm.RoleCodeSre, perm.RoleCodeAdmin)
	if otherRole, err := permMgr.GetRole(ctx, ws.ID, otherCode); err == nil {
		if otherMembers, mErr := permMgr.ListRoleMembers(ctx, otherRole.ID); mErr == nil &&
			lo.Contains(otherMembers, userID) {
			return
		}
	}
	go bkmonitor.NewUserGroupService(h.registry.DB).RemoveMemberForWorkspace(
		context.WithoutCancel(ctx), ws, userID, operator,
	)
}
