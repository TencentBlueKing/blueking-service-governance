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

// Package platconfig 提供平台级配置的查询能力，例如供前端控制页面展示的特性开关。
package platconfig

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"

// 前端特性开关的 key，前端按 key 判断功能是否展示，新增后不要随意改名
const (
	// FeatureMultiTenantMode 是否开启多租户模式
	FeatureMultiTenantMode = "MULTI_TENANT_MODE"
	// FeatureIndependentBkRepo 制品库是否为独立部署（不接入权限中心，成员需在制品库侧自行管理）
	FeatureIndependentBkRepo = "INDEPENDENT_BK_REPO"
	// FeatureWeTerm 是否可以通过 WeTerm 登录容器
	FeatureWeTerm = "WETERM"
)

// GetFrontendFeatures 根据部署配置生成前端特性开关。
//
// 只做白名单映射，不直接透出原始配置，避免泄露敏感配置项。
func GetFrontendFeatures(cfg *config.Config) map[string]bool {
	return map[string]bool{
		FeatureMultiTenantMode:   cfg.Tenant.EnableMultiTenantMode,
		FeatureIndependentBkRepo: cfg.FeatureIntegrations.EnableIndependentBkRepo,
		FeatureWeTerm:            cfg.BCS.WeTermEnvID != "",
	}
}
