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
)

// counterDoc 计数器文档
type counterDoc struct {
	Key      string `bson:"_id"`
	TenantID string `bson:"tenantID"`
	Seq      int64  `bson:"seq"`
}

// Counter 基于集合的按租户自增计数器，文档结构为 {_id: key, tenantID: <ctx 中的租户>, seq: <int64>}。
// _id 仍只是 key，因此 key 必须跨租户唯一，如应用 ID
type Counter struct {
	store *GenericStore[counterDoc]
}

// NewCounter 创建基于 collection 的计数器
func NewCounter(db *mongo.Database, collection string) *Counter {
	return &Counter{store: Of(db, Meta[counterDoc]{Collection: collection, TenantScoped: true})}
}

// Next 将 ctx 中租户下 key 对应的计数加一并返回新值，key 不存在时从 1 开始。
// ctx 中没有租户时返回 ErrTenantIDRequired；ctx 被 RequireExplicitTenantScope 标记过时返回 ErrTenantScopeRequired；
// key 已被其它租户占用时返回 ErrDuplicate
func (c *Counter) Next(ctx context.Context, key string) (int64, error) {
	doc, err := c.store.ModifyOrCreate(ctx, bson.M{"$inc": bson.M{"seq": 1}}, ByID(key))
	if err != nil {
		return 0, errors.Wrapf(err, "increase counter %q", key)
	}
	return doc.Seq, nil
}
