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
	"fmt"

	iamsdk "github.com/TencentBlueKing/iam-go-sdk"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/actions"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/scope"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	cloudapiiam "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam"
	cloudapiiamtypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

// IAM resource type identifiers used to build iam-go-sdk requests.
const (
	resourceTypeWorkspace = "workspace"
	resourceTypeApp       = "app"
	resourceTypeEnv       = "env"

	// subjectTypeUser is the IAM subject type for end users.
	subjectTypeUser = "user"

	// pathAttrKey is the path attribute key recognized by BlueKing IAM
	// when expressing resource hierarchy via _bk_iam_path_.
	pathAttrKey = "_bk_iam_path_"

	// pathAttrWorkspaceTmpl is the parent-path template used for
	// resources nested under a workspace, e.g. apps and envs.
	pathAttrWorkspaceTmpl = "/workspace,%s/"

	// adminRoleDescTmpl is the admin role description template.
	adminRoleDescTmpl = "工作空间(%s)的管理员, 拥有该工作空间下的所有权限"
)

// IAMService orchestrates the BlueKing IAM gateway client and the role
// storage, exposing high-level business operations such as creating a
// workspace admin / built-in roles and checking workspace / app / env
// permissions.
//
// This is the bkms-server in-process IAM orchestration service. It uses
// pure-Go DTOs (see dto.go), so this package does not depend on generated PB
// request/response types.
type IAMService struct {
	iamClient cloudapiiam.IAMClient
	store     role.RoleStore
}

// NewIAMService creates an IAMService.
//
// Argument order follows the "dependencies first, data later" convention:
// the IAM gateway client and the role storage are both dependencies and
// have no business-data parameters at construction time.
func NewIAMService(client cloudapiiam.IAMClient, store role.RoleStore) *IAMService {
	return &IAMService{iamClient: client, store: store}
}

// CreateWorkspaceAdmin creates the workspace admin role (and its initial
// members), as well as the underlying IAM grade manager. If an admin role
// already exists for the given workspace, it is returned as-is (idempotent).
func (s *IAMService) CreateWorkspaceAdmin(
	ctx context.Context,
	data WorkspaceData,
	users []string,
) (*role.Role, error) {
	workspaceID := data.WorkspaceID

	if adminRole, _ := s.getAdminRoleFromStorage(ctx, workspaceID); adminRole != nil {
		return adminRole, nil
	}

	authScopes := s.genAuthScopesByWorkspaceData(data, role.BuiltinRoleCode.Admin)

	gradeManagerID, err := s.iamClient.CreateGradeManager(
		ctx, role.GenGradeManagerName(workspaceID), "", users, authScopes,
	)
	if err != nil {
		return nil, errors.Wrap(err, "create workspace admin: create grade manager")
	}

	if _, err = s.store.CreateWorkspaceGradeManager(
		ctx,
		&role.WorkspaceGradeManager{WorkspaceID: workspaceID, GradeManagerID: *gradeManagerID},
	); err != nil {
		return nil, errors.Wrap(err, "create workspace admin: persist grade manager")
	}

	roleName := role.GenWorkspaceRoleName(workspaceID, role.BuiltinRoleCode.Admin)
	roleDesc := fmt.Sprintf(adminRoleDescTmpl, workspaceID)
	r, err := s.createRole(
		ctx,
		*gradeManagerID,
		workspaceID,
		roleName,
		role.BuiltinRoleCode.Admin,
		roleDesc,
		true,
		role.PermissionScope{ResourceType: role.WorkspaceResourceType, ResourceID: workspaceID},
		authScopes,
	)
	if err != nil {
		return nil, errors.Wrap(err, "create workspace admin: create role")
	}

	if err = s.addRoleMembers(ctx, r, users, cloudapiiam.NeverExpireTimestamp); err != nil {
		return nil, errors.Wrap(err, "create workspace admin: add members")
	}

	return r, nil
}

