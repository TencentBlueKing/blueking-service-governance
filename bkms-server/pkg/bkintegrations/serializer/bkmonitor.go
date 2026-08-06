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

package serializer

import (
	"strings"
	"time"

	bkmmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// DefaultInterval 默认汇聚周期（秒）
const DefaultInterval int64 = 60

// --- BkMonitor URI 参数 ---

// AppEnvURIInput 路径参数
type AppEnvURIInput struct {
	AppID   string `uri:"appID" binding:"required,app_id,min=2"`
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// EnvURIInput 路径参数
type EnvURIInput struct {
	EnvID string `uri:"envID" binding:"required,min=1"`
}

// EnvApmURIInput 路径参数
type EnvApmURIInput struct {
	EnvID string `uri:"envID" binding:"required,min=1"`
	ApmID string `uri:"apmID" binding:"required,min=1"`
}

// WorkspaceURIInput 路径参数
type WorkspaceURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,workspace_id,min=1,max=27"`
}

// --- BkMonitor Output ---

// GetApmServiceNameOutput Apm 服务名称输出
type GetApmServiceNameOutput struct {
	ServiceName string `json:"serviceName"`
}

// GetApmServiceNameResp 获取 Apm 服务名称的响应
type GetApmServiceNameResp struct {
	Data *GetApmServiceNameOutput `json:"data"`
}

// ApmOutput APM 输出
type ApmOutput struct {
	ApmID          int64               `json:"apmID,string"`
	Type           string              `json:"type"`
	BkBizID        int64               `json:"bkBizID,string"`
	Token          string              `json:"token"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Creator        string              `json:"creator"`
	CreatedAt      *time.Time          `json:"createdAt"`
	MetricReady    bool                `json:"metricReady"`
	TraceReady     bool                `json:"traceReady"`
	LogReady       bool                `json:"logReady"`
	ProfilingReady bool                `json:"profilingReady"`
	AssociatedEnvs []*ApmEnvInfoOutput `json:"associatedEnvs"`
}

// ApmEnvInfoOutput APM 关联环境信息输出
type ApmEnvInfoOutput struct {
	EnvID   string `json:"envID"`
	EnvName string `json:"envName"`
}

// FromModel 从领域模型填充输出字段
func (o *ApmEnvInfoOutput) FromModel(info bkmmodel.EnvInfo) *ApmEnvInfoOutput {
	if o == nil {
		return nil
	}
	*o = ApmEnvInfoOutput{
		EnvID:   info.EnvID.Hex(),
		EnvName: info.EnvName,
	}
	return o
}

// ListApmOutput APM 列表输出
type ListApmOutput struct {
	Count   int64        `json:"count,string"`
	Results []*ApmOutput `json:"results"`
}

// ListApmsResp 获取 APM 列表的响应
type ListApmsResp struct {
	Data *ListApmOutput `json:"data"`
}

// GetEnvApmOutput 环境绑定的 APM 输出
type GetEnvApmOutput struct {
	ApmID int64  `json:"apmID,string"`
	Token string `json:"token"`
	Name  string `json:"name"`
}

// FromModel 从领域模型填充输出字段
func (o *GetEnvApmOutput) FromModel(apm bkmmodel.ApmInstConfig) *GetEnvApmOutput {
	if o == nil {
		return nil
	}
	*o = GetEnvApmOutput{
		ApmID: apm.ApmID,
		Token: apm.Token,
		Name:  apm.Name,
	}
	return o
}

// GetEnvApmResp 查询环境绑定的 APM 的响应
type GetEnvApmResp struct {
	Data *GetEnvApmOutput `json:"data"`
}

// CreateEnvApmResp 为环境创建 APM 的响应
type CreateEnvApmResp struct {
	Data *ApmOutput `json:"data"`
}

// --- 实例时序指标查询 ---

// GetInstanceTimeSeriesQueryInput 实例时序指标查询的 Query 参数
// 返回实例 内存、cpu、io、network 数据
type GetInstanceTimeSeriesQueryInput struct {
	// Instances 实例（Pod）名称列表
	Instances []string `form:"instances" binding:"required,min=1"`
	// MetricKey 指定要查询的指标标识，可用值：
	//   - cpu_usage: CPU 使用量（核数）
	//   - cpu_request_usage: CPU Request 使用率
	//   - cpu_limit_usage: CPU Limit 使用率
	//   - memory_usage: 内存使用量（Working Set）
	//   - memory_request_usage: 内存 Request 使用率
	//   - memory_limit_usage: 内存 Limit 使用率
	//   - network_receive: 网络入带宽
	//   - network_transmit: 网络出带宽
	//   - disk_usage: 磁盘使用量
	MetricKey string `form:"metricKey" binding:"required"`
	// Interval 汇聚周期（秒），最小为 60 秒
	Interval int64 `form:"interval"`
	// StartTime 开始时间（Unix 时间戳，秒）
	StartTime int64 `form:"startTime" binding:"required,min=1"`
	// EndTime 结束时间（Unix 时间戳，秒）
	EndTime int64 `form:"endTime" binding:"required,min=1"`
}

// Normalize 对输入进行值修正（过滤空白实例名称、设置默认汇聚周期）
func (r *GetInstanceTimeSeriesQueryInput) Normalize() {
	// 过滤空白实例名称
	instances := make([]string, 0, len(r.Instances))
	for _, inst := range r.Instances {
		if v := strings.TrimSpace(inst); v != "" {
			instances = append(instances, v)
		}
	}
	r.Instances = instances
	// 汇聚周期不能小于 60 秒，太小的话，会产生很多数据。
	if r.Interval < DefaultInterval {
		r.Interval = DefaultInterval
	}
}

// TimeSeriesItemStat 时序数据统计信息
type TimeSeriesItemStat struct {
	// Count 数据点计数
	// [0] 为时间戳，[1] 为值
	Count [2]float64 `json:"count"`
	// Sum 数据点求和
	Sum [2]float64 `json:"sum"`
	// Min 最小值
	Min [2]float64 `json:"min"`
	// Max 最大值
	Max [2]float64 `json:"max"`
	// Avg 平均值
	Avg [2]float64 `json:"avg"`
	// Last 最后一个数据点
	Last [2]float64 `json:"last"`
}

// TimeSeriesItem 单实例的时序数据
type TimeSeriesItem struct {
	// Instance 实例（Pod）名称
	Instance string `json:"instance"`

	// DataPoints 时序数据点列表，每个元素为 [时间戳, 值]
	DataPoints [][2]float64 `json:"dataPoints"`

	// Stat 统计信息，包含 count、sum、min、max、avg、last
	Stat *TimeSeriesItemStat `json:"stat,omitempty"`
}

// MetricTimeSeries 单指标的时序数据（含多个实例）
type MetricTimeSeries struct {
	// DisplayName 指标展示名称
	DisplayName string `json:"displayName"`

	// Unit 指标单位
	Unit string `json:"unit"`

	// Series 各实例的时序数据列表
	Series []*TimeSeriesItem `json:"series"`
}

// InstanceTimeSeriesResp 实例时序指标查询的响应
type InstanceTimeSeriesResp struct {
	// Data 指标名称 -> 时序数据的映射
	Data map[string]*MetricTimeSeries `json:"data"`
}
