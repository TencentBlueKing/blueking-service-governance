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
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// clientFactory 工厂函数类型
type clientFactory func(username string) (bkmapi.Client, error)

// MetricTimeSeriesService 时序指标查询服务
type MetricTimeSeriesService struct {
	// newClient 创建底层 bkmonitor 客户端的工厂函数
	newClient clientFactory
}

// NewMetricTimeSeriesService 创建 MetricTimeSeriesService 实例
func NewMetricTimeSeriesService() *MetricTimeSeriesService {
	return &MetricTimeSeriesService{
		newClient: bkmapi.New,
	}
}

// QueryTimeSeries 查询时序数据
func (s *MetricTimeSeriesService) QueryTimeSeries(
	ctx context.Context, query *MetricsQuery,
) (*MetricsResult, error) {
	client, err := s.newClient(query.Username)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	promQLs := NewPromQLBuilder(query).Build()

	var targetDefs []MetricDefinition
	if len(query.MetricKeys) > 0 {
		targetDefs = lo.Filter(MetricDefinitions, func(def MetricDefinition, _ int) bool {
			return lo.Contains(query.MetricKeys, def.Key)
		})
	} else {
		targetDefs = MetricDefinitions
	}
	result := &MetricsResult{
		Metrics: lo.SliceToMap(targetDefs, func(def MetricDefinition) (string, *MetricTimeSeriesData) {
			return def.Key, &MetricTimeSeriesData{
				DisplayName: def.DisplayName,
				Unit:        def.Unit,
				Series:      make([]*TimeSeries, 0),
			}
		}),
	}

	// 逐指标查询
	for _, def := range targetDefs {
		promql, ok := promQLs[def.Key]
		if !ok {
			continue
		}

		queryConfig := bkmapi.QueryConfig{
			DataSourceLabel: "prometheus",
			DataTypeLabel:   "time_series",
			PromQL:          promql,
			Interval:        query.Interval,
			IntervalUnit:    "s",
		}

		req := bkmapi.NewTimeSeriesUnifyQueryReq(
			query.BkBizID,
			[]bkmapi.QueryConfig{queryConfig},
			// Expression 引用 QueryConfig 中 Metrics 的 Alias，单指标查询时固定为 "a"，表示直接返回该查询结果
			"a",
			query.StartTime,
			query.EndTime,
		)

		resp, err := client.TimeSeriesUnifyQuery(ctx, req)
		if err != nil {
			return nil, errors.Wrapf(err, "query metric %s failed", def.Key)
		}

		// 解析响应，按 pod 维度提取时序数据
		metricData := result.Metrics[def.Key]
		for _, series := range resp.Series {
			podName := lo.CoalesceOrEmpty(
				series.Dimensions["pod"],
				series.Dimensions["pod_name"],
				series.Dimensions["instance"],
			)
			if podName == "" {
				continue
			}

			ts := &TimeSeries{
				Instance:   podName,
				DataPoints: series.Datapoints,
			}
			if series.Stat != nil {
				ts.Stat = series.Stat
			}
			metricData.Series = append(metricData.Series, ts)
		}
	}

	return result, nil
}
