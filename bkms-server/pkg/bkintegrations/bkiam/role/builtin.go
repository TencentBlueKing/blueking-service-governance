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

// BuiltinRoleCode 平台内置角色编码
var BuiltinRoleCode = struct {
	Admin     string
	Developer string
	SRE       string
	Operator  string
}{
	Admin:     "admin",
	Developer: "developer",
	SRE:       "sre",
	Operator:  "operator",
}

// WorkspaceScopeBuiltinRoles 工作空间级别的非管理员内置角色。
//
// 不包含 admin：admin 同时承担 IAM 分级管理员角色，
// 由 CreateWorkspaceAdmin 单独创建和维护，
// 避免在普通工作空间内置角色流程中重复创建。
var WorkspaceScopeBuiltinRoles = []string{
	// 开发者
	BuiltinRoleCode.Developer,
	// SRE
	BuiltinRoleCode.SRE,
	// 运营
	BuiltinRoleCode.Operator,
}
