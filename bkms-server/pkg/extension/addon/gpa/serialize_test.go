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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("buildGPAManifest", func() {
	It("should build a valid GeneralPodAutoscaler manifest with cpu+memory utilization metrics", func() {
		config := &GPAConfig{
			Name:        "web-autoscale",
			AppID:       "app-1",
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics: []GPAMetric{
				{Resource: ResourceCPU, AverageUtilization: 60},
				{Resource: ResourceMemory, AverageUtilization: 70},
			},
		}

		manifest := buildGPAManifest(config, "ws-1", "dev", "web")

		Expect(manifest["apiVersion"]).To(Equal("autoscaling.tkex.tencent.com/v1alpha1"))
		Expect(manifest["kind"]).To(Equal("GeneralPodAutoscaler"))

		metadata := manifest["metadata"].(map[string]any)
		Expect(metadata["name"]).To(Equal("web-autoscale"))
		labels := metadata["labels"].(map[string]any)
		Expect(labels[LabelKeyWorkspaceID]).To(Equal("ws-1"))
		Expect(labels[LabelKeyEnvName]).To(Equal("dev"))
		Expect(labels[LabelKeyAppID]).To(Equal("app-1"))

		spec := manifest["spec"].(map[string]any)
		Expect(spec["minReplicas"]).To(Equal(int32(2)))
		Expect(spec["maxReplicas"]).To(Equal(int32(10)))

		scaleTargetRef := spec["scaleTargetRef"].(map[string]any)
		Expect(scaleTargetRef["kind"]).To(Equal("GameDeployment"))
		Expect(scaleTargetRef["apiVersion"]).To(Equal("tkex.tencent.com/v1alpha1"))
		Expect(scaleTargetRef["name"]).To(Equal("web"))

		metric := spec["metric"].(map[string]any)
		metrics := metric["metrics"].([]any)
		Expect(metrics).To(HaveLen(2))

		cpu := metrics[0].(map[string]any)
		Expect(cpu["type"]).To(Equal("Resource"))
		cpuRes := cpu["resource"].(map[string]any)
		Expect(cpuRes["name"]).To(Equal("cpu"))
		cpuTarget := cpuRes["target"].(map[string]any)
		Expect(cpuTarget["type"]).To(Equal("Utilization"))
		Expect(cpuTarget["averageUtilization"]).To(Equal(int32(60)))

		mem := metrics[1].(map[string]any)
		memRes := mem["resource"].(map[string]any)
		Expect(memRes["name"]).To(Equal("memory"))
		memTarget := memRes["target"].(map[string]any)
		Expect(memTarget["type"]).To(Equal("Utilization"))
		Expect(memTarget["averageUtilization"]).To(Equal(int32(70)))
	})

	It("should build a single-metric manifest", func() {
		config := &GPAConfig{
			Name:        "single",
			AppID:       "app-2",
			MinReplicas: 1,
			MaxReplicas: 5,
			Metrics:     []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 80}},
		}
		manifest := buildGPAManifest(config, "ws-1", "prod", "svc")
		spec := manifest["spec"].(map[string]any)
		metrics := spec["metric"].(map[string]any)["metrics"].([]any)
		Expect(metrics).To(HaveLen(1))
		// 仅指标模式时不应输出 spec.time
		_, hasTime := spec["time"]
		Expect(hasTime).To(BeFalse())
	})

	It("should build a time-only manifest with spec.time.ranges and without spec.metric", func() {
		config := &GPAConfig{
			Name:        "time-only",
			AppID:       "app-3",
			MinReplicas: 1,
			MaxReplicas: 10,
			TimeRanges: []GPATimeRange{
				{DesiredReplicas: 4, Schedule: "* 2-3 * * *", Enabled: true},
				{DesiredReplicas: 6, Schedule: "* 4-5 * * *", Enabled: true},
			},
		}
		manifest := buildGPAManifest(config, "ws-1", "dev", "web")
		spec := manifest["spec"].(map[string]any)

		// 仅定时模式时不应输出 spec.metric
		_, hasMetric := spec["metric"]
		Expect(hasMetric).To(BeFalse())

		ranges := spec["time"].(map[string]any)["ranges"].([]any)
		Expect(ranges).To(HaveLen(2))
		first := ranges[0].(map[string]any)
		Expect(first["desiredReplicas"]).To(Equal(int32(4)))
		Expect(first["schedule"]).To(Equal("* 2-3 * * *"))
	})

	It("should build a manifest with both spec.metric and spec.time when both modes are set", func() {
		config := &GPAConfig{
			Name:        "both",
			AppID:       "app-4",
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics:     []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
			TimeRanges:  []GPATimeRange{{DesiredReplicas: 8, Schedule: "* 9-18 * * 1-5", Enabled: true}},
		}
		manifest := buildGPAManifest(config, "ws-1", "dev", "web")
		spec := manifest["spec"].(map[string]any)

		metrics := spec["metric"].(map[string]any)["metrics"].([]any)
		Expect(metrics).To(HaveLen(1))
		ranges := spec["time"].(map[string]any)["ranges"].([]any)
		Expect(ranges).To(HaveLen(1))
	})

	It("should only write enabled time ranges into spec.time.ranges", func() {
		config := &GPAConfig{
			Name:        "mixed",
			AppID:       "app-5",
			MinReplicas: 1,
			MaxReplicas: 10,
			TimeRanges: []GPATimeRange{
				{DesiredReplicas: 4, Schedule: "* 2-3 * * *", Enabled: true},
				{DesiredReplicas: 6, Schedule: "* 4-5 * * *", Enabled: false},
			},
		}
		manifest := buildGPAManifest(config, "ws-1", "dev", "web")
		spec := manifest["spec"].(map[string]any)

		// 仅启用的规则被写入
		ranges := spec["time"].(map[string]any)["ranges"].([]any)
		Expect(ranges).To(HaveLen(1))
		Expect(ranges[0].(map[string]any)["desiredReplicas"]).To(Equal(int32(4)))
	})

	It("should not emit spec.time when all time ranges are disabled", func() {
		config := &GPAConfig{
			Name:        "all-disabled",
			AppID:       "app-6",
			MinReplicas: 1,
			MaxReplicas: 10,
			Metrics:     []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
			TimeRanges: []GPATimeRange{
				{DesiredReplicas: 4, Schedule: "* 2-3 * * *", Enabled: false},
			},
		}
		manifest := buildGPAManifest(config, "ws-1", "dev", "web")
		spec := manifest["spec"].(map[string]any)

		// 全部未启用时不应输出 spec.time
		_, hasTime := spec["time"]
		Expect(hasTime).To(BeFalse())
	})

	It("should emit compute-by-limits annotation when ComputeByLimits is true", func() {
		config := &GPAConfig{
			Name:            "by-limits",
			AppID:           "app-7",
			MinReplicas:     2,
			MaxReplicas:     10,
			Metrics:         []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 60}},
			ComputeByLimits: true,
		}
		manifest := buildGPAManifest(config, "ws-1", "dev", "web")
		metadata := manifest["metadata"].(map[string]any)

		annotations := metadata["annotations"].(map[string]any)
		Expect(annotations[annotationKeyComputeByLimits]).To(Equal(annotationValueTrue))
	})
})

