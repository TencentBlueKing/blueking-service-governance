package topology

import (
	"context"
	"sync"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	helminfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

var _ = Describe("Refresher helpers", func() {
	Describe("hasOwnerRef", func() {
		It("should return true when ownerReference matches", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "ReplicaSet",
				"metadata": map[string]any{
					"name":      "nginx-abc123",
					"namespace": "default",
					"ownerReferences": []any{
						map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"name":       "nginx",
							"uid":        "12345",
						},
					},
				},
			}}

			Expect(hasOwnerRef(obj, "Deployment", "nginx")).To(BeTrue())
		})

		It("should return false when ownerReference does not match", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "ReplicaSet",
				"metadata": map[string]any{
					"name":      "nginx-abc123",
					"namespace": "default",
					"ownerReferences": []any{
						map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"name":       "nginx",
							"uid":        "12345",
						},
					},
				},
			}}

			Expect(hasOwnerRef(obj, "Deployment", "other-deploy")).To(BeFalse())
			Expect(hasOwnerRef(obj, "StatefulSet", "nginx")).To(BeFalse())
		})

		It("should return false when no ownerReferences exist", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name":      "my-cm",
					"namespace": "default",
				},
			}}

			Expect(hasOwnerRef(obj, "Deployment", "nginx")).To(BeFalse())
		})
	})
})

