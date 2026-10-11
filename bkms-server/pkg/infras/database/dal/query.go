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

package dal

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Query 是一次 Find。Sort/Skip/Limit 只影响 One 和 All，Count 只按 filter 计数。
type Query interface {
	Sort(sort bson.D) Query
	Skip(skip int64) Query
	Limit(limit int64) Query
	One(ctx context.Context, result any) error
	All(ctx context.Context, result any) error
	Count(ctx context.Context) (int64, error)
}

type mongoQuery struct {
	table  *mongoTable
	filter any
	sort   bson.D
	skip   *int64
	limit  *int64
}

func (q *mongoQuery) Sort(sort bson.D) Query {
	cloned := q.clone()
	cloned.sort = sort
	return cloned
}

func (q *mongoQuery) Skip(skip int64) Query {
	cloned := q.clone()
	cloned.skip = &skip
	return cloned
}

func (q *mongoQuery) Limit(limit int64) Query {
	cloned := q.clone()
	cloned.limit = &limit
	return cloned
}

func (q *mongoQuery) One(ctx context.Context, result any) error {
	filter, err := q.filterWithTenant(ctx)
	if err != nil {
		return err
	}
	opts := options.FindOne()
	if len(q.sort) > 0 {
		opts.SetSort(q.sort)
	}
	err = q.table.inner.FindOne(ctx, filter, opts).Decode(result)
	if err != nil {
		return err
	}
	return nil
}

func (q *mongoQuery) All(ctx context.Context, result any) error {
	filter, err := q.filterWithTenant(ctx)
	if err != nil {
		return err
	}
	opts := options.Find()
	if len(q.sort) > 0 {
		opts.SetSort(q.sort)
	}
	if q.skip != nil {
		opts.SetSkip(*q.skip)
	}
	if q.limit != nil {
		opts.SetLimit(*q.limit)
	}
	cursor, err := q.table.inner.Find(ctx, filter, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	if err = cursor.All(ctx, result); err != nil {
		return q.table.wrapErr(errors.Wrap(err, "decode documents"))
	}
	return nil
}

func (q *mongoQuery) Count(ctx context.Context) (int64, error) {
	filter, err := q.filterWithTenant(ctx)
	if err != nil {
		return 0, err
	}
	count, err := q.table.inner.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (q *mongoQuery) filterWithTenant(ctx context.Context) (any, error) {
	if err := q.table.scopeErr(); err != nil {
		return nil, err
	}
	if !q.table.shouldInject() {
		if q.filter == nil {
			return bson.M{}, nil
		}
		return q.filter, nil
	}
	tenantID, err := q.table.requireTenant(ctx)
	if err != nil {
		return nil, err
	}
	filter, err := setTenantField(q.filter, tenantID)
	if err != nil {
		return nil, q.table.wrapErr(err)
	}
	return filter, nil
}

func (q *mongoQuery) clone() *mongoQuery {
	cloned := *q
	return &cloned
}
