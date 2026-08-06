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

package topology

import (
	"context"
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/clusterresources"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("Service", func() {
	var (
		diApp             *fxtest.App
		svc               *Service
		store             *ResourceSnapshotStoreMongo
		scopedEnvVarStore envvars.ScopedEnvVarStore
		appModelStore     appmodel.AppModelStore
		testApp           *bkmsapp.Application
		testEnv           *envmodel.Environment
		testCtx           context.Context
		mocker            *mockey.Mocker
	)

	BeforeEach(func() {
		testCtx = context.Background()

		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			appmodel.FxModule,
			bkmsenv.FxModule,
			envvars.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			polarisenvvars.FxModule,
			polaris.FxModule,
			fx.Populate(&svc, &store, &appModelStore, &scopedEnvVarStore),
		)
		diApp.RequireStart()

		err := testutil.CleanupCollection(ResourceSnapshotsCollection)
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("app_models")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("scoped_env_vars")
		Expect(err).NotTo(HaveOccurred())

		testApp = &bkmsapp.Application{
			ID:          "test-app",
			WorkspaceID: "test-workspace",
			Name:        "test-app",
			Type:        bkmsapp.AppTypeTRPC,
		}
		testEnv = &envmodel.Environment{
			ID:          bson.NewObjectID(),
			Name:        "dev",
			Type:        "development",
			WorkspaceID: "test-workspace",
			Cluster: envmodel.BizCluster{
				ClusterID: "BCS-K8S-00000",
				Namespace: "default",
			},
		}
		err = appModelStore.CreateAppModel(testCtx, &appmodel.AppModel{AppID: testApp.ID})
		Expect(err).NotTo(HaveOccurred())

		cfg, err := testutil.TestClusterConfig("BCS-K8S-00000")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())
		mocker = mockey.Mock(cluster.NewConfig).Return(cfg).Build()
	})

	AfterEach(func() {
		if mocker != nil {
			mocker.Release()
		}
		_ = testutil.CleanupCollection(ResourceSnapshotsCollection)
		_ = testutil.CleanupCollection("app_models")
		_ = testutil.CleanupCollection("scoped_env_vars")
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	Describe("GetTopology", func() {
		It("should return empty topology with partial=true when no ResourceSnapshot exists", func() {
			graph, err := svc.GetTopology(testCtx, "non-existent-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(graph).NotTo(BeNil())
			Expect(graph.IsPartial).To(BeTrue())
			Expect(graph.Nodes).To(BeEmpty())
			Expect(graph.Edges).To(BeEmpty())
			Expect(graph.RootID).To(BeEmpty())
			Expect(graph.Warnings).To(ContainElement(ContainSubstring("no resource snapshot data found")))
			Expect(graph.DataVersion).To(Equal(int64(0)))
		})

		It("should add warning when refresh status is failed", func() {
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     1,
				RefreshStatus:   RefreshStatusFailed,
				WarningSummary:  "connection timeout",
				RefreshedAt:     time.Now(),
				Resources:       []ResourceEntry{},
				Relations:       []ResourceRelation{},
			}
			err := store.UpsertWithVersion(testCtx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())

			graph, err := svc.GetTopology(testCtx, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(graph).NotTo(BeNil())
			Expect(graph.IsPartial).To(BeTrue())
			Expect(graph.Warnings).To(ContainElement(ContainSubstring("resource snapshot data may be stale")))
			Expect(graph.Warnings).To(ContainElement(ContainSubstring("connection timeout")))
		})

		It("should return topology with metadata when ResourceSnapshot exists", func() {
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     3,
				RefreshStatus:   RefreshStatusSuccess,
				RefreshedAt:     time.Now(),
				Resources: []ResourceEntry{
					{
						Kind:       "ConfigMap",
						Namespace:  "default",
						Name:       "test-cm",
						IsManaged:  true,
						SourceType: SourceTypeHelmManifest,
					},
				},
				Relations: []ResourceRelation{},
			}
			err := store.UpsertWithVersion(testCtx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())

			graph, err := svc.GetTopology(testCtx, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(graph).NotTo(BeNil())
			Expect(graph.Metadata.AppID).To(Equal("test-app"))
			Expect(graph.Metadata.EnvName).To(Equal("dev"))
			Expect(graph.Metadata.ClusterID).To(Equal("BCS-K8S-00001"))
			Expect(graph.DataVersion).To(Equal(int64(3)))
			// Build 会自动添加 APP 虚拟根节点 + 1 个 ConfigMap 节点
			Expect(graph.Nodes).To(HaveLen(2))
			// 第一个节点是 APP 虚拟根节点
			Expect(graph.Nodes[0].Kind).To(Equal(NodeKindApp))
			Expect(graph.Nodes[0].Name).To(Equal("test-app"))
			Expect(graph.Nodes[0].Status).To(Equal(k8sstatus.Active))
			// 第二个节点是 ConfigMap，集群中不存在，应标记为 NotFound
			Expect(graph.Nodes[1].Kind).To(Equal("ConfigMap"))
			Expect(graph.Nodes[1].Status).To(Equal(k8sstatus.NotFound))
		})
	})

	Describe("GetNodeDetail", func() {
		var snapshot *ResourceSnapshot

		BeforeEach(func() {
			snapshot = &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     1,
				RefreshStatus:   RefreshStatusSuccess,
				RefreshedAt:     time.Now(),
				Resources: []ResourceEntry{
					{
						Kind:       k8skind.Deploy,
						Namespace:  "default",
						Name:       "my-deploy",
						IsManaged:  true,
						SourceType: SourceTypeHelmManifest,
					},
					{
						Kind:       k8skind.CM,
						Namespace:  "default",
						Name:       "my-cm",
						IsManaged:  true,
						SourceType: SourceTypeHelmManifest,
					},
				},
				Relations: []ResourceRelation{},
			}
			err := store.UpsertWithVersion(testCtx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return node detail for a valid static resource", func() {
			nodeID := EncodeNodeID(k8skind.CM, "default", "my-cm")

			mockey.PatchConvey("get-detail-ok", GinkgoT(), func() {
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       k8skind.CM,
						"metadata": map[string]any{
							"name":              "my-cm",
							"namespace":         "default",
							"creationTimestamp": "2024-06-01T10:00:00Z",
						},
						"data": map[string]any{
							"key1": "value1",
						},
					},
				}, nil).Build()

				detail, err := svc.GetNodeDetail(testCtx, "test-app", "dev", "", nodeID)
				Expect(err).NotTo(HaveOccurred())
				Expect(detail).NotTo(BeNil())
				Expect(detail.Kind).To(Equal(k8skind.CM))
				Expect(detail.Name).To(Equal("my-cm"))
				Expect(detail.Namespace).To(Equal("default"))
				Expect(detail.CreatedAt).NotTo(BeEmpty())
				Expect(detail.Extras).To(HaveKey(ExtrasKeyKeys))
			})
		})

		It("should return error for invalid node ID", func() {
			_, err := svc.GetNodeDetail(testCtx, "test-app", "dev", "", "invalid-base64")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid node ID"))
		})

		It("should return error when resource is out of scope", func() {
			nodeID := EncodeNodeID(k8skind.Ing, "default", "not-in-scope")
			_, err := svc.GetNodeDetail(testCtx, "test-app", "dev", "", nodeID)
			Expect(err).To(HaveOccurred())
		})

		It("should return error when no resource snapshot exists", func() {
			nodeID := EncodeNodeID(k8skind.CM, "default", "my-cm")
			_, err := svc.GetNodeDetail(testCtx, "non-existent", "dev", "", nodeID)
			Expect(err).To(HaveOccurred())
		})

		It("should return detail with empty extras for unsupported kind", func() {
			snapshotWithCustom := &ResourceSnapshot{
				AppID:           "test-app-custom",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     1,
				RefreshStatus:   RefreshStatusSuccess,
				RefreshedAt:     time.Now(),
				Resources: []ResourceEntry{
					{Kind: "CustomResource", Namespace: "default", Name: "my-cr"},
				},
			}
			err := store.UpsertWithVersion(testCtx, snapshotWithCustom, 0)
			Expect(err).NotTo(HaveOccurred())

			nodeID := EncodeNodeID("CustomResource", "default", "my-cr")

			mockey.PatchConvey("custom-kind-detail", GinkgoT(), func() {
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "custom.io", Version: "v1", Resource: "customresources"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "custom.io/v1",
						"kind":       "CustomResource",
						"metadata": map[string]any{
							"name":              "my-cr",
							"namespace":         "default",
							"creationTimestamp": "2024-06-01T10:00:00Z",
						},
					},
				}, nil).Build()

				detail, err := svc.GetNodeDetail(testCtx, "test-app-custom", "dev", "", nodeID)
				Expect(err).NotTo(HaveOccurred())
				Expect(detail.Kind).To(Equal("CustomResource"))
				Expect(detail.Extras).To(BeNil())
			})
		})
	})

	Describe("ListNodeEvents", func() {
		const testProjectCode = "test-project"

		BeforeEach(func() {
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     1,
				RefreshStatus:   RefreshStatusSuccess,
				RefreshedAt:     time.Now(),
				Resources: []ResourceEntry{
					{
						Kind:       k8skind.Deploy,
						Namespace:  "default",
						Name:       "my-deploy",
						IsManaged:  true,
						SourceType: SourceTypeHelmManifest,
					},
				},
			}
			err := store.UpsertWithVersion(testCtx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return events from clusterresources API", func() {
			nodeID := EncodeNodeID(k8skind.Deploy, "default", "my-deploy")
			now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

			mockey.PatchConvey("list-events-ok", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
				mockey.Mock(clusterresources.New).Return(&clusterresources.ApiClient{}, nil).Build()
				mockey.Mock((*clusterresources.ApiClient).ListEvents).Return(&clusterresources.PaginatedEvents{
					Count: 2,
					Data: []clusterresources.EventEntry{
						{
							ClusterID:     "BCS-K8S-00001",
							Namespace:     "default",
							Level:         "Warning",
							Content:       "Back-off restarting failed container",
							Type:          "BackOff",
							ComponentName: "kubelet",
							ResourceKind:  k8skind.Deploy,
							ResourcesName: "my-deploy",
							CreatedAt:     now,
						},
						{
							ClusterID:     "BCS-K8S-00001",
							Namespace:     "default",
							Level:         "Normal",
							Content:       "Scaled up replica set my-deploy-abc to 3",
							Type:          "ScalingReplicaSet",
							ComponentName: "deployment-controller",
							ResourceKind:  k8skind.Deploy,
							ResourcesName: "my-deploy",
							CreatedAt:     now.Add(-2 * time.Hour),
						},
					},
				}, nil).Build()

				result, err := svc.ListNodeEvents(
					testCtx,
					"test-app",
					"dev",
					"",
					testProjectCode,
					nodeID,
					"", 0, 0,
					1,
					10,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Count).To(Equal(int64(2)))
				Expect(result.Data).To(HaveLen(2))
				// 第一个事件字段校验
				Expect(result.Data[0].Level).To(Equal("Warning"))
				Expect(result.Data[0].Content).To(Equal("Back-off restarting failed container"))
				Expect(result.Data[0].Type).To(Equal("BackOff"))
				Expect(result.Data[0].ClusterID).To(Equal("BCS-K8S-00001"))
				Expect(result.Data[0].CreatedAt).To(Equal(now))
				// 第二个事件字段校验
				Expect(result.Data[1].Level).To(Equal("Normal"))
				Expect(result.Data[1].ComponentName).To(Equal("deployment-controller"))
			})
		})

		It("should return empty list when no events exist", func() {
			nodeID := EncodeNodeID(k8skind.Deploy, "default", "my-deploy")

			mockey.PatchConvey("list-events-empty", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
				mockey.Mock(clusterresources.New).Return(&clusterresources.ApiClient{}, nil).Build()
				mockey.Mock((*clusterresources.ApiClient).ListEvents).Return(&clusterresources.PaginatedEvents{
					Count: 0,
					Data:  []clusterresources.EventEntry{},
				}, nil).Build()

				result, err := svc.ListNodeEvents(
					testCtx,
					"test-app",
					"dev",
					"",
					testProjectCode,
					nodeID,
					"", 0, 0,
					1,
					10,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Count).To(Equal(int64(0)))
				Expect(result.Data).To(BeEmpty())
			})
		})

		It("should return error when scope validation fails", func() {
			nodeID := EncodeNodeID(k8skind.Ing, "default", "not-in-scope")
			_, err := svc.ListNodeEvents(testCtx, "test-app", "dev", "", testProjectCode, nodeID, "", 0, 0, 1, 10)
			Expect(err).To(HaveOccurred())
		})

		It("should return error when clusterresources API fails", func() {
			nodeID := EncodeNodeID(k8skind.Deploy, "default", "my-deploy")

			mockey.PatchConvey("list-events-api-error", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
				mockey.Mock(clusterresources.New).Return(&clusterresources.ApiClient{}, nil).Build()
				mockey.Mock((*clusterresources.ApiClient).ListEvents).
					Return(nil, errors.New("connection refused")).
					Build()

				_, err := svc.ListNodeEvents(testCtx, "test-app", "dev", "", testProjectCode, nodeID, "", 0, 0, 1, 10)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("list events"))
			})
		})
	})

	Describe("GetNodeManifest", func() {
		BeforeEach(func() {
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     1,
				RefreshStatus:   RefreshStatusSuccess,
				RefreshedAt:     time.Now(),
				Resources: []ResourceEntry{
					{
						Kind:       k8skind.Deploy,
						Namespace:  "default",
						Name:       "my-deploy",
						IsManaged:  true,
						SourceType: SourceTypeHelmManifest,
					},
					{
						Kind:       k8skind.Secret,
						Namespace:  "default",
						Name:       "my-secret",
						IsManaged:  true,
						SourceType: SourceTypeHelmManifest,
					},
					{
						Kind:       k8skind.CM,
						Namespace:  "default",
						Name:       "my-config",
						IsManaged:  true,
						SourceType: SourceTypeHelmManifest,
					},
				},
			}
			err := store.UpsertWithVersion(testCtx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return raw manifest for non-Secret resources", func() {
			nodeID := EncodeNodeID(k8skind.Deploy, "default", "my-deploy")

			mockey.PatchConvey("get-manifest-raw", GinkgoT(), func() {
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       k8skind.Deploy,
						"metadata": map[string]any{
							"name":      "my-deploy",
							"namespace": "default",
						},
						"spec": map[string]any{
							"replicas": int64(3),
						},
					},
				}, nil).Build()

				manifest, err := svc.GetNodeManifest(testCtx, testApp, testEnv, "", nodeID)
				Expect(err).NotTo(HaveOccurred())
				Expect(manifest).NotTo(BeNil())
				Expect(manifest.Format).To(Equal("yaml"))
				Expect(manifest.Content).To(ContainSubstring("replicas: 3"))
			})
		})

		It("should mask sensitive env var values in manifest", func() {
			nodeID := EncodeNodeID(k8skind.Deploy, "default", "my-deploy")
			secretValue := "1"
			publicValue := "1"
			_, err := scopedEnvVarStore.Create(testCtx, envvars.ScopedEnvVar{
				WorkspaceID: testEnv.WorkspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  testEnv.Name,
				Key:         "SECRET_TOKEN",
				Value:       secretValue,
				IsSensitive: true,
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = scopedEnvVarStore.Create(testCtx, envvars.ScopedEnvVar{
				WorkspaceID: testEnv.WorkspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  testEnv.Name,
				Key:         "PUBLIC_VALUE",
				Value:       publicValue,
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("get-manifest-mask-sensitive-env-vars", GinkgoT(), func() {
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       k8skind.Deploy,
						"metadata": map[string]any{
							"name":      "my-deploy",
							"namespace": "default",
						},
						"spec": map[string]any{
							"replicas": int64(1),
							"template": map[string]any{
								"spec": map[string]any{
									"containers": []any{
										map[string]any{
											"name": "main",
											"env": []any{
												map[string]any{"name": "SECRET_TOKEN", "value": secretValue},
												map[string]any{"name": "PUBLIC_VALUE", "value": publicValue},
											},
										},
									},
								},
							},
						},
					},
				}, nil).Build()

				manifest, err := svc.GetNodeManifest(testCtx, testApp, testEnv, "", nodeID)
				Expect(err).NotTo(HaveOccurred())

				var manifestObj map[string]any
				Expect(yaml.Unmarshal([]byte(manifest.Content), &manifestObj)).To(Succeed())
				replicas, found, err := unstructured.NestedFieldNoCopy(manifestObj, "spec", "replicas")
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(replicas).To(BeNumerically("==", 1))

				containers, found, err := unstructured.NestedSlice(
					manifestObj, "spec", "template", "spec", "containers",
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(containers).To(HaveLen(1))

				container := containers[0].(map[string]any)
				envList, found, err := unstructured.NestedSlice(container, "env")
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(envList).To(ConsistOf(
					HaveKeyWithValue("name", "SECRET_TOKEN"),
					HaveKeyWithValue("name", "PUBLIC_VALUE"),
				))
				Expect(envList).To(ContainElement(SatisfyAll(
					HaveKeyWithValue("name", "SECRET_TOKEN"),
					HaveKeyWithValue("value", envvartypes.SensitiveValueMask),
				)))
				Expect(envList).To(ContainElement(SatisfyAll(
					HaveKeyWithValue("name", "PUBLIC_VALUE"),
					HaveKeyWithValue("value", publicValue),
				)))
			})
		})

		It("should mask sensitive ConfigMap data in manifest", func() {
			nodeID := EncodeNodeID(k8skind.CM, "default", "my-config")
			_, err := scopedEnvVarStore.Create(testCtx, envvars.ScopedEnvVar{
				WorkspaceID: testEnv.WorkspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  testEnv.Name,
				Key:         "SECRET_TOKEN",
				Value:       "plain-secret",
				IsSensitive: true,
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("get-manifest-mask-sensitive-configmap-data", GinkgoT(), func() {
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       k8skind.CM,
						"metadata": map[string]any{
							"name":      "my-config",
							"namespace": "default",
						},
						"data": map[string]any{
							"SECRET_TOKEN": "plain-secret",
							"PUBLIC_VALUE": "public",
						},
						"binaryData": map[string]any{
							"SECRET_TOKEN": "cGxhaW4tc2VjcmV0",
							"PUBLIC_VALUE": "cHVibGlj",
						},
					},
				}, nil).Build()

				manifest, err := svc.GetNodeManifest(testCtx, testApp, testEnv, "", nodeID)
				Expect(err).NotTo(HaveOccurred())

				var manifestObj map[string]any
				Expect(yaml.Unmarshal([]byte(manifest.Content), &manifestObj)).To(Succeed())
				data, found, err := unstructured.NestedMap(manifestObj, "data")
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(data["SECRET_TOKEN"]).To(Equal(envvartypes.SensitiveValueMask))
				Expect(data["PUBLIC_VALUE"]).To(Equal("public"))

				binaryData, found, err := unstructured.NestedMap(manifestObj, "binaryData")
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(binaryData["SECRET_TOKEN"]).To(Equal(envvartypes.SensitiveValueMask))
				Expect(binaryData["PUBLIC_VALUE"]).To(Equal("cHVibGlj"))
			})
		})

		It("should mask sensitive env vars by key when configured value is empty", func() {
			nodeID := EncodeNodeID(k8skind.Deploy, "default", "my-deploy")
			_, err := scopedEnvVarStore.Create(testCtx, envvars.ScopedEnvVar{
				WorkspaceID: testEnv.WorkspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  testEnv.Name,
				Key:         "SECRET_TOKEN",
				Value:       "",
				IsSensitive: true,
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.PatchConvey("get-manifest-mask-sensitive-env-var-empty-value", GinkgoT(), func() {
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "apps/v1",
						"kind":       k8skind.Deploy,
						"metadata": map[string]any{
							"name":      "my-deploy",
							"namespace": "default",
						},
						"spec": map[string]any{
							"template": map[string]any{
								"spec": map[string]any{
									"containers": []any{
										map[string]any{
											"name": "main",
											"env": []any{
												map[string]any{
													"name":  "SECRET_TOKEN",
													"value": "runtime-secret",
												},
											},
										},
									},
								},
							},
						},
					},
				}, nil).Build()

				manifest, err := svc.GetNodeManifest(testCtx, testApp, testEnv, "", nodeID)
				Expect(err).NotTo(HaveOccurred())

				var manifestObj map[string]any
				Expect(yaml.Unmarshal([]byte(manifest.Content), &manifestObj)).To(Succeed())
				containers, found, err := unstructured.NestedSlice(
					manifestObj, "spec", "template", "spec", "containers",
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(containers).To(HaveLen(1))

				container := containers[0].(map[string]any)
				envList, found, err := unstructured.NestedSlice(container, "env")
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(envList).To(ContainElement(SatisfyAll(
					HaveKeyWithValue("name", "SECRET_TOKEN"),
					HaveKeyWithValue("value", envvartypes.SensitiveValueMask),
				)))
			})
		})

		It("should return sanitized manifest for Secret resources", func() {
			nodeID := EncodeNodeID(k8skind.Secret, "default", "my-secret")

			mockey.PatchConvey("get-manifest-secret", GinkgoT(), func() {
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       k8skind.Secret,
						"metadata": map[string]any{
							"name":      "my-secret",
							"namespace": "default",
						},
						"type": "Opaque",
						"data": map[string]any{
							"password": "c2VjcmV0",
						},
						"stringData": map[string]any{
							"token": "plain-token",
						},
					},
				}, nil).Build()

				manifest, err := svc.GetNodeManifest(testCtx, testApp, testEnv, "", nodeID)
				Expect(err).NotTo(HaveOccurred())
				Expect(manifest.Content).NotTo(ContainSubstring("c2VjcmV0"))
				Expect(manifest.Content).NotTo(ContainSubstring("plain-token"))
				Expect(manifest.Content).To(ContainSubstring(envvartypes.SensitiveValueMask))
			})
		})

		It("should return error when scope validation fails", func() {
			nodeID := EncodeNodeID(k8skind.Ing, "default", "not-in-scope")
			_, err := svc.GetNodeManifest(testCtx, testApp, testEnv, "", nodeID)
			Expect(err).To(HaveOccurred())
		})

		It("should return error when resource does not exist in cluster", func() {
			nodeID := EncodeNodeID(k8skind.Deploy, "default", "my-deploy")

			mockey.PatchConvey("get-manifest-not-found", GinkgoT(), func() {
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(nil, errors.New("not found")).Build()

				_, err := svc.GetNodeManifest(testCtx, testApp, testEnv, "", nodeID)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
