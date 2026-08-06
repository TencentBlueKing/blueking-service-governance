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

package serializer_test

import (
	"time"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa/serializer"
)

var _ = Describe("UpsertGPAConfigInput", func() {
	Describe("ToModel", func() {
		It("should map all fields and leave Name empty for the caller to generate", func() {
			input := serializer.UpsertGPAConfigInput{
				MinReplicas: 2,
				MaxReplicas: 10,
				Metrics: []serializer.GPAMetricInput{
					{Resource: "cpu", AverageUtilization: 60},
					{Resource: "memory", AverageUtilization: 80},
				},
			}

			model := input.ToModel("demo-app", "dev")
			Expect(model.AppID).To(Equal("demo-app"))
			Expect(model.EnvName).To(Equal("dev"))
			Expect(model.MinReplicas).To(Equal(int32(2)))
			Expect(model.MaxReplicas).To(Equal(int32(10)))
			// Name 由 handler 通过 GenerateName 生成，序列化层不设置
			Expect(model.Name).To(BeEmpty())
			Expect(model.Metrics).To(HaveLen(2))
			Expect(model.Metrics[0].Resource).To(Equal(gpa.ResourceCPU))
			Expect(model.Metrics[0].AverageUtilization).To(Equal(int32(60)))
			Expect(model.Metrics[1].Resource).To(Equal(gpa.ResourceMemory))
			Expect(model.Metrics[1].AverageUtilization).To(Equal(int32(80)))
			// 默认未开启 limits 基准
			Expect(model.ComputeByLimits).To(BeFalse())
		})

		It("should map ComputeByLimits when enabled", func() {
			input := serializer.UpsertGPAConfigInput{
				MinReplicas:     2,
				MaxReplicas:     10,
				Metrics:         []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 60}},
				ComputeByLimits: true,
			}
			model := input.ToModel("demo-app", "dev")
			Expect(model.ComputeByLimits).To(BeTrue())
		})

		It("should produce a model that passes domain validation and a name that is k8s-safe", func() {
			input := serializer.UpsertGPAConfigInput{
				MinReplicas: 1,
				MaxReplicas: 5,
				Metrics: []serializer.GPAMetricInput{
					{Resource: "cpu", AverageUtilization: 50},
				},
			}
			model := input.ToModel("demo-app", "dev")
			model.Name = model.GenerateName()

			Expect(model.Validate()).To(Succeed())
			Expect(model.Name).To(Equal("gpa-demo-app"))
		})

		It("should map time ranges for a schedule-only config", func() {
			disabled := false
			input := serializer.UpsertGPAConfigInput{
				MinReplicas: 1,
				MaxReplicas: 10,
				TimeRanges: []serializer.GPATimeRangeInput{
					{DesiredReplicas: 4, Schedule: "* 2-3 * * *", Remark: "凌晨扩容"},
					{DesiredReplicas: 6, Schedule: "* 4-5 * * *", Enabled: &disabled},
				},
			}
			model := input.ToModel("demo-app", "dev")
			Expect(model.Metrics).To(BeEmpty())
			Expect(model.TimeRanges).To(HaveLen(2))
			Expect(model.TimeRanges[0].DesiredReplicas).To(Equal(int32(4)))
			Expect(model.TimeRanges[0].Schedule).To(Equal("* 2-3 * * *"))
			// Enabled 不传时默认为 true
			Expect(model.TimeRanges[0].Enabled).To(BeTrue())
			Expect(model.TimeRanges[0].Remark).To(Equal("凌晨扩容"))
			Expect(model.TimeRanges[1].DesiredReplicas).To(Equal(int32(6)))
			// 显式传 false 时保持关闭
			Expect(model.TimeRanges[1].Enabled).To(BeFalse())

			model.Name = model.GenerateName()
			Expect(model.Validate()).To(Succeed())
		})
	})

	Describe("ToUpdateData", func() {
		It("should carry ComputeByLimits as a non-nil pointer reflecting the input", func() {
			input := serializer.UpsertGPAConfigInput{
				MinReplicas:     2,
				MaxReplicas:     10,
				Metrics:         []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 60}},
				ComputeByLimits: true,
			}
			data := input.ToUpdateData()
			Expect(data.ComputeByLimits).NotTo(BeNil())
			Expect(*data.ComputeByLimits).To(BeTrue())
		})

		It("should carry ComputeByLimits=false when not enabled", func() {
			input := serializer.UpsertGPAConfigInput{
				MinReplicas: 2,
				MaxReplicas: 10,
				Metrics:     []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 60}},
			}
			data := input.ToUpdateData()
			Expect(data.ComputeByLimits).NotTo(BeNil())
			Expect(*data.ComputeByLimits).To(BeFalse())
		})
	})

	DescribeTable(
		"validation",
		func(input serializer.UpsertGPAConfigInput, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid input", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics:     []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 60}},
		}, nil),
		Entry("minReplicas below 1", serializer.UpsertGPAConfigInput{
			MinReplicas: 0,
			MaxReplicas: 10,
			Metrics:     []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 60}},
		}, []string{"UpsertGPAConfigInput.MinReplicas"}),
		Entry("maxReplicas less than minReplicas", serializer.UpsertGPAConfigInput{
			MinReplicas: 5,
			MaxReplicas: 3,
			Metrics:     []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 60}},
		}, []string{"UpsertGPAConfigInput.MaxReplicas", "gtefield"}),
		Entry("metrics only is valid", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics:     []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 60}},
		}, nil),
		Entry("time ranges only is valid (no metrics)", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			TimeRanges:  []serializer.GPATimeRangeInput{{DesiredReplicas: 4, Schedule: "* 2-3 * * *"}},
		}, nil),
		Entry("metrics and time ranges together is valid", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics:     []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 60}},
			TimeRanges:  []serializer.GPATimeRangeInput{{DesiredReplicas: 4, Schedule: "* 2-3 * * *"}},
		}, nil),
		Entry("too many metrics", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics: []serializer.GPAMetricInput{
				{Resource: "cpu", AverageUtilization: 60},
				{Resource: "memory", AverageUtilization: 70},
				{Resource: "cpu", AverageUtilization: 80},
			},
		}, []string{"UpsertGPAConfigInput.Metrics", "max"}),
		Entry("invalid metric resource", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics:     []serializer.GPAMetricInput{{Resource: "disk", AverageUtilization: 60}},
		}, []string{"Resource", "oneof"}),
		Entry("averageUtilization above 100", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics:     []serializer.GPAMetricInput{{Resource: "cpu", AverageUtilization: 101}},
		}, []string{"AverageUtilization", "lte"}),
		Entry("time range missing desiredReplicas", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			TimeRanges:  []serializer.GPATimeRangeInput{{Schedule: "* 2-3 * * *"}},
		}, []string{"DesiredReplicas", "required"}),
		Entry("time range missing schedule", serializer.UpsertGPAConfigInput{
			MinReplicas: 2,
			MaxReplicas: 10,
			TimeRanges:  []serializer.GPATimeRangeInput{{DesiredReplicas: 4}},
		}, []string{"Schedule", "required"}),
	)
})

