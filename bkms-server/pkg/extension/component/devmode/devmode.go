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
//
// 开发模式允许开发者在容器运行时动态替换二进制文件并重启服务，无需重新构建和部署镜像。
// 支持 development、test 和 staging 环境，仅 production 环境严禁使用。
// 支持 trpc 和 taf 两种应用类型，两者使用不同的工作目录和脚本集。
//
// 工作原理：将管理脚本打包为 ConfigMap，挂载到容器后替换启动命令为 init.sh，
// 由 monitor.sh 守护进程负责进程监控和自动重启。
package devmode

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// DevMode 开发模式组件接口
type DevMode interface {
	// IsAllowed 检查当前环境是否允许使用开发模式
	IsAllowed() bool

	// Validate 验证配置有效性
	Validate() error

	// Build 构建完整输出
	Build() (*Output, error)

	// BuildConfigMap 构建脚本 ConfigMap
	BuildConfigMap() (*corev1.ConfigMap, error)

	// BuildVolume 构建 Volume
	BuildVolume() corev1.Volume

	// BuildVolumeMount 构建 VolumeMount
	BuildVolumeMount() corev1.VolumeMount

	// BuildCommand 构建启动命令
	BuildCommand() []string
}

// Config 开发模式配置
type Config struct {
	// Enabled 是否启用
	Enabled bool

	// EnvType 环境类型
	EnvType string

	// AppType 应用类型: trpc / taf
	AppType string

	// AppName 应用名称，应与二进制文件名一致
	AppName string

	// WorkPath 开发模式根目录，trpc 为 TrpcWorkPath，taf 为 TafWorkPath
	WorkPath string

	// MountPath 脚本挂载路径，为空时根据 WorkPath 自动计算
	MountPath string

	// StartupCommand 用户 app 启动命令
	StartupCommand string

	// TrpcBinaryPath trpc 二进制文件路径（仅 trpc 类型使用）
	TrpcBinaryPath string
}

// IsTaf 判断是否为 taf 类型应用
func (c *Config) IsTaf() bool {
	return c.AppType == bkmsapp.AppTypeTAF
}

// IsTrpc 判断是否为 trpc 类型应用
func (c *Config) IsTrpc() bool {
	return c.AppType == bkmsapp.AppTypeTRPC
}

// Output 开发模式组件输出
type Output struct {
	// ConfigMap 包含所有管理脚本的 ConfigMap
	ConfigMap *corev1.ConfigMap

	// Volume 引用 ConfigMap 的 Volume
	Volume corev1.Volume

	// VolumeMount 挂载到容器的 VolumeMount
	VolumeMount corev1.VolumeMount

	// Command 容器启动命令（指向 init.sh）
	Command []string
}

// New 创建开发模式构建器实例
func New(config *Config) DevMode {
	if config == nil {
		return &builder{config: &Config{Enabled: false}}
	}

	// 根据 AppType 选择对应的 WorkPath 和 MountPath
	if config.IsTaf() {
		config.WorkPath = TafWorkPath
	} else {
		config.WorkPath = TrpcWorkPath
	}
	config.MountPath = config.WorkPath + "/" + ScriptsSubDir

	return &builder{config: config}
}

// CreateDevModeConfig 从 AppModel 和环境信息创建开发模式配置
func CreateDevModeConfig(appModel *appmodel.AppModel, envType string, enabled bool) *Config {
	config := FromAppModel(appModel, envType)
	config.Enabled = enabled
	return config
}

// FromAppModel 从 AppModel 创建开发模式配置
func FromAppModel(appModel *appmodel.AppModel, envType string) *Config {
	// 从 appModel.Workload.Command、Args 中提取
	var customStartScript string

	// 检查是否有用户 app 启动命令
	if len(appModel.Workload.Command) > 0 || len(appModel.Workload.Args) > 0 {
		temp := make([]string, 0)
		temp = append(temp, appModel.Workload.Command...)
		temp = append(temp, appModel.Workload.Args...)
		customStartScript = strings.Join(temp, " ")
	}

	appType := appModel.Workload.Type
	cfg := &Config{
		Enabled:        false,
		EnvType:        envType,
		AppType:        appType,
		AppName:        appModel.Workload.Name,
		StartupCommand: customStartScript,
	}

	if appType == bkmsapp.AppTypeTAF {
		// taf 类型
		cfg.TrpcBinaryPath = ""
		cfg.WorkPath = TafWorkPath
		cfg.MountPath = TafMountPath
	} else {
		// trpc 类型
		cfg.WorkPath = TrpcWorkPath
		cfg.MountPath = TrpcMountPath
		cfg.TrpcBinaryPath = TrpcBinaryPath
	}

	return cfg
}

// ConfigMapResourceName 根据应用名称生成开发模式资源名称（ConfigMap / Volume）
// 格式: {appName}-devmode-scripts
func ConfigMapResourceName(appName string) string {
	return appName + ConfigMapResourceSuffix
}
