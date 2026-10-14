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

// Package build 提供 Helm Chart 构建触发和构建记录管理功能
package build

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrRecordNotFound Helm Chart 构建记录未找到时返回固定错误
var ErrRecordNotFound = errors.New("helm chart build record not found")

// RecordStore Helm Chart 构建记录存储
type RecordStore = store.GenericStore[Record]

// RecordMeta Helm Chart 构建记录的存储元数据，记录没有 createdAt / updatedAt，开始时间由 CreateRecord 设置。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + buildID
var RecordMeta = store.Meta[Record]{
	Collection: "helm_chart_build_records",
	NotFound:   ErrRecordNotFound,
}

// RecordCounterCollection Helm Chart 构建序号计数器的集合名，文档的 _id 为应用 ID
const RecordCounterCollection = "helm_chart_build_counters"

// ByApp 按应用 ID 匹配构建记录
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ByBuild 按应用 ID 与蓝盾构建 ID 匹配构建记录
func ByBuild(appID, buildID string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("buildID", buildID))
}

// KeywordMatches 按 Chart 版本号、触发人模糊匹配构建记录，keyword 为空时不过滤
func KeywordMatches(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "chartVersion", "operator")
}

// NewestFirst 按开始时间倒序排列构建记录
func NewestFirst() store.Opt {
	return store.Desc("startedAt")
}

// CreateRecord 为记录分配该应用的下一个构建序号（原子操作，并发安全）、设置开始时间后创建构建记录
func CreateRecord(ctx context.Context, db *mongo.Database, record *Record) error {
	num, err := store.NewCounter(db, RecordCounterCollection).Next(ctx, record.AppID)
	if err != nil {
		return errors.Wrapf(err, "get next build num for app %s", record.AppID)
	}
	record.Num = num
	record.StartedAt = time.Now()
	if err = store.Of(db, RecordMeta).Create(ctx, record); err != nil {
		return errors.Wrapf(err, "create helm chart build record for app %s build %s", record.AppID, record.BuildID)
	}
	return nil
}

// UpdateRecord 更新已存在的构建记录（状态、extras、结束时间），记录不存在时返回 ErrRecordNotFound
func UpdateRecord(ctx context.Context, records *RecordStore, record *Record) error {
	if err := records.Update(
		ctx, record, store.UpdateSelected("status", "extras", "endedAt"), ByBuild(record.AppID, record.BuildID),
	); err != nil {
		return errors.Wrapf(err, "update helm chart build record for app %s build %s", record.AppID, record.BuildID)
	}
	return nil
}