var _ = Describe("AppEnvURIInput", func() {
	DescribeTable(
		"validation",
		func(input serializer.AppEnvURIInput, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid app and env", serializer.AppEnvURIInput{
			AppID:   "demo-app",
			EnvName: "dev",
		}, nil),
		Entry("missing appID", serializer.AppEnvURIInput{
			EnvName: "dev",
		}, []string{"AppEnvURIInput.AppID", "required"}),
		Entry("missing envName", serializer.AppEnvURIInput{
			AppID: "demo-app",
		}, []string{"AppEnvURIInput.EnvName", "required"}),
		Entry("appID with illegal characters", serializer.AppEnvURIInput{
			AppID:   "demo app",
			EnvName: "dev",
		}, []string{"AppEnvURIInput.AppID", "uri_slug"}),
	)
})

var _ = Describe("GPAConfigOutputObj", func() {
	Describe("FromModel", func() {
		var config *gpa.GPAConfig

		BeforeEach(func() {
			config = &gpa.GPAConfig{
				Name:        "gpa-demo-app",
				AppID:       "demo-app",
				EnvName:     "dev",
				MinReplicas: 2,
				MaxReplicas: 10,
				Metrics: []gpa.GPAMetric{
					{Resource: gpa.ResourceCPU, AverageUtilization: 60},
				},
				Enabled:   true,
				CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
				UpdatedAt: time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
			}
		})

		It("should map config fields and format timestamps as RFC3339", func() {
			out := new(serializer.GPAConfigOutputObj).FromModel(config, nil)
			Expect(out.Name).To(Equal("gpa-demo-app"))
			Expect(out.AppID).To(Equal("demo-app"))
			Expect(out.EnvName).To(Equal("dev"))
			Expect(out.MinReplicas).To(Equal(int32(2)))
			Expect(out.MaxReplicas).To(Equal(int32(10)))
			Expect(out.Metrics).To(HaveLen(1))
			Expect(out.Metrics[0].Resource).To(Equal("cpu"))
			Expect(out.Metrics[0].AverageUtilization).To(Equal(int32(60)))
			Expect(out.Enabled).To(BeTrue())
			Expect(out.CreatedAt).To(Equal("2026-01-02T03:04:05Z"))
			Expect(out.UpdatedAt).To(Equal("2026-01-02T03:04:06Z"))
		})

		It("should map ComputeByLimits from the model", func() {
			config.ComputeByLimits = true
			out := new(serializer.GPAConfigOutputObj).FromModel(config, nil)
			Expect(out.ComputeByLimits).To(BeTrue())
		})

		It("should leave Status nil when the cluster CR status is absent", func() {
			out := new(serializer.GPAConfigOutputObj).FromModel(config, nil)
			Expect(out.Status).To(BeNil())
		})

		It("should populate Status when a cluster CR status is present", func() {
			status := &gpa.GPAStatus{
				Name:            "gpa-demo-app",
				CurrentReplicas: 3,
				DesiredReplicas: 5,
				LastScaleTime:   "2026-01-02T03:05:00Z",
			}
			out := new(serializer.GPAConfigOutputObj).FromModel(config, status)
			Expect(out.Status).NotTo(BeNil())
			Expect(out.Status.CurrentReplicas).To(Equal(int32(3)))
			Expect(out.Status.DesiredReplicas).To(Equal(int32(5)))
			Expect(out.Status.LastScaleTime).To(Equal("2026-01-02T03:05:00Z"))
		})

		It("should expose phase and statusMessage from the cluster CR status", func() {
			status := &gpa.GPAStatus{
				CurrentReplicas: 3,
				DesiredReplicas: 5,
				Phase:           "Paused",
				StatusMessage:   "no valid metric",
			}
			out := new(serializer.GPAConfigOutputObj).FromModel(config, status)
			Expect(out.Status).NotTo(BeNil())
			Expect(out.Status.Phase).To(Equal("Paused"))
			Expect(out.Status.StatusMessage).To(ContainSubstring("no valid metric"))
		})

		It("should map time ranges from the model", func() {
			config.TimeRanges = []gpa.GPATimeRange{
				{DesiredReplicas: 4, Schedule: "* 2-3 * * *", Enabled: true, Remark: "凌晨扩容"},
			}
			out := new(serializer.GPAConfigOutputObj).FromModel(config, nil)
			Expect(out.TimeRanges).To(HaveLen(1))
			Expect(out.TimeRanges[0].DesiredReplicas).To(Equal(int32(4)))
			Expect(out.TimeRanges[0].Schedule).To(Equal("* 2-3 * * *"))
			Expect(out.TimeRanges[0].Enabled).To(BeTrue())
			Expect(out.TimeRanges[0].Remark).To(Equal("凌晨扩容"))
		})
	})
})