// UpdateWorkspaceAdmin refreshes the auth scopes of the workspace admin
// role, both at the IAM grade-manager level and at the user-group level.
func (s *IAMService) UpdateWorkspaceAdmin(ctx context.Context, data WorkspaceData) error {
	gradeManagerID, err := s.getGradeManagerIDFromStorage(ctx, data.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "update workspace admin: get grade manager id")
	}

	authScopes := s.genAuthScopesByWorkspaceData(data, role.BuiltinRoleCode.Admin)

	if err = s.iamClient.UpdateGradeManager(
		ctx, gradeManagerID, role.GenGradeManagerName(data.WorkspaceID), "", authScopes,
	); err != nil {
		return errors.Wrap(err, "update workspace admin: update grade manager")
	}

	adminRole, err := s.getAdminRoleFromStorage(ctx, data.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "update workspace admin: get admin role")
	}

	if err = s.grantRolePolicies(ctx, adminRole, authScopes); err != nil {
		return errors.Wrap(err, "update workspace admin: grant policies")
	}
	return nil
}

// CreateWorkspaceScopeBuiltinRoles creates the workspace-scope built-in
// roles (developer / sre / operator) under an existing grade manager, and
// returns the full set of roles (including any pre-existing ones).
func (s *IAMService) CreateWorkspaceScopeBuiltinRoles(
	ctx context.Context,
	data WorkspaceData,
) ([]*role.Role, error) {
	workspaceID := data.WorkspaceID

	gradeManagerID, err := s.getGradeManagerIDFromStorage(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "create workspace builtin roles: get grade manager id")
	}

	isGradeManager := false
	rs := role.PermissionScope{ResourceType: role.WorkspaceResourceType, ResourceID: workspaceID}
	queryParams := role.RoleQueryParams{
		WorkspaceID:    &workspaceID,
		IsGradeManager: &isGradeManager,
		Scope:          &rs,
	}
	existedRoles, err := s.store.ListRoles(ctx, &queryParams)
	if err != nil {
		return nil, errors.Wrap(err, "create workspace builtin roles: list existed roles")
	}
	existedNames := make(map[string]struct{}, len(existedRoles))
	for _, r := range existedRoles {
		existedNames[r.Name] = struct{}{}
	}

	for _, roleCode := range role.WorkspaceScopeBuiltinRoles {
		roleName := role.GenWorkspaceRoleName(workspaceID, roleCode)
		if _, ok := existedNames[roleName]; ok {
			continue
		}
		if _, err = s.createRole(
			ctx,
			gradeManagerID,
			workspaceID,
			roleName,
			roleCode,
			"",
			false,
			role.PermissionScope{ResourceType: role.WorkspaceResourceType, ResourceID: workspaceID},
			s.genAuthScopesByWorkspaceData(data, roleCode),
		); err != nil {
			return nil, errors.Wrapf(err, "create workspace builtin role(%s)", roleCode)
		}
	}

	existedRoles, err = s.store.ListRoles(ctx, &queryParams)
	if err != nil {
		return nil, errors.Wrap(err, "create workspace builtin roles: list final roles")
	}
	return existedRoles, nil
}

// UpdateWorkspaceScopeBuiltinRoles refreshes the auth scopes granted to all
// existing workspace-scope built-in roles.
func (s *IAMService) UpdateWorkspaceScopeBuiltinRoles(ctx context.Context, data WorkspaceData) error {
	workspaceID := data.WorkspaceID

	isGradeManager := false
	rs := role.PermissionScope{ResourceType: role.WorkspaceResourceType, ResourceID: workspaceID}
	queryParams := role.RoleQueryParams{
		WorkspaceID:    &workspaceID,
		IsGradeManager: &isGradeManager,
		Scope:          &rs,
	}
	existedRoles, err := s.store.ListRoles(ctx, &queryParams)
	if err != nil {
		return errors.Wrap(err, "update workspace builtin roles: list roles")
	}
	roleMap := make(map[string]*role.Role, len(existedRoles))
	for _, r := range existedRoles {
		roleMap[r.Name] = r
	}

	for _, tplCode := range role.WorkspaceScopeBuiltinRoles {
		roleName := role.GenWorkspaceRoleName(workspaceID, tplCode)
		r, ok := roleMap[roleName]
		if !ok {
			continue
		}
		authScopes := s.genAuthScopesByWorkspaceData(data, tplCode)
		if err = s.grantRolePolicies(ctx, r, authScopes); err != nil {
			return errors.Wrapf(err, "update workspace builtin role(%s)", tplCode)
		}
	}
	return nil
}

