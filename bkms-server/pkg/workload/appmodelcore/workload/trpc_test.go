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

package workload_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gopkg.in/yaml.v3"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// TrpcWorkloadBuilder 包含 TRPC 特有的测试用例（如 environment-specific appConfigFile）
var _ = Describe("TrpcWorkloadBuilder", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var appStore bkmsapp.ApplicationStore
	var appModelStore appmodel.AppModelStore
	var envSvc *env.EnvService
	var builderSvc *workload.BuilderService
	var appConfigFileStore appcfg.AppConfigFileStore
	var appConfigFileVersionStore appcfg.AppConfigFileVersionStore
	var buildConfigStore build.ConfigStore
	var componentDefStore component.ComponentDefStore
	var polarisConfigStore polaris.PolarisConfigStore

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appmodel.FxModule,
			workload.FxModule,
			component.FxModule,
			appcfg.FxModule,
			env.FxModule,
			polaris.FxModule,
			build.FxModule,
			envvars.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(
				&appStore,
				&appModelStore,
				&appConfigFileStore,
				&appConfigFileVersionStore,
				&buildConfigStore,
				&componentDefStore,
				&envSvc,
				&builderSvc,
				&polarisConfigStore,
			),
		)
		diApp.RequireStart()

		// Load builtin components
		dbfactory.LoadBuiltinComponents(ctx, database.Client(), "../../../extension/component/assets/comps")
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	// TRPC 特有测试：environment-specific appConfigFile
	Context("Test Build with environment-specific appConfigFile", func() {
		var app *bkmsapp.Application
		var prodEnv *envmodel.Environment
		var testEnv *envmodel.Environment
		var appModel *appmodel.AppModel

		BeforeEach(func() {
			// 使用 dbfactory.TrpcApplication 创建应用和模型
			// TrpcConfig.FileContent 会作为默认配置文件内容
			app, appModel = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
				AppStore:                  appStore,
				AppModelStore:             appModelStore,
				AppConfigFileStore:        appConfigFileStore,
				AppConfigFileVersionStore: appConfigFileVersionStore,
				BuildConfigStore:          buildConfigStore,
			}, &dbfactory.TrpcApplicationOpts{
				TrpcConfig: &appmodel.TrpcConfig{
					FileName:    "trpc_go.yaml",
					FilePath:    "/etc/trpc",
					Language:    "go",
					FileContent: "server:\n  address: 0.0.0.0:8080\n  timeout: 3000\n",
				},
			})

			// Create two environments
			prodEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)
			testEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// 获取默认配置文件的 ID，用于创建 overlay 配置
			defaultFiles, err := appConfigFileStore.List(ctx, app.ID, appcfg.AcfFilterEnvName(appcfg.EnvNameDefault))
			Expect(err).NotTo(HaveOccurred())
			Expect(defaultFiles).NotTo(BeEmpty())
			defaultFileID := defaultFiles[0].ID

			// Create prod environment-specific AppConfigFile (overlay)
			prodOverlayContent := "server:\n  address: 0.0.0.0:9090"
			_, err = appcfg.NewAppConfigFileService(appConfigFileStore, appConfigFileVersionStore).Create(
				ctx,
				appcfg.CreateCfgFileParams{
					AppID:               app.ID,
					EnvName:             prodEnv.Name,
					Name:                "trpc-prod-config",
					Type:                appcfg.AppConfigFileTypeOverlay,
					ContentSourceType:   appcfg.ContentSourceTypeLocal,
					Format:              appcfg.FileFormatYAML,
					BaseAppConfigFileID: &defaultFileID,
					OverlayContent:      &prodOverlayContent,
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// Clean up app config files
			if app != nil {
				_, _ = appConfigFileStore.DeleteByApp(ctx, app.ID)
			}
		})

		It("should use app-level default config when no env-specific config exists", func() {
			// Build for test environment (no specific config)
			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())
			gd := result.GameDeployment
			extraObjs := result.ExtraObjects

			// Should have one extra ConfigMap
			Expect(extraObjs).To(HaveLen(1))
			Expect(extraObjs[0].GetKind()).To(Equal("ConfigMap"))

			// Verify ConfigMap contains default content
			configMapData := extraObjs[0].Object["data"].(map[string]any)
			configContent := configMapData["trpc_go.yaml"].(string)
			Expect(configContent).To(Equal("server:\n  address: 0.0.0.0:8080\n  timeout: 3000\n"))

			// Verify volume mount exists
			Expect(gd.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := gd.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(1))
			Expect(container.VolumeMounts[0].Name).To(Equal("trpc-config-rendered"))
			Expect(container.VolumeMounts[0].MountPath).To(Equal("/etc/trpc/trpc_go.yaml"))

			// Verify init container exists for runtime variable rendering
			Expect(gd.Spec.Template.Spec.InitContainers).To(HaveLen(1))
			Expect(gd.Spec.Template.Spec.InitContainers[0].Name).To(Equal("trpc-init"))
		})

		It("should use env-specific config when it exists", func() {
			// Build for prod environment (has specific config)
			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, prodEnv)
			Expect(err).NotTo(HaveOccurred())
			gd := result.GameDeployment
			extraObjs := result.ExtraObjects

			// Should have one extra ConfigMap
			Expect(extraObjs).To(HaveLen(1))
			Expect(extraObjs[0].GetKind()).To(Equal("ConfigMap"))

			// Verify ConfigMap contains prod-specific content (not default)
			configMapData := extraObjs[0].Object["data"].(map[string]any)
			configContent := configMapData["trpc_go.yaml"].(string)
			Expect(configContent).To(Equal("server:\n  address: 0.0.0.0:9090\n  timeout: 3000\n"))

			// Verify volume mount exists
			Expect(gd.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := gd.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(1))
		})
	})

	// tRPC 配置变量渲染测试
	Context("Test Build with template variable rendering in tRPC config", func() {
		var app *bkmsapp.Application
		var appEnv *envmodel.Environment
		var appModel *appmodel.AppModel

		AfterEach(func() {
			if app != nil {
				_, _ = appConfigFileStore.DeleteByApp(ctx, app.ID)
			}
		})

		It("should render builtin vars, custom vars, legacy format, and runtime var placeholders", func() {
			// 覆盖：
			// - 新格式内置变量：${{env.BKMS_APP_NAME}}, ${{env.BKMS_ENV_NAME}}
			// - 新格式自定义变量：${{env.MY_REGION}}
			// - tRPC 运行时格式：${SOME_VAR} 原样保留
			// - 运行时变量占位符：${{env.BKMS_POD_IP}}, ${{env.BKMS_NODE_IP}}
			app, appModel = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
				AppStore:                  appStore,
				AppModelStore:             appModelStore,
				AppConfigFileStore:        appConfigFileStore,
				AppConfigFileVersionStore: appConfigFileVersionStore,
				BuildConfigStore:          buildConfigStore,
			}, &dbfactory.TrpcApplicationOpts{
				TrpcConfig: &appmodel.TrpcConfig{
					FileName:    "trpc_go.yaml",
					FilePath:    "/etc/trpc",
					Language:    "go",
					FileContent: "server:\n  app: ${{env.BKMS_APP_NAME}}\n  env: ${{env.BKMS_ENV_NAME}}\n  region: ${{env.MY_REGION}}\n  legacy_region: ${{env.MY_REGION}}\n  trpc_legacy: ${SOME_VAR}\n  pod_ip: ${{env.BKMS_POD_IP}}\n  node_ip: ${{env.BKMS_NODE_IP}}\n",
				},
				EnvVars: []appmodel.Variable{
					{Key: "MY_REGION", Value: "ap-guangzhou"},
				},
			})
			appEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			builder := workload.NewBuilder(builderSvc, app, appModel)
			result, err := builder.Build(ctx, appEnv)
			Expect(err).NotTo(HaveOccurred())
			extraObjs := result.ExtraObjects
			Expect(extraObjs).To(HaveLen(1))

			configMapData := extraObjs[0].Object["data"].(map[string]any)
			configContent := configMapData["trpc_go.yaml"].(string)
			expectYAMLValue := func(expected any, path ...any) {
				actual, err := testutil.YAMLValueAt(configContent, path...)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expected))
			}

			// 内置变量应被渲染为实际值
			expectYAMLValue(app.Name, "server", "app")
			expectYAMLValue(appEnv.Name, "server", "env")
			// 自定义变量（新格式 + 旧格式）都应渲染为实际值
			expectYAMLValue("ap-guangzhou", "server", "region")
			expectYAMLValue("ap-guangzhou", "server", "legacy_region")
			// tRPC 框架运行时格式原样保留下发
			expectYAMLValue("${SOME_VAR}", "server", "trpc_legacy")
			// 运行时变量在编译阶段应保留为占位符，由 Init Container 在 Pod 启动时替换
			expectYAMLValue("__#VAR_PLACEHOLDER#__BKMS_POD_IP__", "server", "pod_ip")
			expectYAMLValue("__#VAR_PLACEHOLDER#__BKMS_NODE_IP__", "server", "node_ip")
		})

		It("should return an error when the tRPC config template is invalid", func() {
			app, appModel = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
				AppStore:                  appStore,
				AppModelStore:             appModelStore,
				AppConfigFileStore:        appConfigFileStore,
				AppConfigFileVersionStore: appConfigFileVersionStore,
				BuildConfigStore:          buildConfigStore,
			}, &dbfactory.TrpcApplicationOpts{
				TrpcConfig: &appmodel.TrpcConfig{FileContent: "${{ env.BROKEN "},
			})
			appEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			_, err := workload.NewBuilder(builderSvc, app, appModel).Build(ctx, appEnv)

			Expect(err).To(MatchError(ContainSubstring("collecting env vars from tRPC config")))
		})
	})

	It("should collect undefined env vars from every workload render source", func() {
		app, appModel := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigFileVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			TrpcConfig: &appmodel.TrpcConfig{
				FileName:    "trpc_go.yaml",
				FilePath:    "/etc/trpc",
				Language:    "go",
				FileContent: "missing: ${{ env.MISSING }}\n",
			},
		})
		DeferCleanup(func() {
			_, _ = appConfigFileStore.DeleteByApp(ctx, app.ID)
			_ = polarisConfigStore.DeleteByApp(ctx, app.ID)
		})

		compDef := dbfactory.CompDef(ctx, componentDefStore, &dbfactory.ComponentDefOpts{
			Properties: []component.Property{{
				Name:         "value",
				Type:         component.PropTypeString,
				DefaultValue: "${{ env.MISSING }}",
			}},
			Patchers: []string{"{}\n"},
		})
		appModel.Components = []*component.Component{{
			Name: "missing-component",
			ComponentInst: component.ComponentInst{
				Type:    compDef.Name,
				Version: compDef.Version,
			},
		}}
		Expect(appModelStore.UpdateAppModel(ctx, appModel)).To(Succeed())

		appEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		Expect(polarisConfigStore.Create(ctx, &polaris.PolarisConfig{
			Name:  "polaris-main",
			AppID: app.ID,
			Properties: polaris.Properties{
				InstanceKey:      "inst-collect",
				PolarisName:      "collect-svc",
				PolarisNamespace: "collect-ns",
				PolarisToken:     "collect-token",
				ServicePort:      8080,
				ServiceLabels:    map[string]string{"missing": "${{ env.MISSING }}"},
			},
			ScopeType:     component.ScopeTypeEnvironment,
			ScopeEnvNames: []string{appEnv.Name},
		})).To(Succeed())

		result, err := workload.NewBuilder(builderSvc, app, appModel).Build(ctx, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UndefinedEnvVars).To(Equal([]envvarrefs.UndefinedEnvVar{{
			Key: "MISSING",
			Sources: []envvarrefs.Source{
				{Type: envvarrefs.SourceAppConfigFile, Name: appcfg.DefaultAppConfigFileName},
				{Type: envvarrefs.SourceComponent, Name: "missing-component"},
				{Type: envvarrefs.SourcePolaris, Name: "polaris-main"},
			},
		}}))
	})

	Context("Build with PolarisRegistryPatcher patching tRPC config file", func() {
		var app *bkmsapp.Application
		var testEnv *envmodel.Environment
		var appModel *appmodel.AppModel

		BeforeEach(func() {
			app, appModel = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
				AppStore:                  appStore,
				AppModelStore:             appModelStore,
				AppConfigFileStore:        appConfigFileStore,
				AppConfigFileVersionStore: appConfigFileVersionStore,
				BuildConfigStore:          buildConfigStore,
			}, &dbfactory.TrpcApplicationOpts{
				TrpcConfig: &appmodel.TrpcConfig{
					FileName:    "trpc_go.yaml",
					FilePath:    "/etc/trpc",
					Language:    "go",
					FileContent: "server:\n  app: myapp\n  timeout: 3000\n",
				},
			})

			testEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		})

		AfterEach(func() {
			if app != nil {
				_, _ = appConfigFileStore.DeleteByApp(ctx, app.ID)
				_ = polarisConfigStore.DeleteByApp(ctx, app.ID)
			}
		})

		It("should inject polaris registry into tRPC config when health check is enabled", func() {
			pc := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-patch",
					PolarisName:       "patch-svc",
					PolarisNamespace:  "patch-ns",
					PolarisToken:      "patch-token",
					ServicePort:       8080,
					EnableHealthCheck: true,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{testEnv.Name},
			}
			err := polarisConfigStore.Create(ctx, pc)
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)
			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())
			extraObjs := result.ExtraObjects

			// tRPC Plugin 应生成 ConfigMap（以及 polaris 组件产生的额外资源）
			Expect(len(extraObjs)).To(BeNumerically(">=", 1))

			// 找到 ConfigMap 并验证 tRPC 配置被 patcher 注入了 polaris registry
			var configContent string
			for _, obj := range extraObjs {
				if obj.GetKind() == "ConfigMap" {
					data := obj.Object["data"].(map[string]any)
					if v, ok := data["trpc_go.yaml"]; ok {
						configContent = v.(string)
					}
				}
			}
			Expect(configContent).NotTo(BeEmpty())

			var configMap map[string]any
			Expect(yaml.Unmarshal([]byte(configContent), &configMap)).To(Succeed())

			// 验证原有配置保留
			server := configMap["server"].(map[string]any)
			Expect(server["app"]).To(Equal("myapp"))

			// 验证 patcher 注入了 plugins.registry.polaris.service
			plugins := configMap["plugins"].(map[string]any)
			registry := plugins["registry"].(map[string]any)
			polarisMap := registry["polaris"].(map[string]any)
			serviceList := polarisMap["service"].([]any)
			Expect(serviceList).To(HaveLen(1))

			svc := serviceList[0].(map[string]any)
			Expect(svc["name"]).To(Equal("patch-svc"))
			Expect(svc["namespace"]).To(Equal("patch-ns"))
			Expect(svc["token"]).To(Equal("patch-token"))
		})

		It("should not patch tRPC config when polaris health check is disabled", func() {
			pc := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-no-hc",
					PolarisName:       "nohc-svc",
					PolarisNamespace:  "nohc-ns",
					PolarisToken:      "nohc-token",
					ServicePort:       8080,
					EnableHealthCheck: false,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{testEnv.Name},
			}
			err := polarisConfigStore.Create(ctx, pc)
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)
			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())
			extraObjs := result.ExtraObjects

			// 找到 ConfigMap 中的 tRPC 配置
			var configContent string
			for _, obj := range extraObjs {
				if obj.GetKind() == "ConfigMap" {
					data := obj.Object["data"].(map[string]any)
					if v, ok := data["trpc_go.yaml"]; ok {
						configContent = v.(string)
					}
				}
			}
			Expect(configContent).NotTo(BeEmpty())

			var configMap map[string]any
			Expect(yaml.Unmarshal([]byte(configContent), &configMap)).To(Succeed())

			// 未启用健康检查时，不应注入 plugins.registry.polaris
			_, hasPlugins := configMap["plugins"]
			Expect(hasPlugins).To(BeFalse(), "should not inject plugins when health check is disabled")
		})
	})
})
