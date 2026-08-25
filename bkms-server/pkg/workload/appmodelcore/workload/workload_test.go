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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/secret"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	componentdevmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model/initdata"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// WorkloadTestCase 定义参数化测试用例
type WorkloadTestCase struct {
	Name             string
	AppType          string // bkmsapp.AppTypeTRPC or bkmsapp.AppTypeTAF
	WorkloadType     string // appmodel.WorkloadTypeTrpc or appmodel.WorkloadTypeTaf
	ConfigFileName   string
	ConfigFilePath   string
	ConfigContent    string
	ConfigVolumeName string // 配置模板 volume 名称，如 "trpc-config-template" 或 "taf-config-template"
	ExtraVolumeCount int    // plugin 产生的额外 volume 数量（如 TAF 有 emptyDir 用于渲染）
	FileFormat       appcfg.FileFormat
}

// 定义测试用例
var workloadTestCases = []WorkloadTestCase{
	{
		Name:             "TRPC",
		AppType:          bkmsapp.AppTypeTRPC,
		WorkloadType:     appmodel.WorkloadTypeTrpc,
		ConfigFileName:   "trpc_config.yaml",
		ConfigFilePath:   "/etc/",
		ConfigContent:    "todo:\n 1",
		ConfigVolumeName: "trpc-config-template",
		ExtraVolumeCount: 1,
		FileFormat:       appcfg.FileFormatYAML,
	},
	{
		Name:             "TAF",
		AppType:          bkmsapp.AppTypeTAF,
		WorkloadType:     appmodel.WorkloadTypeTaf,
		ConfigFileName:   "taf_config.conf",
		ConfigFilePath:   "/etc/",
		ConfigContent:    "<taf>\n  <application>\n    <server>\n      logpath=/data/log\n    </server>\n  </application>\n</taf>\n",
		ConfigVolumeName: "taf-config-template",
		ExtraVolumeCount: 1,
		FileFormat:       appcfg.FileFormatTAF,
	},
}

// SharedTestStores 聚合共享测试依赖，避免测试主体到处传递 store 参数。
type SharedTestStores struct {
	AppStore                  bkmsapp.ApplicationStore
	AppModelStore             appmodel.AppModelStore
	CompDefStore              component.ComponentDefStore
	WorkspaceCompsStore       workspace.WorkspaceCompsStore
	PolarisConfigStore        polaris.PolarisConfigStore
	AppSpecStore              appspec.AppSpecStore
	AppConfigFileStore        appcfg.AppConfigFileStore
	ScopedEnvVarStore         envvars.ScopedEnvVarStore
	AppConfigFileVersionStore appcfg.AppConfigFileVersionStore
	BuildConfigStore          build.ConfigStore
	SvcStore                  depmodel.ServiceStore
	SvcInstStore              depmodel.ServiceInstanceStore
	BindingStore              depmodel.ServiceBindingStore
}

// createApplication 根据测试用例创建应用
func createApplication(
	ctx context.Context,
	tc WorkloadTestCase,
	stores *SharedTestStores,
	envVars []appmodel.Variable,
	components []*component.Component,
) (*bkmsapp.Application, *appmodel.AppModel) {
	switch tc.AppType {
	case bkmsapp.AppTypeTRPC:
		trpcConfig := &appmodel.TrpcConfig{
			FileName:    tc.ConfigFileName,
			FilePath:    tc.ConfigFilePath,
			FileContent: tc.ConfigContent,
		}
		return dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  stores.AppStore,
			AppModelStore:             stores.AppModelStore,
			AppConfigFileStore:        stores.AppConfigFileStore,
			AppConfigFileVersionStore: stores.AppConfigFileVersionStore,
			BuildConfigStore:          stores.BuildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			TrpcConfig: trpcConfig,
			EnvVars:    envVars,
			Components: components,
		})
	case bkmsapp.AppTypeTAF:
		tafConfig := &appmodel.TafConfig{
			FileName:    tc.ConfigFileName,
			FilePath:    tc.ConfigFilePath,
			FileContent: tc.ConfigContent,
		}
		return dbfactory.TafApplication(ctx, &dbfactory.TafApplicationStores{
			AppStore:                  stores.AppStore,
			AppModelStore:             stores.AppModelStore,
			AppConfigFileStore:        stores.AppConfigFileStore,
			AppConfigFileVersionStore: stores.AppConfigFileVersionStore,
			BuildConfigStore:          stores.BuildConfigStore,
		}, &dbfactory.TafApplicationOpts{
			TafConfig:  tafConfig,
			EnvVars:    envVars,
			Components: components,
		})
	default:
		return nil, nil
	}
}

