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
	"encoding/json"
	"sync"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// StubAllowAnyManager 用于本地开发与测试的权限管理器实现，所有权限检查
// 全部放行；角色管理写操作会更新进程内的成员集合，以便 e2e/集成测试能够
// 观察到授权后的读取结果。
//
// **禁止用于生产环境**。
type StubAllowAnyManager struct {
	mu          sync.Mutex
	roleMembers map[string][]string
}

// 编译期检查
var _ Manager = (*StubAllowAnyManager)(nil)

// 4 个固定角色 ID，便于历史调用方在 ID 维度做兼容比对。
const (
	stubAdminRoleID     = "3131869c-8af4-49df-8910-bc2e69ee5e17"
	stubDeveloperRoleID = "c31b1be5-89e7-4890-9f25-cff00aa5d4c9"
	stubSRERoleID       = "1b857531-1af6-4e69-b056-653c6f3c38af"
	stubOperatorRoleID  = "59d7d1be-89a1-410b-b9b1-5ed92dea1054"
)

// stub 角色固定信息
const (
	stubWorkspaceID         = "blueking"
	stubAdminUserGroupID    = 10001
	stubDevUserGroupID      = 10002
	stubSREUserGroupID      = 10003
	stubOperatorUserGroupID = 10004
	stubRoleNameSuffix      = "-name"
	stubRoleDescSuffix      = " role description"
)

// HasCreateAppPerm 总是允许创建应用
func (m *StubAllowAnyManager) HasCreateAppPerm(_ context.Context, _ string) error {
	return nil
}

// HasEditAppPerm 总是允许编辑应用
func (m *StubAllowAnyManager) HasEditAppPerm(_ context.Context, _, _ string) error {
	return nil
}

// HasViewAppPerm 总是允许查看应用
func (m *StubAllowAnyManager) HasViewAppPerm(_ context.Context, _, _ string) error {
	return nil
}

// HasDeleteAppPerm 总是允许删除应用
func (m *StubAllowAnyManager) HasDeleteAppPerm(_ context.Context, _, _ string) error {
	return nil
}

// HasCreateWorkspacePerm 总是允许创建工作空间
func (m *StubAllowAnyManager) HasCreateWorkspacePerm(_ context.Context) error {
	return nil
}

// HasViewWorkspacePerm 总是允许查看工作空间
func (m *StubAllowAnyManager) HasViewWorkspacePerm(_ context.Context, _ string) error {
	return nil
}

// HasEditWorkspacePerm 总是允许编辑工作空间
func (m *StubAllowAnyManager) HasEditWorkspacePerm(_ context.Context, _ string) error {
	return nil
}

// HasDeleteWorkspacePerm 总是允许删除工作空间
func (m *StubAllowAnyManager) HasDeleteWorkspacePerm(_ context.Context, _ string) error {
	return nil
}

// HasCreateEnvPerm 总是允许创建环境
func (m *StubAllowAnyManager) HasCreateEnvPerm(_ context.Context, _ string) error {
	return nil
}

// HasEditEnvPerm 总是允许编辑环境
func (m *StubAllowAnyManager) HasEditEnvPerm(_ context.Context, _, _ string) error {
	return nil
}

// HasViewEnvPerm 总是允许查看环境
func (m *StubAllowAnyManager) HasViewEnvPerm(_ context.Context, _, _ string) error {
	return nil
}

// HasDeleteEnvPerm 总是允许删除环境
func (m *StubAllowAnyManager) HasDeleteEnvPerm(_ context.Context, _, _ string) error {
	return nil
}

// HasDeployEnvPerm 总是允许部署环境
func (m *StubAllowAnyManager) HasDeployEnvPerm(_ context.Context, _, _ string) error {
	return nil
}

// FilterViewableWorkspaces 过滤出用户有权限查看的工作空间列表（测试实现：允许所有）
func (m *StubAllowAnyManager) FilterViewableWorkspaces(
	_ context.Context, workspaceIDs []string,
) (*set.StringSet, error) {
	return set.NewStringSetWithValues(workspaceIDs), nil
}

// FilterViewableApps 过滤出用户有权限查看的应用列表（测试实现：允许所有）
func (m *StubAllowAnyManager) FilterViewableApps(
	_ context.Context, _ string, appIDs []string,
) (*set.StringSet, error) {
	return set.NewStringSetWithValues(appIDs), nil
}

