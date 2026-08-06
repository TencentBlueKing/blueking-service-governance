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
	"strconv"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// SearchAlarmStrategy 查询告警策略列表
func (c *MonitorGatewayClient) SearchAlarmStrategy(
	ctx context.Context,
	req *SearchAlarmStrategyReq,
) (*SearchAlarmStrategyResp, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "search_alarm_strategy_v3",
			Method: http.MethodPost,
			Path:   "/app/alarm_strategy/search/v3/",
		},
		bkapi.OptSetRequestBody(req),
	))
	if err != nil {
		return nil, errors.Wrapf(err, "search alarm strategy failed, bk_biz_id: %d", req.BkBizID)
	}

	result := new(SearchAlarmStrategyResp)
	if err = mapstructure.Decode(resp["data"], result); err != nil {
		return nil, errors.Wrap(err, "decode search alarm strategy response failed")
	}
	return result, nil
}

// SaveAlarmStrategy 创建或更新告警策略
func (c *MonitorGatewayClient) SaveAlarmStrategy(
	ctx context.Context,
	req *SaveAlarmStrategyReq,
) (*SaveAlarmStrategyResp, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "save_alarm_strategy_v3",
			Method: http.MethodPost,
			Path:   "/app/alarm_strategy/save/v3/",
		},
		bkapi.OptSetRequestBody(req),
		bkapi.OptSetRequestHeader(headerBkapiUserName, req.Operator),
	))
	if err != nil {
		return nil, errors.Wrapf(err, "save alarm strategy failed, bk_biz_id: %d, name: %s", req.BkBizID, req.Name)
	}

	result := new(SaveAlarmStrategyResp)
	if err = mapstructure.Decode(resp["data"], result); err != nil {
		return nil, errors.Wrap(err, "decode save alarm strategy response failed")
	}
	return result, nil
}

// SwitchAlarmStrategy 批量启停告警策略
func (c *MonitorGatewayClient) SwitchAlarmStrategy(ctx context.Context, req *SwitchAlarmStrategyReq) error {
	if err := Validate(req); err != nil {
		return err
	}

	_, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "switch_alarm_strategy",
			Method: http.MethodPost,
			Path:   "/app/alarm_strategy/switch/",
		},
		bkapi.OptSetRequestBody(req),
	))
	if err != nil {
		return errors.Wrapf(err, "switch alarm strategy failed, bk_biz_id: %d", req.BkBizID)
	}
	return nil
}

// DeleteAlarmStrategy 批量删除告警策略
func (c *MonitorGatewayClient) DeleteAlarmStrategy(ctx context.Context, req *DeleteAlarmStrategyReq) error {
	if err := Validate(req); err != nil {
		return err
	}

	_, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "delete_alarm_strategy_v3",
			Method: http.MethodPost,
			Path:   "/app/alarm_strategy/delete/v3/",
		},
		bkapi.OptSetRequestBody(req),
	))
	if err != nil {
		return errors.Wrapf(err, "delete alarm strategy failed, bk_biz_id: %d", req.BkBizID)
	}
	return nil
}

// SearchAlert 查询告警事件列表
func (c *MonitorGatewayClient) SearchAlert(ctx context.Context, req *SearchAlertReq) (*SearchAlertResp, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "search_alert",
			Method: http.MethodPost,
			Path:   "/app/alert/search/",
		},
		bkapi.OptSetRequestBody(req),
	))
	if err != nil {
		return nil, errors.Wrapf(err, "search alert failed, bk_biz_ids: %v", req.BkBizIDs)
	}

	return decodeSearchAlertData(resp["data"])
}

func decodeSearchAlertData(data any) (*SearchAlertResp, error) {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil, errors.New("invalid search alert response data")
	}

	result := new(SearchAlertResp)
	if err := mapstructure.Decode(dataMap["alerts"], &result.Alerts); err != nil {
		return nil, errors.Wrap(err, "decode alerts failed")
	}
	if total, exists := dataMap["total"]; exists {
		// total 在接口文档中定义为 int，这里用 cast 兼容 map[string]any 下的实际数值形态。
		result.Total = cast.ToInt64(total)
	}
	return result, nil
}

// GetAlertDetail 查询告警详情
func (c *MonitorGatewayClient) GetAlertDetail(ctx context.Context, req *AlertDetailReq) (map[string]any, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}

	params := map[string]string{
		"bk_biz_id": strconv.FormatInt(req.BkBizID, 10),
		"id":        req.ID,
	}
	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "alert_detail",
			Method: http.MethodGet,
			Path:   "/app/alert/detail/",
		},
	).SetQueryParams(params))
	if err != nil {
		return nil, errors.Wrapf(err, "get alert detail failed, bk_biz_id: %d, id: %s", req.BkBizID, req.ID)
	}

	if data, ok := resp["data"].(map[string]any); ok {
		return data, nil
	}
	return nil, errors.New("invalid alert detail response data")
}
