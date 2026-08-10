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

package strategy

// builtinStrategyMetrics the map for built-in monitor metric bound to a strategy code.
var builtinStrategyMetrics = map[string]string{
	"cpu_request_usage_high":    "container_cpu_usage_seconds_total",
	"memory_request_usage_high": "container_memory_working_set_bytes",
	"pod_restart_frequent":      "kube_pod_container_status_restarts_total",
	"cpu_limit_usage_high":      "container_cpu_usage_seconds_total",
	"memory_limit_usage_high":   "container_memory_working_set_bytes",
}

// defaultTemplates 定义应用创建后自动初始化的默认告警模板。
// 这里只放“应用创建后自动初始化”的默认模板，不等于系统支持创建的全部策略类型。
var defaultTemplates = []DefaultTemplate{
	// CPU Limit 使用率高：用于发现接近 CPU 限流的 Pod。
	{
		StrategyCode:       "cpu_limit_usage_high",
		DisplayName:        "CPU 使用率过高",
		MonitorMetric:      "container_cpu_usage_seconds_total",
		Severity:           AlertSeverityWarning,
		Threshold:          ThresholdConfig{Method: "gte", Value: 80},
		TriggerCondition:   TriggerCondition{Count: 3, CheckWindow: 5},
		RecoverCondition:   RecoverCondition{CheckWindow: 5},
		EffectiveTimeRange: EffectiveTimeRange{StartTime: "00:00:00", EndTime: "23:59:59"},
	},
	// 内存 Limit 使用率高：用于发现接近 OOM Kill 的 Pod。
	{
		StrategyCode:       "memory_limit_usage_high",
		DisplayName:        "内存使用率过高",
		MonitorMetric:      "container_memory_working_set_bytes",
		Severity:           AlertSeverityWarning,
		Threshold:          ThresholdConfig{Method: "gte", Value: 80},
		TriggerCondition:   TriggerCondition{Count: 2, CheckWindow: 5},
		RecoverCondition:   RecoverCondition{CheckWindow: 5},
		EffectiveTimeRange: EffectiveTimeRange{StartTime: "00:00:00", EndTime: "23:59:59"},
	},
	// 容器异常重启：统计观察窗口内的重启增量，避免直接比较累计 counter。
	{
		StrategyCode:       "pod_restart_frequent",
		DisplayName:        "容器异常重启",
		MonitorMetric:      "kube_pod_container_status_restarts_total",
		Severity:           AlertSeverityWarning,
		Threshold:          ThresholdConfig{Method: "gte", Value: 3},
		TriggerCondition:   TriggerCondition{Count: 1, CheckWindow: 5},
		RecoverCondition:   RecoverCondition{CheckWindow: 10},
		EffectiveTimeRange: EffectiveTimeRange{StartTime: "00:00:00", EndTime: "23:59:59"},
	},
}

// MonitorMetricForStrategyCode returns the built-in monitor metric bound to a strategy code.
func MonitorMetricForStrategyCode(strategyCode string) string {
	return builtinStrategyMetrics[strategyCode]
}
