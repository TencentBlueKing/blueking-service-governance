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

// Package template embeds JSON role-scope templates and provides helpers
// to resolve the template path by business system + builtin role code.
package template

import (
	"fmt"
	"path/filepath"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
)

// scopeTemplates 角色权限范围模板
var scopeTemplates = map[string]string{
	role.BuiltinRoleCode.Admin:     fmt.Sprintf("%s.json", role.BuiltinRoleCode.Admin),
	role.BuiltinRoleCode.Developer: fmt.Sprintf("%s.json", role.BuiltinRoleCode.Developer),
	role.BuiltinRoleCode.SRE:       fmt.Sprintf("%s.json", role.BuiltinRoleCode.SRE),
	role.BuiltinRoleCode.Operator:  fmt.Sprintf("%s.json", role.BuiltinRoleCode.Operator),
}

// GetRoleScopeTemplatePath 获取角色权限范围模板路径
func GetRoleScopeTemplatePath(systemFileName, roleCode string) string {
	tpl, ok := scopeTemplates[roleCode]
	if !ok {
		return "anonymous.json"
	}

	return filepath.Join(systemFileName, tpl)
}
