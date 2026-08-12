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

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/samber/lo"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// appScopedAlertGroupBy 将指标聚合到“单个 Pod 实例”维度，避免不同 Pod 的值被合并后失真。
const appScopedAlertGroupBy = "pod_name, bcs_cluster_id, namespace"

const (
	defaultEffectiveStartTime = "00:00:00"
	defaultEffectiveEndTime   = "23:59:59"
)

// buildSaveAlarmStrategyReq 将本地告警策略转换为蓝鲸监控 save_alarm_strategy_v3 请求体。
func (s *Service) buildSaveAlarmStrategyReq(
	strategy *AlertStrategy,
	targets []remoteTargetContext,
	monitorProjectID, remoteStrategyID int64,
	strategyName, operator string,
) *bkmapi.SaveAlarmStrategyReq {
	// 1. 先构造所有 target 共享的触发/恢复/阈值算法配置；
	//    在 1:1 模型下，不同 env/lane 的差异主要体现在 item 级 query_configs。
	triggerConfig := map[string]any{
		"count":        strategy.TriggerCondition.Count,
		"check_window": strategy.TriggerCondition.CheckWindow,
	}
	recoverConfig := map[string]any{"check_window": strategy.RecoverCondition.CheckWindow}
	algorithms := []map[string]any{{
		"level": int(strategy.Severity),
		"type":  "Threshold",
		"config": [][]map[string]any{{{
			"method":    strategy.Threshold.Method,
			"threshold": strategy.Threshold.Value,
		}}},
		"unit_prefix": "",
	}}
	// 2. 将所有 target 的 clusterID/namespace/workloads 合并，生成单个 item：
	//    BKMonitor PromQL 类型策略只支持 1 个 item，多环境通过正则匹配合并到同一条 PromQL。
	scope := mergeTargetScopes(targets)
	promql := buildMergedAlertPromQL(strategy, scope)
	queryAlias := "a"
	items := []map[string]any{{
		"name":           strategy.StrategyCode,
		"expression":     queryAlias,
		"origin_sql":     "",
		"no_data_config": map[string]any{"is_enabled": false, "continuous": 5},
		"query_configs": []map[string]any{{
			"data_source_label": "prometheus",
			"data_type_label":   "time_series",
			"alias":             queryAlias,
			"promql":            promql,
			"agg_interval":      60,
			"interval_unit":     "s",
		}},
		"target":     []map[string]any{},
		"algorithms": algorithms,
	}}
	// 3. 最后组装整条 save_alarm_strategy 请求的公共部分：
	//    detects/notice/apply_instance 等配置在共享远端策略维度只保留一份。
	detects := []map[string]any{{
		"level":           int(strategy.Severity),
		"expression":      "",
		"trigger_config":  triggerConfig,
		"recovery_config": recoverConfig,
		"connector":       "and",
	}}
	noticeStartTime := lo.Ternary(
		strategy.EffectiveTimeRange.StartTime != "",
		strategy.EffectiveTimeRange.StartTime,
		defaultEffectiveStartTime,
	)
	noticeEndTime := lo.Ternary(
		strategy.EffectiveTimeRange.EndTime != "",
		strategy.EffectiveTimeRange.EndTime,
		defaultEffectiveEndTime,
	)
	notice := map[string]any{
		"signal": []string{"abnormal", "recovered"},
		"config": map[string]any{
			"need_poll":       true,
			"notify_interval": 7200,
			"template": []map[string]any{
				{
					"signal":       "abnormal",
					"message_tmpl": "",
					"title_tmpl":   buildDefaultNoticeTitle(strategy.DisplayName, "abnormal"),
				},
				{
					"signal":       "recovered",
					"message_tmpl": "",
					"title_tmpl":   buildDefaultNoticeTitle(strategy.DisplayName, "recovered"),
				},
			},
		},
		"options": map[string]any{
			"start_time": noticeStartTime,
			"end_time":   noticeEndTime,
			"converge_config": map[string]any{
				"need_biz_converge": true,
			},
		},
	}
	if len(strategy.NoticeGroupIDs) > 0 {
		notice["user_groups"] = strategy.NoticeGroupIDs
	}
	return &bkmapi.SaveAlarmStrategyReq{
		BkBizID:   monitorProjectID,
		ID:        remoteStrategyID,
		Name:      strategyName,
		Source:    "bkms",
		Scenario:  "kubernetes",
		IsEnabled: strategy.Enabled,
		Labels:    []string{"bkms"},
		Items:     items,
		Detects:   detects,
		Actions:   []map[string]any{},
		Notice:    notice,
		Operator:  operator,
	}
}

