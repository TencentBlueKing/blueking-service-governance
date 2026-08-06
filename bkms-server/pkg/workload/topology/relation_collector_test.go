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

var _ = Describe("RelationCollector", func() {
	Describe("owner references", func() {
		It("should collect owner reference relations from a ReplicaSet", func() {
			rs := &unstructured.Unstructured{Object: map[string]any{
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

			resources := map[string]*unstructured.Unstructured{
				"ReplicaSet/default/nginx-abc123": rs,
			}
			relations := NewRelationCollector(resources).Collect()

			ownerRefs := filterByType(relations, RelationTypeOwnerReference)
			Expect(ownerRefs).To(HaveLen(1))
			Expect(ownerRefs[0].SourceKind).To(Equal("Deployment"))
			Expect(ownerRefs[0].SourceName).To(Equal("nginx"))
			Expect(ownerRefs[0].TargetKind).To(Equal("ReplicaSet"))
			Expect(ownerRefs[0].TargetName).To(Equal("nginx-abc123"))
		})
	})

	Describe("label selectors", func() {
		It("should collect label selector relations from a Service", func() {
			svc := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]any{
					"name":      "nginx-svc",
					"namespace": "default",
				},
				"spec": map[string]any{
					"selector": map[string]any{
						"app": "nginx",
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"Service/default/nginx-svc": svc,
			}
			relations := NewRelationCollector(resources).Collect()

			selectorRels := filterByType(relations, RelationTypeLabelSelector)
			Expect(selectorRels).To(HaveLen(1))
			Expect(selectorRels[0].SourceKind).To(Equal("Service"))
			Expect(selectorRels[0].SourceName).To(Equal("nginx-svc"))
			Expect(selectorRels[0].TargetKind).To(Equal("Pod"))
			Expect(selectorRels[0].MatchedLabels).To(HaveKeyWithValue("app", "nginx"))
		})

		It("should collect label selector relations from a Deployment", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "nginx",
					"namespace": "default",
				},
				"spec": map[string]any{
					"selector": map[string]any{
						"matchLabels": map[string]any{
							"app": "nginx",
						},
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"Deployment/default/nginx": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			selectorRels := filterByType(relations, RelationTypeLabelSelector)
			Expect(selectorRels).To(HaveLen(1))
			Expect(selectorRels[0].SourceKind).To(Equal("Deployment"))
			Expect(selectorRels[0].SourceFieldPath).To(Equal("spec.selector.matchLabels"))
		})
	})

	Describe("volume mounts", func() {
		It("should collect ConfigMap and Secret volume mount relations", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "web-app",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"volumes": []any{
								map[string]any{
									"name": "config-vol",
									"configMap": map[string]any{
										"name": "app-config",
									},
								},
								map[string]any{
									"name": "secret-vol",
									"secret": map[string]any{
										"secretName": "tls-cert",
									},
								},
							},
						},
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"Deployment/default/web-app": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			volumeRels := filterByType(relations, RelationTypeVolumeMount)
			Expect(volumeRels).To(HaveLen(2))

			cmRel := findRelationByTarget(volumeRels, "ConfigMap", "app-config")
			Expect(cmRel).NotTo(BeNil())
			Expect(cmRel.SourceKind).To(Equal("Deployment"))

			secretRel := findRelationByTarget(volumeRels, "Secret", "tls-cert")
			Expect(secretRel).NotTo(BeNil())
			Expect(secretRel.SourceKind).To(Equal("Deployment"))
		})
	})

	Describe("backend refs", func() {
		It("should collect backend ref relations from Ingress (networking.k8s.io/v1)", func() {
			ingress := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "networking.k8s.io/v1",
				"kind":       "Ingress",
				"metadata": map[string]any{
					"name":      "web-ingress",
					"namespace": "default",
				},
				"spec": map[string]any{
					"rules": []any{
						map[string]any{
							"host": "example.com",
							"http": map[string]any{
								"paths": []any{
									map[string]any{
										"path": "/",
										"backend": map[string]any{
											"service": map[string]any{
												"name": "web-svc",
												"port": map[string]any{
													"number": int64(80),
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

			resources := map[string]*unstructured.Unstructured{
				"Ingress/default/web-ingress": ingress,
			}
			relations := NewRelationCollector(resources).Collect()

			backendRels := filterByType(relations, RelationTypeBackendRef)
			Expect(backendRels).To(HaveLen(1))
			Expect(backendRels[0].SourceKind).To(Equal("Ingress"))
			Expect(backendRels[0].TargetKind).To(Equal("Service"))
			Expect(backendRels[0].TargetName).To(Equal("web-svc"))
		})
	})

	Describe("env refs", func() {
		It("should collect env valueFrom configMapKeyRef and secretKeyRef relations", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "api-server",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "app",
									"env": []any{
										map[string]any{
											"name": "DB_HOST",
											"valueFrom": map[string]any{
												"configMapKeyRef": map[string]any{
													"name": "db-config",
													"key":  "host",
												},
											},
										},
										map[string]any{
											"name": "DB_PASSWORD",
											"valueFrom": map[string]any{
												"secretKeyRef": map[string]any{
													"name": "db-secret",
													"key":  "password",
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

			resources := map[string]*unstructured.Unstructured{
				"Deployment/default/api-server": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			envRefs := filterByType(relations, RelationTypeEnvRef)
			Expect(envRefs).To(HaveLen(2))

			cmRel := findRelationByTarget(envRefs, "ConfigMap", "db-config")
			Expect(cmRel).NotTo(BeNil())
			Expect(cmRel.SourceKind).To(Equal("Deployment"))
			Expect(cmRel.SourceFieldPath).To(ContainSubstring("configMapKeyRef"))

			secretRel := findRelationByTarget(envRefs, "Secret", "db-secret")
			Expect(secretRel).NotTo(BeNil())
			Expect(secretRel.SourceFieldPath).To(ContainSubstring("secretKeyRef"))
		})

		It("should collect envFrom configMapRef and secretRef relations", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "web-app",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "app",
									"envFrom": []any{
										map[string]any{
											"configMapRef": map[string]any{
												"name": "app-env",
											},
										},
										map[string]any{
											"secretRef": map[string]any{
												"name": "app-secrets",
											},
										},
									},
								},
							},
						},
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"Deployment/default/web-app": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			envRefs := filterByType(relations, RelationTypeEnvRef)
			Expect(envRefs).To(HaveLen(2))

			cmRel := findRelationByTarget(envRefs, "ConfigMap", "app-env")
			Expect(cmRel).NotTo(BeNil())
			Expect(cmRel.SourceFieldPath).To(ContainSubstring("envFrom"))
			Expect(cmRel.SourceFieldPath).To(ContainSubstring("configMapRef"))

			secretRel := findRelationByTarget(envRefs, "Secret", "app-secrets")
			Expect(secretRel).NotTo(BeNil())
			Expect(secretRel.SourceFieldPath).To(ContainSubstring("envFrom"))
			Expect(secretRel.SourceFieldPath).To(ContainSubstring("secretRef"))
		})

		It("should deduplicate env refs to the same ConfigMap from multiple containers", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "multi-container",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "app",
									"env": []any{
										map[string]any{
											"name": "CFG",
											"valueFrom": map[string]any{
												"configMapKeyRef": map[string]any{
													"name": "shared-config",
													"key":  "key1",
												},
											},
										},
									},
								},
								map[string]any{
									"name": "sidecar",
									"env": []any{
										map[string]any{
											"name": "CFG",
											"valueFrom": map[string]any{
												"configMapKeyRef": map[string]any{
													"name": "shared-config",
													"key":  "key2",
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

			resources := map[string]*unstructured.Unstructured{
				"Deployment/default/multi-container": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			envRefs := filterByType(relations, RelationTypeEnvRef)
			// 同一个 ConfigMap 只应产生一条关系
			Expect(envRefs).To(HaveLen(1))
			Expect(envRefs[0].TargetName).To(Equal("shared-config"))
		})
	})

	Describe("scale target refs", func() {
		It("should collect scale target ref relation from HPA", func() {
			hpa := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "autoscaling/v2",
				"kind":       "HorizontalPodAutoscaler",
				"metadata": map[string]any{
					"name":      "web-hpa",
					"namespace": "default",
				},
				"spec": map[string]any{
					"scaleTargetRef": map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"name":       "web",
					},
					"minReplicas": int64(2),
					"maxReplicas": int64(10),
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"HorizontalPodAutoscaler/default/web-hpa": hpa,
			}
			relations := NewRelationCollector(resources).Collect()

			scaleRefs := filterByType(relations, RelationTypeScaleTargetRef)
			Expect(scaleRefs).To(HaveLen(1))
			Expect(scaleRefs[0].SourceKind).To(Equal("HorizontalPodAutoscaler"))
			Expect(scaleRefs[0].SourceName).To(Equal("web-hpa"))
			Expect(scaleRefs[0].TargetKind).To(Equal("Deployment"))
			Expect(scaleRefs[0].TargetName).To(Equal("web"))
			Expect(scaleRefs[0].SourceFieldPath).To(Equal("spec.scaleTargetRef"))
			Expect(scaleRefs[0].TargetFieldPath).To(Equal("metadata.name"))
		})

		It("should not generate relation when scaleTargetRef is missing", func() {
			hpa := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "autoscaling/v2",
				"kind":       "HorizontalPodAutoscaler",
				"metadata": map[string]any{
					"name":      "broken-hpa",
					"namespace": "default",
				},
				"spec": map[string]any{
					"minReplicas": int64(1),
					"maxReplicas": int64(5),
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"HorizontalPodAutoscaler/default/broken-hpa": hpa,
			}
			relations := NewRelationCollector(resources).Collect()

			scaleRefs := filterByType(relations, RelationTypeScaleTargetRef)
			Expect(scaleRefs).To(BeEmpty())
		})
	})

	Describe("service account refs", func() {
		It("should collect service account ref relation from Deployment", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "web",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"serviceAccountName": "web-sa",
							"containers": []any{
								map[string]any{
									"name":  "app",
									"image": "nginx:latest",
								},
							},
						},
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"Deployment/default/web": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			saRefs := filterByType(relations, RelationTypeServiceAccountRef)
			Expect(saRefs).To(HaveLen(1))
			Expect(saRefs[0].SourceKind).To(Equal("Deployment"))
			Expect(saRefs[0].SourceName).To(Equal("web"))
			Expect(saRefs[0].TargetKind).To(Equal("ServiceAccount"))
			Expect(saRefs[0].TargetName).To(Equal("web-sa"))
			Expect(saRefs[0].SourceFieldPath).To(Equal("spec.template.spec.serviceAccountName"))
			Expect(saRefs[0].TargetFieldPath).To(Equal("metadata.name"))
		})

		It("should not generate relation when serviceAccountName is empty", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "web",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name":  "app",
									"image": "nginx:latest",
								},
							},
						},
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"Deployment/default/web": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			saRefs := filterByType(relations, RelationTypeServiceAccountRef)
			Expect(saRefs).To(BeEmpty())
		})

		It("should not generate relation when serviceAccountName is default", func() {
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "web",
					"namespace": "default",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"serviceAccountName": "default",
							"containers": []any{
								map[string]any{
									"name":  "app",
									"image": "nginx:latest",
								},
							},
						},
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"Deployment/default/web": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			saRefs := filterByType(relations, RelationTypeServiceAccountRef)
			Expect(saRefs).To(BeEmpty())
		})

		It("should collect service account ref from CronJob", func() {
			cronJob := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "batch/v1",
				"kind":       "CronJob",
				"metadata": map[string]any{
					"name":      "backup",
					"namespace": "default",
				},
				"spec": map[string]any{
					"schedule": "0 2 * * *",
					"jobTemplate": map[string]any{
						"spec": map[string]any{
							"template": map[string]any{
								"spec": map[string]any{
									"serviceAccountName": "backup-sa",
									"containers": []any{
										map[string]any{
											"name":  "backup",
											"image": "backup:latest",
										},
									},
								},
							},
						},
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"CronJob/default/backup": cronJob,
			}
			relations := NewRelationCollector(resources).Collect()

			saRefs := filterByType(relations, RelationTypeServiceAccountRef)
			Expect(saRefs).To(HaveLen(1))
			Expect(saRefs[0].SourceKind).To(Equal("CronJob"))
			Expect(saRefs[0].SourceName).To(Equal("backup"))
			Expect(saRefs[0].TargetKind).To(Equal("ServiceAccount"))
			Expect(saRefs[0].TargetName).To(Equal("backup-sa"))
			Expect(saRefs[0].SourceFieldPath).To(Equal("spec.jobTemplate.spec.template.spec.serviceAccountName"))
		})
	})

	Describe("mixed resources", func() {
		It("should collect relations from multiple resource types", func() {
			svc := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]any{
					"name":      "svc",
					"namespace": "default",
				},
				"spec": map[string]any{
					"selector": map[string]any{
						"app": "web",
					},
				},
			}}
			deploy := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name":      "web",
					"namespace": "default",
				},
				"spec": map[string]any{
					"selector": map[string]any{
						"matchLabels": map[string]any{
							"app": "web",
						},
					},
					"template": map[string]any{
						"spec": map[string]any{
							"volumes": []any{
								map[string]any{
									"name": "cfg",
									"configMap": map[string]any{
										"name": "web-config",
									},
								},
							},
						},
					},
				},
			}}

			resources := map[string]*unstructured.Unstructured{
				"Service/default/svc":    svc,
				"Deployment/default/web": deploy,
			}
			relations := NewRelationCollector(resources).Collect()

			// Service: 1 label_selector, Deployment: 1 label_selector + 1 volume_mount = 3 total
			Expect(len(relations)).To(BeNumerically(">=", 3))
		})
	})
})

// filterByType 按关系类型过滤
func filterByType(relations []ResourceRelation, relType RelationType) []ResourceRelation {
	var result []ResourceRelation
	for _, r := range relations {
		if r.RelationType == relType {
			result = append(result, r)
		}
	}
	return result
}

// findRelationByTarget 按目标资源查找关系
func findRelationByTarget(relations []ResourceRelation, targetKind, targetName string) *ResourceRelation {
	for i, r := range relations {
		if r.TargetKind == targetKind && r.TargetName == targetName {
			return &relations[i]
		}
	}
	return nil
}
