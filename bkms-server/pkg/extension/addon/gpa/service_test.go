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

package gpa

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/ownership"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
)

// newTestEnv 构造一个最小可用的 Environment（含 cluster/namespace/workspace/env name）
func newTestEnv() *bkmsenv.Environment {
	env := &bkmsenv.Environment{
		Name:        "dev",
		WorkspaceID: "ws-1",
	}
	env.Cluster.ClusterID = "cluster-1"
	env.Cluster.Namespace = "ns-1"
	return env
}

var _ = Describe("GPAService K8s interactions", func() {
	var (
		svc *GPAService
		ctx context.Context
		env *bkmsenv.Environment
	)

	BeforeEach(func() {
		svc = NewGPAService(testutil.NewDB(GinkgoT()))
		ctx = testutil.NewDefaultTenantContext()
		env = newTestEnv()
	})

	// mockK8sClientChain mock 出 newK8sClient 链路（cluster.NewConfig / GetGroupVersionResource / NewWithGVR），
	// 返回一个空的 *k8sclient.Client，供各方法调用具体的 K8s 操作（操作本身由各用例单独 mock）。
	mockK8sClientChain := func() {
		mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build()
		mockey.Mock(discovery.GetGroupVersionResource).Return(
			&schema.GroupVersionResource{
				Group: "autoscaling.tkex.tencent.com", Version: "v1alpha1", Resource: "generalpodautoscalers",
			}, nil,
		).Build()
		mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
	}

	// mockNoLiveCR mock 下发前的归属校验：集群中不存在同名 CR
	mockNoLiveCR := func() {
		mockey.Mock((*k8sclient.Client).Get).Return(nil, k8sclient.ErrResourceNotFound).Build()
	}

	Describe("Apply", func() {
		It("should upsert the gpa manifest after passing validation", func() {
			mockey.PatchConvey("apply-success", GinkgoT(), func() {
				// 跳过工作负载名解析，聚焦 K8s 下发编排
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("web", nil).Build()
				mockK8sClientChain()
				mockNoLiveCR()

				var capturedNamespace string
				var capturedManifest map[string]any
				mockey.Mock((*k8sclient.Client).Upsert).To(func(
					_ *k8sclient.Client, _ context.Context, namespace string,
					manifest map[string]any, _ metav1.PatchOptions,
				) (*unstructured.Unstructured, error) {
					capturedNamespace = namespace
					capturedManifest = manifest
					return &unstructured.Unstructured{Object: manifest}, nil
				}).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(err).NotTo(HaveOccurred())

				// 下发到环境对应的 namespace，且 manifest 带上 scaleTargetRef.name
				Expect(capturedNamespace).To(Equal("ns-1"))
				spec := capturedManifest["spec"].(map[string]any)
				Expect(spec["scaleTargetRef"].(map[string]any)["name"]).To(Equal("web"))
			})
		})

		It("should reject apply on a federation env", func() {
			mockey.PatchConvey("apply-federation-unsupported", GinkgoT(), func() {
				env.Cluster.IsFederation = true
				upsertMock := mockey.Mock((*k8sclient.Client).Upsert).Return(nil, nil).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(err).To(MatchError(ErrFederationNotSupported))
				Expect(upsertMock.Times()).To(Equal(0))
			})
		})

		It("should not call k8s when the scale target cannot be resolved", func() {
			mockey.PatchConvey("apply-resolve-target-fail", GinkgoT(), func() {
				resolveErr := errors.New("resolve target failed")
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("", resolveErr).Build()
				upsertMock := mockey.Mock((*k8sclient.Client).Upsert).Return(nil, nil).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(err).To(MatchError(resolveErr))
				Expect(upsertMock.Times()).To(Equal(0))
			})
		})

		It("should return ErrComponentNotInstalled when discovery reports kind not found", func() {
			mockey.PatchConvey("apply-component-not-installed-kind-missing", GinkgoT(), func() {
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("web", nil).Build()
				mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build()
				// group 已注册但缺 GeneralPodAutoscaler 这个 kind，discovery 归一为 ErrKindNotFound
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					nil, errors.Wrapf(discovery.ErrKindNotFound, "kind %s", gpaKind),
				).Build()
				upsertMock := mockey.Mock((*k8sclient.Client).Upsert).Return(nil, nil).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(err).To(MatchError(ErrComponentNotInstalled))
				// discovery 阶段已失败，不应进入 patch 阶段
				Expect(upsertMock.Times()).To(Equal(0))
			})
		})

		It("should return ErrComponentNotInstalled when discovery reports group not registered", func() {
			mockey.PatchConvey("apply-component-not-installed-group-missing", GinkgoT(), func() {
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("web", nil).Build()
				mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build()
				// 整个 group/version 未注册，discovery 同样归一为 ErrKindNotFound
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					nil, errors.Wrapf(discovery.ErrKindNotFound, "group %s", gpaGroupVersion),
				).Build()
				upsertMock := mockey.Mock((*k8sclient.Client).Upsert).Return(nil, nil).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(err).To(MatchError(ErrComponentNotInstalled))
				Expect(upsertMock.Times()).To(Equal(0))
			})
		})

		It("should not report component-not-installed for unrelated discovery errors", func() {
			mockey.PatchConvey("apply-discovery-unrelated-error", GinkgoT(), func() {
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("web", nil).Build()
				mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build()
				// 非 CRD 缺失类错误（如集群连接失败），不应误判为组件未安装
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					nil, errors.New("get group version resource: dial tcp: connection refused"),
				).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(err).To(HaveOccurred())
				Expect(err).NotTo(MatchError(ErrComponentNotInstalled))
			})
		})

		It("should keep the original error when upsert hits a k8s not-found (AC-001)", func() {
			mockey.PatchConvey("apply-upsert-notfound-not-component", GinkgoT(), func() {
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("web", nil).Build()
				mockK8sClientChain()
				mockNoLiveCR()
				k8sNotFound := k8serrors.NewNotFound(
					schema.GroupResource{Resource: "namespaces"}, "ns-1",
				)
				mockey.Mock((*k8sclient.Client).Upsert).Return(
					nil, errors.Wrap(k8sNotFound, "patch generalpodautoscalers"),
				).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(err).To(HaveOccurred())
				Expect(err).NotTo(MatchError(ErrComponentNotInstalled))
			})
		})

		It("should keep the original error when upsert fails manifest validation", func() {
			mockey.PatchConvey("apply-upsert-manifest-invalid", GinkgoT(), func() {
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("web", nil).Build()
				mockK8sClientChain()
				mockNoLiveCR()
				// manifest 校验错误文本含 "not found"（metadata.name not found），不应误判为组件未安装
				mockey.Mock((*k8sclient.Client).Upsert).Return(
					nil, errors.New("validate manifest: metadata.name not found"),
				).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(err).To(HaveOccurred())
				Expect(err).NotTo(MatchError(ErrComponentNotInstalled))
			})
		})

		It("prefixes the CR name with the resource prefix in an env sharing namespace", func() {
			mockey.PatchConvey("apply-shared-namespace", GinkgoT(), func() {
				env.NsSharing.ResPrefix = "fab12"
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("fab12-web", nil).Build()
				mockK8sClientChain()

				var gotName string
				mockey.Mock((*k8sclient.Client).Get).To(func(
					_ *k8sclient.Client, _ context.Context, _, name string, _ metav1.GetOptions,
				) (*unstructured.Unstructured, error) {
					gotName = name
					return nil, k8sclient.ErrResourceNotFound
				}).Build()
				var capturedManifest map[string]any
				mockey.Mock((*k8sclient.Client).Upsert).To(func(
					_ *k8sclient.Client, _ context.Context, _ string,
					manifest map[string]any, _ metav1.PatchOptions,
				) (*unstructured.Unstructured, error) {
					capturedManifest = manifest
					return &unstructured.Unstructured{Object: manifest}, nil
				}).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				Expect(svc.Apply(ctx, env, config)).To(Succeed())

				obj := unstructured.Unstructured{Object: capturedManifest}
				Expect(gotName).To(Equal("fab12-web-autoscale"))
				Expect(obj.GetName()).To(Equal("fab12-web-autoscale"))
				Expect(obj.GetLabels()).To(HaveKeyWithValue(LabelKeyEnvName, "dev"))
				Expect(obj.GetLabels()).To(HaveKeyWithValue(LabelKeyAppID, "app-1"))
				// 调用方传入的配置保持原名
				Expect(config.Name).To(Equal("web-autoscale"))
			})
		})

		It("rejects overwriting a CR owned by another env in an env sharing namespace", func() {
			mockey.PatchConvey("apply-not-owned", GinkgoT(), func() {
				env.NsSharing.ResPrefix = "fab12"
				mockey.Mock((*GPAService).resolveScaleTargetName).Return("fab12-web", nil).Build()
				mockK8sClientChain()
				live := &unstructured.Unstructured{}
				live.SetName("fab12-web-autoscale")
				live.SetLabels(map[string]string{LabelKeyEnvName: "prod", LabelKeyAppID: "app-1"})
				mockey.Mock((*k8sclient.Client).Get).Return(live, nil).Build()
				upsertMock := mockey.Mock((*k8sclient.Client).Upsert).Return(nil, nil).Build()

				config := &GPAConfig{
					Name: "web-autoscale", AppID: "app-1", MinReplicas: 2, MaxReplicas: 10,
					Metrics: []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
				}
				err := svc.Apply(ctx, env, config)
				Expect(errors.Is(err, ownership.ErrNotOwned)).To(BeTrue())
				Expect(upsertMock.Times()).To(Equal(0))
			})
		})
	})

	Describe("Get", func() {
		It("should return parsed status from the cluster CR", func() {
			mockey.PatchConvey("get-success", GinkgoT(), func() {
				mockK8sClientChain()
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"name": "web-autoscale",
						"labels": map[string]any{
							LabelKeyAppID:       "app-1",
							LabelKeyWorkspaceID: "ws-1",
							LabelKeyEnvName:     "dev",
						},
					},
					"status": map[string]any{
						"currentReplicas": int64(3),
						"desiredReplicas": int64(5),
					},
				}}
				mockey.Mock((*k8sclient.Client).Get).Return(obj, nil).Build()

				status, err := svc.Get(ctx, env, "web-autoscale")
				Expect(err).NotTo(HaveOccurred())
				Expect(status.Name).To(Equal("web-autoscale"))
				Expect(status.CurrentReplicas).To(Equal(int32(3)))
				Expect(status.DesiredReplicas).To(Equal(int32(5)))
			})
		})

		It("should translate k8s not-found into ErrCRNotFound", func() {
			mockey.PatchConvey("get-not-found", GinkgoT(), func() {
				mockK8sClientChain()
				mockey.Mock((*k8sclient.Client).Get).Return(nil, k8sclient.ErrResourceNotFound).Build()

				_, err := svc.Get(ctx, env, "missing")
				Expect(err).To(MatchError(ErrCRNotFound))
			})
		})
	})

	Describe("ListByEnv", func() {
		It("should list and parse all gpa CRs filtered by env labels", func() {
			mockey.PatchConvey("list-success", GinkgoT(), func() {
				mockK8sClientChain()
				list := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
					{Object: map[string]any{
						"metadata": map[string]any{"name": "gpa-a", "labels": map[string]any{LabelKeyAppID: "app-1"}},
						"status":   map[string]any{"currentReplicas": int64(2)},
					}},
					{Object: map[string]any{
						"metadata": map[string]any{"name": "gpa-b"},
					}},
				}}

				var capturedSelector string
				mockey.Mock((*k8sclient.Client).List).To(func(
					_ *k8sclient.Client, _ context.Context, _ string, opts metav1.ListOptions,
				) (*unstructured.UnstructuredList, error) {
					capturedSelector = opts.LabelSelector
					return list, nil
				}).Build()

				statuses, err := svc.ListByEnv(ctx, env)
				Expect(err).NotTo(HaveOccurred())
				Expect(statuses).To(HaveLen(2))
				Expect(statuses[0].Name).To(Equal("gpa-a"))
				Expect(statuses[0].CurrentReplicas).To(Equal(int32(2)))
				// label selector 应同时按 workspaceID 与 envName 过滤
				Expect(capturedSelector).To(ContainSubstring(LabelKeyWorkspaceID + "=ws-1"))
				Expect(capturedSelector).To(ContainSubstring(LabelKeyEnvName + "=dev"))
			})
		})
	})

	Describe("Delete", func() {
		It("should delete the gpa CR", func() {
			mockey.PatchConvey("delete-success", GinkgoT(), func() {
				mockK8sClientChain()
				mockey.Mock((*k8sclient.Client).Delete).Return(nil).Build()

				Expect(svc.Delete(ctx, env, "web-autoscale")).To(Succeed())
			})
		})

		It("should translate k8s not-found into ErrCRNotFound", func() {
			mockey.PatchConvey("delete-not-found", GinkgoT(), func() {
				mockK8sClientChain()
				mockey.Mock((*k8sclient.Client).Delete).Return(k8sclient.ErrResourceNotFound).Build()

				Expect(svc.Delete(ctx, env, "missing")).To(MatchError(ErrCRNotFound))
			})
		})
	})
})
