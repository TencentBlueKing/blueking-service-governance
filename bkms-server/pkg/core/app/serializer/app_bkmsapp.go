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

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/lifecycle"
)

// -----------------------------------------------------------------------------
// CreateBKMSApp
// -----------------------------------------------------------------------------

// CreateBKMSAppInput is the JSON body for creating a bkmsApp application.
type CreateBKMSAppInput struct {
	// 应用名称. 长度范围 1~63 个字符，以小写字母开头，可包含小写字母、数字、中划线，不能以中划线结尾
	Name string `json:"name" binding:"required,app_id"`
	// 应用 ID. 应用在全局范围内的唯一标识，默认情况下基于 name 自动生成，在 name 后追加后缀。
	// 长度范围 1~63 个字符，以小写字母开头，可包含小写字母、数字、中划线，不能以中划线结尾
	ID string `json:"id" binding:"required,app_id"`
	// 框架：trpc、taf、blank，可选值见 /app-runtime-definitions
	Framework string `json:"framework" binding:"required"`
	// 语言：go、cpp、nodejs、python，须为框架支持的语言（nodejs、python 仅 blank 支持）
	Language string `json:"language" binding:"required"`
	// 构建配置
	BuildConfig *BuildConfigInput `json:"buildConfig" binding:"required"`
	// 应用模型规范
	AppModelSpec *BKMSAppModelSpecInput `json:"appModelSpec" binding:"required"`
	// 框架配置文件。框架支持框架配置文件时必填，blank 不能传
	FrameworkConfigFile *FrameworkConfigFileInput `json:"frameworkConfigFile"`
}

// BKMSAppModelSpecInput is the app model spec input of a bkmsApp application.
type BKMSAppModelSpecInput struct {
	// 容器启动命令
	Command []string `json:"command"`
	// 容器启动参数
	Args []string `json:"args"`
	// 容器启动脚本，以 /bin/sh 执行，与 command、args 互斥，不能为纯空白
	ShCommand string `json:"shCommand"`
	// 容器环境变量
	EnvVars []VariableInput `json:"envVars" binding:"dive"`
	// 框架扩展配置，按框架严格校验；缺省为空对象
	FrameworkConfig map[string]any `json:"frameworkConfig"`
}

// FrameworkConfigFileInput is the framework config file input.
type FrameworkConfigFileInput struct {
	// 文件名
	Name string `json:"name" binding:"required"`
	// 挂载目录
	MountDir string `json:"mountDir" binding:"required,mount_dir"`
	// 文件内容
	Content string `json:"content"`
}

// ToCreateParams converts the input into lifecycle create params.
func (input *CreateBKMSAppInput) ToCreateParams() *lifecycle.CreateParams {
	spec := input.AppModelSpec
	params := &lifecycle.CreateParams{
		Command:         spec.Command,
		Args:            spec.Args,
		ShCommand:       spec.ShCommand,
		EnvVars:         variableInputsToModel(spec.EnvVars),
		FrameworkConfig: spec.FrameworkConfig,
	}
	if file := input.FrameworkConfigFile; file != nil {
		params.FrameworkFile = &lifecycle.FrameworkFileParams{
			Name:     file.Name,
			MountDir: file.MountDir,
			Content:  file.Content,
		}
	}
	return params
}

// -----------------------------------------------------------------------------
// UpdateAppStartupConfig
// -----------------------------------------------------------------------------

// UpdateAppStartupConfigInput is the JSON body for updating the startup config of a bkmsApp application.
type UpdateAppStartupConfigInput struct {
	// 容器启动命令，整体替换
	Command []string `json:"command"`
	// 容器启动参数，整体替换
	Args []string `json:"args"`
	// 容器启动脚本，整体替换；与 command、args 互斥，不能为纯空白
	ShCommand string `json:"shCommand"`
	// 框架扩展配置：缺省不修改，null 拒绝，对象（含 {}）整体替换并重新校验
	FrameworkConfig json.RawMessage `json:"frameworkConfig,omitempty" swaggertype:"object"`
}

// ToUpdateStartupConfigParams converts the input into lifecycle update params.
func (input *UpdateAppStartupConfigInput) ToUpdateStartupConfigParams() (*lifecycle.UpdateStartupConfigParams, error) {
	params := &lifecycle.UpdateStartupConfigParams{
		Command:   input.Command,
		Args:      input.Args,
		ShCommand: input.ShCommand,
	}
	if len(input.FrameworkConfig) == 0 {
		return params, nil
	}
	var cfg map[string]any
	if err := json.Unmarshal(input.FrameworkConfig, &cfg); err != nil {
		return nil, errors.Wrap(err, "frameworkConfig must be an object")
	}
	if cfg == nil {
		return nil, errors.New("frameworkConfig must not be null")
	}
	params.FrameworkConfig = cfg
	return params, nil
}

// -----------------------------------------------------------------------------
// Runtime definitions & mods
// -----------------------------------------------------------------------------

// GetAppRuntimeDefinitionsOutput is the response for getting app runtime definitions.
type GetAppRuntimeDefinitionsOutput struct {
	Data *AppRuntimeDefinitionsOutputObj `json:"data"`
}

// AppRuntimeDefinitionsOutputObj is the framework catalog and capability matrix.
type AppRuntimeDefinitionsOutputObj struct {
	// 框架目录
	Frameworks []FrameworkDefOutputObj `json:"frameworks"`
	// 每个合法技术栈组合的能力集合
	Matrix []runtimemod.PublicView `json:"matrix"`
}

// FrameworkDefOutputObj is a framework definition.
type FrameworkDefOutputObj struct {
	// 框架名
	Name string `json:"name"`
	// 支持的语言
	SupportedLanguages []string `json:"supportedLanguages"`
	// frameworkConfig 的 schema 版本
	ConfigVersion int `json:"configVersion"`
}

// FromModel fills output fields from the framework catalog and capability matrix.
func (o *AppRuntimeDefinitionsOutputObj) FromModel(
	catalog []appruntime.FrameworkDef,
	matrix []runtimemod.PublicView,
) *AppRuntimeDefinitionsOutputObj {
	o.Frameworks = make([]FrameworkDefOutputObj, 0, len(catalog))
	for _, def := range catalog {
		o.Frameworks = append(o.Frameworks, FrameworkDefOutputObj{
			Name:               def.Name,
			SupportedLanguages: def.SupportedLanguages,
			ConfigVersion:      def.ConfigVersion,
		})
	}
	o.Matrix = matrix
	return o
}

// GetAppRuntimeModsOutput is the response for getting the runtime mods of an application.
type GetAppRuntimeModsOutput struct {
	Data runtimemod.PublicView `json:"data"`
}
