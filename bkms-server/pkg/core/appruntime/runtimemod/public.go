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

package runtimemod

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"

// SchemaVersion 对外结构的版本号，只在结构发生不兼容变更时递增。
const SchemaVersion = 1

// PublicModView 单个能力的对外结构。
type PublicModView struct {
	// Supported 是否支持。
	Supported bool `json:"supported"`
	// ReasonCode 不支持的原因，支持时省略。
	ReasonCode ReasonCode `json:"reasonCode,omitempty"`
	// Props 对外参数，没有时省略。
	Props any `json:"props,omitempty"`
}

// View 把不同参数类型的 Mod 统一转成对外结构。
type View interface {
	View() PublicModView
}

// View 实现 View 接口。
func (m Mod[P]) View() PublicModView {
	if !m.Supported {
		return PublicModView{Supported: false, ReasonCode: m.ReasonCode}
	}
	return PublicModView{Supported: true, Props: m.Props.Public()}
}

// PublicView 应用能力的对外结构。
//
// 只反映技术栈决定的支持情况，不包含环境、集群等运行时条件。
type PublicView struct {
	// SchemaVersion 结构版本号。
	SchemaVersion int `json:"schemaVersion"`
	// Type 应用类型，如 bkmsApp、helm。
	Type string `json:"type"`
	// Framework 框架，helm、agones 应用为空。
	Framework string `json:"framework,omitempty"`
	// Language 语言，未记录时为空。
	Language string `json:"language,omitempty"`
	// Mods 能力 ID 到能力的映射，包含全部能力；新增能力只会增加 key。
	Mods map[ModID]PublicModView `json:"mods"`
}

// BuildPublicView 由能力集合生成对外结构。
func BuildPublicView(stack appruntime.Stack, mods map[ModID]View) PublicView {
	out := make(map[ModID]PublicModView, len(mods))
	for id, mod := range mods {
		out[id] = mod.View()
	}
	return PublicView{
		SchemaVersion: SchemaVersion,
		Type:          stack.Type,
		Framework:     stack.Framework,
		Language:      stack.Language,
		Mods:          out,
	}
}
