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

package serializer

// 本文件是旧 trpc / taf 接口的兼容层：入参、出参结构保持不变，内部一律按新模型处理，
// 不读写 Application.TrpcSpec、Workload.Type / TrpcConfig / TafConfig 等旧字段。

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/lifecycle"
)

// TrpcSpecInput 旧接口的 tRPC 配置。
//
// Deprecated: bkmsApp 迁移兼容，新接口用 language 与 frameworkConfigFile。
type TrpcSpecInput struct {
	// 编程语言。修改时须与应用当前语言一致，不能切换
	Language string `json:"language" binding:"required,oneof=go cpp"`
	// 框架配置文件名。创建时必填；修改时非空才同步到框架配置文件定义
	FileName string `json:"fileName"`
	// 框架配置文件挂载目录。创建时必填；修改时非空才同步到框架配置文件定义
	FilePath string `json:"filePath" binding:"omitempty,mount_dir"`
	// 框架配置文件内容，仅创建时使用
	FileContent string `json:"fileContent"`
}

// TafSpecInput 旧接口的 TAF 配置。
//
// Deprecated: bkmsApp 迁移兼容，新接口用 frameworkConfigFile。
type TafSpecInput struct {
	// 框架配置文件名。创建时必填；修改时非空才同步到框架配置文件定义
	FileName string `json:"fileName"`
	// 框架配置文件挂载目录。创建时必填；修改时非空才同步到框架配置文件定义
	FilePath string `json:"filePath" binding:"omitempty,mount_dir"`
	// 框架配置文件内容，仅创建时使用
	FileContent string `json:"fileContent"`
}

// TrpcSpecOutputObj 详情中的 tRPC 配置，framework 为 trpc 时返回。
//
// Deprecated: bkmsApp 迁移兼容，请读取应用的 language 与框架配置文件定义。
type TrpcSpecOutputObj struct {
	// 编程语言
	Language string `json:"language"`
	// 框架配置文件名
	FileName string `json:"fileName"`
	// 框架配置文件挂载目录
	FilePath string `json:"filePath"`
	// 框架配置文件的默认内容
	FileContent string `json:"fileContent"`
}

// TafSpecOutputObj 详情中的 TAF 配置，framework 为 taf 时返回。
//
// Deprecated: bkmsApp 迁移兼容，请读取框架配置文件定义。
type TafSpecOutputObj struct {
	// 框架配置文件名
	FileName string `json:"fileName"`
	// 框架配置文件挂载目录
	FilePath string `json:"filePath"`
	// 框架配置文件的默认内容
	FileContent string `json:"fileContent"`
}

// FrameworkSpecView 详情中 trpcSpec / tafSpec 的取值，由调用方准备。
//
// Deprecated: bkmsApp 迁移兼容，仅供详情输出 trpcSpec / tafSpec。
type FrameworkSpecView struct {
	// Framework 应用框架：trpc 时输出 trpcSpec，taf 时输出 tafSpec，其余都不输出
	Framework string
	// Language 应用语言，仅 trpcSpec 输出
	Language string
	// FileName 框架配置文件名
	FileName string
	// FilePath 框架配置文件挂载目录
	FilePath string
	// FileContent 框架配置文件的默认内容
	FileContent string
}

// ToBKMSAppCompat 将旧创建接口的 trpc / taf 请求转换为 bkmsApp 的框架、语言与创建参数。
// TAF 请求没有语言，固定为 cpp。
//
// Deprecated: bkmsApp 迁移兼容，仅供旧创建接口使用。
func (input *CreateAppInput) ToBKMSAppCompat() (framework, language string, params *lifecycle.CreateParams) {
	spec := input.AppModelSpec
	params = &lifecycle.CreateParams{
		Command:   spec.Command,
		Args:      spec.Args,
		ShCommand: spec.ShCommand,
		EnvVars:   variableInputsToModel(spec.EnvVars),
	}
	switch input.Type {
	case appruntime.AppTypeTRPC:
		framework, language = appruntime.FrameworkTRPC, spec.TrpcSpec.Language
		params.FrameworkFile = &lifecycle.FrameworkFileParams{
			Name:     spec.TrpcSpec.FileName,
			MountDir: spec.TrpcSpec.FilePath,
			Content:  spec.TrpcSpec.FileContent,
		}
	case appruntime.AppTypeTAF:
		framework, language = appruntime.FrameworkTAF, appruntime.LanguageCpp
		params.FrameworkFile = &lifecycle.FrameworkFileParams{
			Name:     spec.TafSpec.FileName,
			MountDir: spec.TafSpec.FilePath,
			Content:  spec.TafSpec.FileContent,
		}
	}
	return framework, language, params
}

// ToSpecCompatParams 将旧 trpc-spec / taf-spec 请求转换为兼容修改参数。
// 环境变量为兼容旧客户端接收但忽略，应通过独立接口修改。
//
// Deprecated: bkmsApp 迁移兼容，仅供旧 trpc-spec / taf-spec 接口使用。
func (input *AppModelSpecInput) ToSpecCompatParams(framework string) *lifecycle.SpecCompatParams {
	params := &lifecycle.SpecCompatParams{
		Framework: framework,
		Command:   input.Command,
		Args:      input.Args,
		ShCommand: input.ShCommand,
	}
	switch framework {
	case appruntime.FrameworkTRPC:
		if input.TrpcSpec != nil {
			params.Language = input.TrpcSpec.Language
			params.FileName = input.TrpcSpec.FileName
			params.FilePath = input.TrpcSpec.FilePath
		}
	case appruntime.FrameworkTAF:
		if input.TafSpec != nil {
			params.FileName = input.TafSpec.FileName
			params.FilePath = input.TafSpec.FilePath
		}
	}
	return params
}
