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

package dbfactory

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// TrpcApplicationOpts 定义创建 tRPC 应用时的可选参数
type TrpcApplicationOpts struct {
	// Image 容器镜像，默认为 "nginx:latest"
	Image string
	// TrpcConfig tRPC 配置，如果为 nil 则使用默认配置
	TrpcConfig *appmodel.TrpcConfig
	// EnvVars 环境变量列表
	EnvVars []appmodel.Variable
	// Components 组件列表
	Components []*component.Component
	// Replicas 副本数
	Replicas *int32

	// WorkspaceID 工作空间 ID，如果为空，则使用随机 test-ws-*
	WorkspaceID string

	// BuildConfig 构建配置，如果为 nil 则使用默认配置（SourceTypePipeline）
	BuildConfig *build.Config
}

// TrpcApplication 创建一个已持久化的测试用 tRPC 应用（包含 App、AppModel 和 AppConfigFile）
//
// Args:
//   - stores: 必需的 store 集合
//   - opts: 可选参数，用于定制应用配置。如果为 nil 则使用默认值
//
// Returns:
//   - app: 创建的 Application
//   - appModel: 创建的 AppModel
func TrpcApplication(
	ctx context.Context,
	stores *TrpcApplicationStores,
	opts *TrpcApplicationOpts,
) (*bkmsapp.Application, *appmodel.AppModel) {
	if opts == nil {
		opts = &TrpcApplicationOpts{}
	}

	// 创建 Application
	appName := "test-app-" + stringx.Random(6)
	app := &bkmsapp.Application{
		ID:          appName + stringx.Random(6),
		Name:        appName,
		WorkspaceID: "test-ws-" + stringx.Random(6),
		Type:        bkmsapp.AppTypeTRPC,
	}

	if opts.WorkspaceID != "" {
		app.WorkspaceID = opts.WorkspaceID
	}

	err := stores.AppStore.CreateApp(ctx, app)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 设置默认镜像
	image := opts.Image
	if image == "" {
		image = "nginx:latest"
	}

	// 设置默认 TrpcConfig
	trpcConfig := opts.TrpcConfig
	if trpcConfig == nil {
		trpcConfig = &appmodel.TrpcConfig{
			FileName:    "trpc_config.yaml",
			FilePath:    "/etc/",
			FileContent: "server:\n  port: 8080\n",
		}
	}

	// 创建 AppModel
	model := &appmodel.AppModel{
		AppID: app.ID,
		Workload: appmodel.Workload{
			Type:       appmodel.WorkloadTypeTrpc,
			Name:       app.Name,
			Image:      image,
			EnvVars:    opts.EnvVars,
			TrpcConfig: *trpcConfig,
		},
		Components: opts.Components,
		Replicas:   opts.Replicas,
	}
	err = stores.AppModelStore.CreateAppModel(ctx, model)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 创建 app-level 默认 AppConfigFile（tRPC plugin 需要）
	_, err = appcfg.NewAppConfigFileService(stores.AppConfigFileStore, stores.AppConfigFileVersionStore).Create(
		ctx,
		appcfg.CreateCfgFileParams{
			AppID:             app.ID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              appcfg.DefaultAppConfigFileName,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatYAML,
			Content:           &trpcConfig.FileContent,
			Creator:           appcfg.CfgSystemUser,
			Description:       appcfg.CfgSystemVersionDescription,
		},
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 创建 BuildConfig
	buildCfg := opts.BuildConfig
	if buildCfg == nil {
		buildCfg = &build.Config{
			AppID:      app.ID,
			SourceType: build.SourceTypePipeline,
			Pipeline: &build.PipelineConfig{
				PipelineID: "p-xxx",
			},
			TagConfig: build.TagConfig{
				Type: build.VersionTypeCustom,
				CustomOpts: &build.CustomTagOpts{
					WithBuildTime: true,
				},
			},
		}
	}
	err = stores.BuildConfigStore.Create(ctx, buildCfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return app, model
}

// TafApplicationStores 创建 TAF 应用所需的 store 集合
type TafApplicationStores struct {
	AppStore                  bkmsapp.ApplicationStore
	AppModelStore             appmodel.AppModelStore
	AppConfigFileStore        appcfg.AppConfigFileStore
	AppConfigFileVersionStore appcfg.AppConfigFileVersionStore
	BuildConfigStore          build.ConfigStore
}

// TafApplicationOpts 定义创建 TAF 应用时的可选参数
type TafApplicationOpts struct {
	// Image 容器镜像，默认为 "nginx:latest"
	Image string
	// TafConfig TAF 配置，如果为 nil 则使用默认配置
	TafConfig *appmodel.TafConfig
	// EnvVars 环境变量列表
	EnvVars []appmodel.Variable
	// Components 组件列表
	Components []*component.Component
	// Replicas 副本数
	Replicas *int32

	// WorkspaceID 工作空间 ID，如果为空，则使用随机 test-ws-*
	WorkspaceID string

	// BuildConfig 构建配置，如果为 nil 则使用默认配置（SourceTypeCodeRepository）
	BuildConfig *build.Config
}

// TafApplication 创建一个已持久化的测试用 TAF 应用（包含 App、AppModel 和 AppConfigFile）
//
// Args:
//   - stores: 必需的 store 集合
//   - opts: 可选参数，用于定制应用配置。如果为 nil 则使用默认值
//
// Returns:
//   - app: 创建的 Application
//   - appModel: 创建的 AppModel
func TafApplication(
	ctx context.Context,
	stores *TafApplicationStores,
	opts *TafApplicationOpts,
) (*bkmsapp.Application, *appmodel.AppModel) {
	if opts == nil {
		opts = &TafApplicationOpts{}
	}

	// 创建 Application
	appName := "test-taf-app-" + stringx.Random(6)
	app := &bkmsapp.Application{
		ID:          appName + stringx.Random(6),
		Name:        appName,
		WorkspaceID: "test-ws-" + stringx.Random(6),
		Type:        bkmsapp.AppTypeTAF,
	}

	if opts.WorkspaceID != "" {
		app.WorkspaceID = opts.WorkspaceID
	}

	err := stores.AppStore.CreateApp(ctx, app)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 设置默认镜像
	image := opts.Image
	if image == "" {
		image = "nginx:latest"
	}

	// 设置默认 TafConfig
	tafConfig := opts.TafConfig
	if tafConfig == nil {
		tafConfig = &appmodel.TafConfig{
			FileName: "taf_config.conf",
			FilePath: "/etc/",
			FileContent: `<taf>
  <application>
    <server>
      logpath=/data/log
    </server>
  </application>
</taf>
`,
		}
	}

	// 创建 AppModel
	model := &appmodel.AppModel{
		AppID: app.ID,
		Workload: appmodel.Workload{
			Type:      appmodel.WorkloadTypeTaf,
			Name:      app.Name,
			Image:     image,
			EnvVars:   opts.EnvVars,
			TafConfig: *tafConfig,
		},
		Components: opts.Components,
		Replicas:   opts.Replicas,
	}
	err = stores.AppModelStore.CreateAppModel(ctx, model)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 创建 app-level 默认 AppConfigFile（TAF plugin 需要）
	_, err = appcfg.NewAppConfigFileService(stores.AppConfigFileStore, stores.AppConfigFileVersionStore).Create(
		ctx,
		appcfg.CreateCfgFileParams{
			AppID:             app.ID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              appcfg.DefaultAppConfigFileName,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatTAF,
			Content:           &tafConfig.FileContent,
			Creator:           appcfg.CfgSystemUser,
			Description:       appcfg.CfgSystemVersionDescription,
		},
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// 创建 BuildConfig
	buildCfg := opts.BuildConfig
	if buildCfg == nil {
		buildCfg = &build.Config{
			AppID:      app.ID,
			SourceType: build.SourceTypePipeline,
			Pipeline: &build.PipelineConfig{
				PipelineID: "p-xxx",
			},
			TagConfig: build.TagConfig{
				Type: build.VersionTypeCustom,
				CustomOpts: &build.CustomTagOpts{
					WithBuildTime: true,
				},
			},
		}
	}
	err = stores.BuildConfigStore.Create(ctx, buildCfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return app, model
}
