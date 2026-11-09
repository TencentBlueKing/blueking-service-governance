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

package bkmonitor

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// RankType 实例排名方向
type RankType string

const (
	// RankTypeTop 从高到低
	RankTypeTop RankType = "top"
	// RankTypeBottom 从低到高
	RankTypeBottom RankType = "bottom"
)

// minRankStep 排名子查询的最小步长（秒）
const minRankStep int64 = 60

// DefaultRankBy 默认排名指标：CPU 使用率
const DefaultRankBy = MetricCPULimitUsage

// promqlInstanceRank 按排名指标在时间范围内的平均值取 topk / bottomk。
// 占位：topk/bottomk、N、排名指标表达式、子查询范围（秒）、步长（秒）
const promqlInstanceRank = `%[1]s(%[2]d, avg_over_time((%[3]s)[%[4]ds:%[5]ds]))`

// 排名指标表达式，占位：汇聚方法、集群、命名空间、工作负载类型、工作负载名称。
// 用量按工作负载过滤；使用率的分母（limits）不带 workload 标签，只按集群、命名空间过滤，按 pod 与分子对齐
const (
	promqlRankCPUUsage    = `%[1]s by(pod)(rate(container_cpu_usage_seconds_total{bcs_cluster_id="%[2]s",namespace=~"^(%[3]s)$",workload_kind="%[4]s",workload_name="%[5]s",container_name!="POD"}[1m]))` //nolint:lll
	promqlRankMemoryUsage = `%[1]s by(pod)(container_memory_working_set_bytes{bcs_cluster_id="%[2]s",namespace=~"^(%[3]s)$",workload_kind="%[4]s",workload_name="%[5]s",container_name!="POD"})`          //nolint:lll

	promqlRankCPULimits    = `%[1]s by(pod)(kube_pod_container_resource_limits_cpu_cores{bcs_cluster_id="%[2]s",namespace=~"^(%[3]s)$",container_name!="POD"})`    //nolint:lll
	promqlRankMemoryLimits = `%[1]s by(pod)(kube_pod_container_resource_limits_memory_bytes{bcs_cluster_id="%[2]s",namespace=~"^(%[3]s)$",container_name!="POD"})` //nolint:lll
)

// rankExprTemplates 排名指标到表达式模板的映射
var rankExprTemplates = map[string]string{
	MetricCPULimitUsage:    promqlRankCPUUsage + " / " + promqlRankCPULimits,
	MetricMemoryLimitUsage: promqlRankMemoryUsage + " / " + promqlRankMemoryLimits,
	MetricCPUUsage:         promqlRankCPUUsage,
	MetricMemoryUsage:      promqlRankMemoryUsage,
}

// InstanceRankQuery 实例排名查询参数
type InstanceRankQuery struct {
	// BkBizID 蓝鲸监控业务 ID
	BkBizID int64
	// ClusterID BCS 集群 ID
	ClusterID string
	// Namespace 命名空间
	Namespace string
	// WorkloadKind 工作负载类型，如 GameDeployment、Deployment
	WorkloadKind string
	// WorkloadName 工作负载名称
	WorkloadName string
	// RankBy 排名指标，默认 DefaultRankBy
	RankBy string
	// Method 汇聚方法，默认 sum
	Method AggMethod
	// Limit 取前 N 个
	Limit int
	// RankType 排名方向，默认 top
	RankType RankType
	// StartTime 开始时间（秒）
	StartTime int64
	// EndTime 结束时间（秒）
	EndTime int64
	// Interval 子查询步长（秒），最小 60
	Interval int64
	// Username 用户名
	Username string
}

// InstanceRank 单个实例的排名结果
type InstanceRank struct {
	// Instance 实例（Pod）名称
	Instance string
	// Value 排名指标的平均值（原始单位）
	Value float64
	// Unit 排名指标的单位
	Unit string
}

// BuildInstanceRankPromQL 构建实例排名的 PromQL，排名指标不支持时返回错误
func BuildInstanceRankPromQL(q *InstanceRankQuery) (string, error) {
	rankBy := lo.Ternary(q.RankBy == "", DefaultRankBy, q.RankBy)
	exprTpl, ok := rankExprTemplates[rankBy]
	if !ok {
		return "", errors.Errorf("unsupported rank metric: %s", rankBy)
	}
	fn := lo.Ternary(q.RankType == RankTypeBottom, "bottomk", "topk")
	method := lo.Ternary(q.Method == "", AggMethodSum, q.Method)
	step := max(q.Interval, minRankStep)
	// 范围至少一个步长，避免出现 [0s:...]
	rangeSeconds := max(q.EndTime-q.StartTime, step)
	expr := fmt.Sprintf(exprTpl, method, q.ClusterID, q.Namespace, q.WorkloadKind, q.WorkloadName)
	return fmt.Sprintf(promqlInstanceRank, fn, q.Limit, expr, rangeSeconds, step), nil
}

// RankInstances 查询排名指标平均值排名前 Limit 的实例
func (s *MetricTimeSeriesService) RankInstances(ctx context.Context, q *InstanceRankQuery) ([]*InstanceRank, error) {
	promql, err := BuildInstanceRankPromQL(q)
	if err != nil {
		return nil, err
	}
	rankBy := lo.Ternary(q.RankBy == "", DefaultRankBy, q.RankBy)
	unit := lo.FindOrElse(MetricDefinitions, MetricDefinition{}, func(def MetricDefinition) bool {
		return def.Key == rankBy
	}).Unit

	client, err := s.newClient(q.Username)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}

	queryConfig := bkmapi.QueryConfig{
		DataSourceLabel: "prometheus",
		DataTypeLabel:   "time_series",
		PromQL:          promql,
		Interval:        max(q.Interval, minRankStep),
		IntervalUnit:    "s",
	}
	req := bkmapi.NewTimeSeriesUnifyQueryReq(q.BkBizID, []bkmapi.QueryConfig{queryConfig}, "a", q.StartTime, q.EndTime)
	req.Type = "instant"

	resp, err := client.TimeSeriesUnifyQuery(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(
			err, "rank instances of workload %s/%s in namespace %s by %s",
			q.WorkloadKind, q.WorkloadName, q.Namespace, rankBy,
		)
	}

	ranks := lo.FilterMap(resp.Series, func(series bkmapi.TimeSeriesData, _ int) (*InstanceRank, bool) {
		name := podNameOf(series)
		if name == "" || len(series.Datapoints) == 0 {
			return nil, false
		}
		// 取最后一个点的值
		return &InstanceRank{Instance: name, Value: series.Datapoints[len(series.Datapoints)-1][0], Unit: unit}, true
	})

	// 返回顺序不保证，按排名方向重新排序
	bottom := q.RankType == RankTypeBottom
	sort.SliceStable(ranks, func(i, j int) bool {
		if bottom {
			return ranks[i].Value < ranks[j].Value
		}
		return ranks[i].Value > ranks[j].Value
	})
	if q.Limit > 0 && len(ranks) > q.Limit {
		ranks = ranks[:q.Limit]
	}
	return ranks, nil
}
