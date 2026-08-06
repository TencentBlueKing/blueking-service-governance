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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

var _ = Describe("ScopeValidator", func() {
	var (
		ctx   context.Context
		scope *ResourceSnapshot

		validator *ScopeValidator
		cfg       *cluster.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = &cluster.Config{}
		scope = &ResourceSnapshot{
			ClusterID: "test-cluster",
			Resources: []ResourceEntry{
				{Kind: k8skind.Deploy, Namespace: "default", Name: "my-deploy"},
				{Kind: k8skind.SVC, Namespace: "default", Name: "my-svc"},
				{Kind: k8skind.CM, Namespace: "default", Name: "my-cm"},
			},
		}
	})

	Describe("static resource matching", func() {
		It("should pass for a static resource that exists in the scope", func() {
			mockey.PatchConvey("static-match", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(cfg).Build()
				validator = NewScopeValidator(scope, cfg)

				err := validator.Validate(ctx, k8skind.Deploy, "default", "my-deploy")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should fail for a static resource not in scope", func() {
			mockey.PatchConvey("static-reject", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(cfg).Build()
				validator = NewScopeValidator(scope, cfg)

				err := validator.Validate(ctx, k8skind.Deploy, "default", "other-deploy")
				Expect(err).To(MatchError(ErrNodeNotInSnapshot))
			})
		})

		It("should fail for non-dynamic kind not in scope", func() {
			mockey.PatchConvey("non-dynamic-reject", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(cfg).Build()
				validator = NewScopeValidator(scope, cfg)

				err := validator.Validate(ctx, k8skind.Ing, "default", "my-ingress")
				Expect(err).To(MatchError(ErrNodeNotInSnapshot))
			})
		})
	})

	Describe("dynamic resource - 1-level ownerRef tracing (ReplicaSet -> Deployment)", func() {
		It("should pass when RS owner is a Deployment in scope", func() {
			mockey.PatchConvey("rs-to-deploy", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(cfg).Build()

				rsObj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "apps/v1",
					"kind":       k8skind.RS,
					"metadata": map[string]any{
						"name":      "my-deploy-abc123",
						"namespace": "default",
						"ownerReferences": []any{
							map[string]any{
								"apiVersion": "apps/v1",
								"kind":       k8skind.Deploy,
								"name":       "my-deploy",
								"controller": true,
							},
						},
					},
				}}

				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).Return(rsObj, nil).Build()

				validator = NewScopeValidator(scope, cfg)
				err := validator.Validate(ctx, k8skind.RS, "default", "my-deploy-abc123")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("dynamic resource - 2-level ownerRef tracing (Pod -> RS -> Deployment)", func() {
		It("should pass when Pod traces to a Deployment in scope via RS", func() {
			mockey.PatchConvey("pod-to-rs-to-deploy", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(cfg).Build()

				podObj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "v1",
					"kind":       k8skind.Po,
					"metadata": map[string]any{
						"name":      "my-deploy-abc123-xyz",
						"namespace": "default",
						"ownerReferences": []any{
							map[string]any{
								"apiVersion": "apps/v1",
								"kind":       k8skind.RS,
								"name":       "my-deploy-abc123",
								"controller": true,
							},
						},
					},
				}}

				rsObj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "apps/v1",
					"kind":       k8skind.RS,
					"metadata": map[string]any{
						"name":      "my-deploy-abc123",
						"namespace": "default",
						"ownerReferences": []any{
							map[string]any{
								"apiVersion": "apps/v1",
								"kind":       k8skind.Deploy,
								"name":       "my-deploy",
								"controller": true,
							},
						},
					},
				}}

				callCount := 0
				mockey.Mock(discovery.GetGroupVersionResource).To(func(
					_ *cluster.Config, kind, _ string,
				) (*schema.GroupVersionResource, error) {
					switch kind {
					case k8skind.Po:
						return &schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, nil
					case k8skind.RS:
						return &schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}, nil
					default:
						return nil, nil
					}
				}).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).To(func(
					_ *k8sclient.Client, _ context.Context, _, name string, _ metav1.GetOptions,
				) (*unstructured.Unstructured, error) {
					callCount++
					if callCount == 1 {
						return podObj, nil
					}
					return rsObj, nil
				}).Build()

				validator = NewScopeValidator(scope, cfg)
				err := validator.Validate(ctx, k8skind.Po, "default", "my-deploy-abc123-xyz")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("broken ownerRef chain", func() {
		It("should reject when ownerRef chain is broken (owner not in scope)", func() {
			mockey.PatchConvey("broken-chain", GinkgoT(), func() {
				mockey.Mock(cluster.NewConfig).Return(cfg).Build()

				podObj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "v1",
					"kind":       k8skind.Po,
					"metadata": map[string]any{
						"name":      "orphan-pod",
						"namespace": "default",
						"ownerReferences": []any{
							map[string]any{
								"apiVersion": "apps/v1",
								"kind":       k8skind.RS,
								"name":       "unknown-rs",
								"controller": true,
							},
						},
					},
				}}

				rsObj := &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "apps/v1",
					"kind":       k8skind.RS,
					"metadata": map[string]any{
						"name":      "unknown-rs",
						"namespace": "default",
						"ownerReferences": []any{
							map[string]any{
								"apiVersion": "apps/v1",
								"kind":       k8skind.Deploy,
								"name":       "not-in-scope-deploy",
								"controller": true,
							},
						},
					},
				}}

				callCount := 0
				mockey.Mock(discovery.GetGroupVersionResource).Return(
					&schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, nil,
				).Build()
				mockey.Mock(k8sclient.NewWithGVR).Return(&k8sclient.Client{}).Build()
				mockey.Mock((*k8sclient.Client).Get).To(func(
					_ *k8sclient.Client, _ context.Context, _, _ string, _ metav1.GetOptions,
				) (*unstructured.Unstructured, error) {
					callCount++
					if callCount == 1 {
						return podObj, nil
					}
					return rsObj, nil
				}).Build()

				validator = NewScopeValidator(scope, cfg)
				err := validator.Validate(ctx, k8skind.Po, "default", "orphan-pod")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
