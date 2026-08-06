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

// Package semver 提供 Helm Chart semver 版本号生成功能（并发安全）
package semver

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// semverCounterCollName semver 计数器表名
const semverCounterCollName = "helm_chart_semver_counters"

// CounterStore semver 计数器存储接口
type CounterStore interface {
	// Next 原子递增并返回下一个 semver 版本号（格式：major.minor.patch）
	Next(ctx context.Context, appID string, bumpType BumpType) (string, error)
	// Get 查询指定应用的 semver counter 当前值，若记录不存在则返回零值
	Get(ctx context.Context, appID string) (*Counter, error)
}

var _ CounterStore = &CounterStoreMongo{}

// CounterStoreMongo CounterStore 的 MongoDB 实现
type CounterStoreMongo struct {
	collection *mongo.Collection
}

// NewCounterStoreMongo 创建 CounterStoreMongo 实例
func NewCounterStoreMongo(client *mongo.Client, dbName string) (*CounterStoreMongo, error) {
	coll := client.Database(dbName).Collection(semverCounterCollName)
	// _id 即为 appID，无需额外索引
	return &CounterStoreMongo{collection: coll}, nil
}

// Next 原子递增并返回下一个 semver 版本号
// 经典归零语义：递增 major 时 minor+patch 归零，递增 minor 时 patch 归零
// 初始版本：0.0.1（首次调用 BumpPatch 时）
func (s *CounterStoreMongo) Next(ctx context.Context, appID string, bumpType BumpType) (string, error) {
	filter := bson.M{"_id": appID}
	var update bson.M

	switch bumpType {
	case BumpPatch:
		update = bson.M{"$inc": bson.M{"patch": 1}}
	case BumpMinor:
		update = bson.M{
			"$inc": bson.M{"minor": 1},
			"$set": bson.M{"patch": 0},
		}
	case BumpMajor:
		update = bson.M{
			"$inc": bson.M{"major": 1},
			"$set": bson.M{"minor": 0, "patch": 0},
		}
	default:
		return "", errors.Errorf("unknown bump type: %s", bumpType)
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var c Counter
	if err := s.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&c); err != nil {
		return "", errors.Wrapf(err, "next semver for app %s", appID)
	}

	return c.FormatSemver(), nil
}

// Get 查询指定应用的 semver counter 当前值
// 若记录不存在则返回零值 counter（major=0, minor=0, patch=0），而非错误
func (s *CounterStoreMongo) Get(ctx context.Context, appID string) (*Counter, error) {
	filter := bson.M{"_id": appID}
	var c Counter
	err := s.collection.FindOne(ctx, filter).Decode(&c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return &Counter{AppID: appID}, nil
		}
		return nil, errors.Wrapf(err, "get semver for app %s", appID)
	}
	return &c, nil
}
