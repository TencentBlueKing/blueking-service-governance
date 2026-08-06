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

package admin

import (
	"errors"

	"github.com/samber/lo"
)

// ErrInvalidRoleCode indicates the target platform role code is unsupported.
var ErrInvalidRoleCode = errors.New("invalid platform role code")

// RoleCode identifies a platform management role.
type RoleCode string

const (
	// RoleCodeAdmin grants full platform management access.
	RoleCodeAdmin RoleCode = "admin"
)

// RoleInfo describes an available platform administrator role.
type RoleInfo struct {
	RoleCode RoleCode
	Name     string
}

// supportedRoles defines all supported platform administrator roles.
var supportedRoles = []RoleInfo{
	{
		RoleCode: RoleCodeAdmin,
		Name:     "平台管理员",
	},
}

// Roles returns all supported platform administrator roles.
func Roles() []RoleInfo {
	return supportedRoles
}

// isValidRoleCode reports whether the given role code is supported.
func isValidRoleCode(roleCode RoleCode) bool {
	return lo.ContainsBy(supportedRoles, func(role RoleInfo) bool {
		return role.RoleCode == roleCode
	})
}
