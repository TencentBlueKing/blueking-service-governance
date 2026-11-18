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
}

// NewFrontendFeaturesOutput 根据部署配置生成前端功能开关。
func NewFrontendFeaturesOutput(cfg *config.Config) *FrontendFeaturesOutput {
	return &FrontendFeaturesOutput{
		PolarisWorkspaceAccessPoint: cfg.Polaris.AllowWorkspaceAccessPoints,
	}
}

// FrontendFeaturesResponse 是获取前端功能开关接口的 JSON 响应。
type FrontendFeaturesResponse struct {
	Data *FrontendFeaturesOutput `json:"data"`
}