var _ = Describe("parseGPAStatusFromUnstructured", func() {
	It("should parse status fields and labels", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "autoscaling.tkex.tencent.com/v1alpha1",
			"kind":       "GeneralPodAutoscaler",
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
				"lastScaleTime":   "2026-06-18T10:00:00Z",
			},
		}}

		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Name).To(Equal("web-autoscale"))
		Expect(status.AppID).To(Equal("app-1"))
		Expect(status.WorkspaceID).To(Equal("ws-1"))
		Expect(status.EnvName).To(Equal("dev"))
		Expect(status.CurrentReplicas).To(Equal(int32(3)))
		Expect(status.DesiredReplicas).To(Equal(int32(5)))
		Expect(status.LastScaleTime).To(Equal("2026-06-18T10:00:00Z"))
	})

	It("should return Initializing status when CR has no status yet", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "new-gpa"},
		}}
		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Name).To(Equal("new-gpa"))
		Expect(status.CurrentReplicas).To(Equal(int32(0)))
		// status 段整体缺失 → controller 未处理，属正常过渡态 Initializing
		Expect(status.Phase).To(Equal("Initializing"))
	})

	It("should derive phase=Active when AbleToScale=True, ScalingActive=True, ScalingLimited=False", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "gpa-active"},
			"status": map[string]any{
				"conditions": []any{
					map[string]any{"type": "AbleToScale", "status": "True", "message": "ok"},
					map[string]any{"type": "ScalingActive", "status": "True", "message": "ok"},
					// ScalingLimited=False 表示未受限（健康），message 为空避免被计入 statusMessage
					map[string]any{"type": "ScalingLimited", "status": "False", "message": ""},
				},
			},
		}}
		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Phase).To(Equal("Active"))
		// 所有非 True condition 的 message 均为空 → statusMessage 为空
		Expect(status.StatusMessage).To(BeEmpty())
	})

	It("should derive phase=Limited; statusMessage empty when all conditions True", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "gpa-limited"},
			"status": map[string]any{
				"conditions": []any{
					map[string]any{"type": "AbleToScale", "status": "True", "message": "ok"},
					map[string]any{"type": "ScalingActive", "status": "True", "message": "ok"},
					// ScalingLimited=True → Limited；其为 True 故 message 不计入 statusMessage
					map[string]any{"type": "ScalingLimited", "status": "True", "message": "already at maxReplicas"},
				},
			},
		}}
		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Phase).To(Equal("Limited"))
		// 所有 condition 均 True → statusMessage 为空
		Expect(status.StatusMessage).To(BeEmpty())
	})

	It("should derive phase=Paused with ScalingActive message", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "gpa-paused"},
			"status": map[string]any{
				"conditions": []any{
					map[string]any{"type": "AbleToScale", "status": "True", "message": "ok"},
					map[string]any{"type": "ScalingActive", "status": "False", "message": "no valid metric"},
				},
			},
		}}
		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Phase).To(Equal("Paused"))
		Expect(status.StatusMessage).To(ContainSubstring("no valid metric"))
	})

	It(
		"should derive phase=Failed with AbleToScale message, priority over ScalingActive=False",
		func() {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"metadata": map[string]any{"name": "gpa-failed"},
				"status": map[string]any{
					"conditions": []any{
						map[string]any{"type": "AbleToScale", "status": "False", "message": "backoff"},
						map[string]any{"type": "ScalingActive", "status": "False", "message": "no valid metric"},
					},
				},
			}}
			status, err := parseGPAStatusFromUnstructured(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Phase).To(Equal("Failed"))
			// 两条非 True condition 的 message 都应计入
			Expect(status.StatusMessage).To(ContainSubstring("backoff"))
			Expect(status.StatusMessage).To(ContainSubstring("no valid metric"))
			Expect(status.StatusMessage).To(ContainSubstring("; "))
		},
	)

	It("should derive phase=Unknown when conditions field is absent", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "gpa-no-conditions"},
			"status":   map[string]any{"currentReplicas": int64(3)},
		}}
		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		// conditions 缺失 → controller 未处理，属正常过渡态 Initializing（而非异常 Unknown）
		Expect(status.Phase).To(Equal("Initializing"))
		Expect(status.StatusMessage).To(BeEmpty())
		Expect(status.CurrentReplicas).To(Equal(int32(3)))
	})

	It("should derive phase=Initializing when conditions is empty (R-005)", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "gpa-empty-conditions"},
			"status": map[string]any{
				"conditions": []any{},
			},
		}}
		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		// conditions 为空数组 → 同样视为 Initializing
		Expect(status.Phase).To(Equal("Initializing"))
	})

	It("should leave statusMessage empty when all conditions are True", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "gpa-all-true"},
			"status": map[string]any{
				"conditions": []any{
					map[string]any{"type": "AbleToScale", "status": "True", "message": "ok"},
					map[string]any{"type": "ScalingActive", "status": "True", "message": "ok"},
					map[string]any{"type": "ScalingLimited", "status": "True", "message": "at boundary"},
				},
			},
		}}
		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		// 所有 condition 均 True → statusMessage 为空
		Expect(status.StatusMessage).To(BeEmpty())
	})

	It("should include unrecognized condition messages in statusMessage but not affect phase", func() {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "gpa-custom-cond"},
			"status": map[string]any{
				"conditions": []any{
					map[string]any{"type": "AbleToScale", "status": "True", "message": "ok"},
					map[string]any{"type": "ScalingActive", "status": "True", "message": "ok"},
					map[string]any{"type": "ScalingLimited", "status": "False", "message": "within bounds"},
					map[string]any{"type": "CustomCondition", "status": "False", "message": "custom warning"},
				},
			},
		}}
		status, err := parseGPAStatusFromUnstructured(obj)
		Expect(err).NotTo(HaveOccurred())
		// 未识别 condition 不影响 phase 推导
		Expect(status.Phase).To(Equal("Active"))
		// 但其非 True 的 message 仍计入 statusMessage
		Expect(status.StatusMessage).To(ContainSubstring("custom warning"))
	})
})
