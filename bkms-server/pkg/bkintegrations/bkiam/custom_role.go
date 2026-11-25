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

package bkiam

import (
	"context"
	"slices"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/actions"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/scope"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	cloudapiiamtypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrNotCustomRole 角色不是自定义角色，不能修改权限、范围或删除
var ErrNotCustomRole = errors.New("role is not a custom role")

// ResourceRef 授权路径中的资源，名称仅用于权限中心展示
type ResourceRef struct {
	ID   string
	Name string
}

// EnvResourceRef 授权路径中的标准环境，Type 为环境当前类型
type EnvResourceRef struct {
	ID   string
	Name string
	Type string
}

// CustomRoleMonitorData 自定义角色在监控中的授权数据。
//
// BKMS 用 iframe 嵌入 APM 页面，需要用户本人有 APM 查看权限。自定义角色只授予范围内环境
// 绑定的 APM 应用的查看权限，以及它依赖的业务访问（范围是整个监控空间）。
type CustomRoleMonitorData struct {
	SpaceID   string
	SpaceName string
	// AllApms 环境范围为全部时授予空间下全部 APM 应用，包括之后新建的
	AllApms bool
	// Apms 范围内环境绑定的 APM 应用，ID 为监控的 APM 应用 ID
	Apms []ResourceRef
}

// CustomRoleData 生成自定义角色授权所需的数据
type CustomRoleData struct {
	WorkspaceID   string
	WorkspaceName string
	// Permissions 已经过 role.NormalizePermissions 校验与补齐的权限项
	Permissions []role.Permission
	AppScope    role.AppScope
	EnvScope    role.EnvScope
	// Apps 应用范围为指定应用时的应用
	Apps []ResourceRef
	// Envs 环境范围为指定环境时的环境
	Envs []EnvResourceRef
	// BKMonitor 空间的监控信息，监控项目尚未就绪时为空
	BKMonitor *CustomRoleMonitorData
}

// bkmsWorkspaceActions 空间内角色可能授予的全部 BKMS action，修改自定义角色时据此清空旧授权
var bkmsWorkspaceActions = []string{
	actions.WorkspaceAction.View, actions.WorkspaceAction.Edit, actions.WorkspaceAction.Delete,
	actions.AppAction.Create, actions.AppAction.View, actions.AppAction.Edit, actions.AppAction.Delete,
	actions.EnvAction.Create, actions.EnvAction.View, actions.EnvAction.Edit, actions.EnvAction.Delete,
	actions.EnvAction.Deploy,
	actions.AppEnvAction.View, actions.AppEnvAction.Edit, actions.AppEnvAction.Deploy,
}

// customRoleMonitorActions 自定义角色可能授予的全部监控 action，修改自定义角色时据此清空旧授权。
// APM 应用查看依赖业务访问，所以一起授予
var customRoleMonitorActions = []string{actions.SpaceAction.View, actions.APMAction.View}

// GenCustomRoleAuthScopes 将自定义角色的权限与范围转换为 IAM 授权。
//
// 同时涉及应用和环境的权限使用联合 action，应用范围和环境范围写在同一条策略里，
// 因此不同角色的范围不会被拼接；只涉及单一资源的权限沿用原有 action。
func GenCustomRoleAuthScopes(data CustomRoleData) []cloudapiiamtypes.AuthorizationScope {
	system := config.G.BkIAMSystemIDs.Bkms
	wsNode := cloudapiiamtypes.ResourcePath{
		System: system, Type: cloudapiiamtypes.WorkspaceResourceType, ID: data.WorkspaceID, Name: data.WorkspaceName,
	}
	wsRes := cloudapiiamtypes.Resource{
		System: system, Type: cloudapiiamtypes.WorkspaceResourceType,
		Paths: []cloudapiiamtypes.ResourcePaths{{wsNode}},
	}
	appRes := cloudapiiamtypes.Resource{System: system, Type: cloudapiiamtypes.AppResourceType}
	if data.AppScope.IsAll() {
		appRes.Paths = []cloudapiiamtypes.ResourcePaths{{wsNode}}
	} else {
		for _, app := range data.Apps {
			appRes.Paths = append(appRes.Paths, cloudapiiamtypes.ResourcePaths{wsNode, {
				System: system, Type: cloudapiiamtypes.AppResourceType, ID: app.ID, Name: app.Name,
			}})
		}
	}
	envRes := cloudapiiamtypes.Resource{System: system, Type: cloudapiiamtypes.EnvResourceType}
	switch data.EnvScope.Mode {
	case role.EnvScopeTypes:
		for _, t := range data.EnvScope.EnvTypes {
			envRes.Paths = append(envRes.Paths, cloudapiiamtypes.ResourcePaths{
				wsNode, {System: system, Type: cloudapiiamtypes.EnvTypeResourceType, ID: t, Name: t},
			})
		}
	case role.EnvScopeEnvs:
		for _, env := range data.Envs {
			envRes.Paths = append(envRes.Paths, cloudapiiamtypes.ResourcePaths{
				wsNode,
				{System: system, Type: cloudapiiamtypes.EnvTypeResourceType, ID: env.Type, Name: env.Type},
				{System: system, Type: cloudapiiamtypes.EnvResourceType, ID: env.ID, Name: env.Name},
			})
		}
	default:
		envRes.Paths = []cloudapiiamtypes.ResourcePaths{{wsNode}}
	}

	var wsActions, appActions, envActions, appEnvActions []string
	for _, p := range data.Permissions {
		switch p {
		case role.Perm.SpaceView:
			wsActions = append(wsActions, actions.WorkspaceAction.View)
		case role.Perm.SpaceEdit:
			wsActions = append(wsActions, actions.WorkspaceAction.Edit)
		case role.Perm.SpaceDelete:
			wsActions = append(wsActions, actions.WorkspaceAction.Delete)
		case role.Perm.AppCreate:
			wsActions = append(wsActions, actions.AppAction.Create)
		case role.Perm.EnvCreate:
			wsActions = append(wsActions, actions.EnvAction.Create)
		case role.Perm.AppView:
			appActions = append(appActions, actions.AppAction.View)
			appEnvActions = append(appEnvActions, actions.AppEnvAction.View)
		case role.Perm.AppEdit:
			// edit_app 修改的是作用于所有环境的应用级数据，只授予覆盖全部环境的角色
			if data.EnvScope.IsAll() {
				appActions = append(appActions, actions.AppAction.Edit)
			}
			appEnvActions = append(appEnvActions, actions.AppEnvAction.Edit)
		case role.Perm.AppDelete:
			appActions = append(appActions, actions.AppAction.Delete)
		case role.Perm.EnvView:
			envActions = append(envActions, actions.EnvAction.View)
		case role.Perm.EnvEdit:
			envActions = append(envActions, actions.EnvAction.Edit)
		case role.Perm.EnvDelete:
			envActions = append(envActions, actions.EnvAction.Delete)
		case role.Perm.EnvDeploy:
			// 部署依赖在同一范围内编辑应用的环境配置，但不带上应用级的编辑权限
			appEnvActions = append(appEnvActions, actions.AppEnvAction.Edit, actions.AppEnvAction.Deploy)
		}
	}
	appEnvActions = lo.Uniq(appEnvActions)

	var scopes []cloudapiiamtypes.AuthorizationScope
	add := func(actionIDs []string, resources ...cloudapiiamtypes.Resource) {
		if len(actionIDs) == 0 {
			return
		}
		scopes = append(scopes, cloudapiiamtypes.AuthorizationScope{
			System: system,
			Actions: lo.Map(actionIDs, func(id string, _ int) cloudapiiamtypes.Action {
				return cloudapiiamtypes.Action{ID: id}
			}),
			Resources: resources,
		})
	}
	// 指定的应用或环境都已删除时范围为空，表示什么都不覆盖：不授予涉及该资源的 action，
	// 而不是授予一条没有路径的策略
	add(wsActions, wsRes)
	if len(appRes.Paths) > 0 {
		add(appActions, appRes)
	}
	if len(envRes.Paths) > 0 {
		add(envActions, envRes)
	}
	if len(appRes.Paths) > 0 && len(envRes.Paths) > 0 {
		add(appEnvActions, appRes, envRes)
	}
	if data.BKMonitor != nil && lo.Contains(data.Permissions, role.Perm.AppView) {
		scopes = append(scopes, genMonitorAuthScopes(*data.BKMonitor)...)
	}
	return scopes
}

// genMonitorAuthScopes 生成自定义角色在监控中的授权：业务访问，以及范围内的 APM 应用查看
func genMonitorAuthScopes(data CustomRoleMonitorData) []cloudapiiamtypes.AuthorizationScope {
	system := config.G.BkIAMSystemIDs.BkMonitor
	spaceNode := cloudapiiamtypes.ResourcePath{
		System: system, Type: cloudapiiamtypes.SpaceResourceType, ID: data.SpaceID, Name: data.SpaceName,
	}
	scopes := []cloudapiiamtypes.AuthorizationScope{{
		System:  system,
		Actions: []cloudapiiamtypes.Action{{ID: actions.SpaceAction.View}},
		Resources: []cloudapiiamtypes.Resource{{
			System: system, Type: cloudapiiamtypes.SpaceResourceType,
			Paths: []cloudapiiamtypes.ResourcePaths{{spaceNode}},
		}},
	}}

	apmRes := cloudapiiamtypes.Resource{System: system, Type: cloudapiiamtypes.APMResourceType}
	if data.AllApms {
		apmRes.Paths = []cloudapiiamtypes.ResourcePaths{{spaceNode}}
	} else {
		for _, apm := range data.Apms {
			apmRes.Paths = append(apmRes.Paths, cloudapiiamtypes.ResourcePaths{spaceNode, {
				System: system, Type: cloudapiiamtypes.APMResourceType, ID: apm.ID, Name: apm.Name,
			}})
		}
	}
	// 范围内的环境都没有绑定 APM 应用时不授予，避免授出一条没有路径的策略
	if len(apmRes.Paths) > 0 {
		scopes = append(scopes, cloudapiiamtypes.AuthorizationScope{
			System:    system,
			Actions:   []cloudapiiamtypes.Action{{ID: actions.APMAction.View}},
			Resources: []cloudapiiamtypes.Resource{apmRes},
		})
	}
	return scopes
}

// actionPermissions 原有 BKMS action 对应的页面权限项，用于从内置角色模板反推其权限
var actionPermissions = map[string]role.Permission{
	actions.WorkspaceAction.View:   role.Perm.SpaceView,
	actions.WorkspaceAction.Create: role.Perm.SpaceCreate,
	actions.WorkspaceAction.Edit:   role.Perm.SpaceEdit,
	actions.WorkspaceAction.Delete: role.Perm.SpaceDelete,
	actions.AppAction.Create:       role.Perm.AppCreate,
	actions.AppAction.View:         role.Perm.AppView,
	actions.AppAction.Edit:         role.Perm.AppEdit,
	actions.AppAction.Delete:       role.Perm.AppDelete,
	actions.EnvAction.View:         role.Perm.EnvView,
	actions.AppEnvAction.Deploy:    role.Perm.EnvDeploy,
	actions.EnvAction.Create:       role.Perm.EnvCreate,
	actions.EnvAction.Edit:         role.Perm.EnvEdit,
	actions.EnvAction.Delete:       role.Perm.EnvDelete,
}

var builtinRolePermsCache = sync.OnceValue(func() map[string][]role.Permission {
	codes := []string{
		role.BuiltinRoleCode.Admin,
		role.BuiltinRoleCode.Developer,
		role.BuiltinRoleCode.SRE,
		role.BuiltinRoleCode.Operator,
	}
	cache := make(map[string][]role.Permission, len(codes))
	for _, code := range codes {
		cache[code] = parseBuiltinRolePermissions(code)
	}
	return cache
})

// BuiltinRolePermissions 从内置角色的 BKMS 授权模板反推其页面权限项，保证展示的就是实际授予的权限。
// 内置角色的范围固定为全部应用 × 全部环境。
func BuiltinRolePermissions(roleCode string) []role.Permission {
	if perms, ok := builtinRolePermsCache()[roleCode]; ok {
		return slices.Clone(perms)
	}
	return parseBuiltinRolePermissions(roleCode)
}

func parseBuiltinRolePermissions(roleCode string) []role.Permission {
	authScopes := scope.BKMSRoleScopesGenerator{TplRoleCode: roleCode}.Generate()
	set := make(map[role.Permission]struct{})
	for _, s := range authScopes {
		for _, a := range s.Actions {
			if p, ok := actionPermissions[a.ID]; ok {
				set[p] = struct{}{}
			}
		}
	}
	return lo.Filter(role.AllPermissions, func(p role.Permission, _ int) bool {
		_, ok := set[p]
		return ok
	})
}

// GetRole returns the persisted role record of the given role ID.
func (s *IAMService) GetRole(ctx context.Context, roleID string) (*role.Role, error) {
	return s.getRoleFromStorageByID(ctx, roleID)
}

// CreateCustomRole creates a custom role under the workspace grade manager:
// an IAM user group with the policies generated from data, and the local
// role record holding the role definition.
func (s *IAMService) CreateCustomRole(
	ctx context.Context, displayName, description string, data CustomRoleData,
) (*role.Role, error) {
	gradeManagerID, err := s.getGradeManagerIDFromStorage(ctx, data.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "create custom role: get grade manager id")
	}

	id := uuid.NewString()
	groupName := role.GenWorkspaceRoleName(data.WorkspaceID, role.CustomRoleCode+"-"+id[:8])
	userGroups, err := s.iamClient.CreateUserGroups(
		ctx, gradeManagerID,
		cloudapiiamtypes.UserGroupParam{Name: groupName, Description: description, Readonly: true},
	)
	if err != nil {
		return nil, errors.Wrap(err, "create custom role: create user group")
	}
	if len(userGroups) == 0 {
		return nil, errors.New("create custom role: create user group returned empty result")
	}

	groupID := userGroups[0].ID
	r := &role.Role{
		ID:          id,
		Name:        groupName,
		RoleCode:    role.CustomRoleCode,
		Description: description,
		WorkspaceID: data.WorkspaceID,
		Scope:       role.PermissionScope{ResourceType: role.WorkspaceResourceType, ResourceID: data.WorkspaceID},
		UserGroupID: groupID,
		DisplayName: displayName,
		Permissions: data.Permissions,
		AppScope:    &data.AppScope,
		EnvScope:    &data.EnvScope,
	}
	if err = s.roles.Create(ctx, r); err != nil {
		_ = s.iamClient.DeleteUserGroup(ctx, groupID)
		return nil, errors.Wrap(err, "create custom role: persist role")
	}
	if err = s.applyCustomRole(ctx, r, data, false); err != nil {
		_ = s.roles.DeleteIfExists(ctx, role.ByRoleID(r.ID))
		_ = s.iamClient.DeleteUserGroup(ctx, groupID)
		return nil, errors.Wrap(err, "create custom role")
	}
	return r, nil
}

