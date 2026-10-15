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
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Delete 删除匹配的一个文档，不存在时返回 NotFound 错误；需要忽略不存在的场景请使用 DeleteIfExists。
// 支持的选项：CondOpt
func (s *GenericStore[T]) Delete(ctx context.Context, opts ...Opt) error {
	q, err := s.writeQuery("delete from", opts, condOpts)
	if err != nil {
		return err
	}
	n, err := s.deleteOne(ctx, q)
	if err != nil {
		return err
	}
	if n == 0 {
		return s.notFound(q)
	}
	return nil
}

// DeleteIfExists 删除匹配的一个文档，不存在时直接返回 nil。支持的选项：CondOpt
func (s *GenericStore[T]) DeleteIfExists(ctx context.Context, opts ...Opt) error {
	q, err := s.writeQuery("delete from", opts, condOpts)
	if err != nil {
		return err
	}
	_, err = s.deleteOne(ctx, q)
	return err
}

// GetAndDelete 原子地删除匹配的一个文档并返回被删除的文档（已经过 Decode），可通过 Asc / Desc 指定删除哪一个；
// 不存在时返回 NotFound 错误。适用于删除后仍需使用原文档的场景（如记录审计）。支持的选项：CondOpt、SortOpt
func (s *GenericStore[T]) GetAndDelete(ctx context.Context, opts ...Opt) (*T, error) {
	q, err := s.writeQuery("find and delete from", opts, getAndDeleteOpts)
	if err != nil {
		return nil, err
	}
	findOpts := options.FindOneAndDelete()
	if len(q.sort) > 0 {
		findOpts.SetSort(q.sort)
	}

	var obj T
	err = s.coll.FindOneAndDelete(ctx, q.filter(), findOpts).Decode(&obj)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, s.notFound(q)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "find and delete from %s, filter %v", s.coll.Name(), q.filter())
	}
	if err := s.decode(&obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// DeleteMany 删除所有匹配的文档，返回删除的数量。支持的选项：CondOpt
func (s *GenericStore[T]) DeleteMany(ctx context.Context, opts ...Opt) (int64, error) {
	q, err := s.writeQuery("delete many from", opts, condOpts)
	if err != nil {
		return 0, err
	}
	res, err := s.coll.DeleteMany(ctx, q.filter())
	if err != nil {
		return 0, errors.Wrapf(err, "delete many from %s, filter %v", s.coll.Name(), q.filter())
	}
	return res.DeletedCount, nil
}

func (s *GenericStore[T]) deleteOne(ctx context.Context, q *query) (int64, error) {
	res, err := s.coll.DeleteOne(ctx, q.filter())
	if err != nil {
		return 0, errors.Wrapf(err, "delete from %s, filter %v", s.coll.Name(), q.filter())
	}
	return res.DeletedCount, nil
}
