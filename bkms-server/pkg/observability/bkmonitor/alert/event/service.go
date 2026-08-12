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

// Package event 提供蓝鲸监控告警事件查询功能
package event

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert"
)

// defaultSearchAlertLookback 告警事件查询的默认时间回溯窗口。
// 当调用方未指定起止时间时，默认查询从当前时刻往前 24 小时内的告警事件。
const defaultSearchAlertLookback = 24 * time.Hour

// SearchInput 告警事件查询输入。
type SearchInput struct {
	// Status 告警状态过滤，如 ABNORMAL、CLOSED。
	Status []string
	// Severity 告警级别过滤。
	Severity []int
	// StartTime 查询起始时间戳（秒）。
	StartTime int64
	// EndTime 查询结束时间戳（秒）。
	EndTime int64
	// Page 分页页码。
	Page int
	// PageSize 分页大小。
	PageSize int
	// AlertID 按告警 ID 过滤。
	AlertID string
	// Description 按告警内容过滤。
	Description string
	// StrategyName 按策略名称过滤。
	StrategyName string
	// EventID 按事件 ID 过滤。
	EventID string
	// Target 按目标实例过滤。
	Target string
	// ClusterID 按集群 ID 过滤。
	ClusterID string
	// Namespace 按命名空间过滤。
	Namespace string
	// Ordering 排序字段列表，沿用蓝鲸监控 API 的排序语法。
	Ordering []string
}

// Service 告警事件查询业务逻辑层
type Service struct {
	newClient alert.ClientFactory
}

// NewService 创建 Service 实例。
func NewService() *Service {
	return &Service{newClient: bkmapi.NewMonitorClient}
}

// Search 查询工作空间下的告警事件
func (s *Service) Search(
	ctx context.Context,
	ws *workspace.Workspace,
	operator string,
	input SearchInput,
) (*bkmapi.SearchAlertResp, error) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return client.SearchAlert(ctx, s.buildSearchAlertReq(bkMonitorProjectID, input, nil))
}

// SearchByStrategyIDs 按策略 ID 列表查询告警事件
func (s *Service) SearchByStrategyIDs(
	ctx context.Context,
	ws *workspace.Workspace,
	operator string,
	strategyIDs []int64,
	input SearchInput,
) (*bkmapi.SearchAlertResp, error) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return client.SearchAlert(ctx, s.buildSearchAlertReq(bkMonitorProjectID, input, strategyIDs))
}

// GetDetail 查询单条告警详情
func (s *Service) GetDetail(
	ctx context.Context,
	ws *workspace.Workspace,
	operator string,
	alertID string,
) (map[string]any, error) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return client.GetAlertDetail(ctx, &bkmapi.AlertDetailReq{BkBizID: bkMonitorProjectID, ID: alertID})
}

func (s *Service) buildSearchAlertReq(
	bkMonitorProjectID int64,
	input SearchInput,
	strategyIDs []int64,
) *bkmapi.SearchAlertReq {
	endTime := input.EndTime
	if endTime <= 0 {
		endTime = time.Now().Unix()
	}
	startTime := input.StartTime
	if startTime <= 0 {
		startTime = endTime - int64(defaultSearchAlertLookback/time.Second)
	}
	req := &bkmapi.SearchAlertReq{
		BkBizIDs:    []int64{bkMonitorProjectID},
		Status:      input.Status,
		Severity:    input.Severity,
		StartTime:   startTime,
		EndTime:     endTime,
		Page:        input.Page,
		PageSize:    input.PageSize,
		Ordering:    input.Ordering,
		QueryString: buildSearchQueryString(input),
	}
	conditions := buildSearchConditions(input, strategyIDs)
	if len(conditions) > 0 {
		req.Conditions = conditions
	}
	return req
}

// buildSearchConditions 将事件查询输入转换为蓝鲸监控 search_alert 接口使用的 conditions 数组。
// 这里只处理“可选过滤条件”的拼装：策略 ID 走整型数组，其余字符串字段按接口要求包装成单元素字符串数组。
func buildSearchConditions(input SearchInput, strategyIDs []int64) []map[string]any {
	conditions := make([]map[string]any, 0, 8)
	appendIf := func(ok bool, key string, value any) {
		if ok {
			conditions = append(conditions, map[string]any{"key": key, "value": value})
		}
	}
	appendIf(len(strategyIDs) > 0, "strategy_id", strategyIDs)
	for _, item := range []struct {
		key string
		val string
	}{
		{key: "strategy_name", val: input.StrategyName},
		{key: "event_id", val: input.EventID},
		{key: "target", val: input.Target},
		{key: "tags.bcs_cluster_id", val: input.ClusterID},
		{key: "tags.namespace", val: input.Namespace},
	} {
		appendIf(item.val != "", item.key, []string{item.val})
	}
	return conditions
}

// buildSearchQueryString 将查询输入中的告警 ID、告警内容拼接为蓝鲸监控 search_alert 接口
// 使用的 query_string 查询条件。
//
//   - 仅当对应字段非空时才参与拼装，为空则忽略；
//   - 每个字段以 field:"value" 的形式表达（value 经 strconv.Quote 转义，避免特殊字符破坏查询语法）；
//   - 多个字段短语之间以 AND 连接；
//   - 当没有任何字段参与时返回空字符串，交由接口侧忽略该查询项。
func buildSearchQueryString(input SearchInput) string {
	parts := make([]string, 0, 2)
	appendPhrase := func(field, value string) {
		if value == "" {
			return
		}
		parts = append(parts, field+":"+strconv.Quote(value))
	}

	// 告警名称展示与查询统一收敛到 BKMS 本地 displayName 语义，由上层先映射远端 strategyID。
	// 因此这里不再对监控原始 alert_name 做 query_string 过滤，避免前端依赖远端命名规则。
	appendPhrase("id", input.AlertID)
	appendPhrase("description", input.Description)
	// 没有任何过滤条件时返回空字符串，交由接口侧忽略该查询项。
	if len(parts) == 0 {
		return ""
	}
	// 多个短语以 AND 连接，构成蓝鲸监控 query_string 查询条件。
	return strings.Join(parts, " AND ")
}