// buildMergedAlertPromQL 为多环境合并后的 target 构建单条 PromQL。
// 当只有 1 个 target 时退化为精确匹配
func buildMergedAlertPromQL(strategy *AlertStrategy, scope mergedTargetScope) string {
	clusterCond := exactOrRegex("bcs_cluster_id", scope.ClusterIDs)
	nsCond := exactOrRegex("namespace", scope.Namespaces)
	workloadCond := workloadMatcher(scope.Workloads)

	switch strategy.StrategyCode {
	case "cpu_request_usage_high":
		return buildMergedRatioPromQL(
			"container_cpu_usage_seconds_total", "rate", "[2m]",
			"kube_pod_container_resource_requests_cpu_cores", clusterCond, nsCond, workloadCond, "",
		)
	case "cpu_limit_usage_high":
		return buildMergedRatioPromQL(
			"container_cpu_usage_seconds_total", "rate", "[2m]",
			"kube_pod_container_resource_limits_cpu_cores", clusterCond, nsCond, workloadCond, "",
		)
	case "memory_request_usage_high":
		return buildMergedRatioPromQL(
			"container_memory_working_set_bytes", "raw", "",
			"kube_pod_container_resource_requests_memory_bytes", clusterCond, nsCond, workloadCond, "",
		)
	case "memory_limit_usage_high":
		return buildMergedRatioPromQL(
			"container_memory_working_set_bytes", "raw", "",
			"kube_pod_container_resource_limits_memory_bytes", clusterCond, nsCond, workloadCond, "",
		)
	case "pod_restart_frequent":
		windowMinutes := max(strategy.TriggerCondition.CheckWindow, 1)
		return fmt.Sprintf(
			`sum by(%s) (increase(kube_pod_container_status_restarts_total{%s,%s,%s}[%dm]))`,
			appScopedAlertGroupBy,
			clusterCond,
			nsCond,
			podNamePrefixMatcher(scope.Workloads),
			windowMinutes,
		)
	default:
		selector := buildMergedMetricSelector(strategy.MonitorMetric, clusterCond, nsCond, workloadCond)
		if strings.HasSuffix(strategy.MonitorMetric, "_total") {
			return fmt.Sprintf(`sum by(%s) (rate(%s[2m]))`, appScopedAlertGroupBy, selector)
		}
		return fmt.Sprintf(`sum by(%s) (%s)`, appScopedAlertGroupBy, selector)
	}
}

// buildAppScopedAlertPromQL 为应用级告警策略生成按 Pod 聚合的 PromQL（单环境兼容入口）。
func buildAppScopedAlertPromQL(strategy *AlertStrategy, env envmodel.Environment, workloads []string) string {
	return buildMergedAlertPromQL(strategy, mergedTargetScope{
		ClusterIDs: []string{env.Cluster.ClusterID},
		Namespaces: []string{env.Cluster.Namespace},
		Workloads:  workloads,
	})
}

// buildMergedRatioPromQL 生成“使用量 / request|limit * 100”形式的比例型 PromQL（合并模式）。
func buildMergedRatioPromQL(
	numeratorMetric, numeratorAgg, window, denominatorMetric,
	clusterCond, nsCond, numeratorWorkloadCond, denominatorWorkloadCond string,
) string {
	numeratorSelector := mergedContainerMetricSelector(numeratorMetric, clusterCond, nsCond, numeratorWorkloadCond)
	var numerator string
	switch numeratorAgg {
	case "rate":
		numerator = fmt.Sprintf(
			"sum by(%s) (rate(%s%s))",
			appScopedAlertGroupBy, numeratorSelector, window,
		)
	default:
		numerator = fmt.Sprintf(
			"sum by(%s) (%s)",
			appScopedAlertGroupBy, numeratorSelector,
		)
	}
	denominator := fmt.Sprintf(
		"sum by(%s) (%s)",
		appScopedAlertGroupBy,
		mergedContainerMetricSelector(denominatorMetric, clusterCond, nsCond, denominatorWorkloadCond),
	)
	return fmt.Sprintf("(%s / %s) * 100", numerator, denominator)
}

// buildMergedMetricSelector 构造合并模式下的指标选择器。
func buildMergedMetricSelector(metricID, clusterCond, nsCond, workloadCond string) string {
	conditions := []string{clusterCond, nsCond, workloadCond}
	if strings.HasPrefix(metricID, "container_") || strings.HasPrefix(metricID, "kube_pod_container_resource_") {
		conditions = append(conditions, `container_name!="POD"`)
	}
	return fmt.Sprintf("%s{%s}", metricID, strings.Join(conditions, ","))
}

// mergedContainerMetricSelector 构造合并模式下容器级指标选择器（含 container_name!="POD"）。
func mergedContainerMetricSelector(metricID, clusterCond, nsCond, workloadCond string) string {
	conditions := []string{clusterCond, nsCond, `container_name!="POD"`}
	if workloadCond != "" {
		conditions = append(conditions, workloadCond)
	}
	return fmt.Sprintf("%s{%s}", metricID, strings.Join(conditions, ","))
}

