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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	bkmmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
)

var _ = Describe("BkMonitor Serializer - Response Structs", func() {
	Describe("GetApmServiceNameResp", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": {
					"serviceName": "bkms-server-prod"
				}
			}`

			var resp serializer.GetApmServiceNameResp
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ServiceName).To(Equal("bkms-server-prod"))
		})

		It("should parse JSON with null data", func() {
			rawJSON := `{"data": null}`

			var resp serializer.GetApmServiceNameResp
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeNil())
		})
	})

	Describe("ListApmsResp", func() {
		It("should parse raw JSON with APM list", func() {
			rawJSON := `{
				"data": {
					"count": "2",
					"results": [
						{
							"apmID": "1001",
							"type": "custom",
							"bkBizID": "100001",
							"token": "token-abc-123",
							"name": "bkms-server-prod",
							"description": "test env APM",
							"creator": "admin",
							"createdAt": "2026-01-15T10:00:00Z",
							"metricReady": true,
							"traceReady": true,
							"logReady": false,
							"profilingReady": false,
							"associatedEnvs": [
								{"envID": "507f1f77bcf86cd799439011", "envName": "prod"}
							]
						},
						{
							"apmID": "1002",
							"type": "default",
							"bkBizID": "100001",
							"token": "token-def-456",
							"name": "bkms-server-stag",
							"description": "",
							"creator": "dev-user",
							"createdAt": null,
							"metricReady": false,
							"traceReady": false,
							"logReady": false,
							"profilingReady": false,
							"associatedEnvs": null
						}
					]
				}
			}`

			var resp serializer.ListApmsResp
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.Count).To(Equal(int64(2)))
			Expect(resp.Data.Results).To(HaveLen(2))

			Expect(resp.Data.Results[0].ApmID).To(Equal(int64(1001)))
			Expect(resp.Data.Results[0].Type).To(Equal("custom"))
			Expect(resp.Data.Results[0].Token).To(Equal("token-abc-123"))
			Expect(resp.Data.Results[0].Name).To(Equal("bkms-server-prod"))
			Expect(resp.Data.Results[0].MetricReady).To(BeTrue())
			Expect(resp.Data.Results[0].TraceReady).To(BeTrue())
			Expect(resp.Data.Results[0].AssociatedEnvs).To(HaveLen(1))
			Expect(resp.Data.Results[0].AssociatedEnvs[0].EnvName).To(Equal("prod"))

			Expect(resp.Data.Results[1].ApmID).To(Equal(int64(1002)))
			Expect(resp.Data.Results[1].MetricReady).To(BeFalse())
		})
	})

	Describe("GetEnvApmResp", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": {
					"apmID": "2001",
					"token": "token-xyz-789",
					"name": "my-env-apm"
				}
			}`

			var resp serializer.GetEnvApmResp
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ApmID).To(Equal(int64(2001)))
			Expect(resp.Data.Token).To(Equal("token-xyz-789"))
			Expect(resp.Data.Name).To(Equal("my-env-apm"))
		})
	})
})

