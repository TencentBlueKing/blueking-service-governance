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

import "testing"

func TestDefaultTemplatesOnlyContainThreeBuiltins(t *testing.T) {
	type expectedTemplate struct {
		displayName string
		severity    AlertSeverity
		threshold   float64
	}

	expected := map[string]expectedTemplate{
		"cpu_limit_usage_high": {
			displayName: "CPU 使用率过高",
			severity:    AlertSeverityWarning,
			threshold:   80,
		},
		"memory_limit_usage_high": {
			displayName: "内存使用率过高",
			severity:    AlertSeverityWarning,
			threshold:   80,
		},
		"pod_restart_frequent": {
			displayName: "容器异常重启",
			severity:    AlertSeverityWarning,
			threshold:   3,
		},
	}

	if len(defaultTemplates) != len(expected) {
		t.Fatalf("expected %d default templates, got %d", len(expected), len(defaultTemplates))
	}

	for _, tmpl := range defaultTemplates {
		want, ok := expected[tmpl.StrategyCode]
		if !ok {
			t.Fatalf("unexpected default template strategy code: %s", tmpl.StrategyCode)
		}
		if tmpl.DisplayName != want.displayName {
			t.Fatalf("expected %s displayName %s, got %s", tmpl.StrategyCode, want.displayName, tmpl.DisplayName)
		}
		if tmpl.Severity != want.severity {
			t.Fatalf("expected %s severity %v, got %v", tmpl.StrategyCode, want.severity, tmpl.Severity)
		}
		if tmpl.Threshold.Value != want.threshold {
			t.Fatalf("expected %s threshold %v, got %v", tmpl.StrategyCode, want.threshold, tmpl.Threshold.Value)
		}
		delete(expected, tmpl.StrategyCode)
	}

	if len(expected) != 0 {
		t.Fatalf("missing expected default templates: %v", expected)
	}
}

func TestMonitorMetricForStrategyCodeStillSupportsAllFiveBuiltins(t *testing.T) {
	cases := map[string]string{
		"cpu_request_usage_high":    "container_cpu_usage_seconds_total",
		"memory_request_usage_high": "container_memory_working_set_bytes",
		"cpu_limit_usage_high":      "container_cpu_usage_seconds_total",
		"memory_limit_usage_high":   "container_memory_working_set_bytes",
		"pod_restart_frequent":      "kube_pod_container_status_restarts_total",
	}

	for strategyCode, expectedMetric := range cases {
		if got := MonitorMetricForStrategyCode(strategyCode); got != expectedMetric {
			t.Fatalf("expected metric %s for %s, got %s", expectedMetric, strategyCode, got)
		}
	}
}