// UpdateCustomRole replaces the definition of a custom role and re-applies
// its IAM policies. The last update wins.
//
// The new definition is persisted first, then, since IAM merges repeated
// grants, the old policies are revoked before the new ones are granted. If it
// fails midway, the user group holds fewer permissions than persisted (never
// more), and retrying the same update completes it.
func (s *IAMService) UpdateCustomRole(
	ctx context.Context, roleID, displayName, description string, data CustomRoleData,
) (*role.Role, error) {
	r, err := s.getRoleFromStorageByID(ctx, roleID)
	if err != nil {
		return nil, errors.Wrap(err, "update custom role: get role")
	}
	if !r.IsCustom() {
		return nil, errors.Wrapf(ErrNotCustomRole, "update role %s", roleID)
	}

	r.DisplayName = displayName
	r.Description = description
	r.Permissions = data.Permissions
	r.AppScope = &data.AppScope
	r.EnvScope = &data.EnvScope
	err = s.roles.Update(
		ctx,
		r,
		store.UpdateSelected("displayName", "description", "permissions", "appScope", "envScope"),
		role.ByRoleID(roleID),
	)
	if err != nil {
		return nil, errors.Wrap(err, "update custom role: persist role")
	}
	if err = s.applyCustomRole(ctx, r, data, true); err != nil {
		return nil, errors.Wrap(err, "update custom role")
	}
	return r, nil
}

