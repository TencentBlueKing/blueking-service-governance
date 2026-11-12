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

// Package serializer defines Gin input and output serializers for platform workspace cfg switch APIs.
package serializer

import (
	"time"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
)

// WorkspacePath is the URI input for listing cfg switches of a workspace.
type WorkspacePath struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
}

// CfgSwitchPath is the URI input for a single cfg switch of a workspace.
type CfgSwitchPath struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
	// 开关标识：bscp / rainbowTConf
	Key string `uri:"key" binding:"required"`
}

// UpdatePlatCfgSwitchInput is the JSON input for setting a cfg switch.
type UpdatePlatCfgSwitchInput struct {
	// 是否开启；使用指针以区分字段缺失与显式传 false
	Enabled *bool `json:"enabled" binding:"required"`
}

// PlatCfgSwitchOutput is one cfg switch of a workspace.
type PlatCfgSwitchOutput struct {
	// 开关标识：bscp / rainbowTConf
	Key string `json:"key"`
	// 展示名称
	Name string `json:"name"`
	// 是否开启
	Enabled bool `json:"enabled"`
	// 最近修改人，从未设置过时为空
	Updater string `json:"updater"`
	// 最近修改时间，从未设置过时为 null
	UpdatedAt *time.Time `json:"updatedAt"`
}

// NewPlatCfgSwitchOutput builds output from the model.
func NewPlatCfgSwitchOutput(s *model.WorkspaceCfgSwitch) *PlatCfgSwitchOutput {
	def, _ := model.GetSwitchDef(s.Key)
	return &PlatCfgSwitchOutput{
		Key:       string(s.Key),
		Name:      def.Name,
		Enabled:   s.Enabled,
		Updater:   s.Updater,
		UpdatedAt: lo.EmptyableToPtr(s.UpdatedAt),
	}
}

// PlatCfgSwitchListResponse is the JSON response carrying all cfg switches of a workspace.
type PlatCfgSwitchListResponse struct {
	Data []*PlatCfgSwitchOutput `json:"data"`
}

// PlatCfgSwitchResponse is the JSON response carrying one cfg switch of a workspace.
type PlatCfgSwitchResponse struct {
	Data *PlatCfgSwitchOutput `json:"data"`
}
