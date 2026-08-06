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

// Package bkmonitor api client，如：蓝鲸监控的 apm、蓝鲸监控的 metadata
package bkmonitor

import (
	"context"
	"net/http"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// TimeSeriesUnifyQuery 统一时序数据查询
func (c *ApiClient) TimeSeriesUnifyQuery(
	ctx context.Context, req *TimeSeriesUnifyQueryReq,
) (*TimeSeriesUnifyQueryResp, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.newTimeSeriesUnifyQueryOperation(req))
	if err != nil {
		return nil, errors.Wrap(err, "time series unify query failed")
	}

	result := new(TimeSeriesUnifyQueryResp)
	if err = mapstructure.Decode(resp["data"], result); err != nil {
		return nil, errors.Wrap(err, "decode time series unify query response failed")
	}

	return result, nil
}

// newTimeSeriesUnifyQueryOperation 创建统一时序数据查询操作
func (c *ApiClient) newTimeSeriesUnifyQueryOperation(req *TimeSeriesUnifyQueryReq) define.Operation {
	return c.NewOperation(
		bkapi.OperationConfig{
			Name:   "time_series_unify_query",
			Method: http.MethodPost,
			Path:   "/time_series/unify_query/",
		},
		bkapi.OptSetRequestBody(req),
	)
}

// NewTimeSeriesUnifyQueryReq 创建统一时序数据查询请求
func NewTimeSeriesUnifyQueryReq(
	bkBizID int64, queryConfigs []QueryConfig, expression string, startTime, endTime int64,
) *TimeSeriesUnifyQueryReq {
	return &TimeSeriesUnifyQueryReq{
		BkBizID:      bkBizID,
		QueryConfigs: queryConfigs,
		Expression:   expression,
		StartTime:    startTime,
		EndTime:      endTime,
		Target:       []any{},
		Format:       "time_series",
		Type:         "range",
	}
}

// NewQueryConfig 创建单指标查询配置
func NewQueryConfig(dataSourceLabel, metricField, method string) QueryConfig {
	return QueryConfig{
		DataSourceLabel: dataSourceLabel,
		DataTypeLabel:   "time_series",
		Metrics: []QueryMetric{
			{
				Field:  metricField,
				Method: method,
				Alias:  "a",
			},
		},
		Interval:     "auto",
		IntervalUnit: "s",
	}
}