// AddRoleForUsers adds a list of users to the user group of the given role.
// If the role is the workspace's grade-manager role, the users are also
// added to the IAM grade manager's member list.
func (s *IAMService) AddRoleForUsers(ctx context.Context, roleID string, users []string) error {
	r, err := s.getRoleFromStorageByID(ctx, roleID)
	if err != nil {
		return errors.Wrap(err, "add role for users: get role")
	}

	if r.IsGradeManager {
		var gradeManagerID int
		gradeManagerID, err = s.getGradeManagerIDFromStorage(ctx, r.WorkspaceID)
		if err != nil {
			return errors.Wrap(err, "add role for users: get grade manager id")
		}
		if err = s.iamClient.AddGradeManagerMembers(ctx, gradeManagerID, users); err != nil {
			return errors.Wrap(err, "add role for users: add grade manager members")
		}
	}

	if err = s.addRoleMembers(ctx, r, users, cloudapiiam.NeverExpireTimestamp); err != nil {
		return errors.Wrap(err, "add role for users: add user group members")
	}
	return nil
}

// DeleteRoleForUsers removes a list of users from the user group of the
// given role (and from the grade manager when applicable).
func (s *IAMService) DeleteRoleForUsers(ctx context.Context, roleID string, users []string) error {
	r, err := s.getRoleFromStorageByID(ctx, roleID)
	if err != nil {
		return errors.Wrap(err, "delete role for users: get role")
	}

	if r.IsGradeManager {
		var gradeManagerID int
		gradeManagerID, err = s.getGradeManagerIDFromStorage(ctx, r.WorkspaceID)
		if err != nil {
			return errors.Wrap(err, "delete role for users: get grade manager id")
		}
		if err = s.iamClient.DeleteGradeManagerMembers(ctx, gradeManagerID, users); err != nil {
			return errors.Wrap(err, "delete role for users: delete grade manager members")
		}
	}

	if err = s.deleteRoleMembers(ctx, r, users); err != nil {
		return errors.Wrap(err, "delete role for users: delete user group members")
	}
	return nil
}

// ListRoles returns roles that match the given workspace and scope.
func (s *IAMService) ListRoles(
	ctx context.Context, workspaceID string, scp role.PermissionScope,
) ([]*role.Role, error) {
	params := role.RoleQueryParams{WorkspaceID: &workspaceID, Scope: &scp}
	return s.store.ListRoles(ctx, &params)
}

// ListRoleMembers returns the user list of the given role's user group.
func (s *IAMService) ListRoleMembers(ctx context.Context, roleID string) ([]string, error) {
	r, err := s.store.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, errors.Wrap(err, "list role members: get role")
	}
	members, err := s.iamClient.ListUserGroupMembers(ctx, r.UserGroupID)
	if err != nil {
		return nil, errors.Wrap(err, "list role members: list user group members")
	}
	users := make([]string, len(members))
	for i, m := range members {
		users[i] = m.ID
	}
	return users, nil
}

