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

// Package admincommand 声明 admin 命令能力：各框架使用的协议，以及 tRPC 各语言的 admin 地址写法。
package admincommand

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
)

// Protocol admin 命令的协议。
type Protocol string

const (
	// ProtocolTRPC tRPC admin，基于 HTTP。
	ProtocolTRPC Protocol = "trpc"
	// ProtocolTAF TAF 私有协议。
	ProtocolTAF Protocol = "taf"
)

// TrpcConfigLayout tRPC 配置文件中 admin 地址的写法，不同语言写法不同。
type TrpcConfigLayout string

const (
	// TrpcLayoutNested server.admin.ip / server.admin.port，go、python 使用。
	TrpcLayoutNested TrpcConfigLayout = "nested"
	// TrpcLayoutFlat server.admin_ip / server.admin_port，cpp、nodejs 使用。
	TrpcLayoutFlat TrpcConfigLayout = "flat"
	// TrpcLayoutNestedPrefixed server.admin.admin_ip / server.admin.admin_port，java 使用。
	TrpcLayoutNestedPrefixed TrpcConfigLayout = "nestedPrefixed"
)

// trpcAdminLayouts 各语言下 admin 地址的写法。
//
// tRPC 目前只支持 go、cpp、python，其余语言暂不可达；框架目录为 tRPC 开放对应语言后自动生效。
var trpcAdminLayouts = map[string]TrpcConfigLayout{
	appruntime.LanguageGo:     TrpcLayoutNested,
	appruntime.LanguagePython: TrpcLayoutNested,

	appruntime.LanguageCpp:    TrpcLayoutFlat,
	appruntime.LanguageNodeJS: TrpcLayoutFlat,

	appruntime.LanguageJava: TrpcLayoutNestedPrefixed,
}

// Props admin 命令的执行参数。
type Props struct {
	// Protocol 协议。
	Protocol Protocol
	// ConfigLayout admin 地址的写法，仅 tRPC 使用。
	ConfigLayout TrpcConfigLayout
}

// Public 无对外内容。
func (p Props) Public() any { return nil }

// Spec admin 命令：tRPC、TAF 支持。
var Spec = runtimemod.Spec[Props]{
	ID: runtimemod.ModAdminCommand,
	Resolve: func(framework, language string) runtimemod.Mod[Props] {
		switch framework {
		case appruntime.FrameworkTAF:
			return runtimemod.Supported(Props{Protocol: ProtocolTAF})

		case appruntime.FrameworkTRPC:
			if language == "" {
				return runtimemod.Incomplete[Props]()
			}
			layout, ok := trpcAdminLayouts[language]
			if !ok {
				return runtimemod.Unsupported[Props]()
			}
			return runtimemod.Supported(Props{Protocol: ProtocolTRPC, ConfigLayout: layout})
		}
		return runtimemod.Unsupported[Props]()
	},
}
