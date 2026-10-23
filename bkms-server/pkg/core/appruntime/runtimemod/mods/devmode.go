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

// DevModeScriptSet 开发模式注入的脚本集。
type DevModeScriptSet string

const (
	// DevModeScriptSetTRPC tRPC 脚本集。
	DevModeScriptSetTRPC DevModeScriptSet = "trpc"
	// DevModeScriptSetTAF TAF 脚本集。
	DevModeScriptSetTAF DevModeScriptSet = "taf"
)

// DevModePathScriptsSubDir 脚本所在的子目录。
const DevModePathScriptsSubDir = "configmap-scripts"

// 开发模式的容器内路径，由平台决定，用户不能修改。
const (
	// DevModeTrpcWorkPath tRPC 开发模式根目录。
	DevModeTrpcWorkPath = "/data/bkms/dev-mode/trpc"
	// DevModeTrpcMountPath tRPC 脚本挂载目录。
	DevModeTrpcMountPath = DevModeTrpcWorkPath + "/" + DevModePathScriptsSubDir
	// DevModeTrpcBinaryPath tRPC 二进制所在目录。
	DevModeTrpcBinaryPath = "/usr/local/trpc/bin"

	// DevModeTafWorkPath TAF 开发模式根目录。
	DevModeTafWorkPath = "/data/bkms/dev-mode/taf"
	// DevModeTafMountPath TAF 脚本挂载目录。
	DevModeTafMountPath = DevModeTafWorkPath + "/" + DevModePathScriptsSubDir
)

// DevModeProps 开发模式的执行参数。
type DevModeProps struct {
	// ScriptSet 脚本集。
	ScriptSet DevModeScriptSet
	// WorkPath 开发模式根目录。
	WorkPath string
	// MountPath 脚本挂载目录。
	MountPath string
	// BinaryPath 二进制所在目录，TAF 不需要，为空。
	BinaryPath string
}

// Public 对外展示根目录与挂载目录，与 AppSpec 接口返回的一致。
func (p DevModeProps) Public() any {
	return struct {
		WorkPath  string `json:"workPath"`
		MountPath string `json:"mountPath"`
	}{p.WorkPath, p.MountPath}
}

// DevMode 开发模式：tRPC、TAF 支持。是否开启、当前环境是否允许，由 devmode 模块在运行时判断。
var DevMode = runtimemod.Spec[DevModeProps]{
	ID: runtimemod.ModDevMode,
	Resolve: runtimemod.ByFramework(map[string]DevModeProps{
		appruntime.FrameworkTRPC: {
			ScriptSet:  DevModeScriptSetTRPC,
			WorkPath:   DevModeTrpcWorkPath,
			MountPath:  DevModeTrpcMountPath,
			BinaryPath: DevModeTrpcBinaryPath,
		},

		appruntime.FrameworkTAF: {
			ScriptSet: DevModeScriptSetTAF,
			WorkPath:  DevModeTafWorkPath,
			MountPath: DevModeTafMountPath,
		},
	}),
}