// DeleteAllRolesByWorkspaceID removes every role belonging to the given
// workspace, including the IAM user groups, the local role records, the
// IAM grade manager and the local grade-manager record. The function tries
// to make a best effort to delete IAM user groups even if some of them
// fail; the first error encountered is preserved.
func (s *IAMService) DeleteAllRolesByWorkspaceID(ctx context.Context, workspaceID string) error {
	roles, err := s.store.ListRoles(ctx, &role.RoleQueryParams{WorkspaceID: &workspaceID})
	if err != nil {
		return errors.Wrap(err, "delete all roles: list roles")
	}

	var firstErr error
	deletedUserGroups := make([]int, 0, len(roles))
	for _, r := range roles {
		if delErr := s.iamClient.DeleteUserGroup(ctx, r.UserGroupID); delErr != nil {
			if firstErr == nil {
				firstErr = delErr
			}
			continue
		}
		deletedUserGroups = append(deletedUserGroups, r.UserGroupID)
	}

	if err = s.store.DeleteRolesByUserGroupIDs(ctx, workspaceID, deletedUserGroups); err != nil {
		if firstErr == nil {
			firstErr = err
		} else {
			firstErr = errors.Wrapf(firstErr, "delete role rows: %s", err.Error())
		}
	}

	if firstErr != nil {
		return errors.Wrap(firstErr, "delete all roles: delete user groups")
	}

	gradeManagerID, err := s.getGradeManagerIDFromStorage(ctx, workspaceID)
	if err != nil {
		return errors.Wrap(err, "delete all roles: get grade manager id")
	}
	if err = s.iamClient.DeleteGradeManager(ctx, gradeManagerID); err != nil {
		return errors.Wrap(err, "delete all roles: delete grade manager")
	}
	if err = s.store.DeleteWorkspaceGradeManager(ctx, workspaceID); err != nil {
		return errors.Wrap(err, "delete all roles: delete grade manager record")
	}
	return nil
}

// WorkspaceCreateIsAllowed checks whether the user is allowed to create a
// workspace.
func (s *IAMService) WorkspaceCreateIsAllowed(username string) (bool, error) {
	return s.iamClient.IsAllowed(
		iamsdk.NewRequest(
			config.G.BkIAMSystemIDs.Bkms,
			iamsdk.NewSubject(subjectTypeUser, username),
			iamsdk.NewAction(actions.WorkspaceAction.Create),
			[]iamsdk.ResourceNode{},
		),
	)
}

// WorkspaceActionIsAllowed checks whether the user is allowed to perform
// a single action on a single workspace.
func (s *IAMService) WorkspaceActionIsAllowed(
	username, workspaceID, actionID string,
) (bool, error) {
	return s.iamClient.IsAllowed(
		iamsdk.NewRequest(
			config.G.BkIAMSystemIDs.Bkms,
			iamsdk.NewSubject(subjectTypeUser, username),
			iamsdk.NewAction(actionID),
			iamsdk.Resources{
				iamsdk.NewResourceNode(config.G.BkIAMSystemIDs.Bkms, resourceTypeWorkspace, workspaceID, nil),
			},
		),
	)
}

// WorkspacesMultiActionsAllowed checks multiple actions against multiple
// workspaces in a single batched request.
func (s *IAMService) WorkspacesMultiActionsAllowed(
	username string, workspaceIDs, actionIDs []string,
) (map[string]map[string]bool, error) {
	acts := make([]iamsdk.Action, 0, len(actionIDs))
	for _, actionID := range actionIDs {
		acts = append(acts, iamsdk.NewAction(actionID))
	}
	request := iamsdk.MultiActionRequest{
		System:    config.G.BkIAMSystemIDs.Bkms,
		Subject:   iamsdk.NewSubject(subjectTypeUser, username),
		Actions:   acts,
		Resources: []iamsdk.ResourceNode{},
	}
	resources := make([]iamsdk.Resources, 0, len(workspaceIDs))
	for _, id := range workspaceIDs {
		resources = append(
			resources,
			iamsdk.Resources{
				iamsdk.NewResourceNode(config.G.BkIAMSystemIDs.Bkms, resourceTypeWorkspace, id, nil),
			},
		)
	}
	return s.iamClient.BatchResourceMultiActionsAllowed(request, resources)
}

