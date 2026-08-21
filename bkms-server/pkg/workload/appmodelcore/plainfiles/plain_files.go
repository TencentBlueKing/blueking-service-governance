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

package plainfiles

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/runtimerender"
)

// BuildRuntimePlainConfigFiles 获取当前环境实际生效的 plain 配置文件，
// 完成挂载路径校验、环境变量引用收集与模板渲染，并转换为运行时配置渲染所需的文件参数。
func BuildRuntimePlainConfigFiles(
	ctx context.Context,
	store appcfg.AppConfigFileStore,
	appID string,
	envName string,
	envVars map[string]string,
	collector *envvarrefs.Collector,
) ([]runtimerender.ConfigFileParams, error) {
	// 某些应用类型没有接入 AppConfigFileStore，此时视为没有额外挂载文件。
	if store == nil {
		return nil, nil
	}

	// 先解析出当前应用、当前环境下真正生效的 plain 文件内容。
	plainFiles, err := appcfg.ListEnvPlainContents(ctx, store, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "listing env plain config files")
	}

	result := make([]runtimerender.ConfigFileParams, 0, len(plainFiles))
	for _, item := range plainFiles {
		// plain 文件最终会以额外挂载文件的形式下发，因此 mountPath 必须完整存在。
		if item.File.MountPath == "" {
			return nil, errors.Errorf("plain config file %s has empty mount path", item.File.ID.Hex())
		}

		// 记录 plain 文件里引用到的环境变量，供上游统一做依赖分析与校验。
		if err = collector.Collect(item.Content, envvarrefs.Source{
			Type: envvarrefs.SourceAppConfigFile,
			Name: item.File.Name,
		}); err != nil {
			return nil, errors.Wrapf(err, "collecting env vars from plain config %s", item.File.Name)
		}

		// 用当前环境变量上下文渲染模板，得到最终要写入容器的文件内容。
		renderedContent, renderErr := render.New(render.SetEnvContext(envVars)).Render(item.Content)
		if renderErr != nil {
			return nil, errors.Wrapf(renderErr, "rendering plain config %s", item.File.Name)
		}

		// 运行时渲染层分别接收目录和文件名，因此这里从 mountPath 中拆分两者。
		result = append(result, runtimerender.ConfigFileParams{
			FileName:    filepath.Base(item.File.MountPath),
			FilePath:    filepath.Dir(item.File.MountPath),
			FileContent: renderedContent,
		})
	}

	return result, nil
}
