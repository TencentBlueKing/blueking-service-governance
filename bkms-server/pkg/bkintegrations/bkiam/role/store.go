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

// Package role provides IAM role domain types and Mongo storage for bkms-server.
//
// This package defines the role-related domain models (Role, WorkspaceGradeManager,
// PermissionScope, BuiltinRoleCode) and their GenericStore metadata.
package role

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// RoleStore 角色存储
type RoleStore = store.GenericStore[Role]

// GradeManagerStore workspace 分级管理员存储
type GradeManagerStore = store.GenericStore[WorkspaceGradeManager]

// RoleMeta 角色表的存储元信息。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id
// - 查询提速：workspaceID + userGroupID
var RoleMeta = store.Meta[Role]{Collection: "iam_roles"}

// GradeManagerMeta workspace 分级管理员表的存储元信息。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID
var GradeManagerMeta = store.Meta[WorkspaceGradeManager]{Collection: "iam_grade_managers"}

// ByRoleID 按角色 ID 匹配角色
func ByRoleID(roleID string) store.Opt {
	return store.Where("id", roleID)
}

// ByWorkspace 按 workspace 匹配角色或分级管理员
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByGradeManager 按是否为分级管理员角色匹配
func ByGradeManager(isGradeManager bool) store.Opt {
	return store.Where("isGradeManager", isGradeManager)
}

// ByScope 按角色的权限范围匹配
func ByScope(scope PermissionScope) store.Opt {
	return store.Filter(bson.M{"scope.resourceType": scope.ResourceType, "scope.resourceID": scope.ResourceID})
}

// ByUserGroupIDs 匹配某个 workspace 下对应指定权限中心用户组的角色
func ByUserGroupIDs(workspaceID string, userGroupIDs []int) store.Opt {
	// nil 切片会被编码为 null，$in 要求数组
	if userGroupIDs == nil {
		userGroupIDs = []int{}
	}
	return store.Combine(ByWorkspace(workspaceID), store.Where("userGroupID", bson.M{"$in": userGroupIDs}))
}
