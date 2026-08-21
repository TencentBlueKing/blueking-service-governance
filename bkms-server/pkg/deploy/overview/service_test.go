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

package overview

import (
	"context"
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("overview.Service", func() {
	var (
		ctx                        context.Context
		diApp                      *fxtest.App
		appStore                   bkmsapp.ApplicationStore
		appModelStore              appmodel.AppModelStore
		appSpecStore               appspec.AppSpecStore
		appConfigFileStore         appcfg.AppConfigFileStore
		appConfigFileVersionStore  appcfg.AppConfigFileVersionStore
		buildConfigStore           build.ConfigStore
		buildAutoDeployRecordStore autodeploy.RecordStore
		envStore                   envmodel.EnvironmentStore
		envSvc                     *bkmsenv.EnvService
		appModelDeployRecordStore  appmodeldeploy.RecordStore
		helmDeployRecordStore      helmdeploy.RecordStore
		gpaConfigStore             gpa.GPAConfigStore
		svc                        *Service
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Note: gpa.FxModule nests appmodel.FxModule, so appmodel.FxModule must not be
		// listed again here — fx would reject the duplicate providers.
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			gpa.FxModule,
			appspec.FxModule,
			appcfg.FxModule,
			bkmsenv.FxModule,
			autodeploy.FxModule,
			appmodeldeploy.FxModule,
			helmdeploy.FxModule,
			build.FxModule,
			fx.Populate(
				&appStore,
				&appModelStore,
				&appSpecStore,
				&appConfigFileStore,
				&appConfigFileVersionStore,
				&buildConfigStore,
				&envStore,
				&envSvc,
				&buildAutoDeployRecordStore,
				&appModelDeployRecordStore,
				&helmDeployRecordStore,
				&gpaConfigStore,
			),
		)
		diApp.RequireStart()

		svc = NewService(
			envStore,
			appStore,
			appModelStore,
			buildAutoDeployRecordStore,
			appModelDeployRecordStore,
			helmDeployRecordStore,
			gpaConfigStore,
		)
	})

	AfterEach(func() {
		Expect(envStore.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	newTrpcApp := func() *bkmsapp.Application {
		trpcApp, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigFileVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, nil)
		return trpcApp
	}

	// mockGPAClusterClientChain mock 出 GPA 集群客户端的构造链路
	// （cluster.NewConfig / GetGroupVersionResource / NewWithGVR），
	// 具体的 CR 查询由各用例单独 mock。
	mockGPAClusterClientChain := func() {
		mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build()
		mockey.Mock(discovery.GetGroupVersionResource).Return(
			&schema.GroupVersionResource{
				Group: "autoscaling.tkex.tencent.com", Version: "v1alpha1", Resource: "generalpodautoscalers",
			}, nil,
		).Build()
		mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
	}

	overviewMainContainerGD := func(ns, name string) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"namespace": ns, "name": name},
			"spec": map[string]any{
				"replicas": int64(2),
				"template": map[string]any{
					"spec": map[string]any{
						"containers": []any{
							map[string]any{
								"name": "main",
								"resources": map[string]any{
									"limits":   map[string]any{"cpu": "4", "memory": "8Gi"},
									"requests": map[string]any{"cpu": "2", "memory": "4Gi"},
								},
							},
						},
					},
				},
			},
		}}
	}

	Describe("GetOverview", func() {
		It("rejects non AppModel application types", func() {
			_, err := svc.GetOverview(ctx, &bkmsapp.Application{ID: "h1", Type: bkmsapp.AppTypeHelm})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported app type"))
		})

		It("returns empty envs when no environment has the app in AppIDs", func() {
			trpcApp := newTrpcApp()
			_ = dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)

			result, err := svc.GetOverview(ctx, trpcApp)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Envs).To(BeEmpty())
		})

		It("includes only envs with AppIDs membership and default-lane status", func() {
			trpcApp := newTrpcApp()
			tracked := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			untracked := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			Expect(envStore.AddApp(ctx, tracked.ID, trpcApp.ID)).To(Succeed())

			startedAt := time.Now().UTC().Truncate(time.Millisecond)
			_, err := appModelDeployRecordStore.Create(ctx, &appmodeldeploy.Record{
				WorkspaceID: trpcApp.WorkspaceID,
				AppID:       trpcApp.ID,
				EnvName:     tracked.Name,
				Status:      appmodeldeploy.StatusDeployed,
				ImageTag:    "v9",
				StartedAt:   startedAt,
				// 不填 ClusterID：本用例只验证部署状态，不走 K8s 实例查询
			})
			Expect(err).NotTo(HaveOccurred())

			result, err := svc.GetOverview(ctx, trpcApp)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Envs).To(HaveLen(1))
			Expect(result.Envs[0].EnvName).To(Equal(tracked.Name))
			Expect(result.Envs[0].EnvName).NotTo(Equal(untracked.Name))
			Expect(result.Envs[0].DeployStatus).To(Equal(string(appmodeldeploy.StatusDeployed)))
			Expect(result.Envs[0].ImageTag).To(Equal("v9"))
			Expect(result.Envs[0].Cluster.ClusterID).To(Equal(tracked.Cluster.ClusterID))
			Expect(result.Envs[0].Cluster.Namespace).To(Equal(tracked.Cluster.Namespace))
			Expect(result.Envs[0].Cluster.ProjectCode).To(Equal(tracked.Cluster.ProjectCode))
			Expect(result.Envs[0].Cluster.ClusterType).To(Equal(tracked.Cluster.ClusterType))
			Expect(result.Envs[0].LastDeployStartedAt).NotTo(BeNil())
			Expect(result.Envs[0].Instances).To(BeNil())
		})

		It("includes owned feature envs only when AppIDs contains the app", func() {
			trpcApp := newTrpcApp()
			source := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			_ = dbfactory.FeatEnv(ctx, envSvc, trpcApp, source)
			featWithApp := dbfactory.FeatEnv(ctx, envSvc, trpcApp, source)
			Expect(envStore.AddApp(ctx, featWithApp.ID, trpcApp.ID)).To(Succeed())

			result, err := svc.GetOverview(ctx, trpcApp)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Envs).To(HaveLen(1))
			Expect(result.Envs[0].EnvName).To(Equal(featWithApp.Name))
			Expect(result.Envs[0].EnvKind).To(Equal(string(envmodel.EnvironmentKindFeature)))
		})

		It("fills gpa autoscaling summary from config", func() {
			trpcApp := newTrpcApp()
			env := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			Expect(envStore.AddApp(ctx, env.ID, trpcApp.ID)).To(Succeed())

			cfg := &gpa.GPAConfig{
				AppID:       trpcApp.ID,
				EnvName:     env.Name,
				MinReplicas: 4,
				MaxReplicas: 12,
				Metrics: []gpa.GPAMetric{
					{Resource: gpa.ResourceCPU, AverageUtilization: 60},
					{Resource: gpa.ResourceMemory, AverageUtilization: 70},
				},
			}
			Expect(gpaConfigStore.Create(ctx, cfg)).To(Succeed())

			mockey.PatchConvey("gpa CR absent in cluster", GinkgoT(), func() {
				mockGPAClusterClientChain()
				mockey.Mock((*gpa.ClusterClient).GetStatus).Return(nil, gpa.ErrCRNotFound).Build()

				result, err := svc.GetOverview(ctx, trpcApp)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Envs).To(HaveLen(1))
				Expect(result.Envs[0].Autoscaling).NotTo(BeNil())
				Expect(result.Envs[0].Autoscaling.Enabled).To(BeTrue())
				Expect(result.Envs[0].Autoscaling.MinReplicas).To(Equal(int32(4)))
				Expect(result.Envs[0].Autoscaling.MaxReplicas).To(Equal(int32(12)))
				Expect(result.Envs[0].Autoscaling.Metrics).To(ConsistOf(
					AutoscalingMetric{Resource: string(gpa.ResourceCPU), AverageUtilization: 60},
					AutoscalingMetric{Resource: string(gpa.ResourceMemory), AverageUtilization: 70},
				))
				Expect(result.Envs[0].Autoscaling.ComputeByLimits).To(BeFalse())
				// 集群中 CR 不存在时 status 降级为 null，不阻断总览
				Expect(result.Envs[0].Autoscaling.Status).To(BeNil())
			})
		})

		It("fills resources from the cluster GameDeployment instead of app-spec", func() {
			trpcApp := newTrpcApp()
			env := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			Expect(envStore.AddApp(ctx, env.ID, trpcApp.ID)).To(Succeed())

			// AppSpec 与集群实际值刻意设成不同，回归「改了配置未重部署」时总览仍应展示集群值。
			Expect(appspec.SetDefault(ctx, appSpecStore, appModelStore, trpcApp.ID, &appspec.AppSpec{
				Resources: &appspec.ResourcesSpec{
					CPULimits:      lo.ToPtr("2"),
					CPURequests:    lo.ToPtr("1"),
					MemoryLimits:   lo.ToPtr("4Gi"),
					MemoryRequests: lo.ToPtr("2Gi"),
				},
			})).To(Succeed())

			ns := "ns-prod"
			gdName := trpcApp.Name
			_, err := appModelDeployRecordStore.Create(ctx, &appmodeldeploy.Record{
				WorkspaceID:   trpcApp.WorkspaceID,
				AppID:         trpcApp.ID,
				EnvName:       env.Name,
				Status:        appmodeldeploy.StatusDeployed,
				ClusterID:     "cluster-prod",
				Namespace:     ns,
				LabelSelector: map[string]string{"app.kubernetes.io/name": trpcApp.Name},
				ResourceKeys: appmodeldeploy.ResourceKeys{
					{Kind: k8skind.GameDeploy, Name: gdName},
				},
				ImageTag:  "v1",
				StartedAt: time.Now().UTC(),
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("cluster gamedeploy has different resources", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).To(func(id string) *cluster.Config {
					return &cluster.Config{ClusterID: id}
				}).Build()
				mockey.Mock(k8sclient.NewPodClient).Return(&k8sclient.PodClient{}).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{}, nil).Build()
				mockey.Mock((*k8sclient.Client).Get).To(func(
					_ *k8sclient.Client, _ context.Context, gotNS, name string, _ metav1.GetOptions,
				) (*unstructured.Unstructured, error) {
					Expect(gotNS).To(Equal(ns))
					Expect(name).To(Equal(gdName))
					return &unstructured.Unstructured{Object: map[string]any{
						"metadata": map[string]any{"namespace": ns, "name": gdName},
						"spec": map[string]any{
							"replicas": int64(2),
							"template": map[string]any{
								"spec": map[string]any{
									"containers": []any{
										map[string]any{
											"name": "sidecar",
											"resources": map[string]any{
												"limits":   map[string]any{"cpu": "100m", "memory": "128Mi"},
												"requests": map[string]any{"cpu": "50m", "memory": "64Mi"},
											},
										},
										map[string]any{
											"name": "main",
											"resources": map[string]any{
												"limits":   map[string]any{"cpu": "4", "memory": "8Gi"},
												"requests": map[string]any{"cpu": "2", "memory": "4Gi"},
											},
										},
									},
								},
							},
						},
					}}, nil
				}).Build()

				result, err := svc.GetOverview(ctx, trpcApp)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Envs).To(HaveLen(1))
				Expect(result.Envs[0].Resources.CPULimits).To(Equal("4"))
				Expect(result.Envs[0].Resources.CPURequests).To(Equal("2"))
				Expect(result.Envs[0].Resources.MemoryLimits).To(Equal("8Gi"))
				Expect(result.Envs[0].Resources.MemoryRequests).To(Equal("4Gi"))
			})
		})

		It("keeps resources when pod list fails", func() {
			trpcApp := newTrpcApp()
			env := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			Expect(envStore.AddApp(ctx, env.ID, trpcApp.ID)).To(Succeed())

			ns := "ns-prod"
			gdName := trpcApp.Name
			_, err := appModelDeployRecordStore.Create(ctx, &appmodeldeploy.Record{
				WorkspaceID:   trpcApp.WorkspaceID,
				AppID:         trpcApp.ID,
				EnvName:       env.Name,
				Status:        appmodeldeploy.StatusDeployed,
				ClusterID:     "cluster-prod",
				Namespace:     ns,
				LabelSelector: map[string]string{"app.kubernetes.io/name": trpcApp.Name},
				ResourceKeys: appmodeldeploy.ResourceKeys{
					{Kind: k8skind.GameDeploy, Name: gdName},
				},
				ImageTag:  "v1",
				StartedAt: time.Now().UTC(),
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("pod list fails and gamedeploy get succeeds", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).To(func(id string) *cluster.Config {
					return &cluster.Config{ClusterID: id}
				}).Build()
				mockey.Mock(k8sclient.NewPodClient).Return(&k8sclient.PodClient{}).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).List).Return(
					nil, errors.New("list pods failed"),
				).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(overviewMainContainerGD(ns, gdName), nil).Build()

				result, err := svc.GetOverview(ctx, trpcApp)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Envs).To(HaveLen(1))
				Expect(result.Envs[0].Instances).To(BeNil())
				Expect(result.Envs[0].Resources.CPULimits).To(Equal("4"))
				Expect(result.Envs[0].Resources.CPURequests).To(Equal("2"))
				Expect(result.Envs[0].Resources.MemoryLimits).To(Equal("8Gi"))
				Expect(result.Envs[0].Resources.MemoryRequests).To(Equal("4Gi"))
			})
		})

		It("keeps resources when deploy record has no label selector", func() {
			trpcApp := newTrpcApp()
			env := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			Expect(envStore.AddApp(ctx, env.ID, trpcApp.ID)).To(Succeed())

			ns := "ns-prod"
			gdName := trpcApp.Name
			_, err := appModelDeployRecordStore.Create(ctx, &appmodeldeploy.Record{
				WorkspaceID: trpcApp.WorkspaceID,
				AppID:       trpcApp.ID,
				EnvName:     env.Name,
				Status:      appmodeldeploy.StatusDeployed,
				ClusterID:   "cluster-prod",
				Namespace:   ns,
				ResourceKeys: appmodeldeploy.ResourceKeys{
					{Kind: k8skind.GameDeploy, Name: gdName},
				},
				ImageTag:  "v1",
				StartedAt: time.Now().UTC(),
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("no selector fails pod list without calling k8s", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).To(func(id string) *cluster.Config {
					return &cluster.Config{ClusterID: id}
				}).Build()
				mockey.Mock(k8sclient.NewPodClient).Return(&k8sclient.PodClient{}).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).List).To(func(
					*k8sclient.Client, context.Context, string, metav1.ListOptions,
				) (*unstructured.UnstructuredList, error) {
					Fail("pod list must not run without label selector")
					return nil, nil
				}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(overviewMainContainerGD(ns, gdName), nil).Build()

				result, err := svc.GetOverview(ctx, trpcApp)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Envs).To(HaveLen(1))
				Expect(result.Envs[0].Instances).To(BeNil())
				Expect(result.Envs[0].Resources.CPULimits).To(Equal("4"))
				Expect(result.Envs[0].Resources.MemoryLimits).To(Equal("8Gi"))
			})
		})

		It("attaches gpa CR status including Failed phase for frontend display", func() {
			trpcApp := newTrpcApp()
			env := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			Expect(envStore.AddApp(ctx, env.ID, trpcApp.ID)).To(Succeed())

			cfg := &gpa.GPAConfig{
				AppID:       trpcApp.ID,
				EnvName:     env.Name,
				MinReplicas: 4,
				MaxReplicas: 12,
				Metrics: []gpa.GPAMetric{
					{Resource: gpa.ResourceCPU, AverageUtilization: 60},
				},
			}
			Expect(gpaConfigStore.Create(ctx, cfg)).To(Succeed())

			mockey.PatchConvey("gpa status failed", GinkgoT(), func() {
				mockGPAClusterClientChain()
				mockey.Mock((*gpa.ClusterClient).GetStatus).Return(&gpa.GPAStatus{
					CurrentReplicas: 3,
					DesiredReplicas: 4,
					Phase:           "Failed",
					StatusMessage:   "the GPA controller was unable to get the target's current scale: not found",
				}, nil).Build()

				result, err := svc.GetOverview(ctx, trpcApp)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Envs).To(HaveLen(1))
				Expect(result.Envs[0].Autoscaling).NotTo(BeNil())
				Expect(result.Envs[0].Autoscaling.Status).NotTo(BeNil())
				Expect(result.Envs[0].Autoscaling.Status.Phase).To(Equal("Failed"))
				Expect(
					result.Envs[0].Autoscaling.Status.StatusMessage,
				).To(ContainSubstring("unable to get the target"))
				Expect(result.Envs[0].Autoscaling.Status.CurrentReplicas).To(Equal(int32(3)))
				Expect(result.Envs[0].Autoscaling.Status.DesiredReplicas).To(Equal(int32(4)))
			})
		})

		It("counts ready/abnormal pods and GD replicas on successful k8s queries", func() {
			trpcApp := newTrpcApp()
			env := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			Expect(envStore.AddApp(ctx, env.ID, trpcApp.ID)).To(Succeed())

			ns := "ns-prod"
			clusterID := "cluster-prod"
			gdName := trpcApp.Name
			labelSel := map[string]string{"app.kubernetes.io/name": trpcApp.Name}

			_, err := appModelDeployRecordStore.Create(ctx, &appmodeldeploy.Record{
				WorkspaceID:   trpcApp.WorkspaceID,
				AppID:         trpcApp.ID,
				EnvName:       env.Name,
				Status:        appmodeldeploy.StatusDeployed,
				ClusterID:     clusterID,
				Namespace:     ns,
				LabelSelector: labelSel,
				ResourceKeys: appmodeldeploy.ResourceKeys{
					{Kind: k8skind.GameDeploy, Name: gdName},
				},
				ImageTag:  "v1",
				StartedAt: time.Now().UTC(),
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("k8s query success", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).To(func(id string) *cluster.Config {
					return &cluster.Config{ClusterID: id}
				}).Build()
				mockey.Mock(k8sclient.NewPodClient).Return(&k8sclient.PodClient{}).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()

				mockey.Mock((*k8sclient.Client).List).To(func(
					_ *k8sclient.Client, _ context.Context, gotNS string, opts metav1.ListOptions,
				) (*unstructured.UnstructuredList, error) {
					Expect(gotNS).To(Equal(ns))
					Expect(opts.LabelSelector).NotTo(BeEmpty())
					return &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
						{Object: map[string]any{
							"metadata": map[string]any{"namespace": ns, "name": "p1"},
							"status": map[string]any{
								"phase": "Running",
								"conditions": []any{
									map[string]any{"type": "Ready", "status": "True"},
								},
							},
						}},
						{Object: map[string]any{
							"metadata": map[string]any{"namespace": ns, "name": "p2"},
							"status": map[string]any{
								"phase": "Running",
								"conditions": []any{
									map[string]any{"type": "Ready", "status": "False"},
								},
							},
						}},
					}}, nil
				}).Build()

				mockey.Mock((*k8sclient.Client).Get).To(func(
					_ *k8sclient.Client, _ context.Context, gotNS, name string, _ metav1.GetOptions,
				) (*unstructured.Unstructured, error) {
					Expect(gotNS).To(Equal(ns))
					Expect(name).To(Equal(gdName))
					return &unstructured.Unstructured{Object: map[string]any{
						"metadata": map[string]any{"namespace": ns, "name": gdName},
						"spec":     map[string]any{"replicas": int64(3)},
					}}, nil
				}).Build()

				result, err := svc.GetOverview(ctx, trpcApp)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Envs).To(HaveLen(1))
				Expect(result.Envs[0].Instances).NotTo(BeNil())
				Expect(result.Envs[0].Instances.Running).To(Equal(int32(1)))
				Expect(result.Envs[0].Instances.Abnormal).To(Equal(int32(1)))
				Expect(result.Envs[0].Instances.Expected).To(Equal(int32(3)))
			})
		})

		It("sets instances null when env k8s query fails without failing the request", func() {
			trpcApp := newTrpcApp()
			env := dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			Expect(envStore.AddApp(ctx, env.ID, trpcApp.ID)).To(Succeed())

			_, err := appModelDeployRecordStore.Create(ctx, &appmodeldeploy.Record{
				WorkspaceID:   trpcApp.WorkspaceID,
				AppID:         trpcApp.ID,
				EnvName:       env.Name,
				Status:        appmodeldeploy.StatusDeployed,
				ClusterID:     "cluster-down",
				Namespace:     "ns-x",
				LabelSelector: map[string]string{"app": "x"},
				ResourceKeys: appmodeldeploy.ResourceKeys{
					{Kind: k8skind.GameDeploy, Name: "gd-x"},
				},
				ImageTag:  "v1",
				StartedAt: time.Now().UTC(),
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("k8s query failure", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build()
				mockey.Mock(k8sclient.NewPodClient).Return(&k8sclient.PodClient{}).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).List).Return(
					nil, errors.New("cluster unreachable"),
				).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(
					nil, errors.New("cluster unreachable"),
				).Build()

				result, err := svc.GetOverview(ctx, trpcApp)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Envs).To(HaveLen(1))
				Expect(result.Envs[0].Instances).To(BeNil())
				Expect(result.Envs[0].DeployStatus).To(Equal(string(appmodeldeploy.StatusDeployed)))
			})
		})
	})
})
