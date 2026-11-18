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

// Package serializer 定义前端功能开关接口的 Gin Input/Output 结构体。
package serializer

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"

// FrontendFeaturesOutput 前端功能开关，只用于控制页面入口是否展示，后端接口仍各自校验。
type FrontendFeaturesOutput struct {
	// 是否允许空间自建北极星接入点并按集群绑定。为 false 时前端隐藏接入点管理、集群绑定、组件安装等入口，
	// 所有集群都使用平台托管接入点
	PolarisWorkspaceAccessPoint bool `json:"polarisWorkspaceAccessPoint"`
	// 是否为多租户模式。为 true 时前端需要展示租户信息、按用户所属租户选择和展示用户
	EnableMultiTenantMode bool `json:"enableMultiTenantMode"`
	// 制品库是否为独立部署（不接入权限中心，改用制品库账号认证）
	EnableIndependentBkRepo bool `json:"enableIndependentBkRepo"`
	// 是否配置了 WeTerm 环境，为 false 时前端隐藏 WeTerm 入口
	EnableWeTerm bool `json:"enableWeTerm"`
	// 是否启用 BSCP（服务配置中心）对接，为 false 时前端隐藏 BSCP 配置相关入口
	EnableBSCP bool `json:"enableBSCP"`
}

// NewFrontendFeaturesOutput 根据部署配置生成前端功能开关。
func NewFrontendFeaturesOutput(cfg *config.Config) *FrontendFeaturesOutput {
	return &FrontendFeaturesOutput{
		PolarisWorkspaceAccessPoint: cfg.Polaris.AllowWorkspaceAccessPoints,
		EnableMultiTenantMode:       cfg.Tenant.EnableMultiTenantMode,
		EnableIndependentBkRepo:     cfg.FeatureIntegrations.EnableIndependentBkRepo,
		EnableWeTerm:                cfg.BCS.WeTermEnvID != "",
		EnableBSCP:                  !cfg.FeatureIntegrations.DisableBSCP,
	}
}

// FrontendFeaturesResponse 是获取前端功能开关接口的 JSON 响应。
type FrontendFeaturesResponse struct {
	Data *FrontendFeaturesOutput `json:"data"`
}