var _ = Describe("BkMonitor Serializer", func() {
	Describe("GetInstanceTimeSeriesQueryInput", func() {
		var validate *validator.Validate

		BeforeEach(func() {
			validate = validator.New()
			validate.SetTagName("binding")
			// 注册 struct-level 校验
			validate.RegisterStructValidation(func(sl validator.StructLevel) {
				input := sl.Current().Interface().(serializer.GetInstanceTimeSeriesQueryInput)
				if input.StartTime > input.EndTime {
					sl.ReportError(input.EndTime, "EndTime", "EndTime",
						"start_time must be less than or equal to end_time", "")
				}
				if input.MetricKey != "" &&
					!lo.ContainsBy(bkmmodel.MetricDefinitions, func(def bkmmodel.MetricDefinition) bool {
						return def.Key == input.MetricKey
					}) {
					sl.ReportError(input.MetricKey, "MetricKey", "MetricKey",
						fmt.Sprintf("invalid metric key: %s", input.MetricKey), "")
				}
				hasNonEmpty := lo.ContainsBy(input.Instances, func(inst string) bool {
					return strings.TrimSpace(inst) != ""
				})
				if !hasNonEmpty {
					sl.ReportError(input.Instances, "Instances", "Instances",
						"instances must contain at least one non-empty value", "")
				}
			}, serializer.GetInstanceTimeSeriesQueryInput{})
		})

		Context("Struct-level validation", func() {
			DescribeTable(
				"should validate correctly",
				func(input serializer.GetInstanceTimeSeriesQueryInput, shouldFail bool) {
					err := validate.Struct(input)
					if shouldFail {
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err).NotTo(HaveOccurred())
					}
				},
				Entry("valid input with multiple instances", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a", "pod-b"},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, false),
				Entry("empty instances slice", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, true),
				Entry("nil instances", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: nil,
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, true),
				Entry("all instances are empty strings", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"", "  ", ""},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, true),
				Entry("start_time greater than end_time", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a"},
					MetricKey: "cpu_usage",
					StartTime: 1781452799,
					EndTime:   1781366400,
				}, true),
				Entry("invalid metric key", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a"},
					MetricKey: "invalid_key",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, true),
				Entry("valid single instance", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"single-pod"},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, false),
			)
		})

		Context("Normalize", func() {
			DescribeTable(
				"should normalize values correctly",
				func(input serializer.GetInstanceTimeSeriesQueryInput, expectedInstances []string, expectedInterval int64) {
					input.Normalize()
					Expect(input.Instances).To(Equal(expectedInstances))
					Expect(input.Interval).To(Equal(expectedInterval))
				},
				Entry("keeps valid instances unchanged", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a", "pod-b"},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, []string{"pod-a", "pod-b"}, int64(60)),
				Entry("filters empty string elements", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a", "", "pod-c"},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, []string{"pod-a", "pod-c"}, int64(60)),
				Entry("filters whitespace-only elements", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a", "  ", "pod-c"},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, []string{"pod-a", "pod-c"}, int64(60)),
				Entry("default interval when not specified", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a"},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
				}, []string{"pod-a"}, int64(60)),
				Entry("default interval when below minimum", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a"},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
					Interval:  30,
				}, []string{"pod-a"}, int64(60)),
				Entry("keeps specified interval when above minimum", serializer.GetInstanceTimeSeriesQueryInput{
					Instances: []string{"pod-a"},
					MetricKey: "cpu_usage",
					StartTime: 1781366400,
					EndTime:   1781452799,
					Interval:  120,
				}, []string{"pod-a"}, int64(120)),
			)
		})
	})

	Describe("InstanceTimeSeriesResp", func() {
		Context("JSON format", func() {
			It("should serialize to expected JSON structure", func() {
				resp := serializer.InstanceTimeSeriesResp{
					Data: map[string]*serializer.MetricTimeSeries{
						"cpu_usage": {
							DisplayName: "CPU Usage (Cores)",
							Unit:        "cores",
							Series: []*serializer.TimeSeriesItem{
								{
									Instance: "pod-a",
									DataPoints: [][2]float64{
										{1781366400, 23.5},
										{1781366460, 25.1},
										{1781366520, 22.8},
									},
								},
								{
									Instance: "pod-b",
									DataPoints: [][2]float64{
										{1781366400, 45.2},
										{1781366460, 47.8},
										{1781366520, 44.0},
									},
								},
							},
						},
						"memory_usage": {
							DisplayName: "Memory Usage (Working Set)",
							Unit:        "bytes",
							Series: []*serializer.TimeSeriesItem{
								{
									Instance: "pod-a",
									DataPoints: [][2]float64{
										{1781366400, 60.0},
										{1781366460, 61.5},
										{1781366520, 62.3},
									},
								},
							},
						},
					},
				}

				data, err := json.Marshal(resp)
				Expect(err).NotTo(HaveOccurred())

				// 反序列化验证结构完整性
				var decoded serializer.InstanceTimeSeriesResp
				err = json.Unmarshal(data, &decoded)
				Expect(err).NotTo(HaveOccurred())

				Expect(decoded.Data).To(HaveKey("cpu_usage"))
				Expect(decoded.Data).To(HaveKey("memory_usage"))

				cpuMetric := decoded.Data["cpu_usage"]
				Expect(cpuMetric.DisplayName).To(Equal("CPU Usage (Cores)"))
				Expect(cpuMetric.Unit).To(Equal("cores"))
				Expect(cpuMetric.Series).To(HaveLen(2))
				Expect(cpuMetric.Series[0].Instance).To(Equal("pod-a"))
				Expect(cpuMetric.Series[0].DataPoints).To(HaveLen(3))
				Expect(cpuMetric.Series[0].DataPoints[0]).To(Equal([2]float64{1781366400, 23.5}))

				memMetric := decoded.Data["memory_usage"]
				Expect(memMetric.DisplayName).To(Equal("Memory Usage (Working Set)"))
				Expect(memMetric.Unit).To(Equal("bytes"))
				Expect(memMetric.Series).To(HaveLen(1))
			})

			It("should handle empty data map", func() {
				resp := serializer.InstanceTimeSeriesResp{
					Data: map[string]*serializer.MetricTimeSeries{},
				}

				data, err := json.Marshal(resp)
				Expect(err).NotTo(HaveOccurred())

				var decoded serializer.InstanceTimeSeriesResp
				err = json.Unmarshal(data, &decoded)
				Expect(err).NotTo(HaveOccurred())
				Expect(decoded.Data).To(BeEmpty())
			})

			It("should handle metric with empty series", func() {
				resp := serializer.InstanceTimeSeriesResp{
					Data: map[string]*serializer.MetricTimeSeries{
						"network_io": {
							DisplayName: "Network Receive Bandwidth",
							Unit:        "bytes/s",
							Series:      []*serializer.TimeSeriesItem{},
						},
					},
				}

				data, err := json.Marshal(resp)
				Expect(err).NotTo(HaveOccurred())

				var decoded serializer.InstanceTimeSeriesResp
				err = json.Unmarshal(data, &decoded)
				Expect(err).NotTo(HaveOccurred())
				Expect(decoded.Data["network_io"].Series).To(BeEmpty())
			})
		})
	})
})
