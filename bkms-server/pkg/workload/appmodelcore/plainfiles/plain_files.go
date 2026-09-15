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

// Package plainfiles 提供 plain 配置文件在 workload 层的集成能力。
//
// BuildPlainConfigFiles 是面向 plain 配置文件的上层包装，由 Builder 公共步骤调用。
// 根据 EnableEnvVarRender 开关将文件分为两组：
//   - RenderParams：需要经过 cfgrender.RenderConfigContents 做环境变量渲染 + runtimerender 管线
//   - DirectParams：直接以原始内容构建 ConfigMap 挂载，跳过模板渲染
package plainfiles

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/cfgrender"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

// BuildResult 包含按 EnableEnvVarRender 分组后的两组配置文件参数。
type BuildResult struct {
	// RenderParams 需要 env var 渲染 + runtime render 管线处理的文件。
	RenderParams []cfgrender.ConfigFileParams
	// DirectParams 直接以原始内容挂载到 ConfigMap 的文件（跳过模板渲染）。
	DirectParams []cfgrender.ConfigFileParams
}

// BuildPlainConfigFiles 获取当前环境实际生效的 plain 配置文件，
// 按 EnableEnvVarRender 开关分为两组：
//   - EnableEnvVarRender=true → 经 cfgrender.RenderConfigContents 渲染后走 runtimerender 管线
//   - EnableEnvVarRender=false → 跳过渲染，直接构建 ConfigMap 挂载
//
// 该函数由 Builder 公共步骤调用，独立于任何框架 plugin，因此所有 workload 类型
// （trpc、taf、standard）都能自动获得 plain 文件支持。
func BuildPlainConfigFiles(
	ctx context.Context,
	cfgProvider appcfg.MountableFileProvider,
	appID string,
	envName string,
	envVars map[string]string,
	collector *envvarrefs.Collector,
) (*BuildResult, error) {
	if cfgProvider == nil {
		return nil, nil
	}

	plainFiles, err := cfgProvider.ListPlainMountableFiles(ctx, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "listing env plain config files")
	}
	if len(plainFiles) == 0 {
		return nil, nil
	}

	for _, item := range plainFiles {
		if item.MountDir == "" {
			return nil, errors.Errorf("plain config file %s has empty mount dir", item.Name)
		}
	}

	// 按 EnableEnvVarRender 分组
	var renderItems, directItems []appcfg.MountableFile
	for _, item := range plainFiles {
		if item.EnableEnvVarRender {
			renderItems = append(renderItems, item)
		} else {
			directItems = append(directItems, item)
		}
	}

	result := &BuildResult{}

	// 需要渲染的文件走 cfgrender 管线
	if len(renderItems) > 0 {
		rendered, renderErr := cfgrender.RenderConfigContents(renderItems, envVars, collector)
		if renderErr != nil {
			return nil, renderErr
		}
		result.RenderParams = rendered
	}

	// 不需要渲染的文件直接构建参数（原样保留内容）
	for _, item := range directItems {
		result.DirectParams = append(result.DirectParams, cfgrender.ConfigFileParams{
			FileName:    item.Name,
			FilePath:    item.MountDir,
			FileContent: item.Content,
		})
	}

	return result, nil
}
