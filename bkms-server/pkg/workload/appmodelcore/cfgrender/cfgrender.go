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

// Package cfgrender 在 bkms-server 构建 workload 时做编译期配置渲染。
//
// 它只处理 ${{ env.KEY }} 模板：收集引用并替换成当前已知的环境变量值。
// 不创建 ConfigMap，也不生成 init container。
// Pod 启动后才能确定的值（BKMS_POD_IP 等）由 runtimerender 负责。
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

// RenderConfigContents 只做编译期环境变量渲染，framework 和 plain 共用。
//
// 对每条文件：
//  1. 扫描 ${{ env.KEY }}，写入 collector（供上游校验未定义变量）
//  2. 用 envVars 把 ${{ env.KEY }} 替换成当前已知的值
//
// 本函数不会做 runtime 渲染：不写 ConfigMap、不跑 init container。
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
