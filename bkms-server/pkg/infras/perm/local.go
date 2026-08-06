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

package perm

import (
	"context"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// iamServicer 是 LocalManager 内部依赖的 IAMService 编排接口。
//
// 抽象成接口的目的：
//  1. 收敛 LocalManager 真正用到的 IAMService 方法集合，便于阅读；
//  2. 单测中可以替换为同包 fake 实现，避免依赖真实蓝鲸 IAM 网关与 Mongo。
//
// 该接口不对外暴露，仅 perm 包内可见。
type iamServicer interface {
	CreateWorkspaceAdmin(ctx context.Context, data bkiam.WorkspaceData, users []string) (*role.Role, error)
	UpdateWorkspaceAdmin(ctx context.Context, data bkiam.WorkspaceData) error
	CreateWorkspaceScopeBuiltinRoles(ctx context.Context, data bkiam.WorkspaceData) ([]*role.Role, error)
	UpdateWorkspaceScopeBuiltinRoles(ctx context.Context, data bkiam.WorkspaceData) error

	AddRoleForUsers(ctx context.Context, roleID string, users []string) error
	DeleteRoleForUsers(ctx context.Context, roleID string, users []string) error
	ListRoles(ctx context.Context, workspaceID string, scp role.PermissionScope) ([]*role.Role, error)
	ListRoleMembers(ctx context.Context, roleID string) ([]string, error)
	DeleteAllRolesByWorkspaceID(ctx context.Context, workspaceID string) error

	WorkspaceCreateIsAllowed(username string) (bool, error)
	WorkspaceActionIsAllowed(username, workspaceID, actionID string) (bool, error)
	WorkspacesMultiActionsAllowed(
		username string, workspaceIDs, actionIDs []string,
	) (map[string]map[string]bool, error)

	AppCreateIsAllowed(username, workspaceID string) (bool, error)
	AppActionIsAllowed(username, workspaceID, appID, actionID string) (bool, error)
	AppsMultiActionsAllowed(
		username, workspaceID string, appIDs, actionIDs []string,
	) (map[string]map[string]bool, error)

	EnvCreateIsAllowed(username, workspaceID string) (bool, error)
	EnvActionIsAllowed(username, workspaceID, envID, actionID string) (bool, error)
}

// 编译期检查：iam.IAMService 满足 iamServicer 接口。
var _ iamServicer = (*bkiam.IAMService)(nil)

// LocalManager Manager 的进程内本地实现。
//
// 通过转发到 iam.IAMService 的方式提供权限检查与角色管理能力。所有方法
// 内部从 ctx 通过 auth.GetUser 取出当前用户后，再调用 IAMService（其
// *IsAllowed / *ActionIsAllowed 方法接收 username 而非 ctx）。
type LocalManager struct {
	svc iamServicer
}

// 编译期检查：LocalManager 实现 Manager 接口。
var _ Manager = (*LocalManager)(nil)

// 错误信息模板（保持现有对外错误文案稳定，便于业务侧错误识别）。
const (
	errMsgNoCreateApp        = "no permission to create app"
	errMsgNoCreateWorkspace  = "no permission to create workspace"
	errMsgNoCreateEnv        = "no permission to create env"
	errMsgNoActionAppFmt     = "no permission to %s application %s in workspace %s"
	errMsgNoActionWsFmt      = "no permission to %s workspace %s"
	errMsgNoActionEnvFmt     = "no permission to %s env %s in workspace %s"
	errMsgGetAuthUser        = "get auth user"
	errMsgCallIAMServiceFmt  = "call iam service: %s"
	errMsgWorkspaceFilterAct = "filter workspaces by action"
	errMsgAppFilterAct       = "filter apps by action"
)

// HasCreateAppPerm 检查用户是否有创建应用的权限
func (m *LocalManager) HasCreateAppPerm(ctx context.Context, workspaceID string) error {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return errors.Wrap(err, errMsgGetAuthUser)
	}
	allowed, err := m.svc.AppCreateIsAllowed(user.ID, workspaceID)
	if err != nil {
		return errors.Wrapf(err, errMsgCallIAMServiceFmt, "AppCreateIsAllowed")
	}
	if !allowed {
		return errors.New(errMsgNoCreateApp)
	}
	return nil
}

