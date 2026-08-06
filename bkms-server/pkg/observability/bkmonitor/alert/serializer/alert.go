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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
)

// AlertStrategyURIInput 告警策略路径参数
type AlertStrategyURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,min=1,max=27,workspace_id"`
	AppID       string `uri:"appID" binding:"required,uri_slug"`
	StrategyID  string `uri:"strategyID" binding:"required,min=1"`
}

// AlertStrategyWorkspaceURIInput 工作空间级告警路径参数
type AlertStrategyWorkspaceURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,min=1,max=27,workspace_id"`
}

// AlertStrategyAppURIInput 应用级告警策略路径参数
type AlertStrategyAppURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,min=1,max=27,workspace_id"`
	AppID       string `uri:"appID" binding:"required,uri_slug"`
}

// CreateAlertStrategyBody 创建告警策略请求体
type CreateAlertStrategyBody struct {
	StrategyCode       string                  `json:"strategyCode" binding:"required,alert_strategy_code"`
	DisplayName        string                  `json:"displayName" binding:"required"`
	Severity           int                     `json:"severity" binding:"required,oneof=1 2 3"`
	Threshold          ThresholdConfigInput    `json:"threshold" binding:"required"`
	TriggerCondition   TriggerConditionInput   `json:"triggerCondition"`
	RecoverCondition   RecoverConditionInput   `json:"recoverCondition"`
	EffectiveTimeRange EffectiveTimeRangeInput `json:"effectiveTimeRange"`
	EffectiveScope     EffectiveScopeInput     `json:"effectiveScope" binding:"required"`
	NoticeGroupIDs     []int64                 `json:"noticeGroupIDs"`
	Enabled            bool                    `json:"enabled"`
}

// ToCreateReq 将创建请求体转换为领域层 CreateReq。
func (b CreateAlertStrategyBody) ToCreateReq(
	workspaceID, appID, appName, operator string,
) (*alertstrategy.CreateReq, error) {
	envIDs, err := parseEnvIDs(b.EffectiveScope.EnvIDs)
	if err != nil {
		return nil, errors.Wrap(err, "parse effective scope envIDs")
	}
	return &alertstrategy.CreateReq{
		WorkspaceID:  workspaceID,
		AppID:        appID,
		AppName:      appName,
		StrategyCode: b.StrategyCode,
		DisplayName:  b.DisplayName,
		Severity:     alertstrategy.AlertSeverity(b.Severity),
		Threshold: alertstrategy.ThresholdConfig{
			Method: b.Threshold.Method,
			Value:  b.Threshold.Value,
		},
		TriggerCondition: alertstrategy.TriggerCondition{
			Count:       b.TriggerCondition.Count,
			CheckWindow: b.TriggerCondition.CheckWindow,
		},
		RecoverCondition: alertstrategy.RecoverCondition{
			CheckWindow: b.RecoverCondition.CheckWindow,
		},
		EffectiveTimeRange: alertstrategy.EffectiveTimeRange{
			StartTime: b.EffectiveTimeRange.StartTime,
			EndTime:   b.EffectiveTimeRange.EndTime,
		},
		EffectiveScope: alertstrategy.EffectiveScope{
			Type:     alertstrategy.EffectiveScopeType(b.EffectiveScope.Type),
			EnvTypes: b.EffectiveScope.EnvTypes,
			EnvIDs:   envIDs,
		},
		NoticeGroupIDs: b.NoticeGroupIDs,
		Enabled:        b.Enabled,
		Operator:       operator,
	}, nil
}

// ThresholdConfigInput 阈值配置输入
type ThresholdConfigInput struct {
	Method string  `json:"method" binding:"required,oneof=gte gt lte lt eq neq"`
	Value  float64 `json:"value"`
}

// TriggerConditionInput 触发条件输入
type TriggerConditionInput struct {
	Count       int `json:"count"`
	CheckWindow int `json:"checkWindow"`
}

// RecoverConditionInput 恢复条件输入
type RecoverConditionInput struct {
	CheckWindow int `json:"checkWindow"`
}

// EffectiveTimeRangeInput 生效时间段输入
type EffectiveTimeRangeInput struct {
	StartTime string `json:"startTime" binding:"omitempty,datetime=15:04:05"`
	EndTime   string `json:"endTime" binding:"omitempty,datetime=15:04:05"`
}

