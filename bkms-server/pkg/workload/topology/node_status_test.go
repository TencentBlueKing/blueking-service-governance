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

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

var _ = Describe("BuilderStatus", func() {
	Describe("getResourceStatus", func() {
		It("should return Unknown when obj is nil", func() {
			result := getResourceStatus(k8skind.Deploy, nil, false)
			Expect(result.Code).To(Equal(k8sstatus.Unknown))
			Expect(result.Message).To(BeEmpty())
		})

		Context("Deployment", func() {
			It("should return Available when rollout is complete", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(1),
					},
					"spec": map[string]any{
						"replicas": int64(3),
					},
					"status": map[string]any{
						"observedGeneration": int64(1),
						"replicas":           int64(3),
						"updatedReplicas":    int64(3),
						"readyReplicas":      int64(3),
						"availableReplicas":  int64(3),
						"conditions": []any{
							map[string]any{"type": "Available", "status": "True"},
						},
					},
				}}
				result := getResourceStatus(k8skind.Deploy, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Available))
				Expect(result.Message).To(BeEmpty())
			})

			It("should return Degraded when Progressing condition is ProgressDeadlineExceeded", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"status": map[string]any{
						"conditions": []any{
							map[string]any{
								"type":    "Progressing",
								"status":  "False",
								"reason":  "ProgressDeadlineExceeded",
								"message": "deadline exceeded",
							},
						},
					},
				}}
				result := getResourceStatus(k8skind.Deploy, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Degraded))
				Expect(result.Message).To(Equal("ProgressDeadlineExceeded: deadline exceeded"))
			})

			It("should return Progressing when observedGeneration has not caught up", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(2),
					},
					"status": map[string]any{
						"observedGeneration": int64(1),
					},
				}}
				result := getResourceStatus(k8skind.Deploy, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Progressing))
				Expect(result.Message).To(Equal("observedGeneration has not caught up with metadata.generation"))
			})

			It("should return Progressing when replicas is 0 and status.replicas is 0", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(1),
					},
					"spec": map[string]any{
						"replicas": int64(0),
					},
					"status": map[string]any{
						"observedGeneration": int64(1),
						"replicas":           int64(0),
						"updatedReplicas":    int64(0),
						"readyReplicas":      int64(0),
						"availableReplicas":  int64(0),
					},
				}}
				result := getResourceStatus(k8skind.Deploy, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Progressing))
				Expect(result.Message).To(BeEmpty())
			})

			It("should use ParseForFederation when isFederation is true", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"spec": map[string]any{
						"replicas": int64(1),
					},
					"status": map[string]any{
						"replicas":          int64(1),
						"readyReplicas":     int64(1),
						"updatedReplicas":   int64(1),
						"availableReplicas": int64(1),
					},
				}}
				Expect(getResourceStatus(k8skind.Deploy, obj, false).Code).To(Equal(k8sstatus.Progressing))
				Expect(getResourceStatus(k8skind.Deploy, obj, true).Code).To(Equal(k8sstatus.Available))
			})
		})

		Context("StatefulSet", func() {
			It("should return Available when rollout is complete", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(1),
					},
					"spec": map[string]any{
						"replicas": int64(3),
					},
					"status": map[string]any{
						"observedGeneration": int64(1),
						"replicas":           int64(3),
						"readyReplicas":      int64(3),
						"updatedReplicas":    int64(3),
					},
				}}
				result := getResourceStatus(k8skind.STS, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Available))
				Expect(result.Message).To(BeEmpty())
			})

			It("should return Progressing when replicas are not consistent", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(1),
					},
					"spec": map[string]any{
						"replicas": int64(3),
					},
					"status": map[string]any{
						"observedGeneration": int64(1),
						"replicas":           int64(3),
						"readyReplicas":      int64(1),
						"updatedReplicas":    int64(3),
					},
				}}
				result := getResourceStatus(k8skind.STS, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Progressing))
				Expect(result.Message).To(Equal("replicas are not consistent: spec.replicas != status.readyReplicas"))
			})

			It("should return Available when replicas is 0 and status.replicas is 0", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(1),
					},
					"spec": map[string]any{
						"replicas": int64(0),
					},
					"status": map[string]any{
						"observedGeneration": int64(1),
						"replicas":           int64(0),
						"readyReplicas":      int64(0),
						"updatedReplicas":    int64(0),
					},
				}}
				result := getResourceStatus(k8skind.STS, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Available))
				Expect(result.Message).To(BeEmpty())
			})
		})

		Context("DaemonSet", func() {
			It("should return Available when rollout is complete", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(1),
					},
					"status": map[string]any{
						"observedGeneration":     int64(1),
						"desiredNumberScheduled": int64(3),
						"currentNumberScheduled": int64(3),
						"updatedNumberScheduled": int64(3),
						"numberReady":            int64(3),
						"numberUnavailable":      int64(0),
					},
				}}
				result := getResourceStatus(k8skind.DS, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Available))
				Expect(result.Message).To(BeEmpty())
			})

			It("should return Progressing when pods are not consistent", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(1),
					},
					"status": map[string]any{
						"observedGeneration":     int64(1),
						"desiredNumberScheduled": int64(3),
						"currentNumberScheduled": int64(3),
						"updatedNumberScheduled": int64(2),
						"numberReady":            int64(3),
						"numberUnavailable":      int64(0),
					},
				}}
				result := getResourceStatus(k8skind.DS, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Progressing))
				Expect(
					result.Message,
				).To(Equal("pods are not consistent: desiredNumberScheduled != updatedNumberScheduled"))
			})

			It("should return Progressing when some pods are unavailable", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"metadata": map[string]any{
						"generation": int64(1),
					},
					"status": map[string]any{
						"observedGeneration":     int64(1),
						"desiredNumberScheduled": int64(3),
						"currentNumberScheduled": int64(3),
						"updatedNumberScheduled": int64(3),
						"numberReady":            int64(3),
						"numberUnavailable":      int64(1),
					},
				}}
				result := getResourceStatus(k8skind.DS, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Progressing))
				Expect(result.Message).To(Equal("some pods are unavailable"))
			})
		})

		DescribeTable("always-healthy kinds",
			func(kind string) {
				obj := &unstructured.Unstructured{Object: map[string]any{}}
				result := getResourceStatus(kind, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Healthy))
				Expect(result.Message).To(BeEmpty())
			},
			Entry("Service", k8skind.SVC),
			Entry("ConfigMap", k8skind.CM),
			Entry("Secret", k8skind.Secret),
			Entry("ServiceAccount", k8skind.SA),
			Entry("Namespace", k8skind.NS),
		)

		Context("unknown kind", func() {
			It("should return Unknown", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{}}
				result := getResourceStatus("CronJob", obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Unknown))
				Expect(result.Message).To(BeEmpty())
			})
		})

		Context("ReplicaSet", func() {
			It("should return Available when readyReplicas equals replicas", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"spec": map[string]any{
						"replicas": int64(3),
					},
					"status": map[string]any{
						"readyReplicas": int64(3),
					},
				}}
				result := getResourceStatus(k8skind.RS, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Available))
				Expect(result.Message).To(BeEmpty())
			})

			It("should return Progressing when readyReplicas less than replicas", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"spec": map[string]any{
						"replicas": int64(3),
					},
					"status": map[string]any{
						"readyReplicas": int64(1),
					},
				}}
				result := getResourceStatus(k8skind.RS, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Progressing))
				Expect(result.Message).To(BeEmpty())
			})
		})

		Context("HorizontalPodAutoscaler", func() {
			It("should return Healthy when AbleToScale and ScalingActive are both True", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"status": map[string]any{
						"conditions": []any{
							map[string]any{"type": "AbleToScale", "status": "True"},
							map[string]any{"type": "ScalingActive", "status": "True"},
						},
					},
				}}
				result := getResourceStatus(k8skind.HPA, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Healthy))
				Expect(result.Message).To(BeEmpty())
			})

			It("should return Degraded when ScalingActive is False", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"status": map[string]any{
						"conditions": []any{
							map[string]any{"type": "AbleToScale", "status": "True"},
							map[string]any{"type": "ScalingActive", "status": "False"},
						},
					},
				}}
				result := getResourceStatus(k8skind.HPA, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Degraded))
				Expect(result.Message).To(BeEmpty())
			})
		})

		Context("GeneralPodAutoscaler", func() {
			It("should return Healthy when AbleToScale and ScalingActive are both True", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{
					"status": map[string]any{
						"conditions": []any{
							map[string]any{"type": "AbleToScale", "status": "True"},
							map[string]any{"type": "ScalingActive", "status": "True"},
						},
					},
				}}
				result := getResourceStatus(k8skind.GPA, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Healthy))
				Expect(result.Message).To(BeEmpty())
			})

			It("should return Unknown when key conditions are missing", func() {
				obj := &unstructured.Unstructured{Object: map[string]any{}}
				result := getResourceStatus(k8skind.GPA, obj, false)
				Expect(result.Code).To(Equal(k8sstatus.Unknown))
				Expect(result.Message).To(BeEmpty())
			})
		})
	})
})
