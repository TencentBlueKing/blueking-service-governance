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

// Package devmode 提供开发模式组件支持
package devmode

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"

// 环境类型常量，用于判断是否允许启用开发模式
const (
	// EnvTypeTest 测试环境，允许使用
	EnvTypeTest = string(env.TypeTest)

	// EnvTypeDevelopment 开发环境，允许使用
	EnvTypeDevelopment = string(env.TypeDevelopment)

	// EnvTypeStaging 预发布环境，允许使用
	EnvTypeStaging = string(env.TypeStaging)

	// EnvTypeProduction 正式环境，禁止使用
	EnvTypeProduction = string(env.TypeProduction)
)

const (
	// ConfigMapResourceSuffix ConfigMap 资源名称后缀
	ConfigMapResourceSuffix = "-devmode-scripts"

	// ScriptsSubDir 脚本子目录名称
	ScriptsSubDir = "configmap-scripts"
)

// 开发模式 Kubernetes 资源名称和路径常量

// trpc 相关常量
const (
	// TrpcBinaryPath trpc 二进制文件路径
	TrpcBinaryPath = "/usr/local/trpc/bin"
	// TrpcWorkPath trpc 开发模式根目录
	TrpcWorkPath = "/data/bkms/dev-mode/trpc"
	// TrpcMountPath trpc 脚本文件挂载路径
	TrpcMountPath = TrpcWorkPath + "/" + ScriptsSubDir
	// TrpcInitScriptPath trpc init.sh 脚本的完整路径
	TrpcInitScriptPath = TrpcMountPath + "/init.sh"
)

// taf 相关常量
const (
	// TafWorkPath taf 开发模式根目录
	TafWorkPath = "/data/bkms/dev-mode/taf"
	// TafMountPath taf 脚本文件挂载路径
	TafMountPath = TafWorkPath + "/" + ScriptsSubDir
	// TafInitScriptPath taf init.sh 脚本的完整路径
	TafInitScriptPath = TafMountPath + "/init.sh"
)

// ConfigMap Data 中的脚本文件 key（同时也是挂载后的文件名）
const (
	// KeyInitScript 初始化脚本的 key
	KeyInitScript = "init.sh"

	// KeyStartScript 启动脚本的 key
	KeyStartScript = "start.sh"

	// KeyStopScript 停止脚本的 key
	KeyStopScript = "stop.sh"

	// KeyMonitorScript 监控脚本的 key
	KeyMonitorScript = "monitor.sh"

	// KeyRestartScript 重启脚本的 key
	KeyRestartScript = "restart.sh"

	// KeyUtilsScript 公共工具脚本的 key
	KeyUtilsScript = "utils.sh"

	// KeyTafStartScript taf 启动脚本的 key（wrap 方式拉起 taf 进程）
	KeyTafStartScript = "taf-start.sh"

	// KeyBkmsTrpcBinPath bkms trpc 二进制文件路径
	KeyBkmsTrpcBinPath = "BKMS_TRPC_BIN_PATH"

	// KeyBkmsCustomStartScript bkms 自定义启动脚本
	KeyBkmsCustomStartScript = "BKMS_CUSTOM_START_SCRIPT"
)
