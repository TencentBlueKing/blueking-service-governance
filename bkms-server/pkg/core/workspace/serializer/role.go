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

package serializer

import (
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
)

// -----------------------------------------------------------------------------
// 角色管理
// -----------------------------------------------------------------------------

// WorkspaceRoleKeyURIInput is the path input for APIs operating a single role.
type WorkspaceRoleKeyURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 内置角色为角色 Code（admin / developer / sre / operator），自定义角色为角色 ID
	RoleCode string `uri:"roleCode" binding:"required"`
}

// WorkspaceRoleUserURIInput is the path input for removing a user from a role.
type WorkspaceRoleUserURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 内置角色为角色 Code，自定义角色为角色 ID
	RoleCode string `uri:"roleCode" binding:"required"`
	// 用户 ID
	UserID string `uri:"userID" binding:"required,uri_slug"`
}

// AppScopeInput is the app scope of a custom role.
type AppScopeInput struct {
	// 范围类型：all（全部应用，包括之后新建的应用）、apps（指定应用）
	Mode string `json:"mode" binding:"required,oneof=all apps"`
	// 指定的应用 ID，mode 为 apps 时必填
	AppIDs []string `json:"appIDs"`
}

// EnvScopeInput is the env scope of a custom role.
type EnvScopeInput struct {
	// 范围类型：all（全部环境）、types（按环境类型）、envs（指定标准环境）
	Mode string `json:"mode" binding:"required,oneof=all types envs"`
	// 环境类型，mode 为 types 时必填，可选 development、test、staging、production
	EnvTypes []string `json:"envTypes"`
	// 标准环境 ID，mode 为 envs 时必填
	EnvIDs []string `json:"envIDs"`
}

// ToModel converts the input into the role app scope.
func (i *AppScopeInput) ToModel() role.AppScope {
	if i == nil {
		return role.AppScope{Mode: role.AppScopeAll}
	}
	return role.AppScope{Mode: i.Mode, AppIDs: lo.Uniq(i.AppIDs)}
}

// ToModel converts the input into the role env scope.
func (i *EnvScopeInput) ToModel() role.EnvScope {
	if i == nil {
		return role.EnvScope{Mode: role.EnvScopeAll}
	}
	return role.EnvScope{Mode: i.Mode, EnvTypes: lo.Uniq(i.EnvTypes), EnvIDs: lo.Uniq(i.EnvIDs)}
}

// CreateWorkspaceRoleInput is the JSON body for creating a custom role.
type CreateWorkspaceRoleInput struct {
	// 角色名称，空间内唯一；克隆时为空则默认「原名 副本」
	Name string `json:"name" binding:"max=64"`
	// 角色描述
	Description string `json:"description" binding:"max=256"`
	// 克隆来源，内置角色为角色 Code，自定义角色为角色 ID；指定时复制其权限与范围，不复制成员，忽略下列字段
	CloneFrom string `json:"cloneFrom"`
	// 权限项，见 GET /workspaces/{workspaceID}/roles 返回的 permissions；为空表示空白角色
	Permissions []string `json:"permissions"`
	// 应用范围，为空表示全部应用
	AppScope *AppScopeInput `json:"appScope"`
	// 环境范围，为空表示全部环境
	EnvScope *EnvScopeInput `json:"envScope"`
	// 初始成员的用户名；克隆时也生效
	Members []string `json:"members" binding:"dive,required"`
}

// UpdateWorkspaceRoleInput is the JSON body for updating a custom role.
type UpdateWorkspaceRoleInput struct {
	// 角色名称，空间内唯一
	Name string `json:"name" binding:"required,max=64"`
	// 角色描述
	Description string `json:"description" binding:"max=256"`
	// 权限项
	Permissions []string `json:"permissions"`
	// 应用范围
	AppScope *AppScopeInput `json:"appScope" binding:"required"`
	// 环境范围
	EnvScope *EnvScopeInput `json:"envScope" binding:"required"`
}

// RoleResourceOutputObj is an app or env in the scope of a role.
type RoleResourceOutputObj struct {
	// 资源 ID
	ID string `json:"id"`
	// 展示名称
	Name string `json:"name"`
	// 环境类型，仅环境有值
	Type string `json:"type,omitempty"`
}

// RoleAppScopeOutputObj is the app scope of a role.
type RoleAppScopeOutputObj struct {
	// 范围类型：all、apps
	Mode string `json:"mode"`
	// 指定的应用，已删除的应用不再返回
	Apps []*RoleResourceOutputObj `json:"apps"`
}

// RoleEnvScopeOutputObj is the env scope of a role.
type RoleEnvScopeOutputObj struct {
	// 范围类型：all、types、envs
	Mode string `json:"mode"`
	// 环境类型
	EnvTypes []string `json:"envTypes"`
	// 指定的标准环境，已删除的环境不再返回
	Envs []*RoleResourceOutputObj `json:"envs"`
}

