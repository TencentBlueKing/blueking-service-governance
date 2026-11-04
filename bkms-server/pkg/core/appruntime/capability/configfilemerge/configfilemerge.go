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

// Package configfilemerge 声明配置文件合并能力：叠加配置与默认文件按哪种策略合并。
package configfilemerge

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
)

// Strategy YAML 配置文件的合并策略，取值与配置文件合并的 appcfg.MergeStrategy 一致。
type Strategy string

const (
	// StrategyStrategicMerge k8s 默认的 strategic merge：叠加配置中的空值字段会删除该字段，
	// 默认文件为空时合并结果为空。
	StrategyStrategicMerge Strategy = "strategicMerge"
	// StrategyNullPreserving 保留空值的 strategic merge：默认文件与叠加配置中的空值字段都保留为 null，
	// 默认文件为空时仍应用叠加配置。
	StrategyNullPreserving Strategy = "nullPreserving"
)

// Props 配置文件合并的参数。
type Props struct {
	// Strategy 合并策略。
	Strategy Strategy
}

// Public 合并策略属于内部执行参数，不对外展示。
func (p Props) Public() any { return nil }

// Spec 配置文件合并：tRPC、TAF 保留空值，blank 按 k8s 默认的 strategic merge。
// 合并由 core/app/appcfg 执行，作用于应用的全部配置文件；helm / agones 的 values 属独立链路，
// 不经过本能力，按 k8s 默认的 strategic merge 合并。
var Spec = runtimemod.Spec[Props]{
	ID: runtimemod.ModConfigFileMerge,
	Resolve: runtimemod.ByFramework(map[string]Props{
		appruntime.FrameworkTRPC:  {Strategy: StrategyNullPreserving},
		appruntime.FrameworkTAF:   {Strategy: StrategyNullPreserving},
		appruntime.FrameworkBlank: {Strategy: StrategyStrategicMerge},
	}),
}
