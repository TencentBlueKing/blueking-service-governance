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
	"fmt"

	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// ========== 指标定义 ==========

// MetricKey 指标标识常量
const (
	// MetricCPUUsage CPU 使用量（核数）
	MetricCPUUsage = "cpu_usage"
	// MetricCPURequestUsage CPU Request 使用率
	MetricCPURequestUsage = "cpu_request_usage"
	// MetricCPULimitUsage CPU Limit 使用率
	MetricCPULimitUsage = "cpu_limit_usage"

	// MetricMemoryUsage 内存使用量（Working Set）
	MetricMemoryUsage = "memory_usage"
	// MetricMemoryRequestUsage 内存 Request 使用率
	MetricMemoryRequestUsage = "memory_request_usage"
	// MetricMemoryLimitUsage 内存 Limit 使用率
	MetricMemoryLimitUsage = "memory_limit_usage"

	// MetricNetworkReceive 网络入带宽
	MetricNetworkReceive = "network_receive"
	// MetricNetworkTransmit 网络出带宽
	MetricNetworkTransmit = "network_transmit"

	// MetricDiskUsage 磁盘使用量
	MetricDiskUsage = "disk_usage"
)

// MetricDefinitions 有序的指标定义列表
var MetricDefinitions = []MetricDefinition{
	// CPU使用量（核数）
	{Key: MetricCPUUsage, DisplayName: "CPU Usage (Cores)", Unit: "cores"},
	// CPU Request使用率
	{Key: MetricCPURequestUsage, DisplayName: "CPU Request Usage", Unit: "percent"},
	// CPU Limit使用率
	{Key: MetricCPULimitUsage, DisplayName: "CPU Limit Usage", Unit: "percent"},
	// 内存使用量(Working Set)
	{Key: MetricMemoryUsage, DisplayName: "Memory Usage (Working Set)", Unit: "bytes"},
	// 内存 Request使用率
	{Key: MetricMemoryRequestUsage, DisplayName: "Memory Request Usage", Unit: "percent"},
	// 内存 Limit使用率
	{Key: MetricMemoryLimitUsage, DisplayName: "Memory Limit Usage", Unit: "percent"},
	// 网络入带宽
	{Key: MetricNetworkReceive, DisplayName: "Network Receive Bandwidth", Unit: "bytes/s"},
	// 网络出带宽
	{Key: MetricNetworkTransmit, DisplayName: "Network Transmit Bandwidth", Unit: "bytes/s"},
	// 磁盘使用量
	{Key: MetricDiskUsage, DisplayName: "Disk Usage", Unit: "bytes"},
}

// promqlBuilders 指标标识到 PromQL 构建函数的映射
var promqlBuilders = map[string]promqlBuilderFunc{
	MetricCPUUsage: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlCPUUsage, c, ns, pods)
	},
	MetricCPURequestUsage: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlCPURequestUsage, c, ns, pods, c, ns, pods)
	},
	MetricCPULimitUsage: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlCPULimitUsage, c, ns, pods, c, ns, pods)
	},
	MetricMemoryUsage: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlMemoryUsage, c, ns, pods)
	},
	MetricMemoryRequestUsage: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlMemoryRequestUsage, c, ns, pods, c, ns, pods)
	},
	MetricMemoryLimitUsage: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlMemoryLimitUsage, c, ns, pods, c, ns, pods)
	},
	MetricNetworkReceive: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlNetworkReceive, c, ns, pods)
	},
	MetricNetworkTransmit: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlNetworkTransmit, c, ns, pods)
	},
	MetricDiskUsage: func(c, ns, pods string) string {
		return fmt.Sprintf(promqlDiskUsage, c, ns, pods)
	},
}

// promqlBuilderFunc PromQL 构建函数类型，参数为 (clusterID, namespace, podsPattern)
type promqlBuilderFunc func(clusterID, namespace, podsPattern string) string

// MetricDefinition 指标定义信息
type MetricDefinition struct {
	// DisplayName 指标展示名称
	DisplayName string

	// Key 指标标识
	Key string

	// Unit 指标单位
	Unit string
}

// ========== 时序指标查询 ==========

// MetricsQuery 实例指标查询参数
type MetricsQuery struct {
	// BkBizID 蓝鲸监控业务 ID
	BkBizID int64
	// ClusterID BCS 集群 ID
	ClusterID string
	// Namespace 命名空间
	Namespace string
	// Instances 实例（Pod）名称列表
	Instances []string
	// MetricKeys 指定要查询的指标标识列表，为空时查询全部指标
	MetricKeys []string
	// StartTime 开始时间（Unix 时间戳）
	StartTime int64
	// EndTime 结束时间（Unix 时间戳）
	EndTime int64
	// Interval 汇聚周期（秒）
	Interval int64
	// Username 用于创建 bkmonitor 客户端的用户名
	Username string
}

// MetricsResult 实例指标查询结果（按指标分组）
type MetricsResult struct {
	// Metrics 指标标识 -> 时序数据
	Metrics map[string]*MetricTimeSeriesData
}

// MetricTimeSeriesData 单指标的时序数据
type MetricTimeSeriesData struct {
	// DisplayName 指标展示名称
	DisplayName string
	// Unit 指标单位
	Unit string
	// Series 各实例的时序数据
	Series []*TimeSeries
}

// TimeSeries 单实例的时序数据
type TimeSeries struct {
	// Instance 实例名称（Pod 名称）
	Instance string
	// DataPoints 数据点列表，[0] 为值，[1] 为毫秒级时间戳
	DataPoints [][2]float64
	// Stat 统计信息，包含 count、sum、min、max、avg、last
	Stat *bkmapi.TimeSeriesDataStat
}

// ========== APM 业务参数 ==========

// CreateApmInstParams 创建 APM 实例配置所需的参数
type CreateApmInstParams struct {
	WorkspaceID  string
	Username     string
	BkmProjectID int64
}

// ApmInstConfigUpdateData APM 实例配置的可更新字段
type ApmInstConfigUpdateData struct {
	WorkspaceID *string

	ApmID *int64

	Name *string

	Token *string
}