var _ = Describe("Builder Shared Tests", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var stores *SharedTestStores
	var envSvc *env.EnvService
	var builderSvc *workload.BuilderService

	BeforeEach(func() {
		ctx = context.Background()
		stores = &SharedTestStores{}
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appmodel.FxModule,
			workload.FxModule,
			component.FxModule,
			polaris.FxModule,
			appspec.FxModule,
			appcfg.FxModule,
			env.FxModule,
			envvars.FxModule,
			workspace.FxModule,
			build.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(
				&stores.AppStore,
				&stores.AppModelStore,
				&stores.CompDefStore,
				&stores.WorkspaceCompsStore,
				&stores.PolarisConfigStore,
				&stores.AppSpecStore,
				&stores.AppConfigFileStore,
				&stores.AppConfigFileVersionStore,
				&stores.BuildConfigStore,
				&stores.ScopedEnvVarStore,
				&stores.SvcStore,
				&stores.SvcInstStore,
				&stores.BindingStore,
				&envSvc,
				&builderSvc,
			),
		)
		diApp.RequireStart()

		// Load builtin components
		dbfactory.LoadBuiltinComponents(ctx, database.Client(), "../../../extension/component/assets/comps")

		// Seed depservice init data (mysql, polaris, fake service definitions)
		Expect(initdata.Do(stores.SvcStore)).To(Succeed())
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	// Test Build Basic
	DescribeTable("Build Basic - check basic properties",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores,
				[]appmodel.Variable{{Key: "APP_VAR", Value: "app_var"}},
				nil,
			)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// Add env vars
			_, err := stores.ScopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *testEnv, "ENV_VAR", "env_var", "")
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)
			extraObjs := result.ExtraObjects

			// Check basic properties
			Expect(extraObjs).To(HaveLen(1))
			Expect(gd.Name).To(Equal(appModel.Workload.Name))
			Expect(gd.Spec.Template.Spec.Containers[0].Image).To(Equal(appModel.Workload.Image))
			Expect(gd.Spec.Template.Spec.Containers).To(HaveLen(1))
			// Check no UpdateStrategy set by default
			Expect(string(gd.Spec.UpdateStrategy.Type)).To(Equal(""))
			Expect(gd.Spec.UpdateStrategy.MaxUnavailable.String()).To(Equal("25%"))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build Basic - should apply env appspec override",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// The environment override changes only replicas and memory request.
			// Set the AppModel fields expected to remain unchanged explicitly.
			appModel.Replicas = lo.ToPtr(int32(1))
			appModel.UpdateStrategy = &appmodel.UpdateStrategy{
				MaxUnavailable: lo.ToPtr("25%"),
				MaxSurge:       lo.ToPtr("25%"),
			}
			appModel.Workload.Resources = map[string]string{
				"cpu":    "1-2",
				"memory": "2Gi-4Gi",
			}
			err := stores.AppModelStore.UpdateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			err = stores.AppSpecStore.Create(ctx, &appspec.AppSpec{
				AppID:   app.ID,
				EnvName: testEnv.Name,
				Resources: &appspec.ResourcesSpec{
					Replicas:       lo.ToPtr(int32(4)),
					MemoryRequests: lo.ToPtr("64Mi"),
				},
			})
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			Expect(*gd.Spec.Replicas).To(Equal(int32(4)))
			Expect(gd.Spec.UpdateStrategy.MaxUnavailable.String()).To(Equal("25%"), "should keep default value")

			container := gd.Spec.Template.Spec.Containers[0]
			Expect(container.Resources.Requests.Memory().String()).To(Equal("64Mi"))
			// Memory limits should stay intact as default when the env override omits them.
			Expect(container.Resources.Limits.Memory().String()).To(Equal("4Gi"))
			// Cpu resources should stay intact as default
			Expect(container.Resources.Requests.Cpu().String()).To(Equal("1"))
			Expect(container.Resources.Limits.Cpu().String()).To(Equal("2"))

			// Check another env that has no override
			env2 := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
			result2, err := builder.Build(ctx, env2)
			Expect(err).NotTo(HaveOccurred())

			gd2 := asGameDeployment(result2)
			Expect(*gd2.Spec.Replicas).To(Equal(int32(1)))

			container2 := gd2.Spec.Template.Spec.Containers[0]
			Expect(container2.Resources.Requests.Cpu().String()).To(Equal("1"))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build Basic - should apply env appspec update strategy",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// Replicas is outside the update-strategy override and should be preserved.
			appModel.Replicas = lo.ToPtr(int32(1))
			err := stores.AppModelStore.UpdateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			err = stores.AppSpecStore.Create(ctx, &appspec.AppSpec{
				AppID:   app.ID,
				EnvName: testEnv.Name,
				UpdateStrategy: &appspec.UpdateStrategySpec{
					MaxUnavailable: lo.ToPtr("0"),
					MaxSurge:       lo.ToPtr("1"),
				},
			})
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)
			Expect(*gd.Spec.Replicas).To(Equal(int32(1)))
			Expect(gd.Spec.UpdateStrategy.MaxUnavailable.String()).To(Equal("0"))
			Expect(gd.Spec.UpdateStrategy.MaxSurge.String()).To(Equal("1"))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	It("should enable dev mode from appspec in non-production env", func() {
		app, appModel := createApplication(ctx, workloadTestCases[0], stores, nil, nil)
		testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		testEnv.Type = componentdevmode.EnvTypeDevelopment
		appModel.Workload.Command = []string{"./server"}

		err := stores.AppSpecStore.Create(ctx, &appspec.AppSpec{
			AppID:   app.ID,
			EnvName: testEnv.Name,
			DevMode: &appspec.DevModeSpec{
				Enabled: lo.ToPtr(true),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		builder := workload.NewBuilder(builderSvc, app, appModel)

		result, err := builder.Build(ctx, testEnv)
		Expect(err).NotTo(HaveOccurred())
		gd := asGameDeployment(result)
		extraObjs := result.ExtraObjects
		Expect(extraObjs).To(HaveLen(2))
		Expect(lo.Map(extraObjs, func(obj unstructured.Unstructured, _ int) string {
			return obj.GetName()
		})).To(ContainElement(componentdevmode.ConfigMapResourceName(appModel.Workload.Name)))
		Expect(gd.Spec.Template.Spec.Containers[0].Command).To(Equal([]string{componentdevmode.TrpcInitScriptPath}))
		Expect(
			gd.Spec.Template.Spec.Volumes,
		).To(ContainElement(HaveField("Name", Equal(componentdevmode.ConfigMapResourceName(appModel.Workload.Name)))))
	})

	It("should ignore dev mode from appspec in production env", func() {
		app, appModel := createApplication(ctx, workloadTestCases[0], stores, nil, nil)
		testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		testEnv.Type = componentdevmode.EnvTypeProduction
		appModel.Workload.Command = []string{"./server"}

		err := stores.AppSpecStore.Create(ctx, &appspec.AppSpec{
			AppID:   app.ID,
			EnvName: testEnv.Name,
			DevMode: &appspec.DevModeSpec{
				Enabled: lo.ToPtr(true),
			},
		})
		Expect(err).NotTo(HaveOccurred())

		builder := workload.NewBuilder(builderSvc, app, appModel)

		result, err := builder.Build(ctx, testEnv)
		Expect(err).NotTo(HaveOccurred())
		gd := asGameDeployment(result)
		extraObjs := result.ExtraObjects
		Expect(extraObjs).To(HaveLen(1))
		Expect(gd.Spec.Template.Spec.Containers[0].Command).To(Equal([]string{"./server"}))
		Expect(gd.Spec.Template.Spec.Volumes).NotTo(
			ContainElement(HaveField("Name", Equal(componentdevmode.ConfigMapResourceName(appModel.Workload.Name)))),
		)
	})

	DescribeTable("Build Basic - should include all necessary environment variables",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores,
				[]appmodel.Variable{{Key: "APP_VAR", Value: "app_var"}},
				nil,
			)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// Add env vars
			_, err := stores.ScopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *testEnv, "ENV_VAR", "env_var", "")
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)
			extraObjs := result.ExtraObjects

			Expect(extraObjs).To(HaveLen(1))
			// Build the map from env vars for easy checking
			envMap := make(map[string]string)
			for _, envVar := range gd.Spec.Template.Spec.Containers[0].Env {
				envMap[envVar.Name] = envVar.Value
			}
			// Check all kinds of env vars are included
			Expect(envMap).To(HaveKeyWithValue("APP_VAR", "app_var"))
			Expect(envMap).To(HaveKeyWithValue("ENV_VAR", "env_var"))
			Expect(envMap).To(HaveKeyWithValue("BKMS_APP_NAME", app.Name))
			Expect(envMap).To(HaveKeyWithValue("BKMS_ENV_CLUSTER", testEnv.Cluster.ClusterID))
			Expect(envMap).To(HaveKey("BKMS_POD_IP"))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build Basic - should append default image pull secret ref",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			Expect(gd.Spec.Template.Spec.ImagePullSecrets).To(ContainElement(corev1.LocalObjectReference{
				Name: secret.ResolveImagePullSecretName(testEnv.WorkspaceID, "", nil),
			}))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	// Test Build Components
	DescribeTable("Build Components - ListComponents should return the right result",
		func(tc WorkloadTestCase) {
			// Create a test componentDef first
			compDef := dbfactory.CompDef(ctx, stores.CompDefStore, &dbfactory.ComponentDefOpts{
				Properties: []component.Property{
					{Name: "replicas", Type: "INT", DefaultValue: int64(1)},
				},
				Patchers: []string{"spec:\n  replicas: {{ .replicas }}\n"},
			})

			app, appModel := createApplication(
				ctx,
				tc,
				stores,
				nil,
				[]*component.Component{{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"replicas": int64(3),
						},
					},
				}},
			)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			builder := workload.NewBuilder(builderSvc, app, appModel)

			comps, err := builder.ListComponents(ctx, testEnv, appModel)
			Expect(err).NotTo(HaveOccurred())
			Expect(comps).To(HaveLen(1))
			Expect(comps[0].Type).To(Equal(compDef.Name))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build Components - the result should be patched/extended by the components",
		func(tc WorkloadTestCase) {
			compDef := dbfactory.CompDef(ctx, stores.CompDefStore, &dbfactory.ComponentDefOpts{
				Properties: []component.Property{
					{Name: "replicas", Type: "INT", DefaultValue: int64(1)},
				},
				Patchers: []string{"spec:\n  replicas: {{ .replicas }}\n"},
			})

			app, appModel := createApplication(
				ctx,
				tc,
				stores,
				nil,
				[]*component.Component{{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"replicas": int64(3),
						},
					},
				}},
			)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)
			extraObjs := result.ExtraObjects

			By("Check components got applied")
			Expect(gd.Name).To(Equal(appModel.Workload.Name))
			Expect(gd.Spec.Template.Spec.Volumes[0].Name).To(Equal(tc.ConfigVolumeName))
			Expect(gd.Spec.Replicas).To(Equal(lo.ToPtr(int32(3))))

			By("Should return extra objects generated by the config component")
			Expect(extraObjs).To(HaveLen(1))
			Expect(extraObjs[0].GetKind()).To(Equal("ConfigMap"))
			Expect(extraObjs[0].GetName()).To(Equal(app.Name))

			// Convert extraObjs[0] to ConfigMap and check its data
			var configMap corev1.ConfigMap
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(extraObjs[0].Object, &configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Data[tc.ConfigFileName]).To(Equal(tc.ConfigContent))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build Components - should normalize extra objects before DeepCopy",
		func(tc WorkloadTestCase) {
			compDef := dbfactory.CompDef(ctx, stores.CompDefStore, &dbfactory.ComponentDefOpts{
				Patchers: []string{"spec:\n  replicas: 1\n"},
				Specs: []string{`apiVersion: v1
kind: Service
metadata:
  name: deep-copy-check
spec:
  ports:
    - port: 80
      targetPort: 8080
  selector:
    app: deep-copy-check
`},
			})

			app, appModel := createApplication(
				ctx,
				tc,
				stores,
				nil,
				[]*component.Component{{
					Name: "int-extra",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
					},
				}},
			)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			serviceObj, found := lo.Find(result.ExtraObjects, func(obj unstructured.Unstructured) bool {
				return obj.GetKind() == "Service" && obj.GetName() == "deep-copy-check"
			})
			Expect(found).To(BeTrue())
			Expect(func() {
				_ = serviceObj.DeepCopy()
			}).NotTo(Panic())

			ports, found, err := unstructured.NestedSlice(serviceObj.Object, "spec", "ports")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(ports[0].(map[string]any)["targetPort"]).To(Equal(int64(8080)))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build Components - should resolve referenced global workspace components",
		func(tc WorkloadTestCase) {
			compDef := dbfactory.CompDef(ctx, stores.CompDefStore, &dbfactory.ComponentDefOpts{
				Properties: []component.Property{
					{Name: "replicas", Type: "INT", DefaultValue: int64(1)},
				},
				Patchers: []string{"spec:\n  replicas: {{ .replicas }}\n"},
			})

			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			workspaceCompName := "workspace-replicas-foo"
			err := stores.WorkspaceCompsStore.Add(ctx, &workspace.Component{
				Name:        workspaceCompName,
				WorkspaceID: testEnv.WorkspaceID,
				ScopeType:   component.ScopeTypeGlobal,
				ComponentInst: component.ComponentInst{
					Type:    compDef.Name,
					Version: compDef.Version,
					Properties: map[string]any{
						"replicas": int64(12),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Update app model to reference environment component.
			appModel.Components = []*component.Component{{
				ComponentRef: component.ComponentRef{
					RefWorkspaceCompName: workspaceCompName,
				},
			}}

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)
			By("Check the referenced workspace component got applied")
			Expect(gd.Spec.Replicas).To(Equal(lo.ToPtr(int32(12))))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build Components - should resolve referenced environment scope workspace components",
		func(tc WorkloadTestCase) {
			compDef := dbfactory.CompDef(ctx, stores.CompDefStore, &dbfactory.ComponentDefOpts{
				Properties: []component.Property{
					{Name: "replicas", Type: "INT", DefaultValue: int64(1)},
				},
				Patchers: []string{"spec:\n  replicas: {{ .replicas }}\n"},
			})

			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			workspaceCompName := "workspace-replicas-foo"
			workspaceCompNameEnv1 := workspaceCompName + "-env1"
			workspaceCompNameEnv2 := workspaceCompName + "-env2"
			err := stores.WorkspaceCompsStore.Add(ctx, &workspace.Component{
				Name:          workspaceCompNameEnv1,
				WorkspaceID:   testEnv.WorkspaceID,
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{testEnv.Name},
				ComponentInst: component.ComponentInst{
					Type:    compDef.Name,
					Version: compDef.Version,
					Properties: map[string]any{
						"replicas": int64(12),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			err = stores.WorkspaceCompsStore.Add(ctx, &workspace.Component{
				Name:          workspaceCompNameEnv2,
				WorkspaceID:   testEnv.WorkspaceID,
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{testEnv.Name + "-env2"},
				ComponentInst: component.ComponentInst{
					Type:    compDef.Name,
					Version: compDef.Version,
					Properties: map[string]any{
						"replicas": int64(15),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Update app model to reference environment component.
			appModel.Components = []*component.Component{{
				ComponentRef: component.ComponentRef{
					RefWorkspaceCompName: workspaceCompNameEnv1,
				},
			}, {
				ComponentRef: component.ComponentRef{
					RefWorkspaceCompName: workspaceCompNameEnv2,
				},
			}}

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)
			By("Check the referenced workspace component got applied")
			Expect(gd.Spec.Replicas).To(Equal(lo.ToPtr(int32(12))))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	// Test Build Resources
	DescribeTable("Build Resources - should not set container requests and limits when not specified",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			c := gd.Spec.Template.Spec.Containers[0]
			Expect(c.Resources.Requests).To(BeNil())
			Expect(c.Resources.Limits).To(BeNil())
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build Resources - should set container requests and limits",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			appModel.Workload.Resources = map[string]string{
				"cpu":    "100m-200m",
				"memory": "512Mi",
			}
			err := stores.AppModelStore.UpdateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			c := gd.Spec.Template.Spec.Containers[0]
			Expect(c.Resources.Requests).NotTo(BeNil())
			Expect(c.Resources.Limits).NotTo(BeNil())
			Expect(lo.ToPtr(c.Resources.Requests[corev1.ResourceCPU]).Cmp(resource.MustParse("100m"))).To(BeZero())
			Expect(lo.ToPtr(c.Resources.Limits[corev1.ResourceCPU]).Cmp(resource.MustParse("200m"))).To(BeZero())
			Expect(lo.ToPtr(c.Resources.Requests[corev1.ResourceMemory]).Cmp(resource.MustParse("512Mi"))).To(BeZero())
			Expect(lo.ToPtr(c.Resources.Limits[corev1.ResourceMemory]).Cmp(resource.MustParse("512Mi"))).To(BeZero())
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	// Test Misc Parts
	DescribeTable("Misc Parts - should have no probes by default",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			c := gd.Spec.Template.Spec.Containers[0]
			Expect(c.LivenessProbe).To(BeNil())
			Expect(c.ReadinessProbe).To(BeNil())
			Expect(c.StartupProbe).To(BeNil())
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Misc Parts - should set liveness probe",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			appModel.Workload.LivenessProbe = &appmodel.Probe{
				ProbeHandler: &appmodel.ProbeHandler{
					TypeWrapper: appmodel.TypeWrapper{Type: appmodel.ProbeTypeExec},
					ExecAction:  &appmodel.ExecAction{Command: []string{"echo", "ok"}},
				},
				InitialDelaySeconds: 5,
			}
			err := stores.AppModelStore.UpdateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			c := gd.Spec.Template.Spec.Containers[0]
			Expect(c.LivenessProbe).NotTo(BeNil())
			Expect(c.LivenessProbe.Exec.Command).To(Equal([]string{"echo", "ok"}))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Misc Parts - should apply misc(replicas, pull settings, and lifecycle)",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			appModel.Replicas = lo.ToPtr(int32(2))
			appModel.Workload.ImagePullPolicy = appmodel.PullIfNotPresent
			appModel.Workload.ImagePullSecrets = []string{"secret-a", "secret-b"}
			appModel.Workload.Lifecycle = &appmodel.Lifecycle{
				PostStart: &appmodel.LifecycleHandler{
					TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeHTTP},
					HTTPGetAction: &appmodel.HTTPGetAction{
						URL:  "http://example.com/ready",
						Port: 8080,
					},
				},
				PreStop: &appmodel.LifecycleHandler{
					TypeWrapper: appmodel.TypeWrapper{Type: appmodel.LifecycleTypeExec},
					ExecAction:  &appmodel.ExecAction{Command: []string{"sh", "-c", "sleep 1"}},
				},
			}
			err := stores.AppModelStore.UpdateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			// Assert misc settings got applied
			Expect(gd.Spec.Replicas).To(Equal(lo.ToPtr(int32(2))))
			Expect(gd.Spec.Template.Spec.ImagePullSecrets).To(Equal([]corev1.LocalObjectReference{
				{Name: "secret-a"},
				{Name: "secret-b"},
				{Name: secret.ResolveImagePullSecretName(testEnv.WorkspaceID, "", nil)},
			}))

			c := gd.Spec.Template.Spec.Containers[0]
			Expect(c.ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))
			Expect(c.Lifecycle).NotTo(BeNil())
			Expect(c.Lifecycle.PreStop).NotTo(BeNil())
			Expect(c.Lifecycle.PreStop.Exec.Command).To(Equal([]string{"sh", "-c", "sleep 1"}))
			Expect(c.Lifecycle.PostStart).NotTo(BeNil())
			Expect(c.Lifecycle.PostStart.HTTPGet.Path).To(Equal("/ready"))
			Expect(c.Lifecycle.PostStart.HTTPGet.Port.IntValue()).To(Equal(8080))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Misc Parts - should set terminationGracePeriodSeconds",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			appModel.Workload.TerminationGracePeriodSeconds = lo.ToPtr(int64(60))
			err := stores.AppModelStore.UpdateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			Expect(gd.Spec.Template.Spec.TerminationGracePeriodSeconds).To(Equal(lo.ToPtr(int64(60))))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Misc Parts - should apply volume mounts",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			appModel.Workload.VolumeMounts = appmodel.VolumeMounts{
				HostPath: []appmodel.VolumeMountHostPath{
					{
						MountPath: "/data",
						HostPath:  "/var/lib/data",
						Type:      "DirectoryOrCreate",
					},
					{
						MountPath: "/cache",
						HostPath:  "/var/lib/cache",
					},
				},
			}
			err := stores.AppModelStore.UpdateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			c := gd.Spec.Template.Spec.Containers[0]
			// The last volume ...[2] should be the one set by config component
			Expect(c.VolumeMounts).To(HaveLen(3))
			Expect(c.VolumeMounts[0].Name).To(Equal("hostpath-0"))
			Expect(c.VolumeMounts[0].MountPath).To(Equal("/data"))
			Expect(c.VolumeMounts[1].Name).To(Equal("hostpath-1"))
			Expect(c.VolumeMounts[1].MountPath).To(Equal("/cache"))

			Expect(gd.Spec.Template.Spec.Volumes).To(HaveLen(3 + tc.ExtraVolumeCount))
			Expect(gd.Spec.Template.Spec.Volumes[0].HostPath.Path).To(Equal("/var/lib/data"))
			Expect(*gd.Spec.Template.Spec.Volumes[0].HostPath.Type).To(Equal(corev1.HostPathDirectoryOrCreate))
			Expect(gd.Spec.Template.Spec.Volumes[1].HostPath.Path).To(Equal("/var/lib/cache"))
			Expect(*gd.Spec.Template.Spec.Volumes[1].HostPath.Type).To(Equal(corev1.HostPathDirectoryOrCreate))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Misc Parts - should set UpdateStrategy",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			appModel.UpdateStrategy = &appmodel.UpdateStrategy{
				Type:           "InplaceUpdate",
				MaxUnavailable: lo.ToPtr("2"),
				MaxSurge:       lo.ToPtr("50%"),
			}
			err := stores.AppModelStore.UpdateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			strategy := gd.Spec.UpdateStrategy
			Expect(string(strategy.Type)).To(Equal("InplaceUpdate"))
			Expect(strategy.MaxUnavailable.String()).To(Equal("2"))
			Expect(strategy.MaxSurge.String()).To(Equal("50%"))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	// Test Build with PolarisConfig
	DescribeTable("Build with PolarisConfig - should render polaris component with correct environment variables",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// Add env vars
			_, err := stores.ScopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *testEnv, "ENV_VAR", "env_var", "")
			Expect(err).NotTo(HaveOccurred())

			// Create a polaris config for the app
			polarisConfig := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "mypolaris",
					PolarisName:       "test-service",
					PolarisNamespace:  "Test",
					PolarisToken:      "test-token-12345",
					ServicePort:       8080,
					Direct:            true,
					KeepNotReadyPod:   true,
					EnableHealthCheck: false,
					ServiceLabels:     map[string]string{"env": "test", "env-var": "${{env.ENV_VAR}}"},
				},
				ScopeEnvNames: []string{testEnv.Name},
				EnvWeights:    map[string]int32{testEnv.Name: 10},
			}
			err = stores.PolarisConfigStore.Create(ctx, polarisConfig)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = stores.PolarisConfigStore.DeleteByApp(ctx, app.ID) }()

			builder := workload.NewBuilder(builderSvc, app, appModel)

			// Build workload with polaris config store
			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)
			extraObjs := result.ExtraObjects

			// Verify environment variables are added
			envMap := make(map[string]string)
			for _, envVar := range gd.Spec.Template.Spec.Containers[0].Env {
				envMap[envVar.Name] = envVar.Value
			}
			for _, varObj := range polarisConfig.GetVars() {
				Expect(envMap).To(HaveKeyWithValue(varObj.Key, varObj.Value))
			}

			// Verify extra objects are generated (PolarisConfig and Service)
			// Plus config component generates a ConfigMap
			Expect(len(extraObjs)).To(BeNumerically(">=", 3))

			// 验证 PolarisConfig
			polarisConfigObj, found := lo.Find(extraObjs, func(obj unstructured.Unstructured) bool {
				return obj.GetKind() == "PolarisConfig"
			})
			Expect(found).To(BeTrue(), "PolarisConfig resource should be generated")
			expectedBaseName := strings.ToLower(app.Name + "-" + polarisConfig.Name)
			Expect(polarisConfigObj.GetName()).To(Equal(expectedBaseName + "-polaris"))
			Expect(polarisConfigObj.GetAPIVersion()).To(Equal("tkex.tencent.com/v1"))
			// 验证环境上的变量被正确渲染进配置中
			Expect(
				polarisConfigObj.Object["spec"].(map[string]any)["services"].([]any)[0].(map[string]any)["extraMeta"].(map[string]any)["env-var"],
			).To(Equal("env_var"))
			// services[0].name 与模板 {{ .name }}-polaris-service 一致
			Expect(
				polarisConfigObj.Object["spec"].(map[string]any)["services"].([]any)[0].(map[string]any)["name"],
			).To(Equal(expectedBaseName + "-polaris-service"))

			// 验证 Service（命名与模板 {{ .name }}-polaris-service 一致）
			serviceObj, foundServiceObj := lo.Find(extraObjs, func(obj unstructured.Unstructured) bool {
				return obj.GetKind() == "Service" && obj.GetName() == expectedBaseName+"-polaris-service"
			})
			Expect(foundServiceObj).To(BeTrue(), "Service resource should be generated")
			// Service selector 与模板 {{ .bkmsAppName }} 一致
			Expect(
				serviceObj.Object["spec"].(map[string]any)["selector"].(map[string]any)["app.kubernetes.io/name"],
			).To(Equal(app.Name))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build with PolarisConfig - should only include polaris configs available in current environment",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// Create a polaris config only available in a different environment
			polarisConfig := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:      "otherpolaris",
					PolarisName:      "other-service",
					PolarisNamespace: "Production",
					PolarisToken:     "other-token",
					ServicePort:      9090,
				},
				ScopeEnvNames: []string{"other-env"}, // Not current env
			}
			err := stores.PolarisConfigStore.Create(ctx, polarisConfig)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = stores.PolarisConfigStore.DeleteByApp(ctx, app.ID) }()

			builder := workload.NewBuilder(builderSvc, app, appModel)

			// Build workload
			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			// Verify the polaris config env vars are NOT added (since it's not available in this env)
			envMap := make(map[string]string)
			for _, envVar := range gd.Spec.Template.Spec.Containers[0].Env {
				envMap[envVar.Name] = envVar.Value
			}
			Expect(envMap).NotTo(HaveKey("otherpolaris_polarisToken"))
			Expect(envMap).NotTo(HaveKey("otherpolaris_serviceport"))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable(
		"Build with PolarisConfig - should include polaris configs with environment scope matching current env",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// Create a polaris config available in current environment
			polarisConfig := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:      "envpolaris",
					PolarisName:      "env-service",
					PolarisNamespace: "Test",
					PolarisToken:     "env-token-123",
					ServicePort:      7070,
				},
				ScopeEnvNames: []string{testEnv.Name}, // Current env
			}
			err := stores.PolarisConfigStore.Create(ctx, polarisConfig)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = stores.PolarisConfigStore.DeleteByApp(ctx, app.ID) }()

			builder := workload.NewBuilder(builderSvc, app, appModel)

			// Build workload
			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			// Verify the polaris config env vars ARE added
			envMap := make(map[string]string)
			for _, envVar := range gd.Spec.Template.Spec.Containers[0].Env {
				envMap[envVar.Name] = envVar.Value
			}
			Expect(envMap).To(HaveKeyWithValue("envpolaris_polarisToken", "env-token-123"))
			Expect(envMap).To(HaveKeyWithValue("envpolaris_serviceport", "7070"))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	DescribeTable("Build with PolarisConfig - should handle multiple polaris configs",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// Create two polaris configs
			config1 := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:      "polaris1",
					PolarisName:      "service1",
					PolarisNamespace: "Test",
					PolarisToken:     "token1",
					ServicePort:      8081,
				},
				ScopeEnvNames: []string{testEnv.Name},
			}
			config2 := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:      "polaris2",
					PolarisName:      "service2",
					PolarisNamespace: "Production",
					PolarisToken:     "token2",
					ServicePort:      8082,
				},
				ScopeEnvNames: []string{testEnv.Name},
			}
			err := stores.PolarisConfigStore.Create(ctx, config1)
			Expect(err).NotTo(HaveOccurred())
			err = stores.PolarisConfigStore.Create(ctx, config2)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = stores.PolarisConfigStore.DeleteByApp(ctx, app.ID) }()

			builder := workload.NewBuilder(builderSvc, app, appModel)

			// Build workload
			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			// Verify both polaris configs' env vars are added
			envMap := make(map[string]string)
			for _, envVar := range gd.Spec.Template.Spec.Containers[0].Env {
				envMap[envVar.Name] = envVar.Value
			}
			Expect(envMap).To(HaveKeyWithValue("polaris1_polarisToken", "token1"))
			Expect(envMap).To(HaveKeyWithValue("polaris1_serviceport", "8081"))
			Expect(envMap).To(HaveKeyWithValue("polaris2_polarisToken", "token2"))
			Expect(envMap).To(HaveKeyWithValue("polaris2_serviceport", "8082"))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)

	// Test Build with DepServiceInstance
	DescribeTable("Build with DepServiceInstance - should include dep service instance env vars in workload",
		func(tc WorkloadTestCase) {
			app, appModel := createApplication(ctx, tc, stores, nil, nil)
			testEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			// Create a fake service instance and bind it to the app for this env.
			// Binding EnvVars 把 Credentials 按 ${{env.KEY}} 直出，并附带 MYSQL_DSN 衍生变量。
			inst := &depmodel.ServiceInstance{
				Name:         "fake-mysql-inst",
				ServiceName:  "fake",
				PlanName:     "default",
				ProviderType: depmodel.ProviderTypeUserDefined,
				ScopeType:    depmodel.ScopeTypeWorkspace,
				WorkspaceID:  app.WorkspaceID,
				Config:       map[string]any{},
				Credentials: map[string]any{
					"MYSQL_HOST":     "127.0.0.1",
					"MYSQL_PORT":     "3306",
					"MYSQL_USER":     "root",
					"MYSQL_DATABASE": "mydb",
					"MYSQL_PASSWORD": "blueking",
				},
				Status:   depmodel.AvailableStatus,
				Operator: "test-operator",
			}
			instID, err := stores.SvcInstStore.Create(ctx, inst)
			Expect(err).NotTo(HaveOccurred())

			_, err = stores.BindingStore.Create(ctx, &depmodel.ServiceBinding{
				Name:        "mysql",
				AppID:       app.ID,
				WorkspaceID: app.WorkspaceID,
				ServiceName: "fake",
				EnvInstanceMap: map[string]bson.ObjectID{
					testEnv.Name: instID,
				},
				EnvVars: map[string]string{
					"MYSQL_HOST":     "${{env.MYSQL_HOST}}",
					"MYSQL_PORT":     "${{env.MYSQL_PORT}}",
					"MYSQL_USER":     "${{env.MYSQL_USER}}",
					"MYSQL_DATABASE": "${{env.MYSQL_DATABASE}}",
					"MYSQL_PASSWORD": "${{env.MYSQL_PASSWORD}}",
					"MYSQL_DSN":      "mysql://${{env.MYSQL_USER}}:${{env.MYSQL_PASSWORD}}@${{env.MYSQL_HOST}}:${{env.MYSQL_PORT}}/${{env.MYSQL_DATABASE}}",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			builder := workload.NewBuilder(builderSvc, app, appModel)

			result, err := builder.Build(ctx, testEnv)
			Expect(err).NotTo(HaveOccurred())

			gd := asGameDeployment(result)

			// Verify fake provider builtin env vars are included in the workload
			envMap := make(map[string]string)
			for _, envVar := range gd.Spec.Template.Spec.Containers[0].Env {
				envMap[envVar.Name] = envVar.Value
			}
			Expect(envMap).To(HaveKeyWithValue("MYSQL_HOST", "127.0.0.1"))
			Expect(envMap).To(HaveKeyWithValue("MYSQL_PORT", "3306"))
			Expect(envMap).To(HaveKeyWithValue("MYSQL_USER", "root"))
			Expect(envMap).To(HaveKeyWithValue("MYSQL_DATABASE", "mydb"))
			Expect(envMap).To(HaveKeyWithValue("MYSQL_PASSWORD", "blueking"))

			// Verify binding EnvVars rendered via ${{env.KEY}} template syntax
			Expect(envMap).To(HaveKeyWithValue(
				"MYSQL_DSN",
				"mysql://root:blueking@127.0.0.1:3306/mydb",
			))
		},
		Entry("TRPC workload", workloadTestCases[0]),
		Entry("TAF workload", workloadTestCases[1]),
	)
})
