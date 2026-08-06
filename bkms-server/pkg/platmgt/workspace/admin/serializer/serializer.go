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

// Package serializer defines Gin input and output serializers for workspace admin APIs.
package serializer

import "github.com/samber/lo"

// WorkspacePath is the URI input for a single workspace admin API.
type WorkspacePath struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
}

// RoleStatusQuery is the query input for checking whether a user has a workspace role.
type RoleStatusQuery struct {
	// 角色 Code
	RoleCode string `form:"roleCode" binding:"required"`
	// 用户名
	Username string `form:"username" binding:"required"`
}

// RoleStatusOutput is the JSON output for the target user's role status in a workspace.
type RoleStatusOutput struct {
	// 当前目标用户是否拥有指定工作空间角色
	HasRole bool `json:"hasRole"`
}

// GrantAdminInput is the JSON input for granting workspace admin permission.
type GrantAdminInput struct {
	// 是否授予临时管理员。true 表示临时管理员，false 表示永久管理员
	// 使用指针类型以区分字段缺失（nil）和显式传递 false 两种情况；
	// 由于 false 具有明确语义，当前 IsTemporary 字段强制必传， 避免默认值误解
	IsTemporary *bool `json:"isTemporary" binding:"required"`
}

// Temporary returns whether the grant request targets a temporary admin.
func (in GrantAdminInput) Temporary() bool {
	return lo.FromPtr(in.IsTemporary)
}

// NewRoleStatusOutput builds role status output from role state.
func NewRoleStatusOutput(hasRole bool) *RoleStatusOutput {
	return &RoleStatusOutput{HasRole: hasRole}
}

// GetRoleStatusResponse is the JSON response for querying role status.
type GetRoleStatusResponse struct {
	Data *RoleStatusOutput `json:"data"`
}