// DeleteCustomRole deletes a custom role: its IAM user group (with the
// policies and members) and the local role record.
func (s *IAMService) DeleteCustomRole(ctx context.Context, roleID string) error {
	r, err := s.getRoleFromStorageByID(ctx, roleID)
	if err != nil {
		return errors.Wrap(err, "delete custom role: get role")
	}
	if !r.IsCustom() {
		return errors.Wrapf(ErrNotCustomRole, "delete role %s", roleID)
	}
	if err = s.iamClient.DeleteUserGroup(ctx, r.UserGroupID); err != nil {
		return errors.Wrap(err, "delete custom role: delete user group")
	}
	if err = s.roles.Delete(ctx, role.ByRoleID(roleID)); err != nil {
		return errors.Wrap(err, "delete custom role: delete role record")
	}
	return nil
}

// applyCustomRole grants the IAM policies of a custom role, revoking the old
// ones first when revoke is true.
func (s *IAMService) applyCustomRole(ctx context.Context, r *role.Role, data CustomRoleData, revoke bool) error {
	if revoke {
		if err := s.iamClient.RevokeUserGroupActions(
			ctx, r.UserGroupID, config.G.BkIAMSystemIDs.Bkms, bkmsWorkspaceActions,
		); err != nil {
			return errors.Wrapf(err, "revoke policies of role %s", r.ID)
		}
		// 监控项目就绪前不会授予监控权限，也就不需要清空
		if data.BKMonitor != nil {
			if err := s.iamClient.RevokeUserGroupActions(
				ctx, r.UserGroupID, config.G.BkIAMSystemIDs.BkMonitor, customRoleMonitorActions,
			); err != nil {
				return errors.Wrapf(err, "revoke monitor policies of role %s", r.ID)
			}
		}
	}
	if err := s.grantRolePolicies(ctx, r, GenCustomRoleAuthScopes(data)); err != nil {
		return errors.Wrapf(err, "grant policies of role %s", r.ID)
	}
	return nil
}
