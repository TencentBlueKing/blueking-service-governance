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

package dal

// Scope 修改一张表这次访问的租户范围。
type Scope func(Table) Table

// ForTenant 指定这次访问使用的租户，不再读 ctx。
func ForTenant(id string) Scope {
	return func(table Table) Table {
		return table.ForTenant(id)
	}
}

// AllTenants 这次访问不限定租户，不注入 tenant_id。
func AllTenants() Scope {
	return func(table Table) Table {
		return table.AllTenants()
	}
}

// FromContext 这次访问从 ctx 读租户，与不调用 Scope 时的默认行为相同。
func FromContext() Scope {
	return func(table Table) Table {
		return table.FromContext()
	}
}

// ApplyScope 返回应用了 scope 的新 Table，供 Store 实现 Scope 时调用。
// scope 为 nil，或 Table 已经设置过范围时，不在这里报错，错误会在后续操作返回。
func ApplyScope(table Table, scope Scope) Table {
	if scope == nil {
		return withScopeErr(table, ErrScopeRequired)
	}
	return scope(table)
}

func withScopeErr(table Table, err error) Table {
	mt, ok := table.(*mongoTable)
	if !ok || mt == nil {
		return table
	}
	cloned := *mt
	if cloned.err == nil {
		cloned.err = err
	}
	return &cloned
}
