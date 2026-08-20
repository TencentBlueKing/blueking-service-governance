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
	corev1 "k8s.io/api/core/v1"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/secret"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var _ = Describe("Builder", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var appStore bkmsapp.ApplicationStore
	var appModelStore appmodel.AppModelStore
	var appSpecStore appspec.AppSpecStore
	var buildConfigStore build.ConfigStore
	var envSvc *env.EnvService
	var scopedEnvVarStore envvars.ScopedEnvVarStore
	var builderSvc *workload.BuilderService

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appmodel.FxModule,
			appspec.FxModule,
			workload.FxModule,
			env.FxModule,
			envvars.FxModule,
			build.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			polarisenvvars.FxModule,
			polaris.FxModule,
			fx.Populate(
				&appStore,
				&appModelStore,
				&appSpecStore,
				&buildConfigStore,
				&envSvc,
				&scopedEnvVarStore,
				&builderSvc,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Context("standard workload", func() {
		var app *bkmsapp.Application
		var envObj *envmodel.Environment
		var appModel *appmodel.AppModel
		var builder *workload.Builder

		BeforeEach(func() {
			app = dbfactory.Application(ctx, appStore)
			envObj = dbfactory.Env(ctx, envSvc, app.WorkspaceID)

			_, err := scopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *envObj, "ENV_VAR", "env_var", "")
			Expect(err).NotTo(HaveOccurred())

			appModel = &appmodel.AppModel{
				AppID: app.ID,
				Workload: appmodel.Workload{
					Type:  appmodel.WorkloadTypeStandard,
					Name:  app.Name,
					Image: "nginx:latest",
					EnvVars: []appmodel.Variable{
						{Key: "APP_VAR", Value: "app_var"},
					},
				},
			}
			err = appModelStore.CreateAppModel(ctx, appModel)
			Expect(err).NotTo(HaveOccurred())

			By("seed build config")
			err = buildConfigStore.Create(ctx, &build.Config{
				AppID:      app.ID,
				SourceType: build.SourceTypeImageRegistry,
				Image: &build.ImageConfig{
					Name:     "registry.example.com/group/app",
					Username: "user",
					Password: "pass",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			builder = workload.NewBuilder(builderSvc, app, appModel)
		})

		It("should build basic workload without extra resources", func() {
			result, err := builder.Build(ctx, envObj)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.WorkloadKind).To(Equal(kind.GameDeploy))
			gd := asGameDeployment(result)
			extraObjs := result.ExtraObjects

			Expect(extraObjs).To(BeEmpty())
			Expect(gd.Name).To(Equal(appModel.Workload.Name))
			Expect(gd.Spec.UpdateStrategy.Partition.IntValue()).To(Equal(0))
			Expect(gd.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(gd.Spec.Template.Spec.Containers[0].Image).To(Equal(appModel.Workload.Image))
			Expect(gd.Spec.Template.Spec.Containers[0].VolumeMounts).To(BeEmpty())
		})

		It("should build native Deployment when the env is a federation host", func() {
			fedEnv := *envObj
			fedEnv.Cluster.IsFederation = true

			result, err := builder.Build(ctx, &fedEnv)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.WorkloadKind).To(Equal(kind.Deploy))
			Expect(asGameDeployment(result)).To(BeNil())
			Expect(asDeployment(result)).NotTo(BeNil())
			Expect(asDeployment(result).Kind).To(Equal("Deployment"))
			Expect(asDeployment(result).APIVersion).To(Equal("apps/v1"))
			Expect(asDeployment(result).Name).To(Equal(appModel.Workload.Name))
			Expect(asDeployment(result).Spec.Template.Spec.Containers).To(HaveLen(1))
		})

		It("should not modify the source app model when applying environment overrides", func() {
			maxUnavailable := "1"
			maxSurge := "2"
			appModel.UpdateStrategy = &appmodel.UpdateStrategy{
				MaxUnavailable: &maxUnavailable,
				MaxSurge:       &maxSurge,
			}
			Expect(appModelStore.UpdateAppModel(ctx, appModel)).To(Succeed())

			overriddenMaxSurge := "50%"
			err := appspec.SetEnv(ctx, appSpecStore, &appspec.AppSpec{
				AppID:   app.ID,
				EnvName: envObj.Name,
				UpdateStrategy: &appspec.UpdateStrategySpec{
					MaxSurge: &overriddenMaxSurge,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			computed, err := builder.ComputeAppModel(ctx, envObj)
			Expect(err).NotTo(HaveOccurred())
			Expect(*computed.UpdateStrategy.MaxUnavailable).To(Equal("1"))
			Expect(*computed.UpdateStrategy.MaxSurge).To(Equal("50%"))
			Expect(*appModel.UpdateStrategy.MaxUnavailable).To(Equal("1"))
			Expect(*appModel.UpdateStrategy.MaxSurge).To(Equal("2"))
		})

		It("should append app image pull secret when build config has custom credential", func() {
			result, err := builder.Build(ctx, envObj)
			Expect(err).NotTo(HaveOccurred())
			gd := asGameDeployment(result)

			Expect(gd.Spec.Template.Spec.ImagePullSecrets).To(ContainElement(corev1.LocalObjectReference{
				Name: secret.ResolveImagePullSecretName(app.WorkspaceID, app.ID, &build.Config{
					AppID:      app.ID,
					SourceType: build.SourceTypeImageRegistry,
					Image: &build.ImageConfig{
						Name:     "registry.example.com/group/app",
						Username: "user",
						Password: "pass",
					},
				}),
			}))
		})

		It("should keep only system labels when no custom labels/annotations are set", func() {
			result, err := builder.Build(ctx, envObj)
			Expect(err).NotTo(HaveOccurred())
			gd := asGameDeployment(result)

			// System-managed deletion-allow label stays at its default value.
			Expect(gd.ObjectMeta.Labels).To(HaveKeyWithValue("io.tencent.bcs.dev/deletion-allow", "Always"))
			// Pod template keeps its system label and is not polluted by user metadata.
			Expect(gd.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", app.Name))
		})

		It("should inject custom labels/annotations into the pod template metadata only", func() {
			appModel.Labels = map[string]string{"team": "sre", "region": "sh"}
			appModel.Annotations = map[string]string{"note": "hello", "owner": "bkms"}
			Expect(appModelStore.UpdateAppModel(ctx, appModel)).NotTo(HaveOccurred())

			result, err := builder.Build(ctx, envObj)
			Expect(err).NotTo(HaveOccurred())
			gd := asGameDeployment(result)

			By("pod template metadata contains user labels/annotations")
			Expect(gd.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue("team", "sre"))
			Expect(gd.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue("region", "sh"))
			Expect(gd.Spec.Template.ObjectMeta.Annotations).To(HaveKeyWithValue("note", "hello"))
			Expect(gd.Spec.Template.ObjectMeta.Annotations).To(HaveKeyWithValue("owner", "bkms"))

			By("deployment-level metadata is not polluted by user labels/annotations")
			Expect(gd.ObjectMeta.Labels).NotTo(HaveKey("team"))
			Expect(gd.ObjectMeta.Labels).NotTo(HaveKey("region"))
			Expect(gd.ObjectMeta.Annotations).NotTo(HaveKey("note"))
			Expect(gd.ObjectMeta.Annotations).NotTo(HaveKey("owner"))

			By("system-managed labels are preserved")
			Expect(gd.ObjectMeta.Labels).To(HaveKeyWithValue("io.tencent.bcs.dev/deletion-allow", "Always"))
			Expect(gd.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", app.Name))
		})

		It("should not inject tkeRouteEni annotation when section is not configured", func() {
			result, err := builder.Build(ctx, envObj)
			Expect(err).NotTo(HaveOccurred())
			gd := asGameDeployment(result)

			Expect(gd.Spec.Template.ObjectMeta.Annotations).NotTo(HaveKey("tke.cloud.tencent.com/networks"))
		})

		It("should inject tkeRouteEni annotation when appspec tkeRouteEni is enabled", func() {
			enabled := true
			err := appspec.SetDefault(ctx, appSpecStore, appModelStore, app.ID, &appspec.AppSpec{
				TkeRouteEni: &appspec.TkeRouteEniSpec{Enabled: &enabled},
			})
			Expect(err).NotTo(HaveOccurred())

			result, err := builder.Build(ctx, envObj)
			Expect(err).NotTo(HaveOccurred())
			gd := asGameDeployment(result)

			Expect(gd.Spec.Template.ObjectMeta.Annotations).To(
				HaveKeyWithValue("tke.cloud.tencent.com/networks", "tke-route-eni"),
			)
		})

		It("should not inject tkeRouteEni annotation when appspec tkeRouteEni is disabled", func() {
			disabled := false
			err := appspec.SetDefault(ctx, appSpecStore, appModelStore, app.ID, &appspec.AppSpec{
				TkeRouteEni: &appspec.TkeRouteEniSpec{Enabled: &disabled},
			})
			Expect(err).NotTo(HaveOccurred())

			result, err := builder.Build(ctx, envObj)
			Expect(err).NotTo(HaveOccurred())
			gd := asGameDeployment(result)

			Expect(gd.Spec.Template.ObjectMeta.Annotations).NotTo(HaveKey("tke.cloud.tencent.com/networks"))
		})
	})
})
