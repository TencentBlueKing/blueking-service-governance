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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

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

// PlatformTenantID 返回平台数据（如内置组件定义）所在的租户：多租户模式下为 system，否则为 default。
// 多租户环境只支持全新部署，不存在两种模式的数据并存，因此平台租户在一个环境内是固定的
func PlatformTenantID() (string, error) {
	if config.G == nil {
		return "", errors.New("config is not loaded, cannot determine the platform tenant")
	}
	if config.G.Tenant.EnableMultiTenantMode {
		return SystemTenantID, nil
	}
	return DefaultTenantID, nil
}
