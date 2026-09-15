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

// Package cfgrender performs compile-time rendering of application config files.
//
// It collects ${{ env.KEY }} references for undefined-variable validation and
// renders template variables into their actual values. The output
// ConfigFileParams are consumed by runtimerender.BuildConfig to produce K8s
// resources (ConfigMap + init container) for runtime placeholder replacement.
//
// Both framework and plain config files share this pipeline; callers should
// complete any kind-specific pre-processing (e.g. polaris patching, mountDir
// validation) before invoking RenderConfigContents.
package cfgrender

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

// ConfigFileParams describes one rendered config file.
type ConfigFileParams struct {
	// FileName is the config file name.
	FileName string
	// FilePath is the config file directory in the workload container.
	FilePath string
	// FileContent is the config template content written to the ConfigMap.
	FileContent string
}

// RenderConfigContents 对配置文件内容执行环境变量引用收集与模板变量渲染，
// framework 和 plain 共用。
//
// 处理步骤：
//  1. 收集 ${{ env.KEY }} 引用（供上游做未定义环境变量校验）
//  2. 将模板变量渲染为实际环境变量值
//  3. 输出 ConfigFileParams 供 runtimerender.BuildConfig 消费
//
// 注意：collector 是调用方提供的可变状态（入参兼出参），调用后 collector 中会
// 新增本次扫描到的环境变量引用。调用方应在所有 Render 调用完成后再读取 collector
// 的汇总结果（如 UndefinedEnvVars），以确保收集完整。
//
// 调用方在调用前应完成各自独有的预处理（如 framework 的 polaris patcher、plain 的 mountDir 校验）。
func RenderConfigContents(
	items []appcfg.MountableFile,
	envVars map[string]string,
	collector *envvarrefs.Collector,
) ([]ConfigFileParams, error) {
	renderer := render.New(render.SetEnvContext(envVars))
	result := make([]ConfigFileParams, 0, len(items))

	for _, item := range items {
		if err := collector.Collect(item.Content, envvarrefs.Source{
			Type: envvarrefs.SourceAppConfigFile,
			Name: item.Name,
		}); err != nil {
			return nil, errors.Wrapf(err, "collecting env vars from config %s", item.Name)
		}

		renderedContent, err := renderer.Render(item.Content)
		if err != nil {
			return nil, errors.Wrapf(err, "rendering config %s", item.Name)
		}

		result = append(result, ConfigFileParams{
			FileName:    item.Name,
			FilePath:    item.MountDir,
			FileContent: renderedContent,
		})
	}
	return result, nil
}