// EffectiveScopeInput 生效范围输入
type EffectiveScopeInput struct {
	Type     string   `json:"type" binding:"required,oneof=all env_type specific_envs"`
	EnvTypes []string `json:"envTypes"`
	EnvIDs   []string `json:"envIDs"`
}

// UpdateAlertStrategyBody 更新告警策略请求体
type UpdateAlertStrategyBody struct {
	DisplayName        *string                  `json:"displayName"`
	Severity           *int                     `json:"severity" binding:"omitempty,oneof=1 2 3"`
	Threshold          *ThresholdConfigInput    `json:"threshold"`
	TriggerCondition   *TriggerConditionInput   `json:"triggerCondition"`
	RecoverCondition   *RecoverConditionInput   `json:"recoverCondition"`
	EffectiveTimeRange *EffectiveTimeRangeInput `json:"effectiveTimeRange"`
	EffectiveScope     *EffectiveScopeInput     `json:"effectiveScope"`
	NoticeGroupIDs     []int64                  `json:"noticeGroupIDs"`
	Enabled            *bool                    `json:"enabled"`
}

// ToUpdateReq 将更新请求体转换为领域层 UpdateReq。
func (b UpdateAlertStrategyBody) ToUpdateReq(operator string) (*alertstrategy.UpdateReq, error) {
	req := &alertstrategy.UpdateReq{
		DisplayName:    b.DisplayName,
		NoticeGroupIDs: b.NoticeGroupIDs,
		Enabled:        b.Enabled,
		Operator:       operator,
	}
	if b.Severity != nil {
		severity := alertstrategy.AlertSeverity(*b.Severity)
		req.Severity = &severity
	}
	if b.Threshold != nil {
		req.Threshold = &alertstrategy.ThresholdConfig{
			Method: b.Threshold.Method,
			Value:  b.Threshold.Value,
		}
	}
	if b.TriggerCondition != nil {
		req.TriggerCondition = &alertstrategy.TriggerCondition{
			Count:       b.TriggerCondition.Count,
			CheckWindow: b.TriggerCondition.CheckWindow,
		}
	}
	if b.RecoverCondition != nil {
		req.RecoverCondition = &alertstrategy.RecoverCondition{
			CheckWindow: b.RecoverCondition.CheckWindow,
		}
	}
	if b.EffectiveTimeRange != nil {
		req.EffectiveTimeRange = &alertstrategy.EffectiveTimeRange{
			StartTime: b.EffectiveTimeRange.StartTime,
			EndTime:   b.EffectiveTimeRange.EndTime,
		}
	}
	if b.EffectiveScope != nil {
		envIDs, err := parseEnvIDs(b.EffectiveScope.EnvIDs)
		if err != nil {
			return nil, errors.Wrap(err, "parse effective scope envIDs")
		}
		req.EffectiveScope = &alertstrategy.EffectiveScope{
			Type:     alertstrategy.EffectiveScopeType(b.EffectiveScope.Type),
			EnvTypes: b.EffectiveScope.EnvTypes,
			EnvIDs:   envIDs,
		}
	}
	return req, nil
}

// SwitchAlertStrategyBody 启停告警策略请求体
type SwitchAlertStrategyBody struct {
	Enabled bool `json:"enabled"`
}

// RemoteRefOutput 远端策略引用输出
type RemoteRefOutput struct {
	EnvID               string `json:"envID"`
	EnvName             string `json:"envName"`
	TrafficLaneName     string `json:"trafficLaneName,omitempty"`
	RemoteStrategyName  string `json:"remoteStrategyName"`
	BKMonitorStrategyID int64  `json:"bkMonitorStrategyID,string"`
}