// HasEditAppPerm 检查用户是否有编辑应用的权限
func (m *LocalManager) HasEditAppPerm(ctx context.Context, workspaceID, appID string) error {
	return m.hasAppActionPerm(ctx, workspaceID, appID, AppAction.Edit)
}

// HasViewAppPerm 检查用户是否有查看应用的权限
func (m *LocalManager) HasViewAppPerm(ctx context.Context, workspaceID, appID string) error {
	return m.hasAppActionPerm(ctx, workspaceID, appID, AppAction.View)
}

// HasDeleteAppPerm 检查用户是否有删除应用的权限
func (m *LocalManager) HasDeleteAppPerm(ctx context.Context, workspaceID, appID string) error {
	return m.hasAppActionPerm(ctx, workspaceID, appID, AppAction.Delete)
}

// hasAppActionPerm 检查用户是否有应用某项权限
func (m *LocalManager) hasAppActionPerm(ctx context.Context, workspaceID, appID, action string) error {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return errors.Wrap(err, errMsgGetAuthUser)
	}
	allowed, err := m.svc.AppActionIsAllowed(user.ID, workspaceID, appID, action)
	if err != nil {
		return errors.Wrapf(err, errMsgCallIAMServiceFmt, "AppActionIsAllowed")
	}
	if !allowed {
		return errors.Errorf(errMsgNoActionAppFmt, action, appID, workspaceID)
	}
	return nil
}

// HasCreateWorkspacePerm 检查用户是否有创建工作空间的权限
func (m *LocalManager) HasCreateWorkspacePerm(ctx context.Context) error {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return errors.Wrap(err, errMsgGetAuthUser)
	}
	allowed, err := m.svc.WorkspaceCreateIsAllowed(user.ID)
	if err != nil {
		return errors.Wrapf(err, errMsgCallIAMServiceFmt, "WorkspaceCreateIsAllowed")
	}
	if !allowed {
		return errors.New(errMsgNoCreateWorkspace)
	}
	return nil
}

// HasViewWorkspacePerm 检查用户是否有查看工作空间的权限
func (m *LocalManager) HasViewWorkspacePerm(ctx context.Context, workspaceID string) error {
	return m.hasWorkspaceActionPerm(ctx, workspaceID, WorkspaceAction.View)
}

// HasEditWorkspacePerm 检查用户是否有编辑工作空间的权限
func (m *LocalManager) HasEditWorkspacePerm(ctx context.Context, workspaceID string) error {
	return m.hasWorkspaceActionPerm(ctx, workspaceID, WorkspaceAction.Edit)
}

// HasDeleteWorkspacePerm 检查用户是否有删除工作空间的权限
func (m *LocalManager) HasDeleteWorkspacePerm(ctx context.Context, workspaceID string) error {
	return m.hasWorkspaceActionPerm(ctx, workspaceID, WorkspaceAction.Delete)
}

// hasWorkspaceActionPerm 检查用户是否有工作空间某项权限
func (m *LocalManager) hasWorkspaceActionPerm(ctx context.Context, workspaceID, action string) error {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return errors.Wrap(err, errMsgGetAuthUser)
	}
	allowed, err := m.svc.WorkspaceActionIsAllowed(user.ID, workspaceID, action)
	if err != nil {
		return errors.Wrapf(err, errMsgCallIAMServiceFmt, "WorkspaceActionIsAllowed")
	}
	if !allowed {
		return errors.Errorf(errMsgNoActionWsFmt, action, workspaceID)
	}
	return nil
}

// HasCreateEnvPerm 检查用户是否有创建环境的权限
func (m *LocalManager) HasCreateEnvPerm(ctx context.Context, workspaceID string) error {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return errors.Wrap(err, errMsgGetAuthUser)
	}
	allowed, err := m.svc.EnvCreateIsAllowed(user.ID, workspaceID)
	if err != nil {
		return errors.Wrapf(err, errMsgCallIAMServiceFmt, "EnvCreateIsAllowed")
	}
	if !allowed {
		return errors.New(errMsgNoCreateEnv)
	}
	return nil
}

// HasEditEnvPerm 检查用户是否有编辑环境的权限
func (m *LocalManager) HasEditEnvPerm(ctx context.Context, workspaceID, envName string) error {
	return m.hasEnvActionPerm(ctx, workspaceID, envName, EnvAction.Edit)
}

// HasViewEnvPerm 检查用户是否有查看环境的权限
func (m *LocalManager) HasViewEnvPerm(ctx context.Context, workspaceID, envName string) error {
	return m.hasEnvActionPerm(ctx, workspaceID, envName, EnvAction.View)
}

