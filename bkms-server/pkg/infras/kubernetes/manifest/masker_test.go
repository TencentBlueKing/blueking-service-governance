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

package manifest

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const testMaskValue = "******"

var _ = Describe("Masker", func() {
	Describe("Mask", func() {
		It("should mask sensitive env values in workload pod template by env name", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
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
										map[string]any{"name": "SECRET_TOKEN", "value": "1"},
										map[string]any{"name": "PUBLIC_VALUE", "value": "1"},
										map[string]any{"name": "OTHER_SECRET", "value": "changed"},
									},
								},
							},
							"initContainers": []any{
								map[string]any{
									"name": "init",
									"env": []any{
										map[string]any{"name": "INIT_SECRET", "value": "init-secret"},
									},
								},
							},
						},
					},
				},
			}}
			masker := NewMasker(map[string]string{
				"SECRET_TOKEN": "1",
				"INIT_SECRET":  "init-secret",
				"OTHER_SECRET": "original",
			}, testMaskValue)

			masker.Mask(obj)

			Expect(obj.Object["spec"].(map[string]any)["replicas"]).To(Equal(int64(1)))
			Expect(nestedEnvValue(obj.Object, "containers", 0, 0)).To(Equal(testMaskValue))
			Expect(nestedEnvValue(obj.Object, "containers", 0, 1)).To(Equal("1"))
			Expect(nestedEnvValue(obj.Object, "containers", 0, 2)).To(Equal(testMaskValue))
			Expect(nestedEnvValue(obj.Object, "initContainers", 0, 0)).To(Equal(testMaskValue))
		})

		It("should mask sensitive env values in pod specs", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]any{
					"name":      "my-pod",
					"namespace": "default",
				},
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name": "main",
							"env": []any{
								map[string]any{"name": "SECRET_TOKEN", "value": "secret"},
							},
						},
					},
				},
			}}
			masker := NewMasker(map[string]string{"SECRET_TOKEN": "secret"}, testMaskValue)

			masker.Mask(obj)

			containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			container := containers[0].(map[string]any)
			envList, found, err := unstructured.NestedSlice(container, "env")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(envList[0].(map[string]any)["value"]).To(Equal(testMaskValue))
		})

		It("should mask sensitive env values in cronjob pod templates", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "batch/v1",
				"kind":       "CronJob",
				"metadata": map[string]any{
					"name":      "my-cronjob",
					"namespace": "default",
				},
				"spec": map[string]any{
					"schedule": "*/5 * * * *",
					"jobTemplate": map[string]any{
						"spec": map[string]any{
							"template": map[string]any{
								"spec": map[string]any{
									"restartPolicy": "OnFailure",
									"containers": []any{
										map[string]any{
											"name": "main",
											"env": []any{
												map[string]any{"name": "SECRET_TOKEN", "value": "secret"},
												map[string]any{"name": "PUBLIC_VALUE", "value": "secret"},
											},
										},
									},
								},
							},
						},
					},
				},
			}}
			masker := NewMasker(map[string]string{"SECRET_TOKEN": "secret"}, testMaskValue)

			masker.Mask(obj)

			Expect(nestedEnvValueAt(
				obj.Object, []string{"spec", "jobTemplate", "spec", "template", "spec"}, "containers", 0, 0),
			).To(Equal(testMaskValue))
			Expect(nestedEnvValueAt(
				obj.Object, []string{"spec", "jobTemplate", "spec", "template", "spec"}, "containers", 0, 1),
			).To(Equal("secret"))
		})

		It("should mask sensitive env values in GameDeployment pod templates", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "tkex.tencent.com/v1alpha1",
				"kind":       "GameDeployment",
				"metadata": map[string]any{
					"name":      "my-game-deploy",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "main",
									"env": []any{
										map[string]any{"name": "SECRET_TOKEN", "value": "secret"},
										map[string]any{"name": "PUBLIC_VALUE", "value": "secret"},
									},
								},
							},
						},
					},
				},
			}}
			masker := NewMasker(map[string]string{"SECRET_TOKEN": "secret"}, testMaskValue)

			masker.Mask(obj)

			Expect(nestedEnvValueAt(
				obj.Object, []string{"spec", "template", "spec"}, "containers", 0, 0),
			).To(Equal(testMaskValue))
			Expect(nestedEnvValueAt(
				obj.Object, []string{"spec", "template", "spec"}, "containers", 0, 1),
			).To(Equal("secret"))
		})

		It("should mask sensitive env values in ReplicaSet pod templates", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "ReplicaSet",
				"metadata": map[string]any{
					"name":      "my-rs",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "main",
									"env": []any{
										map[string]any{"name": "SECRET_TOKEN", "value": "secret"},
										map[string]any{"name": "PUBLIC_VALUE", "value": "secret"},
									},
								},
							},
						},
					},
				},
			}}
			masker := NewMasker(map[string]string{"SECRET_TOKEN": ""}, testMaskValue)

			masker.Mask(obj)

			Expect(nestedEnvValueAt(
				obj.Object, []string{"spec", "template", "spec"}, "containers", 0, 0),
			).To(Equal(testMaskValue))
			Expect(nestedEnvValueAt(
				obj.Object, []string{"spec", "template", "spec"}, "containers", 0, 1),
			).To(Equal("secret"))
		})

		It("should not mask valueFrom env vars", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "main",
									"env": []any{
										map[string]any{
											"name": "SECRET_FROM_REF",
											"valueFrom": map[string]any{
												"secretKeyRef": map[string]any{
													"name": "secret",
													"key":  "token",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}}
			masker := NewMasker(map[string]string{"SECRET_FROM_REF": "secret"}, testMaskValue)

			masker.Mask(obj)

			envVar := nestedEnvVar(obj.Object, "containers", 0, 0)
			Expect(envVar).NotTo(HaveKey("value"))
			Expect(envVar).To(HaveKey("valueFrom"))
		})

		It("should keep env values when no env names match", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "main",
									"env": []any{
										map[string]any{"name": "PUBLIC_VALUE", "value": "1"},
									},
								},
							},
						},
					},
				},
			}}
			masker := NewMasker(map[string]string{"SECRET_TOKEN": "1"}, testMaskValue)

			masker.Mask(obj)

			Expect(nestedEnvValue(obj.Object, "containers", 0, 0)).To(Equal("1"))
		})

		It("should keep env values when no sensitive env values are configured", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "main",
									"env": []any{
										map[string]any{"name": "SECRET_TOKEN", "value": "secret"},
									},
								},
							},
						},
					},
				},
			}}
			masker := NewMasker(nil, testMaskValue)

			masker.Mask(obj)

			Expect(nestedEnvValue(obj.Object, "containers", 0, 0)).To(Equal("secret"))
		})

		It("should mask ConfigMap data and binaryData by sensitive env names", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
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
			}}
			masker := NewMasker(map[string]string{"SECRET_TOKEN": "plain-secret"}, testMaskValue)

			masker.Mask(obj)

			data, found, err := unstructured.NestedMap(obj.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(data["SECRET_TOKEN"]).To(Equal(testMaskValue))
			Expect(data["PUBLIC_VALUE"]).To(Equal("public"))

			binaryData, found, err := unstructured.NestedMap(obj.Object, "binaryData")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(binaryData["SECRET_TOKEN"]).To(Equal(testMaskValue))
			Expect(binaryData["PUBLIC_VALUE"]).To(Equal("cHVibGlj"))
		})

		It("should mask all Secret data and stringData without sensitive env names", func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]any{
					"name":      "my-secret",
					"namespace": "default",
				},
				"data": map[string]any{
					"password": "c2VjcmV0",
					"token":    "dG9rZW4=",
				},
				"stringData": map[string]any{
					"plain": "secret",
				},
			}}
			masker := NewMasker(nil, testMaskValue)

			masker.Mask(obj)

			data, found, err := unstructured.NestedMap(obj.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(data).To(Equal(map[string]any{
				"password": testMaskValue,
				"token":    testMaskValue,
			}))

			stringData, found, err := unstructured.NestedMap(obj.Object, "stringData")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(stringData).To(Equal(map[string]any{"plain": testMaskValue}))
		})
	})
})

func nestedEnvValue(obj map[string]any, field string, containerIndex, envIndex int) any {
	return nestedEnvVar(obj, field, containerIndex, envIndex)["value"]
}

func nestedEnvValueAt(obj map[string]any, podSpecPath []string, field string, containerIndex, envIndex int) any {
	return nestedEnvVarAt(obj, podSpecPath, field, containerIndex, envIndex)["value"]
}

func nestedEnvVar(obj map[string]any, field string, containerIndex, envIndex int) map[string]any {
	return nestedEnvVarAt(obj, []string{"spec", "template", "spec"}, field, containerIndex, envIndex)
}

func nestedEnvVarAt(
	obj map[string]any, podSpecPath []string, field string, containerIndex, envIndex int,
) map[string]any {
	podSpec, found, err := unstructured.NestedMap(obj, podSpecPath...)
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())

	containers, found, err := unstructured.NestedSlice(podSpec, field)
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())

	container := containers[containerIndex].(map[string]any)
	envList, found, err := unstructured.NestedSlice(container, "env")
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())
	return envList[envIndex].(map[string]any)
}