// AppCreateIsAllowed checks whether the user is allowed to create an app
// under the given workspace.
func (s *IAMService) AppCreateIsAllowed(username, workspaceID string) (bool, error) {
	return s.iamClient.IsAllowed(
		iamsdk.NewRequest(
			config.G.BkIAMSystemIDs.Bkms,
			iamsdk.NewSubject(subjectTypeUser, username),
			iamsdk.NewAction(actions.AppAction.Create),
			iamsdk.Resources{
				iamsdk.NewResourceNode(config.G.BkIAMSystemIDs.Bkms, resourceTypeWorkspace, workspaceID, nil),
			},
		),
	)
}

// AppActionIsAllowed checks whether the user is allowed to perform a
// non-create action on a single app under the given workspace.
func (s *IAMService) AppActionIsAllowed(
	username, workspaceID, appID, actionID string,
) (bool, error) {
	return s.iamClient.IsAllowed(
		iamsdk.NewRequest(
			config.G.BkIAMSystemIDs.Bkms,
			iamsdk.NewSubject(subjectTypeUser, username),
			iamsdk.NewAction(actionID),
			iamsdk.Resources{
				iamsdk.NewResourceNode(
					config.G.BkIAMSystemIDs.Bkms, resourceTypeApp, appID,
					map[string]any{pathAttrKey: fmt.Sprintf(pathAttrWorkspaceTmpl, workspaceID)},
				),
			},
		),
	)
}

// AppsMultiActionsAllowed checks multiple actions against multiple apps
// under the given workspace, in a single batched request.
func (s *IAMService) AppsMultiActionsAllowed(
	username, workspaceID string, appIDs, actionIDs []string,
) (map[string]map[string]bool, error) {
	acts := make([]iamsdk.Action, 0, len(actionIDs))
	for _, actionID := range actionIDs {
		acts = append(acts, iamsdk.NewAction(actionID))
	}
	request := iamsdk.MultiActionRequest{
		System:    config.G.BkIAMSystemIDs.Bkms,
		Subject:   iamsdk.NewSubject(subjectTypeUser, username),
		Actions:   acts,
		Resources: []iamsdk.ResourceNode{},
	}
	resources := make([]iamsdk.Resources, 0, len(appIDs))
	for _, id := range appIDs {
		resources = append(
			resources,
			iamsdk.Resources{
				iamsdk.NewResourceNode(
					config.G.BkIAMSystemIDs.Bkms, resourceTypeApp, id,
					map[string]any{pathAttrKey: fmt.Sprintf(pathAttrWorkspaceTmpl, workspaceID)},
				),
			},
		)
	}
	return s.iamClient.BatchResourceMultiActionsAllowed(request, resources)
}

// EnvCreateIsAllowed checks whether the user is allowed to create an env
// under the given workspace.
func (s *IAMService) EnvCreateIsAllowed(username, workspaceID string) (bool, error) {
	return s.iamClient.IsAllowed(
		iamsdk.NewRequest(
			config.G.BkIAMSystemIDs.Bkms,
			iamsdk.NewSubject(subjectTypeUser, username),
			iamsdk.NewAction(actions.EnvAction.Create),
			iamsdk.Resources{
				iamsdk.NewResourceNode(config.G.BkIAMSystemIDs.Bkms, resourceTypeWorkspace, workspaceID, nil),
			},
		),
	)
}

// EnvActionIsAllowed checks whether the user is allowed to perform a
// non-create action on a single env under the given workspace.
func (s *IAMService) EnvActionIsAllowed(
	username, workspaceID, envID, actionID string,
) (bool, error) {
	return s.iamClient.IsAllowed(
		iamsdk.NewRequest(
			config.G.BkIAMSystemIDs.Bkms,
			iamsdk.NewSubject(subjectTypeUser, username),
			iamsdk.NewAction(actionID),
			iamsdk.Resources{
				iamsdk.NewResourceNode(
					config.G.BkIAMSystemIDs.Bkms, resourceTypeEnv, envID,
					map[string]any{pathAttrKey: fmt.Sprintf(pathAttrWorkspaceTmpl, workspaceID)},
				),
			},
		),
	)
}

