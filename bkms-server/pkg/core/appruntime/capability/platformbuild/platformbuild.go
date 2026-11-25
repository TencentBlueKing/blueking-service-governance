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

// Package platformbuild 声明平台构建能力：哪些技术栈支持由平台生成 Dockerfile 构建镜像。
package platformbuild

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
)

// Props 平台构建的执行参数。
type Props struct {
	// DockerfileLanguage 传给流水线的 Dockerfile 生成语言。
	DockerfileLanguage string
	// RequiresBuilderImage 生成的 Dockerfile 是否有 builder 阶段、因而必须配置构建镜像。
	// python 模板只有运行镜像一个阶段，构建镜像可以留空。
	RequiresBuilderImage bool
}

// Public 对外展示平台构建参数，前端据此判断模板是否包含 builder 阶段
func (p Props) Public() any {
	return struct {
		RequiresBuilderImage bool `json:"requiresBuilderImage"`
	}{
		RequiresBuilderImage: p.RequiresBuilderImage,
	}
}

// Spec 平台构建：由平台生成 Dockerfile 并构建镜像。
// tRPC、blank 的 go，以及 blank 的 python 支持。tRPC 不开放 python。
var Spec = runtimemod.Spec[Props]{
	ID: runtimemod.ModPlatformBuild,
	Resolve: func(framework, language string) runtimemod.Mod[Props] {
		// Dockerfile 只按语言区分。taf 不开放
		if framework != appruntime.FrameworkTRPC && framework != appruntime.FrameworkBlank {
			return runtimemod.Unsupported[Props]()
		}
		switch language {
		case appruntime.LanguageGo:
			return runtimemod.Supported(Props{DockerfileLanguage: appruntime.LanguageGo, RequiresBuilderImage: true})
		case appruntime.LanguagePython:
			// python 只作为通用应用开放，tRPC 不提供该语言的平台构建
			if framework != appruntime.FrameworkBlank {
				return runtimemod.Unsupported[Props]()
			}

			return runtimemod.Supported(Props{
				DockerfileLanguage:   appruntime.LanguagePython,
				RequiresBuilderImage: false,
			})
		case "":
			return runtimemod.Incomplete[Props]()
		default:
			return runtimemod.Unsupported[Props]()
		}
	},
}

// DockerfileLanguage 返回平台生成 Dockerfile 使用的语言；技术栈不支持平台构建时返回错误。
//
// 构建前置校验与实际构建都通过它取语言，两处判断保持一致。
func DockerfileLanguage(mod runtimemod.Mod[Props]) (string, error) {
	props, err := mod.Require()
	if err != nil {
		return "", errors.Wrap(err, "resolve dockerfile language")
	}
	return props.DockerfileLanguage, nil
}

// RequiresBuilderImage 返回保存或触发平台构建时是否必须校验构建镜像。
//
// 技术栈不支持或语言未补全时仍要求构建镜像，保持放开 python 之前的校验口径；
// 是否支持平台构建由 DockerfileLanguage 单独拦截。
func RequiresBuilderImage(mod runtimemod.Mod[Props]) bool {
	if !mod.Supported {
		return true
	}
	return mod.Props.RequiresBuilderImage
}