// mergedTargetScope 将多个 remoteTargetContext 中各自的 clusterID / namespace / workloads
// 去重合并，供 PromQL 构建时按需生成精确匹配或正则匹配条件。
type mergedTargetScope struct {
	ClusterIDs []string
	Namespaces []string
	Workloads  []string
}

// mergeTargetScopes 从一组 target 中提取并去重 clusterID、namespace 和 workloads。
// 其中 workloads 在 Helm 场景下可能来自多个受管 workload；非 Helm 场景通常只有一个默认 workload。
func mergeTargetScopes(targets []remoteTargetContext) mergedTargetScope {
	clusterSet := make(map[string]struct{})
	nsSet := make(map[string]struct{})
	wlSet := make(map[string]struct{})
	for _, t := range targets {
		clusterSet[t.Env.Cluster.ClusterID] = struct{}{}
		nsSet[t.Env.Cluster.Namespace] = struct{}{}
		for _, w := range t.Workloads {
			wlSet[w] = struct{}{}
		}
	}
	return mergedTargetScope{
		ClusterIDs: sortedKeys(clusterSet),
		Namespaces: sortedKeys(nsSet),
		Workloads:  sortedKeys(wlSet),
	}
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// exactOrRegex 为 PromQL label 匹配生成条件：
// 单值时返回精确匹配 key="val"，多值时返回正则匹配 key=~"val1|val2"。
func exactOrRegex(key string, values []string) string {
	if len(values) == 1 {
		return fmt.Sprintf(`%s="%s"`, key, values[0])
	}
	quoted := lo.Map(values, func(item string, _ int) string {
		return regexp.QuoteMeta(item)
	})
	return fmt.Sprintf(`%s=~"%s"`, key, strings.Join(quoted, "|"))
}

// workloadMatcher 根据 workload 数量生成精确匹配或正则匹配条件。
func workloadMatcher(workloads []string) string {
	if len(workloads) == 1 {
		return fmt.Sprintf(`workload_name="%s"`, workloads[0])
	}
	quoted := lo.Map(workloads, func(item string, _ int) string {
		return regexp.QuoteMeta(item)
	})
	return fmt.Sprintf(`workload_name=~"^(%s)$"`, strings.Join(quoted, "|"))
}

// podNamePrefixMatcher 根据 workload 名称列表构造 pod_name 的正则匹配条件。
// AppModel/Helm 生成的 Pod 名通常以 workload 名为前缀，并在后面拼接随机后缀，
// 因此这里统一匹配 `^workload(-.*)?$` 形式；多 workload 时用 `|` 组合成一个正则。
func podNamePrefixMatcher(workloads []string) string {
	quoted := lo.Map(workloads, func(item string, _ int) string {
		return regexp.QuoteMeta(item)
	})
	if len(quoted) == 1 {
		return fmt.Sprintf(`pod_name=~"^%s(-.*)?$"`, quoted[0])
	}
	return fmt.Sprintf(`pod_name=~"^(%s)(-.*)?$"`, strings.Join(quoted, "|"))
}

// buildDefaultNoticeTitle 生成默认告警通知标题。
// 未显式配置模板时，恢复态使用“告警恢复”前缀，其余信号统一视为触发态并使用“告警触发”前缀。
func buildDefaultNoticeTitle(displayName, signal string) string {
	switch signal {
	case "recovered":
		return fmt.Sprintf("【告警恢复】%s", displayName)
	default:
		return fmt.Sprintf("【告警触发】%s", displayName)
	}
}

// buildRemoteStrategyName 使用“策略名称【应用名称】”格式生成远端策略名称。
func buildRemoteStrategyName(strategy *AlertStrategy) string {
	return fmt.Sprintf("%s【%s】", strategy.DisplayName, strategy.AppName)
}

func uniqRemoteTargets(targets []remoteTargetContext) []remoteTargetContext {
	byKey := make(map[string]remoteTargetContext, len(targets))
	for _, target := range targets {
		byKey[remoteRefKey(target.Env.ID, target.TrafficLaneName)] = target
	}
	keys := lo.Keys(byKey)
	sort.Strings(keys)
	result := make([]remoteTargetContext, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result
}

func buildRemoteRefsFromTargets(
	targets []remoteTargetContext,
	remoteStrategyName string,
	remoteStrategyID int64,
) []RemoteStrategyRef {
	refs := make([]RemoteStrategyRef, 0, len(targets))
	for _, target := range targets {
		refs = append(refs, RemoteStrategyRef{
			EnvID:              target.Env.ID,
			EnvName:            target.Env.Name,
			TrafficLaneName:    target.TrafficLaneName,
			RemoteStrategyName: remoteStrategyName,
			RemoteStrategyID:   remoteStrategyID,
		})
	}
	return refs
}
