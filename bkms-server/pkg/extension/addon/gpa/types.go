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

const (
	// ClusterAddonName GPA 集群组件名称
	ClusterAddonName = "general-pod-autoscaler"

	// MinReplicasLowerBound minReplicas 的下限
	MinReplicasLowerBound int32 = 1
	// UtilizationLowerBound 使用率阈值下限（百分比）
	UtilizationLowerBound int32 = 1
	// UtilizationUpperBound 使用率阈值上限（百分比）
	UtilizationUpperBound int32 = 100
	// MaxMetricsCount 指标项最大数量（cpu、memory 各一项）
	MaxMetricsCount = 2
)

// ConfigUpdateData 定义了更新 GPAConfig 时允许修改的数据，nil 字段表示不更新。
type ConfigUpdateData struct {
	// MinReplicas 最小副本数
	MinReplicas *int32
	// MaxReplicas 最大副本数
	MaxReplicas *int32
	// Metrics 指标模式扩缩容指标列表，非 nil 时整体替换（空切片表示清空）
	Metrics []GPAMetric
	// TimeRanges 定时模式扩缩容规则列表，非 nil 时整体替换（空切片表示清空）
	TimeRanges []GPATimeRange
	// ComputeByLimits 利用率计算基准开关，非 nil 时更新
	ComputeByLimits *bool
	// Enabled 开关状态，非 nil 时更新
	Enabled *bool
}

// GPAStatus 从集群中 GeneralPodAutoscaler CR 解析出的运行状态视图。
// 仅用于状态回查，不持久化到 DB。
type GPAStatus struct {
	// Name CR 名称
	Name string
	// AppID 所属应用 ID（从 CR label 解析）
	AppID string
	// WorkspaceID 所属工作空间 ID（从 CR label 解析）
	WorkspaceID string
	// EnvName 所属环境名称（从 CR label 解析）
	EnvName string
	// CurrentReplicas 当前副本数
	CurrentReplicas int32
	// DesiredReplicas 期望副本数
	DesiredReplicas int32
	// LastScaleTime 上次扩缩容时间（RFC3339 字符串，可能为空）
	LastScaleTime string
	// Phase 提炼后的扩缩容健康状态枚举：
	//   Active       - 扩缩正常运作，副本数在 min/max 范围内
	//   Paused       - 指标获取失败或无效，扩缩被暂停
	//   Limited      - 扩缩逻辑正常但已触达 min/max 边界
	//   Failed       - 无法访问 scale 子资源（目标工作负载不存在、API 不可达、权限不足等）
	//   Initializing - CR 刚下发，controller 尚未写入 status.conditions，属正常过渡态，稍候即会转为其他状态
	//   Unknown      - conditions 存在但关键 condition 无法解析（旧版本 GPA 或异常状态）
	Phase string
	// StatusMessage 汇总所有非 True condition 的 message，用 "; " 连接
	// 所有 condition 均为 True 时为空字符串
	StatusMessage string
}
