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

// Package serializer 定义配置开关 Gin API 的请求/响应结构体。
package serializer

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"

// WorkspaceURI 包含 workspaceID 路径参数的 URI 绑定结构体。
type WorkspaceURI struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
}

// CfgSwitchesOutput 空间的配置开关，前端据此决定是否展示对应 tab。
type CfgSwitchesOutput struct {
	// 是否开启配置管理（BSCP）
	Bscp bool `json:"bscp"`
	// 是否展示七彩石
	RainbowTConf bool `json:"rainbowTConf"`
}

// NewCfgSwitchesOutput 由空间的全部配置开关构造输出。
func NewCfgSwitchesOutput(switches []model.WorkspaceCfgSwitch) *CfgSwitchesOutput {
	o := &CfgSwitchesOutput{}
	for _, s := range switches {
		switch s.Key {
		case model.SwitchBscp:
			o.Bscp = s.Enabled
		case model.SwitchRainbowTConf:
			o.RainbowTConf = s.Enabled
		}
	}
	return o
}

// CfgSwitchesResponse 包装 CfgSwitchesOutput 的响应结构。
type CfgSwitchesResponse struct {
	Data *CfgSwitchesOutput `json:"data"`
}
