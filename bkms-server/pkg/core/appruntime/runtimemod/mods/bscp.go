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

package mods

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
)

// BSCPWorkloadMetaProps 无参数。工作负载类型取决于实际渲染结果，由 BSCP 模块按渲染结果填写。
type BSCPWorkloadMetaProps struct{}

// Public 无对外内容。
func (p BSCPWorkloadMetaProps) Public() any { return nil }

// BSCPWorkloadMeta BSCP 元信息自动填充：初始化 BSCP 元信息时，自动填入工作负载名称与类型。
// 目前仅 tRPC 支持，TAF 尚未接入。
var BSCPWorkloadMeta = runtimemod.Spec[BSCPWorkloadMetaProps]{
	ID: runtimemod.ModBSCPWorkloadMeta,
	Resolve: runtimemod.ByFramework(map[string]BSCPWorkloadMetaProps{
		appruntime.FrameworkTRPC: {},
	}),
}
