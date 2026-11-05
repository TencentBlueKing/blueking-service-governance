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

// promqlInstanceRank 按工作负载内 Pod 的平均 CPU 使用量排名。
// 占位：topk/bottomk、N、汇聚方法、集群、命名空间、工作负载类型、工作负载名称、子查询范围（秒）、步长（秒）
const promqlInstanceRank = `%[1]s(%[2]d, avg_over_time((%[3]s by(pod)(rate(container_cpu_usage_seconds_total{bcs_cluster_id="%[4]s",namespace=~"^(%[5]s)$",workload_kind="%[6]s",workload_name="%[7]s",container_name!="POD"}[1m])))[%[8]ds:%[9]ds]))` //nolint:lll

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
	// Value 平均 CPU 使用量（核数）
	Value float64
}

// BuildInstanceRankPromQL 构建实例排名的 PromQL
func BuildInstanceRankPromQL(q *InstanceRankQuery) string {
	fn := lo.Ternary(q.RankType == RankTypeBottom, "bottomk", "topk")
	method := lo.Ternary(q.Method == "", AggMethodSum, q.Method)
	step := max(q.Interval, minRankStep)
	// 范围至少一个步长，避免出现 [0s:...]
	rangeSeconds := max(q.EndTime-q.StartTime, step)
	return fmt.Sprintf(
		promqlInstanceRank,
		fn, q.Limit, method, q.ClusterID, q.Namespace, q.WorkloadKind, q.WorkloadName, rangeSeconds, step,
	)
}

// RankInstances 查询平均 CPU 使用量排名前 Limit 的实例
func (s *MetricTimeSeriesService) RankInstances(ctx context.Context, q *InstanceRankQuery) ([]*InstanceRank, error) {
	client, err := s.newClient(q.Username)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}

	queryConfig := bkmapi.QueryConfig{
		DataSourceLabel: "prometheus",
		DataTypeLabel:   "time_series",
		PromQL:          BuildInstanceRankPromQL(q),
		Interval:        max(q.Interval, minRankStep),
		IntervalUnit:    "s",
	}
	req := bkmapi.NewTimeSeriesUnifyQueryReq(q.BkBizID, []bkmapi.QueryConfig{queryConfig}, "a", q.StartTime, q.EndTime)
	req.Type = "instant"

	resp, err := client.TimeSeriesUnifyQuery(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(
			err, "rank instances of workload %s/%s in namespace %s", q.WorkloadKind, q.WorkloadName, q.Namespace,
		)
	}

	ranks := lo.FilterMap(resp.Series, func(series bkmapi.TimeSeriesData, _ int) (*InstanceRank, bool) {
		name := podNameOf(series)
		if name == "" || len(series.Datapoints) == 0 {
			return nil, false
		}
		// 取最后一个点的值
		return &InstanceRank{Instance: name, Value: series.Datapoints[len(series.Datapoints)-1][0]}, true
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
