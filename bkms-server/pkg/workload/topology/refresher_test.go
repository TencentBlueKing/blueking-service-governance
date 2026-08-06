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
	"sync"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

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
