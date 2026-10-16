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

package build

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrConfigNotFound 构建配置未找到
	ErrConfigNotFound = errors.New("build config not found")
	// ErrRecordNotFound 构建记录未找到。调用方据此区分「记录不存在」与 DB 瞬时故障
	ErrRecordNotFound = errors.New("build record not found")
)

// ConfigStore 构建配置存储
type ConfigStore = store.GenericStore[Config]

// ConfigMeta 构建配置的存储元数据，镜像凭证密码在库中对称加密存储。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID
var ConfigMeta = store.Meta[Config]{
	Collection: "build_configs",
	NotFound:   ErrConfigNotFound,
	Encode:     []func(*Config) error{encryptImagePassword},
	Decode:     []func(*Config) error{decryptImagePassword},
}

func encryptImagePassword(cfg *Config) (err error) {
	if cfg.Image != nil {
		cfg.Image.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESEncrypt, cfg.Image.Password)
	}
	return errors.Wrap(err, "encrypt image password")
}

func decryptImagePassword(cfg *Config) (err error) {
	if cfg.Image != nil {
		cfg.Image.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESDecrypt, cfg.Image.Password)
	}
	return errors.Wrap(err, "decrypt image password")
}

// ByApp 按应用 ID 匹配构建配置或构建记录
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// UpdateConfig 更新已存在的应用构建配置的全部可变字段（appID 与 createdAt 不可变），
// cfg 中为空的可选配置会从库中清除；配置不存在时返回 ErrConfigNotFound
func UpdateConfig(ctx context.Context, configs *ConfigStore, cfg *Config) error {
	if err := configs.Update(ctx, cfg, store.UpdateOmit("appID"), ByApp(cfg.AppID)); err != nil {
		return errors.Wrapf(err, "update app %s build config", cfg.AppID)
	}
	return nil
}

// RecordStore 构建记录存储
type RecordStore = store.GenericStore[Record]

// RecordMeta 构建记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + buildID
var RecordMeta = store.Meta[Record]{
	Collection: "build_records",
	NotFound:   ErrRecordNotFound,
}

// RecordCounterCollection 构建序号计数器的集合名，文档的 _id 为应用 ID
const RecordCounterCollection = "build_counters"

// ByBuild 按应用 ID 与构建 ID 匹配构建记录
func ByBuild(appID, buildID string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("buildID", buildID))
}

// RecordKeywordMatches 按制品信息、操作人模糊匹配构建记录，keyword 为空时不过滤
func RecordKeywordMatches(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "artifact", "operator")
}

// CreateRecord 为记录分配该应用的下一个构建序号（原子操作，并发安全）后创建构建记录
func CreateRecord(ctx context.Context, db *mongo.Database, record *Record) error {
	num, err := store.NewCounter(db, RecordCounterCollection).Next(ctx, record.AppID)
	if err != nil {
		return errors.Wrapf(err, "get next build num for app %s", record.AppID)
	}
	record.Num = num
	if err = store.Of(db, RecordMeta).Create(ctx, record); err != nil {
		return errors.Wrapf(err, "create app %s build record %s", record.AppID, record.BuildID)
	}
	return nil
}

// UpdateRecord 更新已存在的应用构建记录的序号、状态、额外信息与结束时间，记录不存在时返回 ErrRecordNotFound
func UpdateRecord(ctx context.Context, records *RecordStore, record *Record) error {
	if err := records.Update(
		ctx, record, store.UpdateSelected("num", "status", "extras", "endedAt"), ByBuild(record.AppID, record.BuildID),
	); err != nil {
		return errors.Wrapf(err, "update app %s build record %s", record.AppID, record.BuildID)
	}
	return nil
}
