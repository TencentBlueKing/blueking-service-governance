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

package bkmonitor_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
)

var _ = Describe("PromQLBuilder", func() {
	var builder *bkmonitor.PromQLBuilder

	BeforeEach(func() {
		builder = &bkmonitor.PromQLBuilder{
			ClusterID: "BCS-K8S-xxxxx",
			Namespace: "test",
			Pods:      []string{"test-pod-x32rf", "test-pod-rwztv"},
		}
	})

	Describe("Build", func() {
		It("should return 9 metric PromQL expressions", func() {
			result := builder.Build()
			Expect(result).To(HaveLen(9))
			Expect(result).To(HaveKey(bkmonitor.MetricCPUUsage))
			Expect(result).To(HaveKey(bkmonitor.MetricCPURequestUsage))
			Expect(result).To(HaveKey(bkmonitor.MetricCPULimitUsage))
			Expect(result).To(HaveKey(bkmonitor.MetricMemoryUsage))
			Expect(result).To(HaveKey(bkmonitor.MetricMemoryRequestUsage))
			Expect(result).To(HaveKey(bkmonitor.MetricMemoryLimitUsage))
			Expect(result).To(HaveKey(bkmonitor.MetricNetworkReceive))
			Expect(result).To(HaveKey(bkmonitor.MetricNetworkTransmit))
			Expect(result).To(HaveKey(bkmonitor.MetricDiskUsage))
		})

		It("CPU usage PromQL should contain correct cluster, namespace and pod regex", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricCPUUsage]
			Expect(promql).To(ContainSubstring(`bcs_cluster_id="BCS-K8S-xxxxx"`))
			Expect(promql).To(ContainSubstring(`namespace=~"^(test)$"`))
			Expect(promql).To(ContainSubstring(`pod_name=~"^(test-pod-x32rf|test-pod-rwztv)$"`))
			Expect(promql).To(ContainSubstring(`container_name!="POD"`))
			Expect(promql).To(ContainSubstring(`rate(`))
			Expect(promql).To(ContainSubstring(`[1m]`))
			Expect(promql).To(ContainSubstring(`sum by(pod)`))
		})

		It("CPU request usage PromQL should contain division expression", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricCPURequestUsage]
			Expect(promql).To(ContainSubstring(`container_cpu_usage_seconds_total`))
			Expect(promql).To(ContainSubstring(`kube_pod_container_resource_requests_cpu_cores`))
			Expect(promql).To(ContainSubstring(` / `))
		})

		It("CPU limit usage PromQL should contain division expression", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricCPULimitUsage]
			Expect(promql).To(ContainSubstring(`container_cpu_usage_seconds_total`))
			Expect(promql).To(ContainSubstring(`kube_pod_container_resource_limits_cpu_cores`))
			Expect(promql).To(ContainSubstring(` / `))
		})

		It("memory usage PromQL should contain container_memory_working_set_bytes", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricMemoryUsage]
			Expect(promql).To(ContainSubstring(`container_memory_working_set_bytes`))
			Expect(promql).To(ContainSubstring(`sum by(pod)`))
			Expect(promql).NotTo(ContainSubstring(`rate(`))
		})

		It("memory request usage PromQL should contain division expression", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricMemoryRequestUsage]
			Expect(promql).To(ContainSubstring(`container_memory_working_set_bytes`))
			Expect(promql).To(ContainSubstring(`kube_pod_container_resource_requests_memory_bytes`))
			Expect(promql).To(ContainSubstring(` / `))
		})

		It("memory limit usage PromQL should contain division expression", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricMemoryLimitUsage]
			Expect(promql).To(ContainSubstring(`container_memory_working_set_bytes`))
			Expect(promql).To(ContainSubstring(`kube_pod_container_resource_limits_memory_bytes`))
			Expect(promql).To(ContainSubstring(` / `))
		})

		It("network receive PromQL should contain container_network_receive_bytes_total", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricNetworkReceive]
			Expect(promql).To(ContainSubstring(`container_network_receive_bytes_total`))
			Expect(promql).To(ContainSubstring(`rate(`))
			Expect(promql).To(ContainSubstring(`[1m]`))
			// 网络指标不包含 container_name 过滤
			Expect(promql).NotTo(ContainSubstring(`container_name`))
		})

		It("network transmit PromQL should contain container_network_transmit_bytes_total", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricNetworkTransmit]
			Expect(promql).To(ContainSubstring(`container_network_transmit_bytes_total`))
			Expect(promql).To(ContainSubstring(`rate(`))
			Expect(promql).To(ContainSubstring(`[1m]`))
		})

		It("disk usage PromQL should contain bkmonitor:container_fs_usage_bytes and group by pod", func() {
			result := builder.Build()
			promql := result[bkmonitor.MetricDiskUsage]
			Expect(promql).To(ContainSubstring(`bkmonitor:container_fs_usage_bytes`))
			Expect(promql).To(ContainSubstring(`sum by(pod)`))
			// 磁盘指标的 namespace 使用精确匹配而非正则
			Expect(promql).To(ContainSubstring(`namespace="test"`))
		})

		It("should not contain | separator when single pod", func() {
			builder.Pods = []string{"single-pod"}
			result := builder.Build()
			promql := result[bkmonitor.MetricCPUUsage]
			Expect(promql).To(ContainSubstring(`pod_name=~"^(single-pod)$"`))
			Expect(promql).NotTo(ContainSubstring("|"))
		})

		It("should join multiple pods with | separator", func() {
			builder.Pods = []string{"pod-a", "pod-b", "pod-c"}
			result := builder.Build()
			promql := result[bkmonitor.MetricCPUUsage]
			Expect(promql).To(ContainSubstring(`pod_name=~"^(pod-a|pod-b|pod-c)$"`))
		})
	})

	Describe("Build with aggregation method", func() {
		It("should default to sum when method is empty", func() {
			for key, promql := range builder.Build() {
				Expect(promql).To(HavePrefix("sum by(pod)("), key)
			}
		})

		DescribeTable("should apply the method to every metric",
			func(method bkmonitor.AggMethod) {
				builder.Method = method
				for key, promql := range builder.Build() {
					Expect(promql).To(HavePrefix(string(method)+" by(pod)("), key)
					// 模板中不能残留未替换的占位或默认的 sum
					Expect(promql).NotTo(ContainSubstring("%!"), key)
					Expect(promql).NotTo(ContainSubstring("%["), key)
					if method != bkmonitor.AggMethodSum {
						Expect(promql).NotTo(ContainSubstring("sum by(pod)"), key)
					}
				}
			},
			Entry("sum", bkmonitor.AggMethodSum),
			Entry("max", bkmonitor.AggMethodMax),
			Entry("min", bkmonitor.AggMethodMin),
			Entry("count", bkmonitor.AggMethodCount),
			Entry("avg", bkmonitor.AggMethodAvg),
		)

		It("should use the same method on both numerator and denominator of ratio metrics", func() {
			builder.Method = bkmonitor.AggMethodMax
			result := builder.Build()
			for _, key := range []string{
				bkmonitor.MetricCPURequestUsage,
				bkmonitor.MetricCPULimitUsage,
				bkmonitor.MetricMemoryRequestUsage,
				bkmonitor.MetricMemoryLimitUsage,
			} {
				numerator, denominator, found := strings.Cut(result[key], " / ")
				Expect(found).To(BeTrue(), key)
				Expect(numerator).To(HavePrefix("max by(pod)("), key)
				Expect(denominator).To(HavePrefix("max by(pod)("), key)
				// 分母同样带上集群、命名空间与 Pod 过滤
				Expect(denominator).To(ContainSubstring(`bcs_cluster_id="BCS-K8S-xxxxx"`), key)
				Expect(denominator).To(ContainSubstring(`namespace=~"^(test)$"`), key)
				Expect(denominator).To(ContainSubstring(`pod_name=~"^(test-pod-x32rf|test-pod-rwztv)$"`), key)
			}
		})

		It("should take the method from MetricsQuery", func() {
			b := bkmonitor.NewPromQLBuilder(&bkmonitor.MetricsQuery{
				ClusterID:  "BCS-K8S-xxxxx",
				Namespace:  "test",
				Instances:  []string{"pod-a"},
				MetricKeys: []string{bkmonitor.MetricMemoryUsage},
				Method:     bkmonitor.AggMethodAvg,
			})
			Expect(b.Build()[bkmonitor.MetricMemoryUsage]).To(Equal(
				`avg by(pod)(container_memory_working_set_bytes{bcs_cluster_id="BCS-K8S-xxxxx",` +
					`namespace=~"^(test)$",pod_name=~"^(pod-a)$",container_name!="POD"})`,
			))
		})
	})
})
