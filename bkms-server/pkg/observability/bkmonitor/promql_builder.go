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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"strings"

	"github.com/samber/lo"
)

// PromQL 模板
const (
	// CPU 相关
	// container_cpu_usage_seconds_total 以秒为单位消耗的累积 CPU 时间
	// kube_pod_container_resource_requests_cpu_cores Pod 容器 CPU Request 配额
	// kube_pod_container_resource_limits_cpu_cores Pod 容器 CPU Limit 配额
	promqlCPUUsage = `sum by(pod)(rate(container_cpu_usage_seconds_total{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"}[1m]))` //nolint:lll

	promqlCPURequestUsage = `sum by(pod)(rate(container_cpu_usage_seconds_total{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"}[1m])) / sum by(pod)(kube_pod_container_resource_requests_cpu_cores{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"})` //nolint:lll

	promqlCPULimitUsage = `sum by(pod)(rate(container_cpu_usage_seconds_total{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"}[1m])) / sum by(pod)(kube_pod_container_resource_limits_cpu_cores{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"})` //nolint:lll

	// 内存相关
	// container_memory_working_set_bytes 当前工作集使用量；RSS + Active Cache（不含 inactive file cache）；OOM 判定依据；
	// kube_pod_container_resource_requests_memory_bytes Pod 容器内存 Request 配额
	// kube_pod_container_resource_limits_memory_bytes Pod 容器内存 Limit 配额
	promqlMemoryUsage = `sum by(pod)(container_memory_working_set_bytes{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"})` //nolint:lll

	promqlMemoryRequestUsage = `sum by(pod)(container_memory_working_set_bytes{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"}) / sum by(pod)(kube_pod_container_resource_requests_memory_bytes{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"})` //nolint:lll

	promqlMemoryLimitUsage = `sum by(pod)(container_memory_working_set_bytes{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"}) / sum by(pod)(kube_pod_container_resource_limits_memory_bytes{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$",container_name!="POD"})` //nolint:lll

	// 网络相关
	// container_network_receive_bytes_total 容器累计接收的网络字节数（Counter）
	// container_network_transmit_bytes_total 容器累计发送的网络字节数（Counter）
	promqlNetworkReceive  = `sum by(pod)(rate(container_network_receive_bytes_total{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$"}[1m]))`  //nolint:lll
	promqlNetworkTransmit = `sum by(pod)(rate(container_network_transmit_bytes_total{bcs_cluster_id="%s",namespace=~"^(%s)$",pod_name=~"^(%s)$"}[1m]))` //nolint:lll

	// 存储相关
	// bkmonitor:container_fs_usage_bytes 容器文件系统使用量（不含镜像只读层）
	promqlDiskUsage = `sum by(pod)(bkmonitor:container_fs_usage_bytes{bcs_cluster_id="%s",namespace="%s",pod_name=~"^(%s)$"})` //nolint:lll
)

// ========== PromQL 构建 ==========

// PromQLBuilder PromQL 模板构建器
type PromQLBuilder struct {
	// ClusterID BCS 集群 ID
	ClusterID string
	// Namespace 命名空间
	Namespace string
	// Pods 实例（Pod）名称列表
	Pods []string
	// MetricKeys 指定要构建的指标标识列表，为空时构建全部指标
	MetricKeys []string
}

// NewPromQLBuilder 创建一个新的 PromQLBuilder 实例
func NewPromQLBuilder(query *MetricsQuery) *PromQLBuilder {
	return &PromQLBuilder{
		ClusterID:  query.ClusterID,
		Namespace:  query.Namespace,
		Pods:       query.Instances,
		MetricKeys: query.MetricKeys,
	}
}

// Build 构建指定指标的 PromQL 查询语句，返回指标标识到 PromQL 的映射。
// 如果 MetricKeys 为空，则构建所有指标的 PromQL；否则仅按需构建指定指标。
func (b *PromQLBuilder) Build() map[string]string {
	podsPattern := strings.Join(b.Pods, "|")

	// 确定要构建的指标 key 列表
	keys := b.MetricKeys
	if len(keys) == 0 {
		keys = lo.Keys(promqlBuilders)
	}

	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if builder, ok := promqlBuilders[key]; ok {
			result[key] = builder(b.ClusterID, b.Namespace, podsPattern)
		}
	}

	return result
}
