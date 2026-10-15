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

// Create 插入一个文档。ID（零值时）与时间戳会回填到 obj 中；唯一键冲突时返回 Duplicate 错误
func (s *GenericStore[T]) Create(ctx context.Context, obj *T) error {
	doc, err := s.prepareInsert(obj, time.Now())
	if err != nil {
		return err
	}
	if _, err := s.coll.InsertOne(ctx, doc); err != nil {
		return s.wrapWriteErr(err, "insert into")
	}
	return nil
}

// CreateMany 批量插入文档，遇到唯一键冲突时返回 Duplicate 错误（冲突之前的文档已写入）
func (s *GenericStore[T]) CreateMany(ctx context.Context, objs []*T) error {
	docs, err := s.prepareInsertMany(objs)
	if err != nil || len(docs) == 0 {
		return err
	}
	if _, err := s.coll.InsertMany(ctx, docs); err != nil {
		return s.wrapWriteErr(err, "insert many into")
	}
	return nil
}

// CreateManyIfAbsent 批量插入文档并跳过已存在（唯一键冲突）的文档。
// 注意：被跳过的对象同样会被回填 ID 与时间戳，但它们与数据库中已有的文档并不一致
func (s *GenericStore[T]) CreateManyIfAbsent(ctx context.Context, objs []*T) error {
	docs, err := s.prepareInsertMany(objs)
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
// 支持的选项：CondOpt
func (s *GenericStore[T]) Replace(ctx context.Context, obj *T, opts ...Opt) error {
	q, err := s.writeQuery("replace in", opts, condOpts)
	if err != nil {
		return err
	}
	if err = s.beforeSave(obj); err != nil {
		return err
	}
	s.schema.setUpdated(reflect.ValueOf(obj).Elem(), time.Now())
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

// prepareInsert 依次执行 BeforeSave、回填 ID 与时间戳、Encode，返回待插入的文档
func (s *GenericStore[T]) prepareInsert(obj *T, now time.Time) (*T, error) {
	if err := s.beforeSave(obj); err != nil {
		return nil, err
	}
	v := reflect.ValueOf(obj).Elem()
	s.schema.fillID(v)
	s.schema.fillCreated(v, now)
	return s.encode(obj)
}

func (s *GenericStore[T]) prepareInsertMany(objs []*T) ([]any, error) {
	now := time.Now()
	docs := make([]any, 0, len(objs))
	for _, obj := range objs {
		doc, err := s.prepareInsert(obj, now)
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
