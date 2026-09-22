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

// ValidateTenantMode 验证租户模式
func ValidateTenantMode(tenantID string, enableMultiTenantMode bool) (string, error) {
	// 未开启多租户时仅允许租户为 "" 或 default
	if !enableMultiTenantMode {
		switch tenantID {
		case "", DefaultTenantID:
			return DefaultTenantID, nil
		default:
			return "", ErrTenantIDInvalid
		}
	}
	// 开启多租户时，不允许租户为空
	if tenantID == "" {
		return "", ErrTenantIDRequired
	}
	return tenantID, nil
}
