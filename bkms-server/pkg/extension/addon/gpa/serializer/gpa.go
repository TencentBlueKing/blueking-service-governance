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

// Package serializer defines Gin input and output serializers for GPA config APIs.
package serializer

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
)

// -----------------------------------------------------------------------------
// Path inputs
// -----------------------------------------------------------------------------

// AppEnvURIInput is the path input for APIs scoped by application and environment.
type AppEnvURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// ToggleGPAConfigInput is the JSON input for toggling GPA enabled state.
type ToggleGPAConfigInput struct {
	// 是否启用 GPA。true 时下发 CR，false 时删除 CR
	Enabled bool `json:"enabled"`
}

// -----------------------------------------------------------------------------
// Shared outputs
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// -----------------------------------------------------------------------------
// Put GPA config
// -----------------------------------------------------------------------------

// UpsertGPAConfigInput is the JSON input for creating or updating a GPA config.
type UpsertGPAConfigInput struct {
	// 最小副本数（本期强制 >= 1）
	MinReplicas int32 `json:"minReplicas" binding:"required,gte=1"`
	// 最大副本数（须 >= minReplicas）
	MaxReplicas int32 `json:"maxReplicas" binding:"required,gtefield=MinReplicas"`
	// 指标模式扩缩容指标列表，最多 2 条（cpu、memory）。与 timeRanges 二者至少配置其一。
	Metrics []GPAMetricInput `json:"metrics" binding:"omitempty,max=2,dive"`
	// 定时模式扩缩容规则列表。与 metrics 二者至少配置其一。
	TimeRanges []GPATimeRangeInput `json:"timeRanges" binding:"omitempty,dive"`
	// 利用率计算基准开关：true 时该 GPA 下所有 Utilization 指标以 limits（而非默认 requests）为基准计算利用率。
	// 不传默认为 false（沿用 requests）
	ComputeByLimits bool `json:"computeByLimits" binding:"omitempty"`
}

// GPAMetricInput is the JSON input for a single GPA metric.
type GPAMetricInput struct {
	// 指标资源类型：cpu / memory
	Resource string `json:"resource" binding:"required,oneof=cpu memory"`
	// 平均使用率阈值（百分比），取值 1-100
	AverageUtilization int32 `json:"averageUtilization" binding:"required,gte=1,lte=100"`
}

// GPATimeRangeInput is the JSON input for a single GPA time range (scheduled scaling rule).
type GPATimeRangeInput struct {
	// 命中时间段时的期望副本数（>= 1）
	DesiredReplicas int32 `json:"desiredReplicas" binding:"required,gte=1"`
	// 标准 5 段 Crontab 表达式（分 时 日 月 周）。
	// 语法合法性由领域层校验。
	Schedule string `json:"schedule" binding:"required"`
	// 是否启用该定时规则。仅启用的规则会下发到底层 K8s CR。
	// 不传时默认为 true（启用）。
	Enabled *bool `json:"enabled" binding:"omitempty"`
	// 备注说明，仅用于展示，最长 256 字符。
	Remark string `json:"remark" binding:"omitempty,max=256"`
}

// ToModel converts input to a gpa.GPAConfig domain model.
// Name 由 store/handler 通过 GenerateName 生成，此处不设置。
func (i *UpsertGPAConfigInput) ToModel(appID, envName string) *gpa.GPAConfig {
	return &gpa.GPAConfig{
		AppID:           appID,
		EnvName:         envName,
		MinReplicas:     i.MinReplicas,
		MaxReplicas:     i.MaxReplicas,
		Metrics:         i.toMetrics(),
		TimeRanges:      i.toTimeRanges(),
		ComputeByLimits: i.ComputeByLimits,
	}
}

// ToUpdateData 将输入转换为整体替换用的 gpa.ConfigUpdateData。
// PUT 语义为全量覆盖，所有可变字段均非 nil（空切片表示清空对应模式）。
func (i *UpsertGPAConfigInput) ToUpdateData() *gpa.ConfigUpdateData {
	minReplicas := i.MinReplicas
	maxReplicas := i.MaxReplicas
	computeByLimits := i.ComputeByLimits
	return &gpa.ConfigUpdateData{
		MinReplicas:     &minReplicas,
		MaxReplicas:     &maxReplicas,
		Metrics:         i.toMetrics(),
		TimeRanges:      i.toTimeRanges(),
		ComputeByLimits: &computeByLimits,
	}
}

// toMetrics 将输入指标转换为领域模型；始终返回非 nil 切片以支持全量覆盖/清空。
func (i *UpsertGPAConfigInput) toMetrics() []gpa.GPAMetric {
	metrics := make([]gpa.GPAMetric, 0, len(i.Metrics))
	for _, m := range i.Metrics {
		metrics = append(metrics, gpa.GPAMetric{
			Resource:           gpa.ResourceName(m.Resource),
			AverageUtilization: m.AverageUtilization,
		})
	}
	return metrics
}

// toTimeRanges 将输入定时规则转换为领域模型；始终返回非 nil 切片以支持全量覆盖/清空。
// Enabled 不传时默认为 true（启用）。
func (i *UpsertGPAConfigInput) toTimeRanges() []gpa.GPATimeRange {
	ranges := make([]gpa.GPATimeRange, 0, len(i.TimeRanges))
	for _, r := range i.TimeRanges {
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		ranges = append(ranges, gpa.GPATimeRange{
			DesiredReplicas: r.DesiredReplicas,
			Schedule:        r.Schedule,
			Enabled:         enabled,
			Remark:          r.Remark,
		})
	}
	return ranges
}

