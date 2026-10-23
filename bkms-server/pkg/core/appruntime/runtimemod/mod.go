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

// Package runtimemod 定义运行时能力的声明方式与解析结果。
//
// 一个能力（Mod）是否支持、支持时有哪些参数，由框架和语言决定。具体能力在 mods 子包中声明，
// 本包只提供通用的类型与按框架/语言查表的工具。
package runtimemod

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
)

// ModID 能力标识，也是对外 JSON 的 key，发布后不能修改。
type ModID string

const (
	// ModFrameworkFile 框架配置文件。
	ModFrameworkFile ModID = "frameworkFile"
	// ModDevMode 开发模式。
	ModDevMode ModID = "devMode"
	// ModPlatformBuild 平台生成 Dockerfile 构建。
	ModPlatformBuild ModID = "platformBuild"
	// ModAdminCommand admin 命令。
	ModAdminCommand ModID = "adminCommand"
	// ModAPMServiceName APM 服务名提取。
	ModAPMServiceName ModID = "apmServiceName"
	// ModPolarisServiceCheck 北极星服务名校验。
	ModPolarisServiceCheck ModID = "polarisServiceCheck"
	// ModPolarisFrameworkPatch 北极星注册信息注入。
	ModPolarisFrameworkPatch ModID = "polarisFrameworkPatch"
	// ModBSCPWorkloadMeta BSCP 元信息自动填充。
	ModBSCPWorkloadMeta ModID = "bscpWorkloadMeta" //nolint:gosec // 能力标识，不是凭据
)

// ReasonCode 能力不支持的原因。
type ReasonCode string

const (
	// ReasonUnsupportedAppType 非 bkmsApp 应用（helm、agones），所有能力都不支持。
	ReasonUnsupportedAppType ReasonCode = "unsupported_app_type"
	// ReasonUnsupportedStack 该框架（或框架与语言的组合）不支持此能力。
	ReasonUnsupportedStack ReasonCode = "unsupported_stack"
	// ReasonIncompleteLanguage 能否支持取决于语言，但应用未记录语言。
	// BRIDGE-ONLY：只有存量应用会缺少语言，回填后删除。
	ReasonIncompleteLanguage ReasonCode = "incomplete_language"
)

// ErrModUnsupported 能力不支持时，Mod.Require 返回此错误。
var ErrModUnsupported = errors.New("runtime mod unsupported")

// Props 能力参数需实现的接口。
//
// Public 返回可对外展示的部分，内部执行参数不得包含在内；没有可展示内容时返回 nil。
type Props interface {
	Public() any
}

// Mod 某个技术栈对某个能力的解析结果。
type Mod[P Props] struct {
	// ID 能力标识。
	ID ModID
	// Supported 是否支持。
	Supported bool
	// ReasonCode 不支持的原因，支持时为空。
	ReasonCode ReasonCode
	// Props 执行参数，不支持时为零值。
	Props P
}

// Require 支持时返回参数，不支持时返回 ErrModUnsupported。执行器应通过它取参数。
func (m Mod[P]) Require() (P, error) {
	if !m.Supported {
		var zero P
		return zero, errors.Wrapf(ErrModUnsupported, "%s: %s", m.ID, m.ReasonCode)
	}
	return m.Props, nil
}

// Spec 一个能力的声明。
//
// Resolve 根据框架和语言给出解析结果：简单能力用 ByFramework / ByLanguage 生成，
// 参数随框架和语言组合变化的能力自行实现。
type Spec[P Props] struct {
	// ID 能力标识。
	ID ModID
	// Resolve 解析函数。
	Resolve func(framework, language string) Mod[P]
}

// ResolveStack 按技术栈解析该能力，补齐能力标识并处理非 bkmsApp 应用。
func (s Spec[P]) ResolveStack(stack appruntime.Stack) Mod[P] {
	if stack.Type != appruntime.AppTypeBKMSApp {
		return Mod[P]{ID: s.ID, ReasonCode: ReasonUnsupportedAppType}
	}
	mod := s.Resolve(stack.Framework, stack.Language)
	mod.ID = s.ID
	return mod
}

// Supported 构造支持的结果。
func Supported[P Props](props P) Mod[P] {
	return Mod[P]{Supported: true, Props: props}
}

// Unsupported 构造不支持的结果。
func Unsupported[P Props]() Mod[P] {
	return Mod[P]{ReasonCode: ReasonUnsupportedStack}
}

// Incomplete 构造语言缺失、无法判断的结果。
func Incomplete[P Props]() Mod[P] {
	return Mod[P]{ReasonCode: ReasonIncompleteLanguage}
}

// ByFramework 按框架查表：表中有的框架支持，与语言无关，语言为空也支持。
func ByFramework[P Props](table map[string]P) func(framework, language string) Mod[P] {
	return func(framework, _ string) Mod[P] {
		props, ok := table[framework]
		if !ok {
			return Unsupported[P]()
		}
		return Supported(props)
	}
}

// ByLanguage 按语言查表：表中有的语言支持，与框架无关。语言为空时无法判断，返回 Incomplete。
func ByLanguage[P Props](table map[string]P) func(framework, language string) Mod[P] {
	return func(_, language string) Mod[P] {
		if language == "" {
			return Incomplete[P]()
		}
		props, ok := table[language]
		if !ok {
			return Unsupported[P]()
		}
		return Supported(props)
	}
}
