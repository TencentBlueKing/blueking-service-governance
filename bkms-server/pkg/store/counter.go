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

package store

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Counter 基于集合的自增计数器，文档结构为 {_id: key, seq: <int64>}
type Counter struct {
	coll *mongo.Collection
}

// NewCounter 创建基于 collection 的计数器
func NewCounter(db *mongo.Database, collection string) *Counter {
	return &Counter{coll: db.Collection(collection)}
}

// Next 将 key 对应的计数加一并返回新值，key 不存在时从 1 开始
func (c *Counter) Next(ctx context.Context, key string) (int64, error) {
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var doc struct {
		Seq int64 `bson:"seq"`
	}
	for attempt := 0; ; attempt++ {
		err := c.coll.FindOneAndUpdate(ctx, bson.M{"_id": key}, bson.M{"$inc": bson.M{"seq": 1}}, opts).Decode(&doc)
		if err == nil {
			return doc.Seq, nil
		}
		// 并发 upsert 同一个新 key 时只有一个能插入成功，其余会遇到唯一键冲突，重试即可命中已插入的文档
		if mongo.IsDuplicateKeyError(err) && attempt == 0 {
			continue
		}
		return 0, errors.Wrapf(err, "increase counter %q in %s", key, c.coll.Name())
	}
}
