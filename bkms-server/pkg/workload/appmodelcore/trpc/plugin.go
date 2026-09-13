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

package trpc

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/runtimerender"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/trpc/patcher"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/plugin"
)

// Plugin provides workload extensions for tRPC applications.
type Plugin struct {
	mountableFileProvider appcfg.MountableFileProvider
	configPatchers        []appcfg.ConfigPatcher
}

// NewPlugin creates a new tRPC plugin with the given dependencies.
func NewPlugin(
	mountableFileProvider appcfg.MountableFileProvider,
	polarisConfigStore polaris.PolarisConfigStore,
) *Plugin {
	return &Plugin{
		mountableFileProvider: mountableFileProvider,
		configPatchers: []appcfg.ConfigPatcher{
			patcher.NewPolarisRegistryPatcher(polarisConfigStore),
		},
	}
}

// Type returns the workload type handled by this plugin.
func (p *Plugin) Type() string {
	return appmodel.WorkloadTypeTrpc
}

// Start initializes a plugin session for this build.
//
// 处理流程：
//  1. 获取框架配置内容（含 polaris patcher）
//  2. 调用 RenderConfigContents 收集环境变量引用并渲染模板变量
//  3. 通过 runtimerender.BuildConfig 构建 K8s 资源（ConfigMap + init container）
//
// plain 配置文件由 Builder 公共步骤独立处理，plugin 只负责 framework。
func (p *Plugin) Start(
	ctx context.Context,
	env *envmodel.Environment,
	app *bkmsapp.Application,
	appModel *appmodel.AppModel,
	renderCtx plugin.RenderContext,
) (plugin.WorkloadPluginSession, error) {
	frameworkItem, err := p.computeTrpcConfig(ctx, app, env, *appModel)
	if err != nil {
		return nil, errors.Wrap(err, "computing tRPC config")
	}

	if frameworkItem.Name == "" {
		return &runtimerender.Config{}, nil
	}

	// 收集环境变量引用并渲染 ${{ env.KEY }} 模板变量
	frameworkParams, err := runtimerender.RenderConfigContents(
		[]appcfg.MountableFile{frameworkItem},
		renderCtx.EnvVars,
		renderCtx.Collector,
	)
	if err != nil {
		return nil, errors.Wrap(err, "rendering tRPC framework config")
	}

	// 回写渲染后的内容供其他组件使用
	appModel.Workload.TrpcConfig.FileContent = frameworkParams[0].FileContent

	return runtimerender.BuildConfig(runtimerender.ConfigParams{
		WorkloadType:  appmodel.WorkloadTypeTrpc,
		ConfigMapName: appModel.Workload.Name,
		Files:         frameworkParams,
	})
}

// computeTrpcConfig computes the tRPC configuration content based on environment.
// It queries AppConfigFile with the following priority:
// 1. Environment-specific config (envName = current environment name)
// 2. Application-level default config (envName = "")
//
// 获取配置文件内容后，依次调用注册的 appcfg.ConfigPatcher 对配置进行补丁。
// 返回的 MountableFile 已完成 patch，待交给 RenderConfigContents 做环境变量渲染。
func (p *Plugin) computeTrpcConfig(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	appModel appmodel.AppModel,
) (appcfg.MountableFile, error) {
	trpcCfg := appModel.Workload.TrpcConfig
	item := appcfg.MountableFile{
		Name:     trpcCfg.FileName,
		Content:  trpcCfg.FileContent,
		MountDir: trpcCfg.FilePath,
	}

	if p.mountableFileProvider != nil {
		frameworkContent, err := p.mountableFileProvider.GetFrameworkMountableFile(ctx, app.ID, env.Name)
		if err != nil {
			return appcfg.MountableFile{}, err
		}
		// framework 文件名当前仍以 app model 为准，避免把运行时挂载名意外改成 def 名（如 default）。
		item.Content = frameworkContent.Content
		// TODO: MountDir 当前仍由 app model（trpcCfg.FilePath）决定，未使用 def.MountDir。
		// 待挂载路径迁移至 def 后，应改为 item.MountDir = frameworkContent.MountDir。
	}

	// 依次调用注册的 ConfigPatcher 对配置进行补丁（如 polaris 注册信息注入）
	var err error
	for _, cfgPatcher := range p.configPatchers {
		item.Content, err = cfgPatcher.Patch(ctx, app.ID, env.Name, item.Content)
		if err != nil {
			return appcfg.MountableFile{}, errors.Wrap(err, "patching tRPC config")
		}
	}

	return item, nil
}
