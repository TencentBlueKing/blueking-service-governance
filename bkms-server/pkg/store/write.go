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
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const duplicateKeyCode = 11000

// Create 插入一个文档。ID（零值时）、时间戳与租户会回填到 obj 中；唯一键冲突时返回 Duplicate 错误。
// 支持的选项：TenantOf
func (s *GenericStore[T]) Create(ctx context.Context, obj *T, opts ...Opt) error {
	tenantID, err := s.insertTenant(ctx, "insert into", opts)
	if err != nil {
		return err
	}
	doc, err := s.prepareInsert(obj, tenantID, time.Now())
	if err != nil {
		return err
	}
	if _, err := s.coll.InsertOne(ctx, doc); err != nil {
		return s.wrapWriteErr(err, "insert into")
	}
	return nil
}

// CreateMany 批量插入文档，遇到唯一键冲突时返回 Duplicate 错误（冲突之前的文档已写入）。
// 支持的选项：TenantOf
func (s *GenericStore[T]) CreateMany(ctx context.Context, objs []*T, opts ...Opt) error {
	docs, err := s.prepareInsertMany(ctx, "insert many into", objs, opts)
	if err != nil || len(docs) == 0 {
		return err
	}
	if _, err := s.coll.InsertMany(ctx, docs); err != nil {
		return s.wrapWriteErr(err, "insert many into")
	}
	return nil
}

// CreateManyIfAbsent 批量插入文档并跳过已存在（唯一键冲突）的文档。支持的选项：TenantOf。
// 注意：被跳过的对象同样会被回填 ID 与时间戳，但它们与数据库中已有的文档并不一致
func (s *GenericStore[T]) CreateManyIfAbsent(ctx context.Context, objs []*T, opts ...Opt) error {
	docs, err := s.prepareInsertMany(ctx, "insert many into", objs, opts)
	if err != nil || len(docs) == 0 {
		return err
	}
	_, err = s.coll.InsertMany(ctx, docs, options.InsertMany().SetOrdered(false))
	if err == nil || onlyDuplicateKeyErrors(err) {
		return nil
	}
	return s.wrapWriteErr(err, "insert many into")
}

// Replace 用 obj 整体替换匹配的一个文档，并更新 obj 的 updatedAt；不存在时返回 NotFound 错误。
// obj 应是完整的文档（通常来自 Get），其 createdAt 会原样写入；obj 的 _id 被忽略，文档保留原有 _id。
// 支持的选项：CondOpt、TenantOf
func (s *GenericStore[T]) Replace(ctx context.Context, obj *T, opts ...Opt) error {
	q, err := s.writeQuery(ctx, "replace in", opts, replaceOpts)
	if err != nil {
		return err
	}
	// Replace 写入整个文档，租户必须是确定值，否则会把被替换文档的租户抹掉
	tenantID, err := s.tenantForWrite(ctx, "replace in", q)
	if err != nil {
		return err
	}
	if err = s.beforeSave(obj); err != nil {
		return err
	}
	v := reflect.ValueOf(obj).Elem()
	s.schema.setUpdated(v, time.Now())
	s.schema.fillTenant(v, tenantID)
	doc, err := s.encodeDoc(obj, keyID)
	if err != nil {
		return err
	}
	res, err := s.coll.ReplaceOne(ctx, q.filter(), doc)
	if err != nil {
		return s.wrapWriteErr(err, "replace in")
	}
	if res.MatchedCount == 0 {
		return s.notFound(q)
	}
	return nil
}

func (s *GenericStore[T]) updateOne(ctx context.Context, q *query, update bson.M) error {
	res, err := s.coll.UpdateOne(ctx, q.filter(), update)
	if err != nil {
		return s.wrapWriteErr(err, "update")
	}
	if res.MatchedCount == 0 {
		return s.notFound(q)
	}
	return nil
}

// prepareInsert 依次执行 BeforeSave、回填 ID、时间戳与租户、Encode，返回待插入的文档。
// tenantID 为空（全局集合）时不写租户，非空时覆盖 obj 中已有的值，调用方伪造不了租户
func (s *GenericStore[T]) prepareInsert(obj *T, tenantID string, now time.Time) (*T, error) {
	if err := s.beforeSave(obj); err != nil {
		return nil, err
	}
	v := reflect.ValueOf(obj).Elem()
	s.schema.fillID(v)
	s.schema.fillCreated(v, now)
	s.schema.fillTenant(v, tenantID)
	return s.encode(obj)
}

// insertTenant 解析插入类操作的租户。插入没有过滤条件，只需要从 opts 中取出租户范围
func (s *GenericStore[T]) insertTenant(ctx context.Context, op string, opts []Opt) (string, error) {
	q, err := s.parseOpts(ctx, op, opts, insertOpts)
	if err != nil {
		return "", err
	}
	return s.tenantForWrite(ctx, op, q)
}

func (s *GenericStore[T]) prepareInsertMany(ctx context.Context, op string, objs []*T, opts []Opt) ([]any, error) {
	tenantID, err := s.insertTenant(ctx, op, opts)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	docs := make([]any, 0, len(objs))
	for _, obj := range objs {
		doc, err := s.prepareInsert(obj, tenantID, now)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// onlyDuplicateKeyErrors 判断批量写入的错误是否全部是唯一键冲突
func onlyDuplicateKeyErrors(err error) bool {
	var bwe mongo.BulkWriteException
	if !errors.As(err, &bwe) || bwe.WriteConcernError != nil || len(bwe.WriteErrors) == 0 {
		return false
	}
	return lo.EveryBy(bwe.WriteErrors, func(we mongo.BulkWriteError) bool { return we.Code == duplicateKeyCode })
}
