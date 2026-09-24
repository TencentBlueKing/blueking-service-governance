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

package tenant

import (
	"errors"
)

const (
	// HeaderTenantID 多租户请求头，对齐 ConfigCenter TenantHeader。
	HeaderTenantID = "X-Bk-Tenant-Id"

	// DefaultTenantID 单租兼容模式下的保留租户，对齐 ConfigCenter BKSingleTenantID。
	DefaultTenantID = "default"
	// SystemTenantID 平台/系统保留租户，对齐 ConfigCenter BKDefaultTenantID。
	SystemTenantID = "system"
	// SuperTenantID 超级租户保留值，对齐 ConfigCenter BKSuperTenantID。
	SuperTenantID = "superadmin"
)

var (
	// ErrTenantIDRequired indicates multi-tenant mode requires an explicit tenant header.
	ErrTenantIDRequired = errors.New("tenant id is required")
	// ErrTenantIDInvalid indicates the incoming tenant value violates local mode rules.
	ErrTenantIDInvalid = errors.New("tenant id is invalid")
	// ErrTenantAccessDenied indicates the authenticated user cannot access the requested tenant.
	ErrTenantAccessDenied = errors.New("tenant access denied")
	// ErrTenantUserDisabled indicates the bk-user account exists but is not enabled.
	ErrTenantUserDisabled = errors.New("tenant user is disabled")
)