// -----------------------------------------------------------------------------
// Get GPA config
// -----------------------------------------------------------------------------

// GetGPAConfigOutput is the JSON response for querying a GPA config with its K8s status.
type GetGPAConfigOutput struct {
	// GPA 配置（含运行状态）
	Data *GPAConfigOutputObj `json:"data"`
}

// GPAConfigOutputObj is the JSON representation of a GPA config with optional K8s status.
type GPAConfigOutputObj struct {
	// 配置名称（同 GPA CR 的 metadata.name）
	Name string `json:"name"`
	// 所属应用 ID
	AppID string `json:"appID"`
	// 生效环境名称
	EnvName string `json:"envName"`
	// 最小副本数
	MinReplicas int32 `json:"minReplicas"`
	// 最大副本数
	MaxReplicas int32 `json:"maxReplicas"`
	// 指标模式扩缩容指标列表
	Metrics []GPAMetricOutput `json:"metrics"`
	// 定时模式扩缩容规则列表
	TimeRanges []GPATimeRangeOutput `json:"timeRanges"`
	// 利用率计算基准开关：true 时以 limits 为基准计算利用率，false 时以 requests 为基准
	ComputeByLimits bool `json:"computeByLimits"`
	// 是否启用。false 时集群中不存在 GPA CR，对工作负载不生效
	Enabled bool `json:"enabled"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
	// K8s 运行状态，集群中 CR 不存在时为 nil
	Status *GPAStatusOutput `json:"status"`
}

// GPAMetricOutput is the JSON representation of a single GPA metric.
type GPAMetricOutput struct {
	// 指标资源类型：cpu / memory
	Resource string `json:"resource"`
	// 平均使用率阈值（百分比）
	AverageUtilization int32 `json:"averageUtilization"`
}

// GPATimeRangeOutput is the JSON representation of a single GPA time range.
type GPATimeRangeOutput struct {
	// 命中时间段时的期望副本数
	DesiredReplicas int32 `json:"desiredReplicas"`
	// 标准 5 段 Crontab 表达式
	Schedule string `json:"schedule"`
	// 是否启用。false 时该规则不下发到底层 K8s CR
	Enabled bool `json:"enabled"`
	// 备注说明
	Remark string `json:"remark"`
}

// GPAStatusOutput is the JSON representation of the GPA CR runtime status.
type GPAStatusOutput struct {
	// 当前副本数
	CurrentReplicas int32 `json:"currentReplicas"`
	// 期望副本数
	DesiredReplicas int32 `json:"desiredReplicas"`
	// 上次扩缩容时间（RFC3339 字符串，可能为空）
	LastScaleTime string `json:"lastScaleTime"`
	// Phase 提炼后的扩缩容健康状态枚举：
	//   Active       - 扩缩正常运作，副本数在 min/max 范围内
	//   Paused       - 指标获取失败或无效，扩缩被暂停
	//   Limited      - 扩缩逻辑正常但已触达 min/max 边界
	//   Failed       - 无法访问 scale 子资源（目标工作负载不存在、API 不可达、权限不足等）
	//   Initializing - CR 刚下发，controller 尚未写入 status.conditions，属正常过渡态，稍候即会转为其他状态
	//   Unknown      - conditions 存在但关键 condition 无法解析（旧版本 GPA 或异常状态）
	Phase string `json:"phase"`
	// StatusMessage 汇总所有非 True condition 的 message，用 "; " 连接
	// 所有 condition 均为 True 时为空字符串
	StatusMessage string `json:"statusMessage"`
}

// FromModel fills output fields from a GPAConfig domain model and an optional K8s status.
// status 为 nil 时（集群中 CR 不存在）输出的 Status 字段也为 nil。
func (o *GPAConfigOutputObj) FromModel(config *gpa.GPAConfig, status *gpa.GPAStatus) *GPAConfigOutputObj {
	metrics := make([]GPAMetricOutput, 0, len(config.Metrics))
	for _, m := range config.Metrics {
		metrics = append(metrics, GPAMetricOutput{
			Resource:           string(m.Resource),
			AverageUtilization: m.AverageUtilization,
		})
	}
	timeRanges := make([]GPATimeRangeOutput, 0, len(config.TimeRanges))
	for _, r := range config.TimeRanges {
		timeRanges = append(timeRanges, GPATimeRangeOutput{
			DesiredReplicas: r.DesiredReplicas,
			Schedule:        r.Schedule,
			Enabled:         r.Enabled,
			Remark:          r.Remark,
		})
	}
	*o = GPAConfigOutputObj{
		Name:            config.Name,
		AppID:           config.AppID,
		EnvName:         config.EnvName,
		MinReplicas:     config.MinReplicas,
		MaxReplicas:     config.MaxReplicas,
		Metrics:         metrics,
		TimeRanges:      timeRanges,
		ComputeByLimits: config.ComputeByLimits,
		Enabled:         config.Enabled,
		CreatedAt:       config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       config.UpdatedAt.Format(time.RFC3339),
	}
	if status != nil {
		o.Status = &GPAStatusOutput{
			CurrentReplicas: status.CurrentReplicas,
			DesiredReplicas: status.DesiredReplicas,
			LastScaleTime:   status.LastScaleTime,
			Phase:           status.Phase,
			StatusMessage:   status.StatusMessage,
		}
	}
	return o
}
