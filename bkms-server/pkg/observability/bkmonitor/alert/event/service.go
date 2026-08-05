// Package event 提供蓝鲸监控告警事件查询功能
package event

import (
	"context"
	"strconv"
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
	// AlertName 按告警名称过滤。
	AlertName string
	// Description 按告警内容过滤。
	Description string
	// StrategyName 按策略名称过滤。
	StrategyName string
	// EventID 按事件 ID 过滤。
	EventID string
	// Target 按目标实例过滤。
	Target string
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
	bkBizID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return client.SearchAlert(ctx, s.buildSearchAlertReq(bkBizID, input, nil))
}

// SearchByStrategyIDs 按策略 ID 列表查询告警事件
func (s *Service) SearchByStrategyIDs(
	ctx context.Context,
	ws *workspace.Workspace,
	operator string,
	strategyIDs []int64,
	input SearchInput,
) (*bkmapi.SearchAlertResp, error) {
	bkBizID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return client.SearchAlert(ctx, s.buildSearchAlertReq(bkBizID, input, strategyIDs))
}

// GetDetail 查询单条告警详情
func (s *Service) GetDetail(
	ctx context.Context,
	ws *workspace.Workspace,
	operator string,
	alertID string,
) (map[string]any, error) {
	bkBizID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return client.GetAlertDetail(ctx, &bkmapi.AlertDetailReq{BkBizID: bkBizID, ID: alertID})
}

func (s *Service) buildSearchAlertReq(
	bkBizID int64,
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
		BkBizIDs:    []int64{bkBizID},
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
	conditions := make([]map[string]any, 0, 6)
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
		{key: "id", val: input.AlertID},
		{key: "alert_name", val: input.AlertName},
		{key: "strategy_name", val: input.StrategyName},
		{key: "event_id", val: input.EventID},
		{key: "target", val: input.Target},
	} {
		appendIf(item.val != "", item.key, []string{item.val})
	}
	return conditions
}

func buildSearchQueryString(input SearchInput) string {
	if input.Description == "" {
		return ""
	}
	return "description:" + strconv.Quote(input.Description)
}