// ListRoles 返回预设角色信息
func (m *StubAllowAnyManager) ListRoles(_ context.Context, workspaceID string) ([]*role.Role, error) {
	return []*role.Role{
		{
			ID:             stubAdminRoleID,
			Name:           RoleCodeAdmin + stubRoleNameSuffix,
			RoleCode:       RoleCodeAdmin,
			Description:    "admin" + stubRoleDescSuffix,
			WorkspaceID:    stubWorkspaceID,
			IsGradeManager: true,
			Scope: role.PermissionScope{
				ResourceType: role.WorkspaceResourceType,
				ResourceID:   workspaceID,
			},
			UserGroupID: stubAdminUserGroupID,
		},
		{
			ID:             stubDeveloperRoleID,
			Name:           RoleCodeDeveloper + stubRoleNameSuffix,
			RoleCode:       RoleCodeDeveloper,
			Description:    "developer" + stubRoleDescSuffix,
			WorkspaceID:    stubWorkspaceID,
			IsGradeManager: false,
			Scope: role.PermissionScope{
				ResourceType: role.WorkspaceResourceType,
				ResourceID:   workspaceID,
			},
			UserGroupID: stubDevUserGroupID,
		},
		{
			ID:             stubSRERoleID,
			Name:           RoleCodeSre + stubRoleNameSuffix,
			RoleCode:       RoleCodeSre,
			Description:    "sre" + stubRoleDescSuffix,
			WorkspaceID:    stubWorkspaceID,
			IsGradeManager: false,
			Scope: role.PermissionScope{
				ResourceType: role.WorkspaceResourceType,
				ResourceID:   workspaceID,
			},
			UserGroupID: stubSREUserGroupID,
		},
		{
			ID:             stubOperatorRoleID,
			Name:           RoleCodeOperator + stubRoleNameSuffix,
			RoleCode:       RoleCodeOperator,
			Description:    "operator" + stubRoleDescSuffix,
			WorkspaceID:    stubWorkspaceID,
			IsGradeManager: false,
			Scope: role.PermissionScope{
				ResourceType: role.WorkspaceResourceType,
				ResourceID:   workspaceID,
			},
			UserGroupID: stubOperatorUserGroupID,
		},
	}, nil
}

// GetRole 返回预设角色信息
func (m *StubAllowAnyManager) GetRole(
	_ context.Context, workspaceID, roleCode string,
) (*role.Role, error) {
	var stubRoleID string
	switch roleCode {
	case RoleCodeAdmin:
		stubRoleID = stubAdminRoleID
	case RoleCodeDeveloper:
		stubRoleID = stubDeveloperRoleID
	case RoleCodeSre:
		stubRoleID = stubSRERoleID
	case RoleCodeOperator:
		stubRoleID = stubOperatorRoleID
	}
	return &role.Role{
		ID:             stubRoleID,
		Name:           roleCode + stubRoleNameSuffix,
		RoleCode:       roleCode,
		Description:    roleCode + stubRoleDescSuffix,
		WorkspaceID:    stubWorkspaceID,
		IsGradeManager: true,
		Scope: role.PermissionScope{
			ResourceType: role.WorkspaceResourceType,
			ResourceID:   workspaceID,
		},
		UserGroupID: stubAdminUserGroupID,
	}, nil
}

// ListRoleMembers 列出指定角色的成员
func (m *StubAllowAnyManager) ListRoleMembers(_ context.Context, roleID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	baseMembers := defaultStubRoleMembers(roleID)
	dynamicMembers := m.roleMembers[roleID]
	members := lo.Union(baseMembers, dynamicMembers)
	if len(members) == 0 {
		return nil, nil
	}
	return members, nil
}

func defaultStubRoleMembers(roleID string) []string {
	switch roleID {
	case stubAdminRoleID:
		return []string{"admin", "blueking"}
	case stubDeveloperRoleID:
		return []string{"developer"}
	case stubSRERoleID:
		return []string{"sre"}
	case stubOperatorRoleID:
		return []string{"operator"}
	}
	return nil
}

// CreateWorkspaceAdmin stub 函数不做实际创建动作
func (m *StubAllowAnyManager) CreateWorkspaceAdmin(
	_ context.Context,
	_, _ string,
	_ []string,
	_, _, _ string,
) error {
	return nil
}

// CreateWorkspaceScopeBuiltinRoles stub 函数不做实际创建动作
func (m *StubAllowAnyManager) CreateWorkspaceScopeBuiltinRoles(
	_ context.Context,
	_, _ string,
	_, _, _ string,
) error {
	return nil
}

// AddRoleForUsers 将用户添加到进程内的角色成员集合。
func (m *StubAllowAnyManager) AddRoleForUsers(_ context.Context, roleID string, users []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.roleMembers == nil {
		m.roleMembers = make(map[string][]string)
	}
	m.roleMembers[roleID] = lo.Union(m.roleMembers[roleID], users)
	return nil
}

// DeleteRoleForUsers 从进程内的角色成员集合中移除用户。
func (m *StubAllowAnyManager) DeleteRoleForUsers(_ context.Context, roleID string, users []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.roleMembers[roleID]) == 0 {
		return nil
	}
	filtered := lo.Without(m.roleMembers[roleID], users...)
	if len(filtered) == 0 {
		delete(m.roleMembers, roleID)
		return nil
	}
	m.roleMembers[roleID] = filtered
	return nil
}

// DeleteAllRolesByWorkspaceID stub 函数不做实际删除动作
func (m *StubAllowAnyManager) DeleteAllRolesByWorkspaceID(_ context.Context, _ string) error {
	return nil
}

// UpdateWorkspaceAdmin stub 函数仅打日志，不做实际更新动作
func (m *StubAllowAnyManager) UpdateWorkspaceAdmin(ctx context.Context, data bkiam.WorkspaceData) error {
	req, _ := json.Marshal(data)
	log.Debugf(ctx, "Stub: UpdateWorkspaceAdmin: %s", string(req))
	return nil
}

// UpdateWorkspaceScopeBuiltinRoles stub 函数仅打日志，不做实际更新动作
func (m *StubAllowAnyManager) UpdateWorkspaceScopeBuiltinRoles(
	ctx context.Context, data bkiam.WorkspaceData,
) error {
	req, _ := json.Marshal(data)
	log.Debugf(ctx, "Stub: UpdateWorkspaceScopeBuiltinRoles: %s", string(req))
	return nil
}
