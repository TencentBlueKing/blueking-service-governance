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
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrNotFound 告警策略未找到
var ErrNotFound = errors.New("AlertStrategy not found")

// Store 告警策略存储
type Store = store.GenericStore[AlertStrategy]

// Meta 告警策略模型的存储元信息。
//
// 集合 bkmonitor_alert_strategy 在 bkms 中保存告警策略记录，
// 存储策略与 workspace/app 的关联范围、启用状态、strategyCode 等本地元数据，用于默认策略下发或应用部署后策略下发。
//
// 索引（由 golang-migrate 维护）：
// - 查询提速：workspaceID + appID + enabled
// - 查询提速：workspaceID + appID + remoteRefs.envID + remoteRefs.trafficLaneName
// - 唯一约束：workspaceID + appID + displayName（集合中唯一的唯一索引，因此冲突即表示策略名称重复）
var Meta = store.Meta[AlertStrategy]{
	Collection: "bkmonitor_alert_strategy",
	NotFound:   ErrNotFound,
	Duplicate:  ErrDisplayNameAlreadyExists,
	BeforeSave: []func(*AlertStrategy) error{validateStrategy, defaultEmptySlices},
}

// validateStrategy 写入完整策略前校验结构体与生效范围
func validateStrategy(rule *AlertStrategy) error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(rule); err != nil {
		return errors.Wrap(err, "AlertStrategy validation failed")
	}
	if err := rule.EffectiveScope.Validate(); err != nil {
		return errors.Wrap(err, "AlertStrategy effectiveScope validation failed")
	}
	return nil
}

// defaultEmptySlices 将 nil 切片补为空切片，使文档中保存 [] 而不是 null
func defaultEmptySlices(rule *AlertStrategy) error {
	if rule.RemoteRefs == nil {
		rule.RemoteRefs = make([]RemoteStrategyRef, 0)
	}
	if rule.NoticeGroupIDs == nil {
		rule.NoticeGroupIDs = make([]int64, 0)
	}
	return nil
}

// ByWorkspace 匹配指定工作空间下的告警策略
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByApp 匹配指定工作空间下某应用的告警策略
func ByApp(workspaceID, appID string) store.Opt {
	return store.Filter(bson.M{"workspaceID": workspaceID, "appID": appID})
}

// ByDisplayName 匹配同一工作空间、同一应用下指定展示名称的策略；excludeID 非空时排除该策略自身
func ByDisplayName(workspaceID, appID, displayName string, excludeID *bson.ObjectID) store.Opt {
	opts := []store.Opt{ByApp(workspaceID, appID), store.Where("displayName", displayName)}
	if excludeID != nil && !excludeID.IsZero() {
		opts = append(opts, store.Where("_id", bson.M{"$ne": *excludeID}))
	}
	return store.Combine(opts...)
}

// ByRemoteEnv 匹配某应用下、关联了指定远程环境与泳道的告警策略
func ByRemoteEnv(workspaceID, appID string, envID bson.ObjectID, trafficLaneName string) store.Opt {
	return store.Combine(
		ByApp(workspaceID, appID),
		store.Where("remoteRefs", bson.M{
			"$elemMatch": bson.M{
				"envID":           envID,
				"trafficLaneName": trafficLaneName,
			},
		}),
	)
}

// EnabledMatchingEnv 匹配某应用中已启用、且生效范围覆盖给定环境的告警策略
func EnabledMatchingEnv(workspaceID, appID, envType string, envID bson.ObjectID) store.Opt {
	return store.Combine(
		ByApp(workspaceID, appID),
		store.Filter(bson.M{
			"enabled": true,
			"$or": bson.A{
				bson.M{"effectiveScope.type": EffectiveScopeAll},
				bson.M{"effectiveScope.type": EffectiveScopeEnvType, "effectiveScope.envTypes": envType},
				bson.M{"effectiveScope.type": EffectiveScopeSpecificEnvs, "effectiveScope.envIDs": envID},
			},
		}),
	)
}

// NewestFirst 按创建时间倒序排列，创建时间相同时按 _id 倒序，保证顺序稳定
func NewestFirst() store.Opt {
	return store.Combine(store.Desc("createdAt"), store.Desc("_id"))
}
