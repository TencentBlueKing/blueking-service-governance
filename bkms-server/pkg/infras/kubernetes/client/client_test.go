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

package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
)

var _ = Describe("Client", func() {
	var (
		cli       *Client
		ctx       context.Context
		namespace string
		name      string
		manifest  map[string]any
		clientSet *kubernetes.Clientset
	)

	BeforeEach(func() {
		ctx = context.Background()

		cfg, err := testutil.TestClusterConfig("")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())

		clientSet, err = kubernetes.NewForConfig(cfg.Rest)
		Expect(err).NotTo(HaveOccurred())

		// 预先初始化命名空间
		namespace = stringx.Random(8)
		_, err = clientSet.CoreV1().Namespaces().Create(
			ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{},
		)
		Expect(err).NotTo(HaveOccurred())

		// 准备测试用资源
		cli = NewWithGVR(cfg, schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"})
		name = fmt.Sprintf("nginx-deployment-%s", stringx.Random(8))
		manifest = map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata":   map[string]any{"name": name},
			"spec": map[string]any{
				"replicas": 3,
				"selector": map[string]any{
					"matchLabels": map[string]any{"app": "nginx"},
				},
				"template": map[string]any{
					"metadata": map[string]any{
						"labels": map[string]any{"app": "nginx"},
					},
					"spec": map[string]any{
						"containers": []map[string]any{
							{
								"name":  "nginx",
								"image": "nginx:1.14.2",
								"ports": []map[string]any{{"containerPort": 80}},
							},
						},
					},
				},
			},
		}
	})

	AfterEach(func() {
		// 回收命名空间
		err := clientSet.CoreV1().Namespaces().Delete(
			ctx, namespace, metav1.DeleteOptions{
				GracePeriodSeconds: lo.ToPtr(int64(0)),
			},
		)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("create, update, retrieve, delete", func() {
		It("should not errors", func() {
			// 新建资源
			_, err := cli.Create(ctx, namespace, manifest, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// 获取 Deployment 列表
			uList, err := cli.List(ctx, namespace, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(uList.Items)).NotTo(BeZero())

			// 修改副本数量
			err = mapx.SetItems(manifest, "spec.replicas", 2)
			Expect(err).NotTo(HaveOccurred())

			// 更新到集群
			_, err = cli.Update(ctx, namespace, name, manifest, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// 从集群获取资源
			ret, err := cli.Get(ctx, namespace, name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ret.GetNamespace()).To(Equal(namespace))
			Expect(ret.GetName()).To(Equal(name))
			Expect(mapx.GetInt64(ret.Object, "spec.replicas")).To(Equal(int64(2)))

			// 删除资源
			err = cli.Delete(ctx, namespace, name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			// 重复删除资源是被允许的
			err = cli.Delete(ctx, namespace, name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("get action error occurred", func() {
			// 模拟获取操作报错
			_, err := cli.Get(ctx, namespace, name+"abcd", metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrResourceNotFound))
		})

		It("create action error occurred", func() {
			// 模拟重复创建操作报错
			_, err := cli.Create(ctx, namespace, manifest, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			_, err = cli.Create(ctx, namespace, manifest, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrResourceAlreadyExists))
		})

		It("update action error occurred", func() {
			_, err := cli.Create(ctx, namespace, manifest, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// 模拟更新操作报错
			manifest["spec"] = map[string]any{"replicas": 2}
			_, err = cli.Update(ctx, namespace, name, manifest, metav1.UpdateOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("update apps/v1, Resource=deployments in namespace"))
		})
	})

	Context("PaginateList", func() {
		BeforeEach(func() {
			for i := 0; i < 2; i++ {
				// 复用 manifest 配置，只修改名称
				manifest["metadata"] = map[string]any{
					"name":   fmt.Sprintf("nginx-deployment-%s-%d", stringx.Random(8), i),
					"labels": map[string]string{"app": "nginx"},
				}

				_, err := cli.Create(ctx, namespace, manifest, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should return first page with correct page size", func() {
			// 获取第 1 页，每页 2 条
			result, err := cli.PaginateList(ctx, namespace, 1, 2, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(result.Items)).To(Equal(2))
		})

		It("should return second page with correct page size", func() {
			// 获取第 1 页，每页 2 条
			result, err := cli.PaginateList(ctx, namespace, 2, 1, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(result.Items)).To(Equal(1))
		})

		It("should return empty list when page exceeds total pages", func() {
			// 获取超出范围的页码
			result, err := cli.PaginateList(ctx, namespace, 100, 2, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(result.Items)).To(Equal(0))
		})

		It("should work with label selector", func() {
			// 测试带标签选择器的分页查询
			result, err := cli.PaginateList(ctx, namespace, 1, 10, metav1.ListOptions{
				LabelSelector: "app=nginx",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(len(result.Items)).To(Equal(2))
		})
	})
})

var _ = Describe("withNamespace", func() {
	It("should inject namespace without mutating the input manifest", func() {
		metadata := map[string]any{"name": "nginx"}
		manifest := map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata":   metadata,
			// 调用方常写 Go 字面量 int，这类值无法通过 unstructured.DeepCopy
			"spec": map[string]any{"replicas": 3},
		}

		obj := withMeta(manifest, "default")
		Expect(mapx.GetStr(obj, "metadata.namespace")).To(Equal("default"))
		Expect(mapx.GetStr(obj, "metadata.name")).To(Equal("nginx"))

		Expect(manifest["metadata"]).To(Equal(map[string]any{"name": "nginx"}))
		Expect(metadata).NotTo(HaveKey("namespace"))
	})

	It("should not inject namespace when it is empty", func() {
		manifest := map[string]any{"metadata": map[string]any{"name": "nginx"}}
		Expect(withMeta(manifest, "")).To(Equal(manifest))
		Expect(mapx.GetStr(manifest, "metadata.namespace")).To(BeEmpty())
	})

	It("should create metadata when it is missing", func() {
		obj := withMeta(map[string]any{"kind": "Deployment"}, "default")
		Expect(mapx.GetStr(obj, "metadata.namespace")).To(Equal("default"))
	})
})

var _ = Describe("Test upsert method", func() {
	var (
		cli       *Client
		deployCli *Client
		clientSet *kubernetes.Clientset
		ctx       context.Context
	)

	BeforeEach(func() {
		cfg, err := testutil.TestClusterConfig("")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())

		cli = NewWithGVR(cfg, gvr.SVC)
		deployCli = NewWithGVR(cfg, gvr.Deploy)
		clientSet, err = kubernetes.NewForConfig(cfg.Rest)
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
	})

	It("should create a service when it does not exist", func() {
		namespace := "default"
		svcName := fmt.Sprintf("svc-%s", stringx.Random(8))

		_, err := cli.Get(ctx, namespace, svcName, metav1.GetOptions{})
		Expect(errors.Is(err, ErrResourceNotFound)).To(BeTrue())

		svcManifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": svcName,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "api",
						"port":       80,
						"targetPort": 80,
						"protocol":   "TCP",
					},
				},
			},
		}

		_, err = cli.Upsert(ctx, namespace, svcManifest, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		result, err := cli.Get(ctx, namespace, svcName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.GetName()).To(Equal(svcName))
		Expect(result.GetKind()).To(Equal("Service"))

		ports := mapx.GetList(result.Object, "spec.ports")
		Expect(ports).To(HaveLen(1))
		Expect(ports[0].(map[string]any)["name"]).To(Equal("api"))

		// cleanup
		err = cli.Delete(ctx, namespace, svcName, metav1.DeleteOptions{GracePeriodSeconds: lo.ToPtr(int64(0))})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should update a service when it already exists", func() {
		namespace := "default"
		svcName := fmt.Sprintf("svc-%s", stringx.Random(8))

		svcManifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": svcName,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "api",
						"port":       80,
						"targetPort": 80,
						"protocol":   "TCP",
					},
				},
			},
		}

		// create the service first
		_, err := cli.Upsert(ctx, namespace, svcManifest, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		// upsert with updated ports
		updatedManifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": svcName,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "metrics",
						"port":       8080,
						"targetPort": 8080,
						"protocol":   "TCP",
					},
					map[string]any{
						"name":       "web",
						"port":       80,
						"targetPort": 80,
						"protocol":   "TCP",
					},
				},
			},
		}

		result, err := cli.Upsert(ctx, namespace, updatedManifest, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.GetName()).To(Equal(svcName))

		ports := mapx.GetList(result.Object, "spec.ports")
		Expect(ports).To(HaveLen(2))
		Expect(ports[0].(map[string]any)["port"]).To(BeNumerically("==", 8080))
		Expect(ports[0].(map[string]any)["name"]).To(Equal("metrics"))
		Expect(ports[1].(map[string]any)["port"]).To(BeNumerically("==", 80))
		Expect(ports[1].(map[string]any)["name"]).To(Equal("web"))

		// cleanup
		err = cli.Delete(ctx, namespace, svcName, metav1.DeleteOptions{GracePeriodSeconds: lo.ToPtr(int64(0))})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should preserve clusterIP and clusterIPs after update", func() {
		namespace := "default"
		svcName := fmt.Sprintf("svc-%s", stringx.Random(8))

		svcManifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": svcName,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "api",
						"port":       80,
						"targetPort": 80,
						"protocol":   "TCP",
					},
				},
			},
		}

		// create the service
		_, err := cli.Upsert(ctx, namespace, svcManifest, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		// retrieve server-assigned clusterIP and clusterIPs
		created, err := cli.Get(ctx, namespace, svcName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		originalClusterIP := mapx.GetStr(created.Object, "spec.clusterIP")
		originalClusterIPs := mapx.GetList(created.Object, "spec.clusterIPs")
		Expect(originalClusterIP).NotTo(BeEmpty())
		Expect(originalClusterIPs).NotTo(BeEmpty())

		// upsert again with different ports (no clusterIP in manifest)
		updatedManifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": svcName,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "new-api",
						"port":       9090,
						"targetPort": 9090,
						"protocol":   "TCP",
					},
				},
			},
		}

		_, err = cli.Upsert(ctx, namespace, updatedManifest, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		// verify clusterIP and clusterIPs are preserved
		updated, err := cli.Get(ctx, namespace, svcName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(mapx.GetStr(updated.Object, "spec.clusterIP")).To(Equal(originalClusterIP))
		Expect(mapx.GetList(updated.Object, "spec.clusterIPs")).To(Equal(originalClusterIPs))

		// cleanup
		err = cli.Delete(ctx, namespace, svcName, metav1.DeleteOptions{GracePeriodSeconds: lo.ToPtr(int64(0))})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not modify the input manifest (no side effects)", func() {
		namespace := "default"
		svcName := fmt.Sprintf("svc-%s", stringx.Random(8))

		svcManifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": svcName,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "api",
						"port":       80,
						"targetPort": 80,
						"protocol":   "TCP",
					},
				},
			},
		}

		// deep copy the manifest to compare later
		originalMetadata := map[string]any{"name": svcName}

		_, err := cli.Upsert(ctx, namespace, svcManifest, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		// verify manifest was not mutated (no resourceVersion injected, no extra fields)
		Expect(svcManifest["metadata"]).To(Equal(originalMetadata))
		Expect(mapx.GetStr(svcManifest, "metadata.resourceVersion")).To(BeEmpty())

		// cleanup
		err = cli.Delete(ctx, namespace, svcName, metav1.DeleteOptions{GracePeriodSeconds: lo.ToPtr(int64(0))})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should preserve clusterIP when using merge patch for federation clusters", func() {
		namespace := "default"
		svcName := fmt.Sprintf("svc-%s", stringx.Random(8))

		svcManifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": svcName,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "api",
						"port":       80,
						"targetPort": 80,
						"protocol":   "TCP",
					},
				},
			},
		}

		_, err := cli.upsertMergePatch(ctx, namespace, svcManifest, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		created, err := cli.Get(ctx, namespace, svcName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		originalClusterIP := mapx.GetStr(created.Object, "spec.clusterIP")
		Expect(originalClusterIP).NotTo(BeEmpty())

		updatedManifest := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name": svcName,
			},
			"spec": map[string]any{
				"ports": []any{
					map[string]any{
						"name":       "new-api",
						"port":       9090,
						"targetPort": 9090,
						"protocol":   "TCP",
					},
				},
			},
		}
		_, err = cli.upsertMergePatch(ctx, namespace, updatedManifest, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())

		updated, err := cli.Get(ctx, namespace, svcName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(mapx.GetStr(updated.Object, "spec.clusterIP")).To(Equal(originalClusterIP))

		err = cli.Delete(ctx, namespace, svcName, metav1.DeleteOptions{GracePeriodSeconds: lo.ToPtr(int64(0))})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should drop omitted initContainers and labels while keeping injected labels", func() {
		namespace := "default"
		deployName := fmt.Sprintf("fed-upsert-%s", stringx.Random(8))
		podLabels := map[string]any{"app": "fed-upsert"}
		mkDeploy := func(labels map[string]any, withInit bool) map[string]any {
			podSpec := map[string]any{
				"containers": []any{
					map[string]any{"name": "nginx", "image": "nginx:latest"},
				},
			}
			if withInit {
				podSpec["initContainers"] = []any{
					map[string]any{
						"name":    "init",
						"image":   "busybox:latest",
						"command": []any{"sh", "-c", "true"},
					},
				}
			}
			return map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":   deployName,
					"labels": labels,
				},
				"spec": map[string]any{
					"replicas": 1,
					"selector": map[string]any{"matchLabels": podLabels},
					"template": map[string]any{
						"metadata": map[string]any{"labels": podLabels},
						"spec":     podSpec,
					},
				},
			}
		}

		_, err := deployCli.upsertMergePatch(
			ctx, namespace,
			mkDeploy(map[string]any{"app": "fed-upsert", "component": "sidecar"}, true),
			metav1.PatchOptions{},
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = clientSet.AppsV1().Deployments(namespace).Patch(
			ctx, deployName, types.MergePatchType,
			[]byte(`{"metadata":{"labels":{"injected":"keep"}}}`),
			metav1.PatchOptions{},
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = deployCli.upsertMergePatch(
			ctx, namespace,
			mkDeploy(map[string]any{"app": "fed-upsert"}, false),
			metav1.PatchOptions{},
		)
		Expect(err).NotTo(HaveOccurred())

		got, err := deployCli.Get(ctx, namespace, deployName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(got.GetLabels()).To(HaveKeyWithValue("app", "fed-upsert"))
		Expect(got.GetLabels()).To(HaveKeyWithValue("injected", "keep"))
		Expect(got.GetLabels()).NotTo(HaveKey("component"))
		Expect(mapx.GetList(got.Object, "spec.template.spec.initContainers")).To(BeEmpty())

		err = deployCli.Delete(
			ctx,
			namespace,
			deployName,
			metav1.DeleteOptions{GracePeriodSeconds: lo.ToPtr(int64(0))},
		)
		Expect(err).NotTo(HaveOccurred())
	})
})
