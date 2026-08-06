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

// Package strategy 提供蓝鲸监控告警策略相关功能
package strategy

import (
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// EffectiveScopeType 告警策略生效范围类型
type EffectiveScopeType string

const (
	// EffectiveScopeAll 所有环境（包括新增的环境也会自动生效）
	EffectiveScopeAll EffectiveScopeType = "all"
	// EffectiveScopeEnvType 按环境类型（development/test/production）
	EffectiveScopeEnvType EffectiveScopeType = "env_type"
	// EffectiveScopeSpecificEnvs 特定环境
	EffectiveScopeSpecificEnvs EffectiveScopeType = "specific_envs"
)

// AlertSeverity 告警级别
type AlertSeverity int

const (
	// AlertSeverityFatal 致命
	AlertSeverityFatal AlertSeverity = 1
	// AlertSeverityWarning 预警
	AlertSeverityWarning AlertSeverity = 2
	// AlertSeverityInfo 提醒
	AlertSeverityInfo AlertSeverity = 3
)

// EffectiveScope 生效范围
type EffectiveScope struct {
	// Type 生效范围类型
	Type EffectiveScopeType `bson:"type" validate:"required,oneof=all env_type specific_envs"`
	// EnvTypes 按环境类型生效时的类型列表（development/test/production）
	EnvTypes []string `bson:"envTypes,omitempty"`
	// EnvIDs 特定环境生效时的环境 ID 列表
	EnvIDs []bson.ObjectID `bson:"envIDs,omitempty"`
}

// Validate 校验生效范围结构是否合法。
func (s EffectiveScope) Validate() error {
	switch s.Type {
	case EffectiveScopeAll:
		if len(s.EnvTypes) > 0 || len(s.EnvIDs) > 0 {
			return errors.Errorf("effectiveScope type %q must not contain envTypes or envIDs", s.Type)
		}
	case EffectiveScopeEnvType:
		if len(s.EnvTypes) == 0 {
			return errors.Errorf("effectiveScope type %q requires non-empty envTypes", s.Type)
		}
		if len(s.EnvIDs) > 0 {
			return errors.Errorf("effectiveScope type %q must not contain envIDs", s.Type)
		}
	case EffectiveScopeSpecificEnvs:
		if len(s.EnvIDs) == 0 {
			return errors.Errorf("effectiveScope type %q requires non-empty envIDs", s.Type)
		}
		if len(s.EnvTypes) > 0 {
			return errors.Errorf("effectiveScope type %q must not contain envTypes", s.Type)
		}
	}
	return nil
}

// EffectiveTimeRange 生效时间段
type EffectiveTimeRange struct {
	StartTime string `bson:"startTime" validate:"omitempty,datetime=15:04:05"`
	EndTime   string `bson:"endTime" validate:"omitempty,datetime=15:04:05"`
}

// Validate 校验生效时间段格式是否合法。
func (r EffectiveTimeRange) Validate() error {
	return validator.New().Struct(r)
}

// TriggerCondition 触发条件
type TriggerCondition struct {
	Count       int `bson:"count"`
	CheckWindow int `bson:"checkWindow"`
}

// RecoverCondition 恢复条件
type RecoverCondition struct {
	CheckWindow int `bson:"checkWindow"`
}

// ThresholdConfig 阈值配置
type ThresholdConfig struct {
	Method string  `bson:"method" validate:"required,oneof=gte gt lte lt eq neq"`
	Value  float64 `bson:"value"`
}

// RemoteStrategyRef 记录一条远端策略与本地策略的映射关系。
// 同一环境下，不同流量泳道会维护各自独立的远端引用。
type RemoteStrategyRef struct {
	EnvID              bson.ObjectID `bson:"envID"`
	EnvName            string        `bson:"envName"`
	TrafficLaneName    string        `bson:"trafficLaneName"`
	RemoteStrategyName string        `bson:"remoteStrategyName"`
	RemoteStrategyID   int64         `bson:"remoteStrategyID"`
}

// AlertStrategy 应用级告警策略，BKMS 本地主模型
type AlertStrategy struct {
	// ID 策略 ID。
	ID bson.ObjectID `bson:"_id,omitempty"`

	// WorkspaceID 所属工作空间 ID。
	WorkspaceID string `bson:"workspaceID" validate:"required"`
	// AppID 所属应用 ID。
	AppID string `bson:"appID" validate:"required"`
	// AppName 所属应用名称。
	AppName string `bson:"appName" validate:"required"`
	// StrategyCode 策略模板标识，对应蓝鲸监控的策略类型/算法模板。
	StrategyCode string `bson:"strategyCode" validate:"required"`
	// DisplayName 策略展示名称。
	DisplayName string `bson:"displayName" validate:"required"`

	// MonitorMetric 监控指标（如 CPU、内存、请求错误率等）。
	MonitorMetric string `bson:"monitorMetric"`
	// Severity 告警级别。
	Severity AlertSeverity `bson:"severity"`
	// Threshold 阈值配置。
	Threshold ThresholdConfig `bson:"threshold"`

	// TriggerCondition 告警触发条件。
	TriggerCondition TriggerCondition `bson:"triggerCondition"`
	// RecoverCondition 告警恢复条件。
	RecoverCondition RecoverCondition `bson:"recoverCondition"`
	// EffectiveTimeRange 生效时间段。
	EffectiveTimeRange EffectiveTimeRange `bson:"effectiveTimeRange"`
	// EffectiveScope 生效范围（全部环境 / 按环境类型 / 指定环境）。
	EffectiveScope EffectiveScope `bson:"effectiveScope"`

	// NoticeGroupIDs 通知用户组 ID 列表。
	NoticeGroupIDs []int64 `bson:"noticeGroupIDs"`
	// Enabled 是否启用该策略。
	Enabled bool `bson:"enabled"`
	// RemoteRefs 与蓝鲸监控远端策略的引用列表，按环境+泳道记录远端策略 ID。
	RemoteRefs []RemoteStrategyRef `bson:"remoteRefs"`

	// Creator 创建者。
	Creator string `bson:"creator"`
	// CreatedAt 创建时间。
	CreatedAt time.Time `bson:"createdAt"`
	// Updater 更新者。
	Updater string `bson:"updater"`
	// UpdatedAt 更新时间。
	UpdatedAt time.Time `bson:"updatedAt"`
}

// CreateReq 创建告警策略请求
type CreateReq struct {
	WorkspaceID        string
	AppID              string
	AppName            string
	StrategyCode       string
	DisplayName        string
	Severity           AlertSeverity
	Threshold          ThresholdConfig
	TriggerCondition   TriggerCondition
	RecoverCondition   RecoverCondition
	EffectiveTimeRange EffectiveTimeRange
	EffectiveScope     EffectiveScope
	NoticeGroupIDs     []int64
	Enabled            bool
	Operator           string
}

// UpdateReq 更新告警策略请求
type UpdateReq struct {
	DisplayName        *string
	Severity           *AlertSeverity
	Threshold          *ThresholdConfig
	TriggerCondition   *TriggerCondition
	RecoverCondition   *RecoverCondition
	EffectiveTimeRange *EffectiveTimeRange
	EffectiveScope     *EffectiveScope
	NoticeGroupIDs     []int64
	Enabled            *bool
	Operator           string
}

// ToBSON converts UpdateReq to a MongoDB update document.
func (r *UpdateReq) ToBSON() (bson.M, bool, error) {
	updateData := bson.M{}
	changed := false

	setIfPresent := func(bsonField string, present bool, value any) {
		if present {
			updateData[bsonField] = value
			changed = true
		}
	}

	setIfPresent("displayName", r.DisplayName != nil, lo.FromPtr(r.DisplayName))
	setIfPresent("severity", r.Severity != nil, lo.FromPtr(r.Severity))
	setIfPresent("threshold", r.Threshold != nil, lo.FromPtr(r.Threshold))
	setIfPresent("triggerCondition", r.TriggerCondition != nil, lo.FromPtr(r.TriggerCondition))
	setIfPresent("recoverCondition", r.RecoverCondition != nil, lo.FromPtr(r.RecoverCondition))
	if r.EffectiveTimeRange != nil {
		if err := r.EffectiveTimeRange.Validate(); err != nil {
			return nil, false, errors.Wrap(err, "validate effectiveTimeRange")
		}
		updateData["effectiveTimeRange"] = *r.EffectiveTimeRange
		changed = true
	}
	if r.EffectiveScope != nil {
		if err := r.EffectiveScope.Validate(); err != nil {
			return nil, false, errors.Wrap(err, "validate effectiveScope")
		}
		updateData["effectiveScope"] = *r.EffectiveScope
		changed = true
	}
	if r.NoticeGroupIDs != nil {
		updateData["noticeGroupIDs"] = r.NoticeGroupIDs
		changed = true
	}
	setIfPresent("enabled", r.Enabled != nil, lo.FromPtr(r.Enabled))

	return updateData, changed, nil
}

// ApplyTo 将 UpdateReq 中已传入的字段覆盖到目标告警策略对象。
func (r *UpdateReq) ApplyTo(strategy *AlertStrategy) {
	applyIfPresent(r.DisplayName, &strategy.DisplayName)
	applyIfPresent(r.Severity, &strategy.Severity)
	applyIfPresent(r.Threshold, &strategy.Threshold)
	applyIfPresent(r.TriggerCondition, &strategy.TriggerCondition)
	applyIfPresent(r.RecoverCondition, &strategy.RecoverCondition)
	applyIfPresent(r.EffectiveTimeRange, &strategy.EffectiveTimeRange)
	applyIfPresent(r.EffectiveScope, &strategy.EffectiveScope)
	applyIfPresentSlice(r.NoticeGroupIDs, &strategy.NoticeGroupIDs)
	applyIfPresent(r.Enabled, &strategy.Enabled)
	strategy.Updater = r.Operator
}

// applyIfPresent 在部分更新场景下，仅当源指针非空时才将值拷贝到目标指针指向的对象
func applyIfPresent[T any](src, dst *T) {
	if src != nil {
		*dst = *src
	}
}

// applyIfPresentSlice 在部分更新场景下，仅当源切片非 nil 时才将其深拷贝到目标切片
// 使用 append([]T(nil), src...) 创建独立副本，避免源与目标共享底层数组
func applyIfPresentSlice[T any](src []T, dst *[]T) {
	if src != nil {
		*dst = append([]T(nil), src...)
	}
}

// DefaultTemplate 预置告警策略模板
type DefaultTemplate struct {
	// StrategyCode 模板标识，用于绑定默认模板与 PromQL 生成逻辑。
	StrategyCode string
	// DisplayName 模板默认展示名称。
	DisplayName string
	// MonitorMetric 模板默认使用的监控指标。
	MonitorMetric string
	// Severity 模板默认告警级别。
	Severity AlertSeverity
	// Threshold 模板默认阈值配置。
	Threshold ThresholdConfig
	// TriggerCondition 模板默认触发条件。
	TriggerCondition TriggerCondition
	// RecoverCondition 模板默认恢复条件。
	RecoverCondition RecoverCondition
	// EffectiveTimeRange 模板默认生效时间段。
	EffectiveTimeRange EffectiveTimeRange
}

type strategyLockEntry struct {
	mu   sync.Mutex
	refs int
}

type strategyLockTable struct {
	mu      sync.Mutex
	entries map[string]*strategyLockEntry
}

// strategyMu 按策略 ID 维护互斥锁，确保同一条策略的并发同步/清理操作串行执行，
// 防止并发部署多个环境时 remoteRefs 互相覆盖。key: strategyID hex string, value: *strategyLockEntry
//
// TODO(alertstrategy): 当前互斥锁为进程内锁，仅能保护单个 worker Pod 内的并发。
// 当多个 worker Pod 同时处理同一应用的不同环境部署时，跨 Pod 并发仍存在 remoteRefs
// 互相覆盖的风险。后续应将策略同步/清理迁移到 asynq 任务队列（taskq 框架），
// 利用 asynq 的唯一任务（UniqueTask）或队列串行化能力，确保同一条策略的操作
// 在分布式场景下也能串行执行。
var strategyMu = strategyLockTable{
	entries: map[string]*strategyLockEntry{},
}

// Service 告警策略业务逻辑层
type Service struct {
	store         Store
	envStore      envmodel.EnvironmentStore
	appStore      bkmsapp.ApplicationStore
	snapshotStore topology.ResourceSnapshotStore
	newClient     alert.ClientFactory
}
