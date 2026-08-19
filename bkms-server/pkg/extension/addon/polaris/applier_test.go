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

package polaris_test

import (
	"context"
	"errors"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var _ = Describe("Polaris CR applier", func() {
	var (
		ctx               context.Context
		diApp             *fxtest.App
		appStore          bkmsapp.ApplicationStore
		envStore          bkmsenv.EnvironmentStore
		envService        *env.EnvService
		store             polaris.PolarisConfigStore
		appModelStore     appmodel.AppModelStore
		scopedEnvVarStore envvars.ScopedEnvVarStore
		appDepsVarReader  *depenvvars.Reader
		polarisVarReader  *polarisenvvars.Reader
		depSvcStore       depsvcmodel.ServiceStore
		depSvcInstStore   depsvcmodel.ServiceInstanceStore
		envStateManager   *polaris.PolarisEnvStateManager
		service           *polaris.PolarisConfigService
		app               *bkmsapp.Application
		environment       *bkmsenv.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			env.FxModule,
			appmodel.FxModule,
			envvars.FxModule,
			depsvcmodel.FxModule,
			depenvvars.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(
				&appStore,
				&envStore,
				&envService,
				&store,
				&appModelStore,
				&scopedEnvVarStore,
				&appDepsVarReader,
				&polarisVarReader,
				&depSvcStore,
				&depSvcInstStore,
				&envStateManager,
			),
		)
		diApp.RequireStart()

		service = polaris.NewPolarisConfigService(
			store,
			polaris.NewPolarisPlatformManager(depSvcStore, depSvcInstStore, store),
			envStateManager,
			envStore,
			appModelStore,
			envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader),
			nil,
		)
		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, app.ID)
		diApp.RequireStop()
	})

	It("should return an error without persisting weight or changing env state", func() {
		applied := redeployFields("k1", "t1", 8080)
		config := newTestConfig(
			app.ID,
			"cfg-apply-error",
			[]string{environment.Name},
			map[string]polaris.PolarisEnvState{environment.Name: envState(applied)},
		)
		Expect(store.Create(ctx, config)).To(Succeed())
		Expect(store.UpsertEnvState(ctx, app.ID, config.Name, environment.Name, polaris.PolarisEnvStateUpdate{
			AppliedFields: applied,
		})).To(Succeed())

		mockey.PatchConvey("cluster discovery fails", GinkgoT(), func() {
			mockPolarisDiscoveryFailure()

			updated, err := service.UpdateEnvWeight(ctx, app, config, environment.Name, 20)
			Expect(err).To(MatchError(ContainSubstring("patch env weight")))
			Expect(updated).To(BeNil())
			stored, getErr := store.Get(ctx, app.ID, config.Name)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.EnvWeights).NotTo(HaveKey(environment.Name))
			state := stored.GetEnvState(environment.Name)
			Expect(state.LastError).To(BeEmpty())
			Expect(state.AppliedFields).To(Equal(applied))
		})
	})

	Context("when the test cluster supports PolarisConfig", Label("k8s"), func() {
		var (
			clusterCfg          *cluster.Config
			client              *k8sclient.Client
			clusterEnv          *bkmsenv.Environment
			config              *polaris.PolarisConfig
			applied             *polaris.RedeployRequiredFields
			manifest            map[string]any
			crName              string
			expectedServiceName string
		)

		BeforeEach(func() {
			var err error
			clusterCfg, err = testutil.TestClusterConfig("test-cluster")
			if errors.Is(err, testutil.ErrKubeConfigNotFound) {
				Skip(err.Error())
			}
			Expect(err).NotTo(HaveOccurred())

			gvr, err := discovery.GetGroupVersionResource(clusterCfg, "PolarisConfig", "tkex.tencent.com/v1")
			if err != nil {
				Skip("PolarisConfig CRD not registered in test cluster: " + err.Error())
			}
			client = k8sclient.NewWithGVR(clusterCfg, *gvr)
			clusterEnv = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
				WorkspaceID: app.WorkspaceID,
				Cluster: &bkmsenv.BizCluster{
					ProjectCode: "test-project",
					ClusterID:   clusterCfg.ClusterID,
					ClusterType: "test",
					Namespace:   "default",
				},
			})
			Expect(appModelStore.CreateAppModel(ctx, &appmodel.AppModel{AppID: app.ID})).To(Succeed())
			DeferCleanup(func() { _ = appModelStore.DeleteAppModel(ctx, app.ID) })

			applied = redeployFields("key", "token", 8080)
			config = &polaris.PolarisConfig{
				AppID: app.ID,
				Name:  "cfg-auto-apply",
				Properties: polaris.Properties{
					InstanceKey: "key", PolarisName: "polaris-service",
					PolarisNamespace: "Test", PolarisToken: "token",
					ServicePort: 8080, Direct: true, KeepNotReadyPod: true,
					EnableHealthCheck: true,
				},
				ScopeEnvNames: []string{clusterEnv.Name},
				EnvWeights:    map[string]int32{clusterEnv.Name: 10},
				EnvStates: map[string]polaris.PolarisEnvState{
					clusterEnv.Name: {AppliedFields: applied, LastError: "previous error"},
				},
			}
			Expect(store.Create(ctx, config)).To(Succeed())
			previousError := "previous error"
			Expect(store.UpsertEnvState(ctx, app.ID, config.Name, clusterEnv.Name, polaris.PolarisEnvStateUpdate{
				AppliedFields: applied,
				LastError:     &previousError,
			})).To(Succeed())

			buildResult, err := polaris.NewWorkloadBuilder(store).Build(
				ctx, app, clusterEnv, nil, corev1.PodSpec{}, nil,
			)
			Expect(err).NotTo(HaveOccurred())
			manifest = nil
			for idx := range buildResult.ExtraObjects {
				if buildResult.ExtraObjects[idx].GetKind() == "PolarisConfig" {
					manifest = buildResult.ExtraObjects[idx].Object
					break
				}
			}
			Expect(manifest).NotTo(BeNil())
			crName = mapx.GetStr(manifest, "metadata.name")
			services := mapx.GetList(manifest, "spec.services")
			Expect(services).To(HaveLen(1))
			serviceSpec := services[0].(map[string]any)
			expectedServiceName = mapx.GetStr(serviceSpec, "name")
			delete(serviceSpec, "weight")
			_, err = client.Upsert(ctx, "default", manifest, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = client.Delete(ctx, "default", crName, metav1.DeleteOptions{})
			})
		})

		It("should return a service mismatch without persisting weight or changing env state", func() {
			services := mapx.GetList(manifest, "spec.services")
			serviceSpec := services[0].(map[string]any)
			serviceSpec["name"] = "unexpected-service"
			_, err := client.Upsert(ctx, "default", manifest, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("use the configured test cluster", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()

				updated, updateErr := service.UpdateEnvWeight(ctx, app, config, clusterEnv.Name, 20)
				Expect(updateErr).To(MatchError(ContainSubstring("patch env weight")))
				Expect(updated).To(BeNil())

				obj, getErr := client.Get(ctx, "default", crName, metav1.GetOptions{})
				Expect(getErr).NotTo(HaveOccurred())
				currentServices := mapx.GetList(obj.Object, "spec.services")
				Expect(currentServices).To(HaveLen(1))
				Expect(currentServices[0].(map[string]any)).NotTo(HaveKey("weight"))
				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.EnvWeights[clusterEnv.Name]).To(Equal(int32(10)))
				state := stored.GetEnvState(clusterEnv.Name)
				Expect(state.LastError).To(Equal("previous error"))
				Expect(state.AppliedFields).To(Equal(applied))
			})
		})

		It("should patch only weight without clearing the previous error", func() {
			mockey.PatchConvey("use the configured test cluster", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()

				updated, updateErr := service.UpdateEnvWeight(ctx, app, config, clusterEnv.Name, 0)
				Expect(updateErr).NotTo(HaveOccurred())
				Expect(updated.EnvWeights[clusterEnv.Name]).To(BeZero())

				obj, getErr := client.Get(ctx, "default", crName, metav1.GetOptions{})
				Expect(getErr).NotTo(HaveOccurred())
				currentServices := mapx.GetList(obj.Object, "spec.services")
				Expect(currentServices).To(HaveLen(1))
				currentService := currentServices[0].(map[string]any)
				Expect(mapx.GetStr(currentService, "name")).To(Equal(expectedServiceName))
				Expect(mapx.GetInt64(currentService, "weight")).To(BeZero())
				Expect(currentService["port"]).To(BeEquivalentTo(8080))
				Expect(currentService["direct"]).To(BeTrue())
				Expect(currentService["keepNotReadyPod"]).To(BeTrue())
				Expect(currentService["enableHealthCheck"]).To(BeTrue())
				Expect(mapx.GetStr(obj.Object, "spec.polaris.token")).To(Equal("token"))
				state := updated.GetEnvState(clusterEnv.Name)
				Expect(state.LastError).To(Equal("previous error"))
				Expect(state.AppliedFields).To(Equal(applied))
			})
		})

		It("should register an immediate config without a deployment and clean up on unbind", func() {
			serviceClient, err := k8sServiceClient(clusterCfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(client.Delete(ctx, "default", crName, metav1.DeleteOptions{})).To(Succeed())

			immediate := &polaris.PolarisConfig{
				AppID: app.ID,
				Name:  "cfg-immediate-apply",
				Properties: polaris.Properties{
					InstanceKey: "immediatekey", PolarisName: "immediate-polaris-service",
					PolarisNamespace: "Test", PolarisToken: "immediate-token",
					ServicePort: 9090, RegisterMode: polaris.RegisterModeImmediate,
				},
				ScopeEnvNames: []string{clusterEnv.Name},
			}
			immediateCRName, immediateServiceName := polaris.PolarisResourceNames(app.Name, immediate.Name)
			DeferCleanup(func() {
				_ = client.Delete(ctx, "default", immediateCRName, metav1.DeleteOptions{})
				_ = serviceClient.Delete(ctx, "default", immediateServiceName, metav1.DeleteOptions{})
			})

			mockey.PatchConvey("use the configured test cluster", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()

				Expect(service.Create(ctx, app, immediate, false)).To(Succeed())

				// 未经部署，CR 与配套 Service 都应已存在于集群中
				cr, getErr := client.Get(ctx, "default", immediateCRName, metav1.GetOptions{})
				Expect(getErr).NotTo(HaveOccurred())
				Expect(mapx.GetStr(cr.Object, "spec.polaris.token")).To(Equal("immediate-token"))
				_, getErr = serviceClient.Get(ctx, "default", immediateServiceName, metav1.GetOptions{})
				Expect(getErr).NotTo(HaveOccurred())

				stored, storeErr := store.Get(ctx, app.ID, immediate.Name)
				Expect(storeErr).NotTo(HaveOccurred())
				state := stored.GetEnvState(clusterEnv.Name)
				Expect(state.LastError).To(BeEmpty())
				Expect(polaris.PolarisEnvStatus(stored, clusterEnv.Name, state)).
					To(Equal(polaris.PolarisEnvStatusDeployed))

				updated, updateErr := service.Update(ctx, app, stored, &polaris.ConfigUpdateData{
					ScopeEnvNames: []string{},
				})
				Expect(updateErr).NotTo(HaveOccurred())
				Expect(updated.EnvStates).NotTo(HaveKey(clusterEnv.Name))

				_, getErr = client.Get(ctx, "default", immediateCRName, metav1.GetOptions{})
				Expect(getErr).To(HaveOccurred())
				_, getErr = serviceClient.Get(ctx, "default", immediateServiceName, metav1.GetOptions{})
				Expect(getErr).To(HaveOccurred())
			})
		})

		It("should not persist weight when the PolarisConfig resource is missing", func() {
			Expect(client.Delete(ctx, "default", crName, metav1.DeleteOptions{})).To(Succeed())

			mockey.PatchConvey("use the configured test cluster", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()

				updated, updateErr := service.UpdateEnvWeight(ctx, app, config, clusterEnv.Name, 25)
				Expect(updateErr).To(MatchError(ContainSubstring("patch env weight")))
				Expect(updated).To(BeNil())
				stored, getErr := store.Get(ctx, app.ID, config.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(stored.EnvWeights[clusterEnv.Name]).To(Equal(int32(10)))
				state := stored.GetEnvState(clusterEnv.Name)
				Expect(state.LastError).To(Equal("previous error"))
				Expect(state.AppliedFields).To(Equal(applied))
			})
		})
	})
})
