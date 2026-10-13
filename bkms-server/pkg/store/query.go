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
	"slices"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Opt 描述一次操作的过滤条件、排序与数量限制，多个 Opt 的过滤条件按 AND 合并。
//
// 领域包可基于 Where / Filter 定义具名的 Opt，让字段名只出现在一处，例如：
//
//	// 包 bkci 中
//	func ByWorkspace(workspaceID string) store.Opt { return store.Where("workspaceID", workspaceID) }
//
//	// 调用方
//	projects.Get(ctx, bkci.ByWorkspace(wsID))
//	pipelines.List(ctx, bkci.ByWorkspace(wsID), store.Desc("createdAt"), store.Limit(10))
type Opt func(*query)

type query struct {
	conds    bson.D
	matchAll bool
	sort     bson.D
	limit    int64
}

var (
	// errNoFilter 写操作必须带过滤条件，避免漏传条件时误改、误删整个集合
	errNoFilter = errors.New("filter is required for write operations, use store.All() to match all documents")
	// errSortLimit 排序与数量限制只对 Get / List 有意义；对其它操作静默忽略可能违背调用方预期，
	// 例如 DeleteMany(..., Limit(10)) 实际会删除全部匹配文档
	errSortLimit = errors.New("sort and limit are only supported by Get and List")
)

// Where 增加一个字段条件，value 可以是普通值，也可以是 bson.M{"$in": ...} 等操作符表达式
func Where(field string, value any) Opt {
	return func(q *query) { q.conds = append(q.conds, bson.E{Key: field, Value: value}) }
}

// Filter 将 m 中的每个条件加入过滤条件
func Filter(m bson.M) Opt {
	return func(q *query) {
		// 按 key 排序，保证生成的过滤条件与错误信息稳定
		keys := lo.Keys(m)
		slices.Sort(keys)
		for _, k := range keys {
			q.conds = append(q.conds, bson.E{Key: k, Value: m[k]})
		}
	}
}

// Combine 将多个 Opt 组合为一个，便于领域包在已有的具名 Opt 上构造新的，例如：
//
//	func ByWorkspaceAndType(workspaceID string, typ PipelineType) store.Opt {
//		return store.Combine(ByWorkspace(workspaceID), store.Where("type", typ))
//	}
func Combine(opts ...Opt) Opt {
	return func(q *query) {
		for _, opt := range opts {
			if opt != nil {
				opt(q)
			}
		}
	}
}

// All 显式声明匹配全部文档，写操作在没有任何过滤条件时必须使用它
func All() Opt {
	return func(q *query) { q.matchAll = true }
}

// Asc 按字段升序排序，多个排序条件按传入顺序生效，仅 Get / List 支持
func Asc(field string) Opt {
	return func(q *query) { q.sort = append(q.sort, bson.E{Key: field, Value: 1}) }
}

// Desc 按字段降序排序，多个排序条件按传入顺序生效，仅 Get / List 支持
func Desc(field string) Opt {
	return func(q *query) { q.sort = append(q.sort, bson.E{Key: field, Value: -1}) }
}

// Limit 限制返回的文档数量，n <= 0 表示不限制，仅 List 支持
func Limit(n int64) Opt {
	return func(q *query) { q.limit = n }
}

func newQuery(opts []Opt) *query {
	q := &query{}
	Combine(opts...)(q)
	return q
}

// filter 生成过滤条件。同一字段出现多次时（如分别指定上下限）改用 $and，避免后者覆盖前者
func (q *query) filter() bson.D {
	if len(q.conds) == 0 {
		return bson.D{}
	}
	keys := lo.Map(q.conds, func(e bson.E, _ int) string { return e.Key })
	if len(lo.Uniq(keys)) == len(keys) {
		return q.conds
	}
	return bson.D{{Key: "$and", Value: lo.Map(q.conds, func(e bson.E, _ int) bson.D { return bson.D{e} })}}
}

func (q *query) hasKey(key string) bool {
	return lo.ContainsBy(q.conds, func(e bson.E) bool { return e.Key == key })
}

func (q *query) hasSortOrLimit() bool {
	return len(q.sort) > 0 || q.limit > 0
}

// readQuery 用于不支持排序与数量限制的读操作（Exists / Count）
func (s *GenericStore[T]) readQuery(op string, opts []Opt) (*query, error) {
	q := newQuery(opts)
	if q.hasSortOrLimit() {
		return nil, errors.Wrapf(errSortLimit, "%s %s", op, s.coll.Name())
	}
	return q, nil
}

// writeQuery 用于写操作：必须带过滤条件（或显式 All），且不支持排序与数量限制
func (s *GenericStore[T]) writeQuery(op string, opts []Opt) (*query, error) {
	q := newQuery(opts)
	if len(q.conds) == 0 && !q.matchAll {
		return nil, errors.Wrapf(errNoFilter, "%s %s", op, s.coll.Name())
	}
	if q.hasSortOrLimit() {
		return nil, errors.Wrapf(errSortLimit, "%s %s", op, s.coll.Name())
	}
	return q, nil
}
