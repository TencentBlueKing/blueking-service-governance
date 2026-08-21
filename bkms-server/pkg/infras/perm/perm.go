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

// Package perm 提供 bkms-server 业务侧的权限管理入口，衔接 bkiam 角色与 IAM 云 API
package perm

import (
	"context"
	"sync"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	cloudapiiam "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

// Manager 权限管理器接口
//
// Manager 是 bkms-server 业务侧的权限管理器入口（v2 风格）。方法签名中的
// 领域类型均取自 pkg/bkintegrations/bkiam 的纯 Go DTO。
//
// 当前两种实现：
//   - LocalManager：进程内本地实现，转发到 bkiam.IAMService -> cloudapi/iam
//     -> 蓝鲸 IAM 网关
//   - StubAllowAnyManager：本地开发桩，全部放行
type Manager interface {
	// HasCreateAppPerm 检查用户是否有创建应用的权限
	HasCreateAppPerm(ctx context.Context, workspaceID string) error
	// HasEditAppPerm 检查用户是否有编辑应用的权限
	HasEditAppPerm(ctx context.Context, workspaceID, appID string) error
	// HasViewAppPerm 检查用户是否有查看应用的权限
	HasViewAppPerm(ctx context.Context, workspaceID, appID string) error
	// HasDeleteAppPerm 检查用户是否有删除应用的权限
	HasDeleteAppPerm(ctx context.Context, workspaceID, appID string) error

	// HasCreateWorkspacePerm 检查用户是否有创建工作空间的权限
	HasCreateWorkspacePerm(ctx context.Context) error
	// HasViewWorkspacePerm 检查用户是否有查看工作空间的权限
	HasViewWorkspacePerm(ctx context.Context, workspaceID string) error
	// HasEditWorkspacePerm 检查用户是否有编辑工作空间的权限
	HasEditWorkspacePerm(ctx context.Context, workspaceID string) error
	// HasDeleteWorkspacePerm 检查用户是否有删除工作空间的权限
	HasDeleteWorkspacePerm(ctx context.Context, workspaceID string) error

	// HasCreateEnvPerm 检查用户是否有创建环境的权限
	HasCreateEnvPerm(ctx context.Context, workspaceID string) error
	// HasEditEnvPerm 检查用户是否有编辑环境的权限
	HasEditEnvPerm(ctx context.Context, workspaceID, envName string) error
	// HasViewEnvPerm 检查用户是否有查看环境的权限
	HasViewEnvPerm(ctx context.Context, workspaceID, envName string) error
	// HasDeleteEnvPerm 检查用户是否有删除环境的权限
	HasDeleteEnvPerm(ctx context.Context, workspaceID, envName string) error
	// HasDeployEnvPerm 检查用户是否有部署环境的权限
	HasDeployEnvPerm(ctx context.Context, workspaceID, envName string) error

	// FilterViewableWorkspaces 过滤出用户有权限查看的工作空间列表
	FilterViewableWorkspaces(ctx context.Context, workspaceIDs []string) (*set.StringSet, error)
	// FilterViewableApps 过滤出用户有权限查看的应用列表
	FilterViewableApps(ctx context.Context, workspaceID string, appIDs []string) (*set.StringSet, error)

	// ListRoles 获取角色列表
	ListRoles(ctx context.Context, workspaceID string) ([]*role.Role, error)
	// ListRoleMembers 获取角色成员列表
	ListRoleMembers(ctx context.Context, roleID string) ([]string, error)
	// GetRole 获取角色信息
	GetRole(ctx context.Context, workspaceID, roleCode string) (*role.Role, error)

	// CreateWorkspaceAdmin 创建工作空间管理员
	CreateWorkspaceAdmin(
		ctx context.Context,
		workspaceID, workspaceDisplayName string,
		users []string,
		bkCIProjectID, bcsProjectID, bkRepoProjectID string,
	) error
	// CreateWorkspaceScopeBuiltinRoles 创建工作空间内置角色
	CreateWorkspaceScopeBuiltinRoles(
		ctx context.Context,
		workspaceID, workspaceDisplayName string,
		bkCIProjectID, bcsProjectID, bkRepoProjectID string,
	) error
	// AddRoleForUsers 为角色用户组添加用户, roleID 通常为工作空间下的角色 ID
	AddRoleForUsers(ctx context.Context, roleID string, users []string) error
	// DeleteRoleForUsers 从角色用户组删除用户, roleID 通常为工作空间下的角色 ID
	DeleteRoleForUsers(ctx context.Context, roleID string, users []string) error

	// DeleteAllRolesByWorkspaceID 删除工作空间所有角色
	DeleteAllRolesByWorkspaceID(ctx context.Context, workspaceID string) error

	// UpdateWorkspaceAdmin 更新工作空间管理员权限范围
	UpdateWorkspaceAdmin(ctx context.Context, data bkiam.WorkspaceData) error
	// UpdateWorkspaceScopeBuiltinRoles 更新工作空间内置角色权限范围
	UpdateWorkspaceScopeBuiltinRoles(ctx context.Context, data bkiam.WorkspaceData) error
}

// NewManager 创建权限管理器实现。
//
// NewManager 使用懒加载单例，确保进程内权限管理器及其底层 IAM client、
// Mongo 角色存储和 IAMService 只初始化一次，避免在请求路径重复创建依赖。
//
// 内部分支顺序：
//  1. config.G.Development.UseStubPerm == true：返回 StubAllowAnyManager（仅
//     用于本地开发，绕过所有权限检查，**禁止用于生产环境**）；
//  2. 否则：依次构造 cloudapi/iam.IAMClient + role.RoleStore + iam.IAMService，
//     注入到 LocalManager 后返回。
//
// 如果底层依赖构造失败（如蓝鲸 IAM 网关 client 创建失败、Mongo 索引创建
// 失败），将通过 log.Fatal 终止启动，与项目其他基础设施初始化保持一致。
func NewManager() Manager {
	managerOnce.Do(func() {
		manager, managerErr = buildManager()
	})
	if managerErr != nil {
		log.Fatalf("perm: build manager: %v", managerErr)
	}
	return manager
}

var (
	managerOnce sync.Once
	manager     Manager
	managerErr  error
)

func buildManager() (Manager, error) {
	if config.G.Development.UseStubPerm {
		log.WarnNoContext("Using the stub perm manager for permission checking, " +
			"ENSURE THIS IS A DEVELOPMENT ENVIRONMENT!",
		)
		return &StubAllowAnyManager{}, nil
	}

	svc, err := newIAMService()
	if err != nil {
		return nil, errors.Wrap(err, "build iam service")
	}
	return &LocalManager{svc: svc}, nil
}

// newIAMService 装配 LocalManager 内部依赖：cloudapi/iam.IAMClient +
// role.RoleStoreMongo（复用全局 mongo client）+ iam.IAMService。
func newIAMService() (*bkiam.IAMService, error) {
	cli, err := cloudapiiam.NewIAMClient()
	if err != nil {
		return nil, errors.Wrap(err, "new iam client")
	}
	store, err := role.NewRoleStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "new role store")
	}
	return bkiam.NewIAMService(cli, store), nil
}