// RoleOutputObj is the JSON representation of a workspace role.
type RoleOutputObj struct {
	// 角色 ID
	ID string `json:"id"`
	// 角色 Code，内置角色为 admin / developer / sre / operator，自定义角色为 custom
	RoleCode string `json:"roleCode"`
	// 角色名称
	Name string `json:"name"`
	// 角色描述
	Description string `json:"description"`
	// 是否为内置角色；内置角色只能管理成员和克隆
	Builtin bool `json:"builtin"`
	// 权限项
	Permissions []string `json:"permissions"`
	// 应用范围
	AppScope *RoleAppScopeOutputObj `json:"appScope"`
	// 环境范围
	EnvScope *RoleEnvScopeOutputObj `json:"envScope"`
	// 成员数量
	MemberCount int `json:"memberCount"`
	// 成员的用户名
	Members []string `json:"members"`
}

// ListWorkspaceRolesOutput is the JSON response for listing workspace roles.
type ListWorkspaceRolesOutput struct {
	Data []*RoleOutputObj `json:"data"`
	// 全部可选的权限项，按展示顺序排列
	Permissions []string `json:"permissions"`
}

// WorkspaceRoleOutput is the JSON response for a single workspace role.
type WorkspaceRoleOutput struct {
	Data *RoleOutputObj `json:"data"`
}

// ListWorkspaceRoleUsersOutput is the JSON response for listing role members.
type ListWorkspaceRoleUsersOutput struct {
	Data []string `json:"data"`
}

// -----------------------------------------------------------------------------
// 成员视角与可部署的人
// -----------------------------------------------------------------------------

// ListWorkspaceUsersQueryInput is the query input for listing workspace users.
type ListWorkspaceUsersQueryInput struct {
	// 只返回该角色的成员；内置角色为角色 Code，自定义角色为角色 ID，为空返回全部成员
	RoleCode string `form:"roleCode"`
}

// UserRoleOutputObj is a role held by a user.
type UserRoleOutputObj struct {
	// 角色 ID
	ID string `json:"id"`
	// 角色 Code，自定义角色为 custom
	RoleCode string `json:"roleCode"`
	// 角色名称
	Name string `json:"name"`
	// 是否为内置角色
	Builtin bool `json:"builtin"`
}

// MergedAppScopeOutputObj is the app scope merged from multiple roles, for display only.
type MergedAppScopeOutputObj struct {
	// 是否覆盖全部应用
	All bool `json:"all"`
	// 指定的应用，All 为 true 时为空
	Apps []*RoleResourceOutputObj `json:"apps"`
}

// MergedEnvScopeOutputObj is the env scope merged from multiple roles, for display only.
type MergedEnvScopeOutputObj struct {
	// 是否覆盖全部环境
	All bool `json:"all"`
	// 按环境类型覆盖的类型
	EnvTypes []string `json:"envTypes"`
	// 指定的标准环境
	Envs []*RoleResourceOutputObj `json:"envs"`
}

// WorkspaceUserOutputObj is the JSON representation of a workspace member.
type WorkspaceUserOutputObj struct {
	// 用户名
	Username string `json:"username"`
	// 用户在空间内的全部角色
	Roles []*UserRoleOutputObj `json:"roles"`
	// 各角色应用范围的合并结果，仅用于展示；实际权限按单个角色判断，不会跨角色拼接
	AppScope *MergedAppScopeOutputObj `json:"appScope"`
	// 各角色环境范围的合并结果，仅用于展示
	EnvScope *MergedEnvScopeOutputObj `json:"envScope"`
	// 是否有角色带部署到环境的权限
	CanDeploy bool `json:"canDeploy"`
}

// ListWorkspaceUsersOutput is the JSON response for listing workspace users.
type ListWorkspaceUsersOutput struct {
	Data []*WorkspaceUserOutputObj `json:"data"`
}

// EnvDeployerOutputObj is a user who can deploy to an env.
type EnvDeployerOutputObj struct {
	// 用户名
	Username string `json:"username"`
	// 赋予该用户部署这个环境权限的角色
	Roles []*UserRoleOutputObj `json:"roles"`
	// 可以部署全部应用
	AllApps bool `json:"allApps"`
	// AllApps 为 false 时，可以部署的应用
	Apps []*RoleResourceOutputObj `json:"apps"`
}

// EnvDeployersOutputObj is the deployers of a standard env.
type EnvDeployersOutputObj struct {
	// 环境 ID
	EnvID string `json:"envID"`
	// 环境名称
	EnvName string `json:"envName"`
	// 能部署该环境的人，为空表示无人可部署
	Deployers []*EnvDeployerOutputObj `json:"deployers"`
}

// ListWorkspaceEnvDeployersOutput is the JSON response for listing env deployers.
type ListWorkspaceEnvDeployersOutput struct {
	Data []*EnvDeployersOutputObj `json:"data"`
}
