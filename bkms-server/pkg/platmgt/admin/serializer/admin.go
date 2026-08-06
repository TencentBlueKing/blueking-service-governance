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

// Package serializer defines Gin input and output serializers for platform administrator APIs.
package serializer

import (
	"time"

	platmgtadmin "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/admin"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// RoleBindingOutput is the JSON representation of a platform administrator role binding.
type RoleBindingOutput struct {
	// 平台管理员用户名
	Username string `json:"username"`
	// 平台管理员角色编码
	RoleCode platmgtadmin.RoleCode `json:"roleCode"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 添加人
	Creator string `json:"creator"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
	// 更新人
	Updater string `json:"updater"`
}

// NewRoleBindingOutput builds a platform administrator role binding response object from model.
func NewRoleBindingOutput(roleBinding platmgtadmin.RoleBinding) *RoleBindingOutput {
	return &RoleBindingOutput{
		Username:  roleBinding.Username,
		RoleCode:  roleBinding.RoleCode,
		CreatedAt: roleBinding.CreatedAt,
		Creator:   roleBinding.Creator,
		UpdatedAt: roleBinding.UpdatedAt,
		Updater:   roleBinding.Updater,
	}
}

// RoleBindingPath is the path input for platform administrator APIs.
type RoleBindingPath struct {
	// 平台管理员用户名
	Username string `uri:"username" binding:"required,max=128,uri_slug"`
}

// AssignRolesInput is the JSON body for assigning platform administrator roles in batch.
type AssignRolesInput struct {
	// 平台管理员用户名列表
	Usernames []string `json:"usernames" binding:"required,min=1,dive,required,max=128,uri_slug"`
	// 平台管理员角色编码
	RoleCode platmgtadmin.RoleCode `json:"roleCode" binding:"required"`
}

// ListRoleBindingsQuery is the query input for listing platform administrator role bindings.
type ListRoleBindingsQuery struct {
	// 用户名关键字，大小写不敏感模糊匹配
	Keyword string `form:"keyword"`
}

// ListRoleBindingsResponse is the JSON response for listing platform administrator role bindings.
type ListRoleBindingsResponse struct {
	Data []*RoleBindingOutput `json:"data"`
}

// RoleOutput is the JSON representation of an available platform administrator role.
type RoleOutput struct {
	// 角色编码
	RoleCode platmgtadmin.RoleCode `json:"roleCode"`
	// 角色名称
	Name string `json:"name"`
}

// NewRoleOutput builds a role response object from model.
func NewRoleOutput(roleInfo platmgtadmin.RoleInfo) *RoleOutput {
	return &RoleOutput{
		RoleCode: roleInfo.RoleCode,
		Name:     roleInfo.Name,
	}
}

// ListRolesResponse is the JSON response for listing available platform administrator roles.
type ListRolesResponse struct {
	Data []*RoleOutput `json:"data"`
}
