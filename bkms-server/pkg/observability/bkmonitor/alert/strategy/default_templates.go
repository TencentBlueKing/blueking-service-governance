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

// defaultTemplates 定义应用创建后自动初始化的默认告警模板。
// 这些模板通过 StrategyCode 与 buildAppScopedAlertPromQL() 的分支逻辑一一对应。
var defaultTemplates = []DefaultTemplate{
	// CPU Request 使用率高：关注容器实际 CPU 使用量相对 request 的比例。
	{
		StrategyCode:       "cpu_request_usage_high",
		DisplayName:        "CPU Request 使用率过高",
		MonitorMetric:      "container_cpu_usage_seconds_total",
		Severity:           AlertSeverityWarning,
		Threshold:          ThresholdConfig{Method: "gte", Value: 80},
		TriggerCondition:   TriggerCondition{Count: 3, CheckWindow: 5},
		RecoverCondition:   RecoverCondition{CheckWindow: 5},
		EffectiveTimeRange: EffectiveTimeRange{StartTime: "00:00:00", EndTime: "23:59:59"},
	},
	// 内存 Request 使用率高：关注 working set 相对 request 的比例。
	{
		StrategyCode:       "memory_request_usage_high",
		DisplayName:        "内存 Request 使用率过高",
		MonitorMetric:      "container_memory_working_set_bytes",
		Severity:           AlertSeverityWarning,
		Threshold:          ThresholdConfig{Method: "gte", Value: 80},
		TriggerCondition:   TriggerCondition{Count: 3, CheckWindow: 5},
		RecoverCondition:   RecoverCondition{CheckWindow: 5},
		EffectiveTimeRange: EffectiveTimeRange{StartTime: "00:00:00", EndTime: "23:59:59"},
	},
	// Pod 频繁重启：统计观察窗口内的重启增量，避免直接比较累计 counter。
	{
		StrategyCode:       "pod_restart_frequent",
		DisplayName:        "Pod 频繁重启",
		MonitorMetric:      "kube_pod_container_status_restarts_total",
		Severity:           AlertSeverityFatal,
		Threshold:          ThresholdConfig{Method: "gte", Value: 3},
		TriggerCondition:   TriggerCondition{Count: 1, CheckWindow: 5},
		RecoverCondition:   RecoverCondition{CheckWindow: 10},
		EffectiveTimeRange: EffectiveTimeRange{StartTime: "00:00:00", EndTime: "23:59:59"},
	},
	// CPU Limit 使用率高：用于发现接近 CPU 限流的 Pod。
	{
		StrategyCode:       "cpu_limit_usage_high",
		DisplayName:        "CPU Limit 使用率过高（即将被限流）",
		MonitorMetric:      "container_cpu_usage_seconds_total",
		Severity:           AlertSeverityFatal,
		Threshold:          ThresholdConfig{Method: "gte", Value: 90},
		TriggerCondition:   TriggerCondition{Count: 3, CheckWindow: 5},
		RecoverCondition:   RecoverCondition{CheckWindow: 5},
		EffectiveTimeRange: EffectiveTimeRange{StartTime: "00:00:00", EndTime: "23:59:59"},
	},
	// 内存 Limit 使用率高：用于发现接近 OOM Kill 的 Pod。
	{
		StrategyCode:       "memory_limit_usage_high",
		DisplayName:        "内存 Limit 使用率过高（即将被 OOM Kill）",
		MonitorMetric:      "container_memory_working_set_bytes",
		Severity:           AlertSeverityFatal,
		Threshold:          ThresholdConfig{Method: "gte", Value: 90},
		TriggerCondition:   TriggerCondition{Count: 2, CheckWindow: 5},
		RecoverCondition:   RecoverCondition{CheckWindow: 5},
		EffectiveTimeRange: EffectiveTimeRange{StartTime: "00:00:00", EndTime: "23:59:59"},
	},
}