// EnvsMultiActionsAllowed checks multiple actions against multiple envs
// under the given workspace, in a single batched request.
func (s *IAMService) EnvsMultiActionsAllowed(
	username, workspaceID string, envIDs, actionIDs []string,
) (map[string]map[string]bool, error) {
	acts := make([]iamsdk.Action, 0, len(actionIDs))
	for _, actionID := range actionIDs {
		acts = append(acts, iamsdk.NewAction(actionID))
	}
	request := iamsdk.MultiActionRequest{
		System:    config.G.BkIAMSystemIDs.Bkms,
		Subject:   iamsdk.NewSubject(subjectTypeUser, username),
		Actions:   acts,
		Resources: []iamsdk.ResourceNode{},
	}
	resources := make([]iamsdk.Resources, 0, len(envIDs))
	for _, id := range envIDs {
		resources = append(
			resources,
			iamsdk.Resources{
				iamsdk.NewResourceNode(
					config.G.BkIAMSystemIDs.Bkms, resourceTypeEnv, id,
					map[string]any{pathAttrKey: fmt.Sprintf(pathAttrWorkspaceTmpl, workspaceID)},
				),
			},
		)
	}
	return s.iamClient.BatchResourceMultiActionsAllowed(request, resources)
}

// createRole creates a single role under the given grade manager: it
// creates the IAM user group, persists the role record and grants the
// initial set of policies to the user group.
func (s *IAMService) createRole(
	ctx context.Context,
	gradeManagerID int,
	workspaceID, roleName, roleCode, roleDesc string,
	isGradeManager bool,
	scp role.PermissionScope,
	authScopes []cloudapiiamtypes.AuthorizationScope,
) (*role.Role, error) {
	userGroups, err := s.iamClient.CreateUserGroups(
		ctx, gradeManagerID,
		cloudapiiamtypes.UserGroupParam{Name: roleName, Description: roleDesc, Readonly: true},
	)
	if err != nil {
		return nil, errors.Wrap(err, "create user group")
	}
	if len(userGroups) == 0 {
		return nil, errors.New("create user group returned empty result")
	}

	groupID := userGroups[0].ID
	r := &role.Role{
		ID:             uuid.NewString(),
		Name:           roleName,
		RoleCode:       roleCode,
		Description:    roleDesc,
		WorkspaceID:    workspaceID,
		IsGradeManager: isGradeManager,
		Scope:          scp,
		UserGroupID:    groupID,
	}
	if _, err = s.store.CreateRole(ctx, r); err != nil {
		return nil, errors.Wrap(err, "persist role")
	}
	if err = s.grantRolePolicies(ctx, r, authScopes); err != nil {
		return nil, errors.Wrap(err, "grant role policies")
	}
	return r, nil
}

// addRoleMembers adds members to the role's user group.
func (s *IAMService) addRoleMembers(
	ctx context.Context, r *role.Role, members []string, expiredAt int,
) error {
	return s.iamClient.AddUserGroupMembers(ctx, r.UserGroupID, members, expiredAt)
}

// deleteRoleMembers removes members from the role's user group.
func (s *IAMService) deleteRoleMembers(ctx context.Context, r *role.Role, members []string) error {
	return s.iamClient.DeleteUserGroupMembers(ctx, r.UserGroupID, members)
}

// grantRolePolicies grants a set of auth scopes to the role's user group.
func (s *IAMService) grantRolePolicies(
	ctx context.Context, r *role.Role, authScopes []cloudapiiamtypes.AuthorizationScope,
) error {
	return s.iamClient.GrantUserGroupPolicies(ctx, r.UserGroupID, authScopes)
}

