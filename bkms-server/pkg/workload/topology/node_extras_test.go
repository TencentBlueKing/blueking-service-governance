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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("BuilderExtras", func() {
	Describe("extractPodExtras", func() {
		It("should return empty map when Pod has no containers", func() {
			pod := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{},
			}}
			extras := extractPodExtras(pod)
			Expect(extras).To(BeEmpty())
		})

		DescribeTable("podIP extraction",
			func(status map[string]any, expectKey bool, expectedIP string) {
				pod := &unstructured.Unstructured{Object: map[string]any{
					"spec": map[string]any{
						"containers": []any{
							map[string]any{"name": "app", "image": "app:v1"},
						},
					},
				}}
				if status != nil {
					pod.Object["status"] = status
				}
				extras := extractPodExtras(pod)
				if expectKey {
					Expect(extras).To(HaveKeyWithValue(ExtrasKeyPodIP, expectedIP))
				} else {
					Expect(extras).NotTo(HaveKey(ExtrasKeyPodIP))
				}
			},
			Entry("present", map[string]any{"podIP": "127.0.0.1"}, true, "127.0.0.1"),
			Entry("missing", nil, false, ""),
			Entry("empty string", map[string]any{"podIP": ""}, false, ""),
		)

		DescribeTable(
			"image extraction",
			func(containers []any, expectKey bool, expectedImage string) {
				pod := &unstructured.Unstructured{Object: map[string]any{
					"spec": map[string]any{
						"containers": containers,
					},
				}}
				extras := extractPodExtras(pod)
				if expectKey {
					Expect(extras).To(HaveKeyWithValue(ExtrasKeyImage, expectedImage))
				} else {
					Expect(extras).NotTo(HaveKey(ExtrasKeyImage))
				}
			},
			Entry(
				"single container",
				[]any{map[string]any{"name": "nginx", "image": "nginx:1.25"}},
				true,
				"nginx:1.25",
			),
			Entry("empty image string", []any{map[string]any{"name": "c1", "image": ""}}, false, ""),
			Entry("multiple containers - takes first", []any{
				map[string]any{"name": "main", "image": "app:v1"},
				map[string]any{"name": "sidecar", "image": "envoy:latest"},
			}, true, "app:v1"),
		)
	})

	Describe("extractWorkloadExtras", func() {
		It("should extract image, replicas and readyReplicas", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"replicas": int64(3),
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name":  "web",
									"image": "myapp:v2",
								},
							},
						},
					},
				},
				"status": map[string]any{
					"readyReplicas": int64(2),
				},
			}}
			extras := extractWorkloadExtras(deploy)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyImage, "myapp:v2"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReplicas, "3"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReadyReplicas, "2"))
		})

		It("should handle missing template gracefully", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"replicas": int64(1),
				},
			}}
			extras := extractWorkloadExtras(deploy)
			Expect(extras).NotTo(HaveKey(ExtrasKeyImage))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReplicas, "1"))
		})

		It("should handle zero replicas", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"replicas": int64(0),
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{"name": "app", "image": "img:v1"},
							},
						},
					},
				},
				"status": map[string]any{
					"readyReplicas": int64(0),
				},
			}}
			extras := extractWorkloadExtras(deploy)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReplicas, "0"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReadyReplicas, "0"))
		})
	})

	Describe("extractServiceExtras", func() {
		It("should extract ports, selector, clusterIP and type", func() {
			svc := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"ports": []any{
						map[string]any{
							"port":     int64(80),
							"protocol": "TCP",
						},
						map[string]any{
							"port":     int64(443),
							"protocol": "TCP",
						},
					},
					"selector": map[string]any{
						"app": "nginx",
					},
					"clusterIP": "127.0.0.1",
					"type":      "ClusterIP",
				},
			}}
			extras := extractServiceExtras(svc)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyPorts, "80/TCP,443/TCP"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeySelector, "app=nginx"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyClusterIP, "127.0.0.1"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyServiceType, "ClusterIP"))
		})

		It("should default protocol to TCP when not specified", func() {
			svc := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"ports": []any{
						map[string]any{"port": int64(8080)},
					},
				},
			}}
			extras := extractServiceExtras(svc)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyPorts, "8080/TCP"))
		})

		It("should handle Service with no ports", func() {
			svc := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"type": "ExternalName",
				},
			}}
			extras := extractServiceExtras(svc)
			Expect(extras).NotTo(HaveKey(ExtrasKeyPorts))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyServiceType, "ExternalName"))
		})

		It("should handle Service with empty selector", func() {
			svc := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"clusterIP": "None",
					"type":      "ClusterIP",
				},
			}}
			extras := extractServiceExtras(svc)
			Expect(extras).NotTo(HaveKey(ExtrasKeySelector))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyClusterIP, "None"))
		})
	})

	Describe("extractIngressExtras", func() {
		It("should extract multiple hosts", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{"host": "example.com"},
						map[string]any{"host": "api.example.com"},
					},
				},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyHost, "example.com,api.example.com"))
		})

		It("should skip rules without host", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{"host": "example.com"},
						map[string]any{}, // 无 host
					},
				},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyHost, "example.com"))
		})

		It("should return empty map when no rules exist", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(BeEmpty())
		})

		It("should extract rules with path routing info", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{
							"host": "example.com",
							"http": map[string]any{
								"paths": []any{
									map[string]any{
										"path": "/api",
										"backend": map[string]any{
											"service": map[string]any{
												"name": "api-svc",
											},
										},
									},
								},
							},
						},
					},
				},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(HaveKey(ExtrasKeyRules))
			Expect(extras[ExtrasKeyRules]).To(ContainSubstring("example.com/api->api-svc"))
		})

		It("should extract TLS configuration", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"tls": []any{
						map[string]any{
							"hosts":      []any{"example.com"},
							"secretName": "tls-secret",
						},
					},
				},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(HaveKey(ExtrasKeyTLS))
			Expect(extras[ExtrasKeyTLS]).To(ContainSubstring("example.com"))
			Expect(extras[ExtrasKeyTLS]).To(ContainSubstring("tls-secret"))
		})

		It("should extract rules from legacy backend.serviceName (extensions/v1beta1)", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{
							"host": "legacy.example.com",
							"http": map[string]any{
								"paths": []any{
									map[string]any{
										"path": "/app",
										"backend": map[string]any{
											"serviceName": "legacy-svc",
										},
									},
								},
							},
						},
					},
				},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyHost, "legacy.example.com"))
			Expect(extras).To(HaveKey(ExtrasKeyRules))
			Expect(extras[ExtrasKeyRules]).To(Equal("legacy.example.com/app->legacy-svc"))
		})

		It("should prefer backend.service.name over backend.serviceName when both exist", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{
							"host": "both.example.com",
							"http": map[string]any{
								"paths": []any{
									map[string]any{
										"path": "/v1",
										"backend": map[string]any{
											"service": map[string]any{
												"name": "new-svc",
											},
											"serviceName": "old-svc",
										},
									},
								},
							},
						},
					},
				},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(HaveKey(ExtrasKeyRules))
			Expect(extras[ExtrasKeyRules]).To(Equal("both.example.com/v1->new-svc"))
		})

		It("should extract multiple rules with mixed backend formats", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{
							"host": "app.example.com",
							"http": map[string]any{
								"paths": []any{
									map[string]any{
										"path": "/new",
										"backend": map[string]any{
											"service": map[string]any{
												"name": "new-backend",
											},
										},
									},
									map[string]any{
										"path": "/legacy",
										"backend": map[string]any{
											"serviceName": "legacy-backend",
										},
									},
								},
							},
						},
					},
				},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(HaveKey(ExtrasKeyRules))
			Expect(extras[ExtrasKeyRules]).To(ContainSubstring("app.example.com/new->new-backend"))
			Expect(extras[ExtrasKeyRules]).To(ContainSubstring("app.example.com/legacy->legacy-backend"))
		})

		It("should skip backend with neither service.name nor serviceName", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"rules": []any{
						map[string]any{
							"host": "empty.example.com",
							"http": map[string]any{
								"paths": []any{
									map[string]any{
										"path":    "/noop",
										"backend": map[string]any{},
									},
								},
							},
						},
					},
				},
			}}
			extras := extractIngressExtras(ingress)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyHost, "empty.example.com"))
			Expect(extras).NotTo(HaveKey(ExtrasKeyRules))
		})
	})

	Describe("extractDeploymentExtras", func() {
		It("should extract availableReplicas and strategy", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"replicas": int64(3),
					"strategy": map[string]any{
						"type": "RollingUpdate",
					},
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{"name": "app", "image": "app:v1"},
							},
						},
					},
				},
				"status": map[string]any{
					"readyReplicas":     int64(3),
					"availableReplicas": int64(3),
				},
			}}
			extras := extractDeploymentExtras(deploy)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyAvailableReplicas, "3"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyStrategy, "RollingUpdate"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReplicas, "3"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyImage, "app:v1"))
		})

		It("should handle missing strategy gracefully", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"replicas": int64(1),
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{"name": "app", "image": "app:v1"},
							},
						},
					},
				},
				"status": map[string]any{},
			}}
			extras := extractDeploymentExtras(deploy)
			Expect(extras).NotTo(HaveKey(ExtrasKeyStrategy))
			Expect(extras).NotTo(HaveKey(ExtrasKeyAvailableReplicas))
		})
	})

	Describe("extractReplicaSetExtras", func() {
		It("should extract ownerDeployment from ownerReferences", func() {
			rs := &unstructured.Unstructured{Object: map[string]any{
				"metadata": map[string]any{
					"name":      "deploy-abc123",
					"namespace": "default",
					"ownerReferences": []any{
						map[string]any{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"name":       "my-deploy",
						},
					},
				},
				"spec": map[string]any{
					"replicas": int64(2),
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{"name": "app", "image": "app:v2"},
							},
						},
					},
				},
				"status": map[string]any{
					"readyReplicas": int64(2),
				},
			}}
			extras := extractReplicaSetExtras(rs)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyOwnerDeployment, "my-deploy"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReplicas, "2"))
		})

		It("should handle ReplicaSet without ownerReferences", func() {
			rs := &unstructured.Unstructured{Object: map[string]any{
				"metadata": map[string]any{
					"name":      "standalone-rs",
					"namespace": "default",
				},
				"spec": map[string]any{
					"replicas": int64(1),
				},
			}}
			extras := extractReplicaSetExtras(rs)
			Expect(extras).NotTo(HaveKey(ExtrasKeyOwnerDeployment))
		})
	})

	Describe("extractConfigMapExtras", func() {
		It("should extract keys and data size", func() {
			cm := &unstructured.Unstructured{Object: map[string]any{
				"data": map[string]any{
					"config.yaml": "key: value",
					"env":         "FOO=bar",
				},
			}}
			extras := extractConfigMapExtras(cm)
			Expect(extras).To(HaveKey(ExtrasKeyKeys))
			Expect(extras).To(HaveKey(ExtrasKeyDataSize))
			// keys 应包含两个 key（顺序可能不同）
			keys := extras[ExtrasKeyKeys]
			Expect(keys).To(ContainSubstring("config.yaml"))
			Expect(keys).To(ContainSubstring("env"))
		})

		It("should handle ConfigMap with empty data", func() {
			cm := &unstructured.Unstructured{Object: map[string]any{}}
			extras := extractConfigMapExtras(cm)
			Expect(extras).NotTo(HaveKey(ExtrasKeyKeys))
			Expect(extras).NotTo(HaveKey(ExtrasKeyDataSize))
		})

		It("should extract binaryData size", func() {
			cm := &unstructured.Unstructured{Object: map[string]any{
				"data": map[string]any{
					"key1": "value1",
				},
				"binaryData": map[string]any{
					"binary-key": "YmluYXJ5",
				},
			}}
			extras := extractConfigMapExtras(cm)
			Expect(extras).To(HaveKey(ExtrasKeyBinaryDataSize))
			Expect(extras[ExtrasKeyKeys]).To(ContainSubstring("key1"))
			Expect(extras[ExtrasKeyKeys]).To(ContainSubstring("binary-key"))
		})

		It("should extract keys from binaryData only when data is absent", func() {
			cm := &unstructured.Unstructured{Object: map[string]any{
				"binaryData": map[string]any{
					"cert.pem": "Y2VydA==",
					"key.pem":  "a2V5",
				},
			}}
			extras := extractConfigMapExtras(cm)
			Expect(extras).To(HaveKey(ExtrasKeyBinaryDataSize))
			Expect(extras).NotTo(HaveKey(ExtrasKeyDataSize))
			Expect(extras).To(HaveKey(ExtrasKeyKeys))
			Expect(extras[ExtrasKeyKeys]).To(ContainSubstring("cert.pem"))
			Expect(extras[ExtrasKeyKeys]).To(ContainSubstring("key.pem"))
		})
	})

	Describe("extractSecretExtras", func() {
		It("should extract secretType and keys but not values", func() {
			secret := &unstructured.Unstructured{Object: map[string]any{
				"type": "Opaque",
				"data": map[string]any{
					"password": "c2VjcmV0",
					"token":    "dG9rZW4=",
				},
			}}
			extras := extractSecretExtras(secret)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeySecretType, "Opaque"))
			Expect(extras).To(HaveKey(ExtrasKeyKeys))
			keys := extras[ExtrasKeyKeys]
			Expect(keys).To(ContainSubstring("password"))
			Expect(keys).To(ContainSubstring("token"))
			// 不能包含值
			Expect(keys).NotTo(ContainSubstring("c2VjcmV0"))
		})

		It("should handle Secret with no data", func() {
			secret := &unstructured.Unstructured{Object: map[string]any{
				"type": "kubernetes.io/service-account-token",
			}}
			extras := extractSecretExtras(secret)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeySecretType, "kubernetes.io/service-account-token"))
			Expect(extras).NotTo(HaveKey(ExtrasKeyKeys))
		})
	})

	Describe("extractServiceAccountExtras", func() {
		It("should extract secrets and automountToken", func() {
			sa := &unstructured.Unstructured{Object: map[string]any{
				"automountServiceAccountToken": true,
				"secrets": []any{
					map[string]any{"name": "sa-token-abc"},
					map[string]any{"name": "sa-token-xyz"},
				},
			}}
			extras := extractServiceAccountExtras(sa)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyAutomountToken, "true"))
			Expect(extras).To(HaveKey(ExtrasKeySecrets))
			Expect(extras[ExtrasKeySecrets]).To(ContainSubstring("sa-token-abc"))
			Expect(extras[ExtrasKeySecrets]).To(ContainSubstring("sa-token-xyz"))
		})

		It("should handle ServiceAccount with no secrets", func() {
			sa := &unstructured.Unstructured{Object: map[string]any{
				"automountServiceAccountToken": false,
			}}
			extras := extractServiceAccountExtras(sa)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyAutomountToken, "false"))
			Expect(extras).NotTo(HaveKey(ExtrasKeySecrets))
		})

		It("should handle ServiceAccount with empty object", func() {
			sa := &unstructured.Unstructured{Object: map[string]any{}}
			extras := extractServiceAccountExtras(sa)
			Expect(extras).NotTo(HaveKey(ExtrasKeyAutomountToken))
			Expect(extras).NotTo(HaveKey(ExtrasKeySecrets))
		})
	})

	Describe("extractPodExtras - extended fields", func() {
		It("should extract nodeName, phase, restartCount and ready", func() {
			pod := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"nodeName": "node-1",
					"containers": []any{
						map[string]any{"name": "app", "image": "app:v1"},
					},
				},
				"status": map[string]any{
					"podIP": "127.0.0.1",
					"phase": "Running",
					"containerStatuses": []any{
						map[string]any{
							"name":         "app",
							"ready":        true,
							"restartCount": int64(3),
						},
					},
				},
			}}
			extras := extractPodExtras(pod)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyNodeName, "node-1"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyPhase, "Running"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyRestartCount, "3"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReady, "true"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyPodIP, "127.0.0.1"))
		})

		It("should report not ready when any container is not ready", func() {
			pod := &unstructured.Unstructured{Object: map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{"name": "app", "image": "app:v1"},
					},
				},
				"status": map[string]any{
					"containerStatuses": []any{
						map[string]any{
							"name":         "app",
							"ready":        true,
							"restartCount": int64(0),
						},
						map[string]any{
							"name":         "sidecar",
							"ready":        false,
							"restartCount": int64(5),
						},
					},
				},
			}}
			extras := extractPodExtras(pod)
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyReady, "false"))
			Expect(extras).To(HaveKeyWithValue(ExtrasKeyRestartCount, "5"))
		})
	})
})