// HasDeleteEnvPerm 检查用户是否有删除环境的权限
func (m *LocalManager) HasDeleteEnvPerm(ctx context.Context, workspaceID, envName string) error {
	return m.hasEnvActionPerm(ctx, workspaceID, envName, EnvAction.Delete)
}

// HasDeployEnvPerm 检查用户是否有部署环境的权限
func (m *LocalManager) HasDeployEnvPerm(ctx context.Context, workspaceID, envName string) error {
	return m.hasEnvActionPerm(ctx, workspaceID, envName, EnvAction.Deploy)
}

// hasEnvActionPerm 检查用户是否有环境某项权限
func (m *LocalManager) hasEnvActionPerm(ctx context.Context, workspaceID, envName, action string) error {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return errors.Wrap(err, errMsgGetAuthUser)
	}
	allowed, err := m.svc.EnvActionIsAllowed(user.ID, workspaceID, envName, action)
	if err != nil {
		return errors.Wrapf(err, errMsgCallIAMServiceFmt, "EnvActionIsAllowed")
	}
	if !allowed {
		return errors.Errorf(errMsgNoActionEnvFmt, action, envName, workspaceID)
	}
	return nil
}

// FilterViewableWorkspaces 过滤出用户有权限查看的工作空间列表
func (m *LocalManager) FilterViewableWorkspaces(
	ctx context.Context, workspaceIDs []string,
) (*set.StringSet, error) {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errMsgGetAuthUser)
	}
	resp, err := m.svc.WorkspacesMultiActionsAllowed(
		user.ID, workspaceIDs, []string{WorkspaceAction.View},
	)
	if err != nil {
		return nil, errors.Wrap(err, errMsgWorkspaceFilterAct)
	}
	hasPerm := set.NewStringSet()
	for wsID, actions := range resp {
		if actions[WorkspaceAction.View] {
			hasPerm.Add(wsID)
		}
	}
	return hasPerm, nil
}

// FilterViewableApps 过滤出用户有权限查看的应用列表
func (m *LocalManager) FilterViewableApps(
	ctx context.Context, workspaceID string, appIDs []string,
) (*set.StringSet, error) {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, errMsgGetAuthUser)
	}
	resp, err := m.svc.AppsMultiActionsAllowed(
		user.ID, workspaceID, appIDs, []string{AppAction.View},
	)
	if err != nil {
		return nil, errors.Wrap(err, errMsgAppFilterAct)
	}
	hasPerm := set.NewStringSet()
	for appID, actions := range resp {
		if actions[AppAction.View] {
			hasPerm.Add(appID)
		}
	}
	return hasPerm, nil
}

// ListRoles 列出指定工作空间下的角色信息
func (m *LocalManager) ListRoles(ctx context.Context, workspaceID string) ([]*role.Role, error) {
	scp := role.PermissionScope{ResourceType: role.WorkspaceResourceType, ResourceID: workspaceID}
	roles, err := m.svc.ListRoles(ctx, workspaceID, scp)
	if err != nil {
		return nil, errors.Wrap(err, "list roles")
	}
	return roles, nil
}

// GetRole 获取指定工作空间下指定角色的详细信息
//
// 行为保持稳定：先列举工作空间的角色，再基于 RoleCode 匹配，找不到时返回
// "role not found"。
func (m *LocalManager) GetRole(ctx context.Context, workspaceID, roleCode string) (*role.Role, error) {
	roles, err := m.ListRoles(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "list roles")
	}
	for _, r := range roles {
		if r.RoleCode == roleCode {
			return r, nil
		}
	}
	return nil, errors.New("role not found")
}

// ListRoleMembers 列出指定角色的成员
func (m *LocalManager) ListRoleMembers(ctx context.Context, roleID string) ([]string, error) {
	members, err := m.svc.ListRoleMembers(ctx, roleID)
	if err != nil {
		return nil, errors.Wrap(err, "list role members")
	}
	return members, nil
}