// getGradeManagerIDFromStorage returns the persisted IAM grade-manager ID
// for the given workspace, or an error if not found.
func (s *IAMService) getGradeManagerIDFromStorage(ctx context.Context, workspaceID string) (int, error) {
	gm, err := s.store.GetWorkspaceGradeManager(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	if gm.GradeManagerID == 0 {
		return 0, errors.Errorf("workspace(%s) grade manager not found", workspaceID)
	}
	return gm.GradeManagerID, nil
}

// getAdminRoleFromStorage returns the persisted admin role of the given
// workspace, or an error if not found.
func (s *IAMService) getAdminRoleFromStorage(ctx context.Context, workspaceID string) (*role.Role, error) {
	isGradeManager := true
	roles, err := s.store.ListRoles(
		ctx, &role.RoleQueryParams{WorkspaceID: &workspaceID, IsGradeManager: &isGradeManager},
	)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, errors.Errorf("workspace(%s) admin role not found", workspaceID)
	}
	return roles[0], nil
}

// getRoleFromStorageByID returns the persisted role record for the given
// role ID, or an error if it has no associated user group.
func (s *IAMService) getRoleFromStorageByID(ctx context.Context, roleID string) (*role.Role, error) {
	r, err := s.store.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if r.UserGroupID == 0 {
		return nil, errors.Errorf("role(%s) not found", roleID)
	}
	return r, nil
}

// genAuthScopesByWorkspaceData fan-outs WorkspaceData into a list of scope
// generators (one per integrated business system) and concatenates their
// produced authorization scopes.
func (s *IAMService) genAuthScopesByWorkspaceData(
	data WorkspaceData, roleCode string,
) []cloudapiiamtypes.AuthorizationScope {
	generators := []scope.AuthScopesGenerator{
		scope.BKMSRoleScopesGenerator{
			WorkspaceID:   data.WorkspaceID,
			WorkspaceName: data.WorkspaceName,
			TplRoleCode:   roleCode,
		},
	}
	if data.BKCI != nil {
		generators = append(generators, scope.BKCIRoleScopesGenerator{
			ProjectID: data.BKCI.ProjectID, ProjectName: data.BKCI.ProjectName, TplRoleCode: roleCode,
		})
	}
	if data.BCS != nil {
		generators = append(generators, scope.BCSRoleScopesGenerator{
			ProjectID: data.BCS.ProjectID, ProjectName: data.BCS.ProjectName, TplRoleCode: roleCode,
		})
	}
	if data.BKMonitor != nil {
		generators = append(generators, scope.BKMonitorRoleScopesGenerator{
			SpaceID: data.BKMonitor.SpaceID, SpaceName: data.BKMonitor.SpaceName, TplRoleCode: roleCode,
		})
	}
	if data.BKLog != nil {
		generators = append(generators, scope.BKLogRoleScopesGenerator{
			SpaceID: data.BKLog.SpaceID, SpaceName: data.BKLog.SpaceName, TplRoleCode: roleCode,
		})
	}
	if data.BKRepo != nil {
		generators = append(generators, scope.BKRepoRoleScopesGenerator{
			ProjectID: data.BKRepo.ProjectID, ProjectName: data.BKRepo.ProjectName, TplRoleCode: roleCode,
		})
	}
	if data.BSCP != nil {
		generators = append(generators, scope.BSCPRoleScopesGenerator{
			BizID:       data.BSCP.BizID,
			BizName:     data.BSCP.BizName,
			TplRoleCode: roleCode,
			Services:    toScopeBSCPServices(data.BSCP.Services),
		})
	}
	return scope.GenerateAuthScopes(generators...)
}

// toScopeBSCPServices converts the public DTO BSCPService slice into the
// scope-package internal representation.
func toScopeBSCPServices(services []BSCPService) []scope.BSCPService {
	result := make([]scope.BSCPService, len(services))
	for i, svc := range services {
		result[i] = scope.BSCPService{ID: svc.ID, Name: svc.Name}
	}
	return result
}
