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

package serializer

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/timex"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	alertevent "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/event"
)

// AlertQueryInput 告警事件查询参数
type AlertQueryInput struct {
	Status    []string `form:"status"`
	Severity  []int    `form:"severity"`
	StartTime int64    `form:"startTime"`
	EndTime   int64    `form:"endTime"`
	Page      int      `form:"page" binding:"required,gte=1"`
	PageSize  int      `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
	AlertID   string   `form:"alertID"`
	// AlertDisplayName 对外只暴露 BKMS 本地展示名，例如 `CPU 使用率过高`。
	// 与之对应的 BKMonitor 原始名称格式为 `CPU 使用率过高【demo-app】`。
	AlertDisplayName string   `form:"alertDisplayName"`
	Description      string   `form:"description"`
	StrategyName     string   `form:"strategyName"`
	EventID          string   `form:"eventID"`
	Target           string   `form:"target"`
	Ordering         []string `form:"ordering"`
}

// Normalize 对查询参数做默认排序补充。
func (q *AlertQueryInput) Normalize() {
	if len(q.Ordering) == 0 {
		q.Ordering = []string{"-create_time"}
	}
}

// ToSearchInput 将查询参数转换为告警事件服务层输入。
func (q AlertQueryInput) ToSearchInput() alertevent.SearchInput {
	return alertevent.SearchInput{
		Status:       q.Status,
		Severity:     q.Severity,
		StartTime:    q.StartTime,
		EndTime:      q.EndTime,
		Page:         q.Page,
		PageSize:     q.PageSize,
		AlertID:      q.AlertID,
		Description:  q.Description,
		StrategyName: q.StrategyName,
		EventID:      q.EventID,
		Target:       q.Target,
		Ordering:     q.Ordering,
	}
}

// AppScopedAlertQueryInput 应用维度告警事件查询参数。
type AppScopedAlertQueryInput struct {
	AlertQueryInput
	EnvName string `form:"envName"`
}

// AlertEventOutput 告警事件输出
type AlertEventOutput struct {
	StrategyID string `json:"strategyID,omitempty"`
	AlertID    string `json:"alertID"`
	EventID    string `json:"eventID,omitempty"`
	// AlertDisplayName 是 BKMS 本地策略展示名，例如 `CPU 使用率过高`；
	// 对应的监控原始名称格式为 `CPU 使用率过高【demo-app】`。
	AlertDisplayName    string   `json:"alertDisplayName"`
	Assignee            []string `json:"assignee,omitempty"`
	Status              string   `json:"status"`
	Severity            int      `json:"severity"`
	Description         string   `json:"description"`
	BKMonitorStrategyID int64    `json:"bkMonitorStrategyID,string"`
	StrategyName        string   `json:"strategyName"`
	TargetType          string   `json:"targetType"`
	Target              string   `json:"target"`
	Dimensions          any      `json:"dimensions,omitempty"`
	CurrentValue        any      `json:"currentValue,omitempty"`
	DataSource          string   `json:"dataSource,omitempty"`
	Content             string   `json:"content,omitempty"`
	Detail              any      `json:"detail,omitempty"`
	RelatedInfo         any      `json:"relatedInfo,omitempty"`
	Duration            string   `json:"duration,omitempty"`
	BeginTime           int64    `json:"beginTime"`
	EndTime             int64    `json:"endTime"`
	LatestTime          int64    `json:"latestTime"`
	CreateTime          int64    `json:"createTime"`
}

// ListAlertEventsOutput 告警事件列表输出
type ListAlertEventsOutput struct {
	Count   int64               `json:"count,string"`
	Results []*AlertEventOutput `json:"results"`
}

// ListAlertEventsResp 告警事件列表响应
type ListAlertEventsResp struct {
	Data *ListAlertEventsOutput `json:"data"`
}

// AlertDetailURIInput 单条告警详情路径参数
type AlertDetailURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,min=1,max=27,workspace_id"`
	AlertID     string `uri:"alertID" binding:"required,min=1"`
}

// GetAlertDetailResp 单条告警详情响应
type GetAlertDetailResp struct {
	Data map[string]any `json:"data"`
}

// NewAlertEventOutput 从云 API 告警事件转换为输出。
// 这里显式改写为本地 displayName，是为了让前端不必理解监控侧 alertName 的拼装规则。
func NewAlertEventOutput(a bkmapi.AlertEvent, strategyID, alertDisplayName string) *AlertEventOutput {
	return &AlertEventOutput{
		StrategyID:          strategyID,
		AlertID:             a.ID,
		EventID:             a.EventID,
		AlertDisplayName:    alertDisplayName,
		Assignee:            a.Assignee,
		Status:              a.Status,
		Severity:            a.Severity,
		Description:         a.Description,
		BKMonitorStrategyID: a.StrategyID,
		StrategyName:        a.StrategyName,
		TargetType:          a.TargetType,
		Target:              a.Target,
		Dimensions:          a.Dimensions,
		CurrentValue:        a.CurrentValue,
		DataSource:          a.DataSource,
		Content:             a.Content,
		Detail:              a.Detail,
		RelatedInfo:         a.RelatedInfo,
		Duration:            calcAlertDuration(a.BeginTime, a.EndTime, a.LatestTime),
		BeginTime:           a.BeginTime,
		EndTime:             a.EndTime,
		LatestTime:          a.LatestTime,
		CreateTime:          a.CreateTime,
	}
}

// NewGetAlertDetailResp 规范化详情接口输出字段，统一对外使用 alertID / alertDisplayName。
// alert_name为 策略名【应用名】，alertDisplayName为策略名
func NewGetAlertDetailResp(detail map[string]any, alertDisplayName string) *GetAlertDetailResp {
	if detail == nil {
		return &GetAlertDetailResp{Data: nil}
	}

	normalized := make(map[string]any, len(detail))
	for key, value := range detail {
		if key == "id" {
			if _, ok := detail["alertID"]; ok {
				continue
			}
			normalized["alertID"] = value
			continue
		}
		// BKMonitor 详情原始字段仍是 alert_name，这里转换成对外统一的 alertDisplayName。
		if key == "alert_name" {
			displayValue := value
			if alertDisplayName != "" {
				displayValue = alertDisplayName
			}
			normalized["alertDisplayName"] = displayValue
			continue
		}
		normalized[key] = value
	}
	return &GetAlertDetailResp{Data: normalized}
}

func calcAlertDuration(beginTime, endTime, latestTime int64) string {
	if beginTime <= 0 {
		return ""
	}
	end := endTime
	if end <= 0 {
		end = latestTime
	}
	if end <= 0 || end < beginTime {
		return ""
	}
	startStr := time.Unix(beginTime, 0).Format(timex.TimeLayout)
	endStr := time.Unix(end, 0).Format(timex.TimeLayout)
	return timex.CalcDuration(startStr, endStr)
}