// CreateWorkspaceAdmin 创建工作空间管理员
//
// 字段映射注意事项（保持现有 IAM 数据兼容行为）：
//   - BKCI.ProjectName = bkCIProjectID
//   - BCS.ProjectName  = bkCIProjectID（注意：不是 bcsProjectID，历史兼容 quirk，原因是 bkCIProjectID 更具可读性）
//   - BKRepo.ProjectName = bkRepoProjectID
func (m *LocalManager) CreateWorkspaceAdmin(
	ctx context.Context,
	workspaceID, workspaceDisplayName string,
	users []string,
	bkCIProjectID, bcsProjectID, bkRepoProjectID string,
) error {
	data := bkiam.WorkspaceData{
		WorkspaceID:   workspaceID,
		WorkspaceName: workspaceDisplayName,
		BKCI: &bkiam.BKCIOptions{
			ProjectID:   bkCIProjectID,
			ProjectName: bkCIProjectID,
		},
		BCS: &bkiam.BCSOptions{
			ProjectID: bcsProjectID,
			// 历史兼容 quirk：BCS.ProjectName 取 bkCIProjectID 而非 bcsProjectID。
			ProjectName: bkCIProjectID,
		},
		BKRepo: &bkiam.BKRepoOptions{
			ProjectID:   bkRepoProjectID,
			ProjectName: bkRepoProjectID,
		},
	}
	if _, err := m.svc.CreateWorkspaceAdmin(ctx, data, users); err != nil {
		return errors.Wrap(err, "create workspace admin")
	}
	return nil
}

// CreateWorkspaceScopeBuiltinRoles 创建工作空间内置角色
//
// 字段映射 quirk 同 CreateWorkspaceAdmin。
func (m *LocalManager) CreateWorkspaceScopeBuiltinRoles(
	ctx context.Context,
	workspaceID, workspaceDisplayName string,
	bkCIProjectID, bcsProjectID, bkRepoProjectID string,
) error {
	data := bkiam.WorkspaceData{
		WorkspaceID:   workspaceID,
		WorkspaceName: workspaceDisplayName,
		BKCI: &bkiam.BKCIOptions{
			ProjectID:   bkCIProjectID,
			ProjectName: bkCIProjectID,
		},
		BCS: &bkiam.BCSOptions{
			ProjectID: bcsProjectID,
			// 历史兼容 quirk：BCS.ProjectName 取 bkCIProjectID 而非 bcsProjectID。
			ProjectName: bkCIProjectID,
		},
		BKRepo: &bkiam.BKRepoOptions{
			ProjectID:   bkRepoProjectID,
			ProjectName: bkRepoProjectID,
		},
	}
	if _, err := m.svc.CreateWorkspaceScopeBuiltinRoles(ctx, data); err != nil {
		return errors.Wrap(err, "create workspace builtin roles")
	}
	return nil
}

// AddRoleForUsers 为角色添加用户
func (m *LocalManager) AddRoleForUsers(ctx context.Context, roleID string, users []string) error {
	if err := m.svc.AddRoleForUsers(ctx, roleID, users); err != nil {
		return errors.Wrap(err, "add role for users")
	}
	return nil
}

// DeleteRoleForUsers 为角色删除用户
func (m *LocalManager) DeleteRoleForUsers(ctx context.Context, roleID string, users []string) error {
	if err := m.svc.DeleteRoleForUsers(ctx, roleID, users); err != nil {
		return errors.Wrap(err, "delete role for users")
	}
	return nil
}

// DeleteAllRolesByWorkspaceID 删除工作空间所有角色
func (m *LocalManager) DeleteAllRolesByWorkspaceID(ctx context.Context, workspaceID string) error {
	if err := m.svc.DeleteAllRolesByWorkspaceID(ctx, workspaceID); err != nil {
		return errors.Wrap(err, "delete all roles by workspace id")
	}
	return nil
}

// UpdateWorkspaceAdmin 更新工作空间管理员权限范围
func (m *LocalManager) UpdateWorkspaceAdmin(ctx context.Context, data bkiam.WorkspaceData) error {
	if err := m.svc.UpdateWorkspaceAdmin(ctx, data); err != nil {
		return errors.Wrap(err, "update workspace admin")
	}
	return nil
}

// UpdateWorkspaceScopeBuiltinRoles 更新工作空间内置角色权限范围
func (m *LocalManager) UpdateWorkspaceScopeBuiltinRoles(
	ctx context.Context, data bkiam.WorkspaceData,
) error {
	if err := m.svc.UpdateWorkspaceScopeBuiltinRoles(ctx, data); err != nil {
		return errors.Wrap(err, "update workspace builtin roles")
	}
	return nil
}
