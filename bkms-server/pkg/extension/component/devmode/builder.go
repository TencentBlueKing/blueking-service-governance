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

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
)

var _ DevMode = new(builder)

type builder struct {
	config *Config
}

// scriptDefinition 脚本定义（key、模板内容、名称）
type scriptDefinition struct {
	key string

	template string

	name string
}

// IsAllowed 检查当前环境是否允许使用开发模式
func (b *builder) IsAllowed() bool {
	if b.config == nil || !b.config.Enabled {
		return false
	}
	if !bkmsenv.IsValidEnvType(b.config.EnvType) {
		return false
	}
	return !bkmsenv.IsProductionType(bkmsenv.Type(b.config.EnvType))
}

// Validate 验证开发模式配置
func (b *builder) Validate() error {
	if b.config == nil || !b.config.Enabled {
		return nil
	}
	if !b.IsAllowed() {
		return errors.Wrapf(ErrNotAllowed, "environment %s", b.config.EnvType)
	}
	if !bkmsapp.IsAppModelType(b.config.AppType) {
		return errors.Wrapf(ErrUnsupportedAppType, "%s only trpc and taf are supported", b.config.AppType)
	}
	if b.config.AppName == "" {
		return ErrAppNameRequired
	}
	if b.config.StartupCommand == "" {
		return ErrStartupCommandRequired
	}
	// taf 类型不校验 TrpcBinaryPath，trpc 类型需要校验
	if b.config.IsTrpc() && b.config.TrpcBinaryPath == "" {
		return ErrTrpcBinaryPathRequired
	}
	return nil
}

// Build 构建开发模式组件完整输出
func (b *builder) Build() (*Output, error) {
	if b.config == nil || !b.config.Enabled {
		return nil, nil
	}
	// trpc 类型需要处理 TrpcBinaryPath
	if b.config.IsTrpc() {
		b.config.TrpcBinaryPath = strings.TrimSuffix(b.config.TrpcBinaryPath, "/")
	}

	configMap, err := b.BuildConfigMap()
	if err != nil {
		return nil, err
	}

	return &Output{
		ConfigMap:   configMap,
		Volume:      b.BuildVolume(),
		VolumeMount: b.BuildVolumeMount(),
		Command:     b.BuildCommand(),
	}, nil
}

// BuildConfigMap 构建包含所有管理脚本的 ConfigMap
func (b *builder) BuildConfigMap() (*corev1.ConfigMap, error) {
	if b.config == nil || !b.config.Enabled {
		return nil, nil
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	var initSh string
	var scripts []scriptDefinition
	data := make(map[string]string)

	if b.config.IsTaf() {
		// taf 类型
		initSh = TafInitScript
		scripts = []scriptDefinition{
			{key: KeyUtilsScript, template: TafUtilsScript, name: KeyUtilsScript},
			{key: KeyStartScript, template: TafStartScript, name: KeyStartScript},
			{key: KeyStopScript, template: TafStopScript, name: KeyStopScript},
			{key: KeyMonitorScript, template: TafMonitorScript, name: KeyMonitorScript},
			{key: KeyRestartScript, template: TafRestartScript, name: KeyRestartScript},
		}
	} else {
		// trpc 类型
		initSh = TrpcInitScript
		data[KeyBkmsTrpcBinPath] = b.config.TrpcBinaryPath
		data[KeyBkmsCustomStartScript] = b.config.StartupCommand
		scripts = []scriptDefinition{
			{key: KeyUtilsScript, template: TrpcUtilsScript, name: KeyUtilsScript},
			{key: KeyStartScript, template: TrpcStartScript, name: KeyStartScript},
			{key: KeyStopScript, template: TrpcStopScript, name: KeyStopScript},
			{key: KeyMonitorScript, template: TrpcMonitorScript, name: KeyMonitorScript},
			{key: KeyRestartScript, template: TrpcRestartScript, name: KeyRestartScript},
		}
	}

	configMapData := map[string]string{
		// init.sh 不需要模板渲染
		KeyInitScript: initSh,
	}

	for _, script := range scripts {
		rendered, err := b.renderScript(script.template, script.name, data)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to render %s script", script.name)
		}
		configMapData[script.key] = rendered
	}

	if b.config.IsTaf() {
		// taf 类型额外包含 taf-start.sh（wrap 启动脚本）
		configMapData[KeyTafStartScript] = fmt.Sprintf("#!/bin/bash\n\nexec %s", b.config.StartupCommand)
	}

	resourceName := ConfigMapResourceName(b.config.AppName)

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
		},
		Data: configMapData,
	}

	return configMap, nil
}

// BuildVolume 构建引用 ConfigMap 的 Volume（权限 0755）
func (b *builder) BuildVolume() corev1.Volume {
	resourceName := ConfigMapResourceName(b.config.AppName)
	return corev1.Volume{
		Name: resourceName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					// 引用的 ConfigMap 名称
					Name: resourceName,
				},
				DefaultMode: func() *int32 {
					// 设置脚本文件权限为 0755（可执行）
					mode := int32(0o755)
					return &mode
				}(),
			},
		},
	}
}

// BuildVolumeMount 构建 VolumeMount
func (b *builder) BuildVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		// 引用的 Volume 名称
		Name: ConfigMapResourceName(b.config.AppName),
		// 挂载路径
		MountPath: b.config.MountPath,
	}
}

// BuildCommand 构建启动命令，指向挂载路径下的 init.sh
func (b *builder) BuildCommand() []string {
	return []string{filepath.Join(b.config.MountPath, "init.sh")}
}

// renderScript 使用 Go template 渲染脚本内容
func (b *builder) renderScript(scriptTemplate, scriptName string, data map[string]string) (string, error) {
	tmpl, err := template.New(scriptName).Parse(scriptTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
