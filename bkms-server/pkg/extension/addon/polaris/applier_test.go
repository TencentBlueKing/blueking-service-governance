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
	"time"

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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
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

		reader := envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader)
		service = polaris.NewPolarisConfigService(
			store,
			polaris.NewPolarisPlatformManager(depSvcStore, depSvcInstStore, store),
			envStateManager,
			envStore,
			appModelStore,
			reader,
		)
		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, app.ID)
		diApp.RequireStop()
	})

	It("should record only the error when asynchronous apply fails", func() {
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
		weight := int32(20)
		_, err := service.Update(ctx, app, config, &polaris.ConfigUpdateData{Weight: &weight})
		Expect(err).NotTo(HaveOccurred())
		Eventually(func(g Gomega) {
			stored, getErr := store.Get(ctx, app.ID, config.Name)
			g.Expect(getErr).NotTo(HaveOccurred())
			state := stored.GetEnvState(environment.Name)
			g.Expect(state.LastError).NotTo(BeEmpty())
			g.Expect(state.AppliedFields).To(Equal(applied))
		}).Should(Succeed())
	})

	Context("when the test cluster supports PolarisConfig", Label("k8s"), func() {
		It("should apply a CR and clear only the previous error", func() {
			clusterCfg, err := testutil.TestClusterConfig("test-cluster")
			if errors.Is(err, testutil.ErrKubeConfigNotFound) {
				Skip(err.Error())
			}
			Expect(err).NotTo(HaveOccurred())

			gvr, err := discovery.GetGroupVersionResource(clusterCfg, "PolarisConfig", "tkex.tencent.com/v1")
			if err != nil {
				Skip("PolarisConfig CRD not registered in test cluster: " + err.Error())
			}
			client := k8sclient.NewWithGVR(clusterCfg, *gvr)
			clusterEnv := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
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

			applied := redeployFields("key", "token", 8080)
			config := &polaris.PolarisConfig{
				AppID: app.ID,
				Name:  "cfg-auto-apply",
				Properties: polaris.Properties{
					InstanceKey: "key", PolarisName: "polaris-service",
					PolarisNamespace: "Test", PolarisToken: "token",
					ServicePort: 8080, Weight: 10,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{clusterEnv.Name},
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
				ctx, app, clusterEnv, nil, corev1.PodSpec{}, "", nil,
			)
			Expect(err).NotTo(HaveOccurred())
			var manifest map[string]any
			for idx := range buildResult.ExtraObjects {
				if buildResult.ExtraObjects[idx].GetKind() == "PolarisConfig" {
					manifest = buildResult.ExtraObjects[idx].Object
					break
				}
			}
			Expect(manifest).NotTo(BeNil())
			crName := mapx.GetStr(manifest, "metadata.name")
			_, err = client.Upsert(ctx, "default", manifest, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = client.Delete(ctx, "default", crName, metav1.DeleteOptions{})
			})

			mockey.PatchConvey("use the configured test cluster", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()

				weight := int32(20)
				_, updateErr := service.Update(ctx, app, config, &polaris.ConfigUpdateData{Weight: &weight})
				Expect(updateErr).NotTo(HaveOccurred())

				Eventually(func(g Gomega) {
					obj, getErr := client.Get(ctx, "default", crName, metav1.GetOptions{})
					g.Expect(getErr).NotTo(HaveOccurred())
					services := mapx.GetList(obj.Object, "spec.services")
					g.Expect(services).To(HaveLen(1))
					g.Expect(mapx.GetInt64(services[0].(map[string]any), "weight")).To(Equal(int64(weight)))

					stored, getErr := store.Get(ctx, app.ID, config.Name)
					g.Expect(getErr).NotTo(HaveOccurred())
					state := stored.GetEnvState(clusterEnv.Name)
					g.Expect(state.LastError).To(BeEmpty())
					g.Expect(state.AppliedFields).To(Equal(applied))
				}).WithTimeout(30 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())
			})
		})
	})
})