var _ = Describe("supplementOwnerRefChain", func() {
	var (
		ctx              context.Context
		refresher        *Refresher
		clusterResources map[string]*unstructured.Unstructured
		mu               sync.Mutex
		mocker           *mockey.Mocker
	)

	BeforeEach(func() {
		ctx = context.Background()
		refresher = &Refresher{}
		clusterResources = make(map[string]*unstructured.Unstructured)
		mu = sync.Mutex{}

		cfg := &cluster.Config{ClusterID: "test-cluster"}
		mocker = mockey.Mock(cluster.NewConfig).Return(cfg).Build()
	})

	AfterEach(func() {
		mocker.Release()
	})

	Describe("Deployment scenario", func() {
		It("should supplement only ReplicaSet without Pod", func() {
			mockey.PatchConvey("deploy supplements RS only", GinkgoT(), func() {
				mockGVR := &schema.GroupVersionResource{
					Group: "apps", Version: "v1", Resource: "replicasets",
				}
				mockey.Mock(discovery.GetGroupVersionResource).Return(mockGVR, nil).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()

				// 模拟 List 返回一个属于 nginx Deployment 的活跃 RS
				rsList := &unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "ReplicaSet",
							"metadata": map[string]any{
								"name":      "nginx-rs-abc",
								"namespace": "default",
								"ownerReferences": []any{
									map[string]any{
										"apiVersion": "apps/v1",
										"kind":       "Deployment",
										"name":       "nginx",
										"uid":        "deploy-uid-1",
									},
								},
							},
							"spec": map[string]any{
								"replicas": int64(2),
							},
						}},
						// 不活跃的 RS（replicas=0），应被过滤
						{Object: map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "ReplicaSet",
							"metadata": map[string]any{
								"name":      "nginx-rs-old",
								"namespace": "default",
								"ownerReferences": []any{
									map[string]any{
										"apiVersion": "apps/v1",
										"kind":       "Deployment",
										"name":       "nginx",
										"uid":        "deploy-uid-1",
									},
								},
							},
							"spec": map[string]any{
								"replicas": int64(0),
							},
						}},
					},
				}
				mockey.Mock((*k8sclient.Client).List).Return(rsList, nil).Build()

				entry := ResourceEntry{
					Kind: k8skind.Deploy, Namespace: "default", Name: "nginx",
				}
				clusterCfg := cluster.NewConfig("test-cluster")
				err := refresher.supplementOwnerRefChain(
					ctx, clusterCfg, "default", entry, clusterResources, &mu,
				)

				Expect(err).NotTo(HaveOccurred())

				// 活跃 RS 应被补充到 clusterResources
				rsKey := ResourceKey(k8skind.RS, "default", "nginx-rs-abc")
				Expect(clusterResources).To(HaveKey(rsKey))

				// 不活跃 RS 不应被补充
				oldRSKey := ResourceKey(k8skind.RS, "default", "nginx-rs-old")
				Expect(clusterResources).NotTo(HaveKey(oldRSKey))

				// Pod 不应被补充（由 Builder 实时发现）
				for key := range clusterResources {
					Expect(key).NotTo(ContainSubstring("Pod/"))
				}
			})
		})
	})

	Describe("CronJob scenario", func() {
		It("should supplement only Job without Pod", func() {
			mockey.PatchConvey("cronjob supplements Job only", GinkgoT(), func() {
				mockGVR := &schema.GroupVersionResource{
					Group: "batch", Version: "v1", Resource: "jobs",
				}
				mockey.Mock(discovery.GetGroupVersionResource).Return(mockGVR, nil).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()

				jobList := &unstructured.UnstructuredList{
					Items: []unstructured.Unstructured{
						{Object: map[string]any{
							"apiVersion": "batch/v1",
							"kind":       "Job",
							"metadata": map[string]any{
								"name":      "my-cj-12345",
								"namespace": "default",
								"ownerReferences": []any{
									map[string]any{
										"apiVersion": "batch/v1",
										"kind":       "CronJob",
										"name":       "my-cj",
										"uid":        "cj-uid-1",
									},
								},
							},
						}},
					},
				}
				mockey.Mock((*k8sclient.Client).List).Return(jobList, nil).Build()

				entry := ResourceEntry{
					Kind: k8skind.CJ, Namespace: "default", Name: "my-cj",
				}
				clusterCfg := cluster.NewConfig("test-cluster")
				err := refresher.supplementOwnerRefChain(
					ctx, clusterCfg, "default", entry, clusterResources, &mu,
				)

				Expect(err).NotTo(HaveOccurred())

				// Job 应被补充到 clusterResources
				jobKey := ResourceKey(k8skind.Job, "default", "my-cj-12345")
				Expect(clusterResources).To(HaveKey(jobKey))

				// Pod 不应被补充
				for key := range clusterResources {
					Expect(key).NotTo(ContainSubstring("Pod/"))
				}
			})
		})
	})

	Describe("workloads without intermediate layer", func() {
		It("should not supplement any resource for StatefulSet", func() {
			entry := ResourceEntry{
				Kind: k8skind.STS, Namespace: "default", Name: "my-sts",
			}
			clusterCfg := cluster.NewConfig("test-cluster")
			err := refresher.supplementOwnerRefChain(
				ctx, clusterCfg, "default", entry, clusterResources, &mu,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(clusterResources).To(BeEmpty())
		})

		It("should not supplement any resource for DaemonSet", func() {
			entry := ResourceEntry{
				Kind: k8skind.DS, Namespace: "default", Name: "my-ds",
			}
			clusterCfg := cluster.NewConfig("test-cluster")
			err := refresher.supplementOwnerRefChain(
				ctx, clusterCfg, "default", entry, clusterResources, &mu,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(clusterResources).To(BeEmpty())
		})

		It("should not supplement any resource for unregistered kind", func() {
			entry := ResourceEntry{
				Kind: k8skind.SVC, Namespace: "default", Name: "my-svc",
			}
			clusterCfg := cluster.NewConfig("test-cluster")
			err := refresher.supplementOwnerRefChain(
				ctx, clusterCfg, "default", entry, clusterResources, &mu,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(clusterResources).To(BeEmpty())
		})
	})
})

var _ = Describe("Refresher", func() {
	Describe("Refresh", func() {
		var (
			ctx   context.Context
			store *ResourceSnapshotStoreMongo
		)

		BeforeEach(func() {
			ctx = context.Background()
			var err error
			// 构建真实 ResourceSnapshotStoreMongo，并在每个用例前清空测试数据
			store, err = NewResourceSnapshotStoreMongo(database.Client(), database.Name())
			Expect(err).NotTo(HaveOccurred())
			err = store.DeleteAll(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = store.DeleteAll(ctx)
		})

		It("should generate success snapshot when all declared resources exist", func() {
			refresher := NewRefresher(store)

			mockey.PatchConvey("all-declared-resources-exist", GinkgoT(), func() {
				cfg := &cluster.Config{ClusterID: "test-cluster"}
				mockey.Mock(cluster.NewConfig).Return(cfg).Build()
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(&unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "v1",
					"kind":       k8skind.CM,
					"metadata": map[string]any{
						"name":      "existing-cm",
						"namespace": "default",
					},
				}}, nil).Build()

				err := refresher.Refresh(ctx, RefreshArgs{
					AppID:     "test-app",
					EnvName:   "dev",
					ClusterID: "test-cluster",
					Namespace: "default",
					ResourceKeys: []ResourceKeyEntry{
						{Kind: k8skind.CM, Name: "existing-cm"},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				// 直接从真实 store 回读 snapshot 断言
				got, err := store.Get(ctx, "test-app", "dev", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(got).NotTo(BeNil())
				Expect(got.RefreshStatus).To(Equal(RefreshStatusSuccess))
				Expect(got.WarningSummary).To(BeEmpty())
				// 首次刷新：DataVersion 从初始 0 推进到 1
				Expect(got.DataVersion).To(Equal(int64(1)))
				Expect(got.RefreshedAt).NotTo(BeZero())
				Expect(got.Resources).To(HaveLen(1))
				Expect(got.Resources[0].Name).To(Equal("existing-cm"))
			})
		})

		It("should generate partial success snapshot when some declared resources are missing", func() {
			refresher := NewRefresher(store)

			mockey.PatchConvey("appmodel-partial-success", GinkgoT(), func() {
				cfg := &cluster.Config{ClusterID: "test-cluster"}
				mockey.Mock(cluster.NewConfig).Return(cfg).Build()
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).To(func(
					_ *k8sclient.Client, _ context.Context, _, name string, _ metav1.GetOptions,
				) (*unstructured.Unstructured, error) {
					if name == "existing-cm" {
						return &unstructured.Unstructured{Object: map[string]any{
							"apiVersion": "v1",
							"kind":       k8skind.CM,
							"metadata": map[string]any{
								"name":      "existing-cm",
								"namespace": "default",
							},
						}}, nil
					}
					return nil, k8sclient.ErrResourceNotFound
				}).Build()

				err := refresher.Refresh(ctx, RefreshArgs{
					AppID:     "test-app",
					EnvName:   "dev",
					ClusterID: "test-cluster",
					Namespace: "default",
					ResourceKeys: []ResourceKeyEntry{
						{Kind: k8skind.CM, Name: "existing-cm"},
						{Kind: k8skind.CM, Name: "missing-cm"},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				got, err := store.Get(ctx, "test-app", "dev", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(got).NotTo(BeNil())
				Expect(got.RefreshStatus).To(Equal(RefreshStatusPartialSuccess))
				Expect(got.WarningSummary).To(
					Equal("partial success: 1 of 2 declared resources missing in cluster"),
				)
				Expect(got.DataVersion).To(Equal(int64(1)))
				Expect(got.Resources).To(HaveLen(1))
				Expect(got.Resources[0].Name).To(Equal("existing-cm"))
			})
		})

		It("should keep previous snapshot when helm release is not found", func() {
			previousSnapshot := &ResourceSnapshot{
				AppID:         "test-app",
				EnvName:       "dev",
				ClusterID:     "test-cluster",
				Namespace:     "default",
				DataVersion:   3,
				RefreshStatus: RefreshStatusSuccess,
				Resources: []ResourceEntry{
					{Kind: k8skind.CM, Namespace: "default", Name: "existing-cm"},
				},
			}
			err := store.UpsertWithVersion(ctx, previousSnapshot, 2)
			Expect(err).NotTo(HaveOccurred())

			refresher := NewRefresher(store)

			mockey.PatchConvey("helm-release-not-found", GinkgoT(), func() {
				mockey.Mock(helminfra.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				mockey.Mock(helminfra.GetReleaseManifest).Return(
					"", errors.Wrap(driver.ErrReleaseNotFound, "wrapped release error"),
				).Build()

				refreshErr := refresher.Refresh(ctx, RefreshArgs{
					AppID:       "test-app",
					EnvName:     "dev",
					ClusterID:   "test-cluster",
					Namespace:   "default",
					ReleaseName: "missing-release",
				})
				Expect(refreshErr).NotTo(HaveOccurred())

				got, err := store.Get(ctx, "test-app", "dev", "")
				Expect(err).NotTo(HaveOccurred())
				Expect(got).NotTo(BeNil())
				Expect(got.DataVersion).To(Equal(int64(3)))
				Expect(got.RefreshStatus).To(Equal(RefreshStatusSuccess))
				Expect(got.WarningSummary).To(BeEmpty())
				Expect(got.Resources).To(HaveLen(1))
				Expect(got.Resources[0].Name).To(Equal("existing-cm"))
			})
		})
	})
})
