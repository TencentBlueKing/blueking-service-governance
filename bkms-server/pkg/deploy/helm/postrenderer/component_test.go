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

package postrenderer

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	helmcomp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/helm"
)

var _ = Describe("ComponentPostRenderer", func() {
	Describe("NewComponentPostRenderer", func() {
		It("should return nil when items is empty", func() {
			r := NewComponentPostRenderer(nil)
			Expect(r).To(BeNil())
		})

		It("should return non-nil when items provided", func() {
			items := []ComponentPatch{{Name: "test"}}
			r := NewComponentPostRenderer(items)
			Expect(r).NotTo(BeNil())
		})
	})

	Describe("Run", func() {
		Context("nil or empty receiver", func() {
			It("should return input as-is for nil receiver", func() {
				var r *ComponentPostRenderer
				input := bytes.NewBufferString("apiVersion: v1\nkind: Service\n")
				output, err := r.Run(input)
				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(Equal(input))
			})
		})

		Context("sidecar injection (US1)", func() {
			It("should patch Deployment spec.template.spec with sidecar container", func() {
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: main
          image: nginx:latest
`
				items := []ComponentPatch{
					{
						Name: "sidecar-envoy",
						Target: helmcomp.TargetResourceSelector{
							Kind: "Deployment",
							Name: "my-app",
						},
						Patchers: []map[string]any{
							{
								"spec": map[string]any{"template": map[string]any{"spec": map[string]any{
									"containers": []any{
										map[string]any{"name": "main", "image": "nginx:latest"},
										map[string]any{"name": "envoy-sidecar", "image": "envoy:v1.28"},
									},
								}}},
							},
						},
					},
				}

				r := NewComponentPostRenderer(items)
				output, err := r.Run(bytes.NewBufferString(manifest))
				Expect(err).NotTo(HaveOccurred())

				// 解析输出验证 sidecar 已注入
				var doc map[string]any
				Expect(yaml.Unmarshal(output.Bytes(), &doc)).To(Succeed())

				spec := doc["spec"].(map[string]any)
				template := spec["template"].(map[string]any)
				podSpec := template["spec"].(map[string]any)
				containers := podSpec["containers"].([]any)
				Expect(containers).To(HaveLen(2))

				sidecar := containers[1].(map[string]any)
				Expect(sidecar["name"]).To(Equal("envoy-sidecar"))
				Expect(sidecar["image"]).To(Equal("envoy:v1.28"))
			})
		})

		Context("multi-component priority ordering (US1)", func() {
			It("should apply components in order and later patches override earlier ones", func() {
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
`
				items := []ComponentPatch{
					{
						Name:   "comp-first",
						Target: helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "web"},
						Patchers: []map[string]any{
							{
								"metadata": map[string]any{
									"annotations": map[string]any{
										"first": "true",
									},
								},
							},
						},
					},
					{
						Name:   "comp-second",
						Target: helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "web"},
						Patchers: []map[string]any{
							{
								"metadata": map[string]any{
									"annotations": map[string]any{
										"second": "true",
									},
								},
							},
						},
					},
				}

				r := NewComponentPostRenderer(items)
				output, err := r.Run(bytes.NewBufferString(manifest))
				Expect(err).NotTo(HaveOccurred())

				var doc map[string]any
				Expect(yaml.Unmarshal(output.Bytes(), &doc)).To(Succeed())

				metadata := doc["metadata"].(map[string]any)
				annotations := metadata["annotations"].(map[string]any)
				// 两个组件的 annotations 都应该存在（JSON Merge Patch 合并）
				Expect(annotations["first"]).To(Equal("true"))
				Expect(annotations["second"]).To(Equal("true"))
			})
		})

		Context("multiple patchers in one component", func() {
			It("should apply patchers in array order", func() {
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
`
				items := []ComponentPatch{{
					Name:   "ordered-patchers",
					Target: helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "web"},
					Patchers: []map[string]any{
						{"metadata": map[string]any{"annotations": map[string]any{"order": "first"}}},
						{"metadata": map[string]any{"annotations": map[string]any{"order": "second"}}},
					},
				}}

				renderer := NewComponentPostRenderer(items)
				output, err := renderer.Run(bytes.NewBufferString(manifest))
				Expect(err).NotTo(HaveOccurred())

				var document map[string]any
				Expect(yaml.Unmarshal(output.Bytes(), &document)).To(Succeed())
				metadata := document["metadata"].(map[string]any)
				annotations := metadata["annotations"].(map[string]any)
				Expect(annotations["order"]).To(Equal("second"))
			})
		})

		Context("spec append (US2)", func() {
			It("should append extra resources from spec to manifest", func() {
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
`
				items := []ComponentPatch{
					{
						Name:   "config-injector",
						Target: helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "web"},
						Specs: []map[string]any{
							{
								"apiVersion": "v1",
								"kind":       "ConfigMap",
								"metadata": map[string]any{
									"name": "injected-config",
								},
								"data": map[string]any{
									"key": "value",
								},
							},
						},
					},
				}

				r := NewComponentPostRenderer(items)
				output, err := r.Run(bytes.NewBufferString(manifest))
				Expect(err).NotTo(HaveOccurred())

				// 解析 multi-doc YAML
				docs, err := parseMultiDocYAML(output)
				Expect(err).NotTo(HaveOccurred())
				Expect(docs).To(HaveLen(2))

				// 第二个文档应该是追加的 ConfigMap
				Expect(docs[1]["kind"]).To(Equal("ConfigMap"))
				metadata := docs[1]["metadata"].(map[string]any)
				Expect(metadata["name"]).To(Equal("injected-config"))
			})
		})

		Context("target not found (US3)", func() {
			It("should not append extra resources when target resource does not exist", func() {
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: existing-app
spec:
  replicas: 1
`
				items := []ComponentPatch{
					{
						Name:   "ghost-patcher",
						Target: helmcomp.TargetResourceSelector{Kind: "StatefulSet", Name: "non-existent"},
						Patchers: []map[string]any{
							{
								"metadata": map[string]any{"annotations": map[string]any{"x": "y"}},
							},
						},
						Specs: []map[string]any{
							{
								"apiVersion": "v1",
								"kind":       "ConfigMap",
								"metadata":   map[string]any{"name": "orphaned-config"},
							},
						},
					},
				}

				r := NewComponentPostRenderer(items)
				output, err := r.Run(bytes.NewBufferString(manifest))
				Expect(err).NotTo(HaveOccurred())

				docs, err := parseMultiDocYAML(output)
				Expect(err).NotTo(HaveOccurred())
				Expect(docs).To(HaveLen(1))
				Expect(docs[0]["kind"]).To(Equal("Deployment"))
				metadata := docs[0]["metadata"].(map[string]any)
				Expect(metadata["name"]).To(Equal("existing-app"))
				Expect(metadata).NotTo(HaveKey("annotations"))
			})
		})

		Context("component without explicit target", func() {
			It("should not patch workload resources or append extra resources", func() {
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  template:
    spec:
      containers:
        - name: main
          image: app:v1
---
apiVersion: v1
kind: Service
metadata:
  name: web
`
				items := []ComponentPatch{
					{
						Name: "polaris",
						Patchers: []map[string]any{
							{
								"spec": map[string]any{"template": map[string]any{"spec": map[string]any{
									"containers": []any{
										map[string]any{
											"name":  "main",
											"image": "app:v1",
											"env": []any{
												map[string]any{"name": "POLARIS_TOKEN", "value": "token"},
											},
										},
									},
								}}},
							},
						},
						Specs: []map[string]any{
							{
								"apiVersion": "v1",
								"kind":       "ConfigMap",
								"metadata":   map[string]any{"name": "extra"},
							},
						},
					},
				}

				r := NewComponentPostRenderer(items)
				output, err := r.Run(bytes.NewBufferString(manifest))
				Expect(err).NotTo(HaveOccurred())

				docs, err := parseMultiDocYAML(output)
				Expect(err).NotTo(HaveOccurred())
				Expect(docs).To(HaveLen(2))

				spec := docs[0]["spec"].(map[string]any)
				template := spec["template"].(map[string]any)
				podSpec := template["spec"].(map[string]any)
				containers := podSpec["containers"].([]any)
				container := containers[0].(map[string]any)
				Expect(container).NotTo(HaveKey("env"))
				Expect(docs[1]["kind"]).To(Equal("Service"))
			})
		})

		Context("patch execution error (US3)", func() {
			It("should return error when YAML input is invalid", func() {
				invalidManifest := `not: [valid: yaml`

				items := []ComponentPatch{
					{
						Name:     "any-comp",
						Target:   helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "x"},
						Patchers: []map[string]any{{"metadata": map[string]any{"x": "y"}}},
					},
				}

				r := NewComponentPostRenderer(items)
				_, err := r.Run(bytes.NewBufferString(invalidManifest))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("parse multi-doc YAML"))
			})
		})

		Context("apiVersion matching", func() {
			It("should match with apiVersion when specified in selector", func() {
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
---
apiVersion: kruise.io/v1alpha1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 2
`
				items := []ComponentPatch{
					{
						Name: "kruise-patcher",
						Target: helmcomp.TargetResourceSelector{
							APIVersion: "kruise.io/v1alpha1",
							Kind:       "Deployment",
							Name:       "web",
						},
						Patchers: []map[string]any{
							{
								"metadata": map[string]any{
									"annotations": map[string]any{"patched": "true"},
								},
							},
						},
					},
				}

				r := NewComponentPostRenderer(items)
				output, err := r.Run(bytes.NewBufferString(manifest))
				Expect(err).NotTo(HaveOccurred())

				docs, err := parseMultiDocYAML(output)
				Expect(err).NotTo(HaveOccurred())
				Expect(docs).To(HaveLen(2))

				// 第一个 Deployment (apps/v1) 不应被 patch
				meta1 := docs[0]["metadata"].(map[string]any)
				Expect(meta1).NotTo(HaveKey("annotations"))

				// 第二个 Deployment (kruise.io/v1alpha1) 应被 patch
				meta2 := docs[1]["metadata"].(map[string]any)
				annotations := meta2["annotations"].(map[string]any)
				Expect(annotations["patched"]).To(Equal("true"))
			})
		})

		Context("preview error reporting (US4)", func() {
			It("should include component name in error message", func() {
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: target
spec:
  replicas: 1
`
				// 使用 channel 类型的值触发 JSON 序列化失败。
				items := []ComponentPatch{
					{
						Name:   "bad-component",
						Target: helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "target"},
						Patchers: []map[string]any{
							{
								"spec": make(chan int), // 无法序列化为 JSON
							},
						},
					},
				}

				r := NewComponentPostRenderer(items)
				_, err := r.Run(bytes.NewBufferString(manifest))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("bad-component"))
				Expect(err.Error()).To(ContainSubstring("patcher[0] failed"))
			})
		})

		Context("multi-doc YAML with mixed resources", func() {
			It("should only patch matching resources and leave others unchanged", func() {
				manifest := `apiVersion: v1
kind: Service
metadata:
  name: my-svc
spec:
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value
`
				items := []ComponentPatch{
					{
						Name:   "replicas-patcher",
						Target: helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "my-app"},
						Patchers: []map[string]any{
							{
								"spec": map[string]any{"replicas": 3},
							},
						},
					},
				}

				r := NewComponentPostRenderer(items)
				output, err := r.Run(bytes.NewBufferString(manifest))
				Expect(err).NotTo(HaveOccurred())

				docs, err := parseMultiDocYAML(output)
				Expect(err).NotTo(HaveOccurred())
				Expect(docs).To(HaveLen(3))

				// Service 不变
				Expect(docs[0]["kind"]).To(Equal("Service"))
				// Deployment 被 patch
				spec := docs[1]["spec"].(map[string]any)
				Expect(spec["replicas"]).To(BeEquivalentTo(3))
				// ConfigMap 不变
				Expect(docs[2]["kind"]).To(Equal("ConfigMap"))
			})
		})
	})
})
