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

package runtimerender

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

// RenderConfigContents 对配置文件内容执行环境变量引用收集与模板变量渲染，
// framework 和 plain 共用。
//
// 处理步骤：
//  1. 收集 ${{ env.KEY }} 引用（供上游做未定义环境变量校验）
//  2. 将模板变量渲染为实际环境变量值
//  3. 输出 ConfigFileParams 供 BuildConfig 消费
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
