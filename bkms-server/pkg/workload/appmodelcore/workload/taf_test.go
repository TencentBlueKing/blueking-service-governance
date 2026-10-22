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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// TafWorkloadBuilder 包含 TAF 特有的测试用例（如 environment-specific appConfigFile）
var _ = Describe("TafWorkloadBuilder", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var appStore *bkmsapp.ApplicationStore
	var envSvc *env.EnvService
	var builderSvc *workload.BuilderService
	var appConfigFileDefStore *appcfg.AppConfigFileDefStore

	BeforeEach(func() {
		ctx = context.Background()
		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		diApp = fxtest.New(
			GinkgoT(),
			workload.FxModule,
			appcfg.FxModule,
			env.FxModule,
			envvars.FxModule,
			depenvvars.FxModule,
			polarisenvvars.FxModule,
			polaris.FxModule,
			fx.Populate(
				&envSvc,
				&builderSvc,
			),
		)
		diApp.RequireStart()
		appConfigFileDefStore = testutil.NewStore(GinkgoT(), appcfg.AppConfigFileDefMeta)
	})

	AfterEach(func() {
		cleanupWorkspaces(ctx)
		diApp.RequireStop()
	})

	singleConfigMapContent := func(extraObjs []unstructured.Unstructured) string {
		for _, obj := range extraObjs {
			if obj.GetKind() != "ConfigMap" {
				continue
			}
			data := obj.Object["data"].(map[string]any)
			Expect(data).To(HaveLen(1))
			for _, v := range data {
				content, ok := v.(string)
				Expect(ok).To(BeTrue())
				return content
			}
		}
		Fail("config map not found")
		return ""
	}

	// TAF 特有测试：environment-specific appConfigFile
	Context("Test Build with environment-specific appConfigFile", func() {
		var app *bkmsapp.Application
		var prodEnv *envmodel.Environment
		var testEnv *envmodel.Environment
		var appModel *appmodel.AppModel
		var prodContent string

		BeforeEach(func() {
			// 使用 dbfactory.TafApplication 创建应用和模型
			// TafConfig.FileContent 会作为默认配置文件内容
			app, appModel = dbfactory.TafApplication(ctx, &dbfactory.TafApplicationStores{
				AppStore: appStore,
			}, &dbfactory.TafApplicationOpts{
				TafConfig: &appmodel.TafConfig{
					FileName:    "taf_config.conf",
					FilePath:    "/etc/taf",
					FileContent: "<taf>\n  <application>\n    <server>\n      logpath=/data/log\n      timeout=3000\n    </server>\n  </application>\n</taf>\n",
				},
				WorkspaceID: newWorkspaceID(ctx),
			})

			// Create two environments
			prodEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)
			testEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// 切换到独立配置后，为 prod 写入环境完整配置。TAF 框架文件使用 overwrite。
			defs, err := appcfg.ListDefsByApp(
				ctx,
				appConfigFileDefStore,
				app.ID,
				appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(defs).To(HaveLen(1))

			cfgSvc := appcfg.NewAppConfigFileService(
				testutil.NewDB(GinkgoT()),
			)

			// 切换为独立配置模式
			isUnified := false
			err = cfgSvc.UpdateAppCfgFileDef(ctx, &defs[0], appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        appcfg.CfgSystemUser,
			})
			Expect(err).NotTo(HaveOccurred())

			// 获取已有框架 def 的默认文件，在其上创建环境完整配置
			defaultFileWithDef, err := cfgSvc.GetDefaultFileWithDef(ctx, defs[0].ID)
			Expect(err).NotTo(HaveOccurred())

			prodContent = "<taf>\n  <application>\n    <server>\n      logpath=/data/prod/log\n    </server>\n  </application>\n</taf>\n"
			_, err = cfgSvc.CreateEnvInstance(ctx, *defaultFileWithDef, appcfg.CreateEnvInstanceParams{
				EnvName:     prodEnv.Name,
				Content:     &prodContent,
				Operator:    appcfg.CfgSystemUser,
				Description: "prod env content for test",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// Clean up app config files
			if app != nil {
				_, _ = store.Of(database.DB(), appcfg.AppConfigFileMeta).
					DeleteMany(ctx, appcfg.FilesByApp(app.ID))
			}
		})

		It("should use app-level default config when no env-specific config exists", func() {
			// Build for test environment (no specific config)
			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())
			gd := asGameDeployment(result)
			extraObjs := result.ExtraObjects

			// Should have one extra ConfigMap
			Expect(extraObjs).To(HaveLen(1))
			Expect(extraObjs[0].GetKind()).To(Equal("ConfigMap"))

			// Verify ConfigMap contains default content
			configContent := singleConfigMapContent(extraObjs)
			expectedDefaultContent := "<taf>\n  <application>\n    <server>\n" +
				"      logpath=/data/log\n      timeout=3000\n    </server>\n  </application>\n</taf>\n"
			Expect(configContent).To(Equal(expectedDefaultContent))

			// Verify volume mount exists
			Expect(gd.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := gd.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(1))
			Expect(container.VolumeMounts[0].Name).To(Equal("taf-config-rendered"))
			Expect(container.VolumeMounts[0].MountPath).To(Equal("/etc/taf/taf_config.conf"))

			// Verify init container exists for runtime variable rendering
			Expect(gd.Spec.Template.Spec.InitContainers).To(HaveLen(1))
			Expect(gd.Spec.Template.Spec.InitContainers[0].Name).To(Equal("taf-init"))
		})

		It("should use env-specific config when it exists", func() {
			// Build for prod environment (has specific config)
			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, prodEnv)
			Expect(err).NotTo(HaveOccurred())
			gd := asGameDeployment(result)
			extraObjs := result.ExtraObjects

			// Should have one extra ConfigMap
			Expect(extraObjs).To(HaveLen(1))
			Expect(extraObjs[0].GetKind()).To(Equal("ConfigMap"))

			// 环境内容整体替换默认内容，默认文件中的 timeout 不再保留。
			configContent := singleConfigMapContent(extraObjs)
			Expect(configContent).To(Equal(prodContent))

			// Verify volume mount exists
			Expect(gd.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := gd.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(1))
		})
	})

	// TAF 配置变量渲染测试
	Context("Test Build with template variable rendering in TAF config", func() {
		var app *bkmsapp.Application
		var appEnv *envmodel.Environment
		var appModel *appmodel.AppModel

		AfterEach(func() {
			if app != nil {
				_, _ = store.Of(database.DB(), appcfg.AppConfigFileMeta).
					DeleteMany(ctx, appcfg.FilesByApp(app.ID))
			}
		})

		It("should render builtin vars, custom vars, legacy format, and runtime var placeholders", func() {
			// 覆盖：
			// - 新格式内置变量：${{env.BKMS_APP_NAME}}, ${{env.BKMS_ENV_NAME}}
			// - 新格式自定义变量：${{env.MY_REGION}}
			// - 运行时变量占位符：${{env.BKMS_POD_IP}}, ${{env.BKMS_NODE_IP}}
			app, appModel = dbfactory.TafApplication(ctx, &dbfactory.TafApplicationStores{
				AppStore: appStore,
			}, &dbfactory.TafApplicationOpts{
				TafConfig: &appmodel.TafConfig{
					FileName:    "taf_config.conf",
					FilePath:    "/etc/taf",
					FileContent: "<taf>\n  app=${{env.BKMS_APP_NAME}}\n  env=${{env.BKMS_ENV_NAME}}\n  region=${{env.MY_REGION}}\n  legacy_region=${{env.MY_REGION}}\n  pod_ip=${{env.BKMS_POD_IP}}\n  node_ip=${{env.BKMS_NODE_IP}}\n</taf>\n",
				},
				EnvVars: []appmodel.Variable{
					{Key: "MY_REGION", Value: "ap-guangzhou"},
				},
				WorkspaceID: newWorkspaceID(ctx),
			})
			appEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			builder := workload.NewBuilder(builderSvc, app, appModel)
			result, err := builder.Build(ctx, appEnv)
			Expect(err).NotTo(HaveOccurred())
			extraObjs := result.ExtraObjects
			Expect(extraObjs).To(HaveLen(1))

			configContent := singleConfigMapContent(extraObjs)

			// 内置变量应被渲染为实际值
			Expect(configContent).To(ContainSubstring("app=" + app.Name))
			Expect(configContent).To(ContainSubstring("env=" + appEnv.Name))
			// 自定义变量（新格式 + 旧格式）都应渲染为实际值
			Expect(configContent).To(ContainSubstring("region=ap-guangzhou"))
			Expect(configContent).To(ContainSubstring("legacy_region=ap-guangzhou"))
			// 运行时变量在编译阶段应保留为占位符，由 Init Container 在 Pod 启动时替换
			Expect(configContent).To(ContainSubstring("pod_ip=__#VAR_PLACEHOLDER#__BKMS_POD_IP__"))
			Expect(configContent).To(ContainSubstring("node_ip=__#VAR_PLACEHOLDER#__BKMS_NODE_IP__"))
		})

		It("should collect undefined env vars from the TAF config", func() {
			app, appModel = dbfactory.TafApplication(ctx, &dbfactory.TafApplicationStores{
				AppStore: appStore,
			}, &dbfactory.TafApplicationOpts{
				TafConfig: &appmodel.TafConfig{
					FileName:    "taf_config.conf",
					FilePath:    "/etc/taf",
					FileContent: "missing=${{ env.MISSING }}\n",
				},
				WorkspaceID: newWorkspaceID(ctx),
			})
			appEnv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			result, err := workload.NewBuilder(builderSvc, app, appModel).Build(ctx, appEnv)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.UndefinedEnvVars).To(Equal([]envvarrefs.UndefinedEnvVar{{
				Key: "MISSING",
				Sources: []envvarrefs.Source{{
					Type: envvarrefs.SourceAppConfigFile,
					Name: "taf_config.conf",
				}},
			}}))
		})
	})
})
