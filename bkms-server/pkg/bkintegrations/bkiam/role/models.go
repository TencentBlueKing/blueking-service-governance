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

package role

// ResourceType 资源类型
type ResourceType string

const WorkspaceResourceType ResourceType = "workspace"

// PermissionScope is the permission scope of the role
type PermissionScope struct {
	// ResourceType 资源类型
	ResourceType ResourceType `bson:"resourceType"`
	// ResourceID 资源 ID
	ResourceID string `bson:"resourceID"`
}

// Role is the user role with permission
type Role struct {
	ID   string `bson:"id"`
	Name string `bson:"name"`
	// RoleCode 角色码. 目前的内置取值有: admin(管理员), developer(开发者), sre, operator(运维)
	RoleCode    string `bson:"roleCode"`
	Description string `bson:"description"`
	// WorkspaceID 角色所属的 workspace id
	WorkspaceID string `bson:"workspaceID"`
	// IsGradeManager 是否是当前 workspace 的分级管理员角色
	IsGradeManager bool `bson:"isGradeManager"`
	// Scope 角色的权限范围
	Scope PermissionScope `bson:"scope"`
	// UserGroupID 角色对应的权限中心的用户组 id
	UserGroupID int `bson:"userGroupID"`
}

// WorkspaceGradeManager workspace 分级管理员
type WorkspaceGradeManager struct {
	WorkspaceID    string `bson:"workspaceID"`
	GradeManagerID int    `bson:"gradeManagerID"`
}

// RoleQueryParams is the query params of role
type RoleQueryParams struct {
	WorkspaceID    *string
	IsGradeManager *bool
	Scope          *PermissionScope
}
