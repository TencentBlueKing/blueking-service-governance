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

package overview

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

var _ = Describe("instance helpers", func() {
	Describe("countPodStates", func() {
		newPod := func(phase, readyStatus string) unstructured.Unstructured {
			return unstructured.Unstructured{Object: map[string]any{
				"status": map[string]any{
					"phase":      phase,
					"conditions": []any{map[string]any{"type": "Ready", "status": readyStatus}},
				},
			}}
		}

		It("counts running and ready pods as running, the others as abnormal", func() {
			running, abnormal := countPodStates([]unstructured.Unstructured{
				newPod("Running", "True"), newPod("Running", "False"), newPod("Running", "True"),
			})
			Expect(running).To(Equal(int32(2)))
			Expect(abnormal).To(Equal(int32(1)))
		})

		It("counts terminated pods as abnormal even when reported as completed", func() {
			running, abnormal := countPodStates([]unstructured.Unstructured{
				{Object: map[string]any{"status": map[string]any{
					"phase": "Failed",
					"conditions": []any{
						map[string]any{"type": "Ready", "status": "False", "reason": "PodCompleted"},
					},
				}}},
			})
			Expect(running).To(BeZero())
			Expect(abnormal).To(Equal(int32(1)))
		})

		It("counts pods without a Ready condition as abnormal", func() {
			running, abnormal := countPodStates([]unstructured.Unstructured{
				{Object: map[string]any{"status": map[string]any{"phase": "Running"}}},
			})
			Expect(running).To(BeZero())
			Expect(abnormal).To(Equal(int32(1)))
		})

		It("returns zeros for an empty pod list", func() {
			running, abnormal := countPodStates(nil)
			Expect(running).To(BeZero())
			Expect(abnormal).To(BeZero())
		})
	})

	Describe("instanceCountsFromReplicas", func() {
		pods := []unstructured.Unstructured{
			{Object: map[string]any{"status": map[string]any{
				"phase":      "Running",
				"conditions": []any{map[string]any{"type": "Ready", "status": "True"}},
			}}},
			{Object: map[string]any{"status": map[string]any{"phase": "Pending"}}},
		}

		It("returns nil when pod list failed", func() {
			Expect(instanceCounts(lo.ToPtr(int32(3)), pods, false)).To(BeNil())
		})

		It("returns nil when replicas are missing", func() {
			Expect(instanceCounts(nil, pods, true)).To(BeNil())
		})

		It("fills expected from replicas and running/abnormal from pods", func() {
			Expect(instanceCounts(lo.ToPtr(int32(3)), pods, true)).To(Equal(&InstanceCounts{
				Running: 1, Expected: 3, Abnormal: 1,
			}))
		})
	})

	Describe("extractMainWorkload", func() {
		It("picks Deployment from the recorded resources", func() {
			kind, name := extractMainWorkload(&appmodel.Record{ResourceKeys: appmodel.ResourceKeys{
				{Kind: k8skind.SVC, Name: "svc-app"},
				{Kind: k8skind.Deploy, Name: "app"},
			}})
			Expect(kind).To(Equal(k8skind.Deploy))
			Expect(name).To(Equal("app"))
		})

		It("uses WorkloadKind when the record has it", func() {
			kind, name := extractMainWorkload(&appmodel.Record{
				WorkloadKind: k8skind.Deploy,
				ResourceKeys: appmodel.ResourceKeys{
					{Kind: k8skind.Deploy, Name: "app"},
					{Kind: k8skind.SVC, Name: "svc-app"},
				},
			})
			Expect(kind).To(Equal(k8skind.Deploy))
			Expect(name).To(Equal("app"))
		})
	})

	Describe("groupDeployRecordsByCluster", func() {
		It("groups records of the same cluster together", func() {
			byCluster := groupDeployRecordsByCluster([]deployRecordForEnv{
				{EnvName: "dev", Record: &appmodel.Record{ClusterID: "cls-1"}},
				{EnvName: "test", Record: &appmodel.Record{ClusterID: "cls-1"}},
				{EnvName: "prod", Record: &appmodel.Record{ClusterID: "cls-2"}},
			})
			Expect(byCluster).To(HaveLen(2))
			Expect(byCluster["cls-1"]).To(HaveLen(2))
			Expect(byCluster["cls-2"]).To(HaveLen(1))
		})

		It("drops records that cannot locate a cluster", func() {
			Expect(groupDeployRecordsByCluster([]deployRecordForEnv{
				{EnvName: "no-record"},
				{EnvName: "no-cluster", Record: &appmodel.Record{}},
			})).To(BeEmpty())
		})
	})

	Describe("listPods", func() {
		It("returns an error when the deploy record has no label selector", func() {
			_, err := listPods(context.Background(), &clusterQuerier{}, deployRecordForEnv{
				Record: &appmodel.Record{Namespace: "ns-1"},
			})
			Expect(err).To(MatchError("deploy record has no label selector"))
		})
	})

	// 缺 GameDeployment 在发起集群调用之前返回，因此 querier 无需持有真实客户端。
	Describe("queryEnvClusterDataForEnv", func() {
		It("reports unavailable when the deploy record has no GameDeployment", func() {
			data, err := queryEnvClusterDataForEnv(context.Background(), &clusterQuerier{}, deployRecordForEnv{
				EnvName: "prod",
				Record: &appmodel.Record{
					Namespace:     "ns-1",
					LabelSelector: map[string]string{"app.kubernetes.io/name": "app"},
					ResourceKeys:  appmodel.ResourceKeys{{Kind: k8skind.SVC, Name: "app"}},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(BeNil())
		})
	})

	Describe("extractMainContainerResourcesFromContainers", func() {
		qty := func(s string) resource.Quantity { return resource.MustParse(s) }
		container := func(name, cpuReq, cpuLim, memReq, memLim string) corev1.Container {
			return corev1.Container{
				Name: name,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    qty(cpuReq),
						corev1.ResourceMemory: qty(memReq),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    qty(cpuLim),
						corev1.ResourceMemory: qty(memLim),
					},
				},
			}
		}

		It("returns an empty spec when the workload has no containers", func() {
			Expect(extractMainContainerResources(
				context.Background(), "ns", "app", nil,
			)).To(Equal(ResourceSpec{}))
		})

		It("reads cpu and memory from the main container and ignores sidecars", func() {
			out := extractMainContainerResources(
				context.Background(), "ns", "app", []corev1.Container{
					container("sidecar", "50m", "100m", "64Mi", "128Mi"),
					container("main", "2", "4", "4Gi", "8Gi"),
				},
			)
			Expect(out).To(Equal(ResourceSpec{
				CPULimits:      "4",
				CPURequests:    "2",
				MemoryLimits:   "8Gi",
				MemoryRequests: "4Gi",
			}))
		})

		It("returns an empty spec when only sidecar containers exist", func() {
			out := extractMainContainerResources(
				context.Background(), "ns", "app", []corev1.Container{
					container("sidecar", "50m", "100m", "64Mi", "128Mi"),
				},
			)
			Expect(out).To(Equal(ResourceSpec{}))
		})
	})
})