// AlertStrategyOutput 告警策略输出
type AlertStrategyOutput struct {
	ID                 string                  `json:"id"`
	WorkspaceID        string                  `json:"workspaceID"`
	AppID              string                  `json:"appID"`
	AppName            string                  `json:"appName"`
	StrategyCode       string                  `json:"strategyCode"`
	DisplayName        string                  `json:"displayName"`
	MonitorMetric      string                  `json:"monitorMetric"`
	Severity           int                     `json:"severity"`
	Threshold          ThresholdConfigInput    `json:"threshold"`
	TriggerCondition   TriggerConditionInput   `json:"triggerCondition"`
	RecoverCondition   RecoverConditionInput   `json:"recoverCondition"`
	EffectiveTimeRange EffectiveTimeRangeInput `json:"effectiveTimeRange"`
	EffectiveScope     EffectiveScopeInput     `json:"effectiveScope"`
	NoticeGroupIDs     []int64                 `json:"noticeGroupIDs"`
	Enabled            bool                    `json:"enabled"`
	RemoteRefs         []RemoteRefOutput       `json:"remoteRefs"`
	Creator            string                  `json:"creator"`
	Updater            string                  `json:"updater"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
}

// FromModel 从领域模型转换为输出
func (o *AlertStrategyOutput) FromModel(strategy alertstrategy.AlertStrategy) *AlertStrategyOutput {
	envIDs := make([]string, 0, len(strategy.EffectiveScope.EnvIDs))
	for _, id := range strategy.EffectiveScope.EnvIDs {
		envIDs = append(envIDs, id.Hex())
	}

	remoteRefs := make([]RemoteRefOutput, 0, len(strategy.RemoteRefs))
	for _, ref := range strategy.RemoteRefs {
		remoteRefs = append(remoteRefs, RemoteRefOutput{
			EnvID:               ref.EnvID.Hex(),
			EnvName:             ref.EnvName,
			TrafficLaneName:     ref.TrafficLaneName,
			RemoteStrategyName:  ref.RemoteStrategyName,
			BKMonitorStrategyID: ref.RemoteStrategyID,
		})
	}

	*o = AlertStrategyOutput{
		ID:            strategy.ID.Hex(),
		WorkspaceID:   strategy.WorkspaceID,
		AppID:         strategy.AppID,
		AppName:       strategy.AppName,
		StrategyCode:  strategy.StrategyCode,
		DisplayName:   strategy.DisplayName,
		MonitorMetric: strategy.MonitorMetric,
		Severity:      int(strategy.Severity),
		Threshold: ThresholdConfigInput{
			Method: strategy.Threshold.Method,
			Value:  strategy.Threshold.Value,
		},
		TriggerCondition: TriggerConditionInput{
			Count:       strategy.TriggerCondition.Count,
			CheckWindow: strategy.TriggerCondition.CheckWindow,
		},
		RecoverCondition: RecoverConditionInput{
			CheckWindow: strategy.RecoverCondition.CheckWindow,
		},
		EffectiveTimeRange: EffectiveTimeRangeInput{
			StartTime: strategy.EffectiveTimeRange.StartTime,
			EndTime:   strategy.EffectiveTimeRange.EndTime,
		},
		EffectiveScope: EffectiveScopeInput{
			Type:     string(strategy.EffectiveScope.Type),
			EnvTypes: strategy.EffectiveScope.EnvTypes,
			EnvIDs:   envIDs,
		},
		NoticeGroupIDs: strategy.NoticeGroupIDs,
		Enabled:        strategy.Enabled,
		RemoteRefs:     remoteRefs,
		Creator:        strategy.Creator,
		Updater:        strategy.Updater,
		CreatedAt:      strategy.CreatedAt,
		UpdatedAt:      strategy.UpdatedAt,
	}
	return o
}

func parseEnvIDs(ids []string) ([]bson.ObjectID, error) {
	result := make([]bson.ObjectID, 0, len(ids))
	for _, id := range ids {
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		result = append(result, oid)
	}
	return result, nil
}

// CreateAlertStrategyResp 创建告警策略响应
type CreateAlertStrategyResp struct {
	Data *AlertStrategyOutput `json:"data"`
}

// GetAlertStrategyResp 获取告警策略响应
type GetAlertStrategyResp struct {
	Data *AlertStrategyOutput `json:"data"`
}

// ListAlertStrategiesOutput 告警策略列表输出
type ListAlertStrategiesOutput struct {
	Count   int64                  `json:"count,string"`
	Results []*AlertStrategyOutput `json:"results"`
}

// ListAlertStrategiesResp 告警策略列表响应
type ListAlertStrategiesResp struct {
	Data *ListAlertStrategiesOutput `json:"data"`
}
