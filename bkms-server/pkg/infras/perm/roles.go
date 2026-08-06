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

const (
	// 对应蓝鲸 IAM 中的 RoleCode 取值

	// RoleCodeAdmin 管理员
	RoleCodeAdmin = "admin"
	// RoleCodeSre SRE
	RoleCodeSre = "sre"
	// RoleCodeDeveloper 开发者
	RoleCodeDeveloper = "developer"
	// RoleCodeOperator 运营者
	RoleCodeOperator = "operator"
)

// RoleCodes return all valid role codes
func RoleCodes() []string {
	return []string{
		RoleCodeAdmin,
		RoleCodeSre,
		RoleCodeDeveloper,
		RoleCodeOperator,
	}
}
