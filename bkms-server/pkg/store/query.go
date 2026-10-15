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
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// =============================================================================
// Query 对象与解析
// =============================================================================

// query 汇总从 Opt 中解析出的操作参数
type query struct {
	conds    bson.D
	matchAll bool
	sort     bson.D
	limit    int64
	// fields 仅 Update / Upsert 使用，nil 表示没有传入任何 UpdateOpt
	fields *fieldSelection
	// omitted 读操作不读出的字段，parseOpts 之后为 bson 路径
	omitted []string
	// withoutTotal 仅 ListPage 使用
	withoutTotal bool
	// keepUpdatedAt 写操作不刷新 updatedAt，见 KeepUpdatedAt
	keepUpdatedAt bool
	// arrayFilters 仅 Modify / ModifyMany 使用
	arrayFilters []any
}

var (
	// errNoFilter 写操作必须带过滤条件，避免漏传条件时误改、误删整个集合
	errNoFilter = errors.New("filter is required for write operations, use store.All() to match all documents")
	// errUnsupportedOpt 操作收到了它不支持的选项
	errUnsupportedOpt = errors.New("option not supported by this operation")
)

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
	return documentHasKey(q.conds, key)
}

// projection 生成排除 omitted 字段的投影，没有要排除的字段时返回 nil（读出全部字段）
func (q *query) projection() bson.D {
	if len(q.omitted) == 0 {
		return nil
	}
	return lo.Map(q.omitted, func(f string, _ int) bson.E { return bson.E{Key: f, Value: 0} })
}

// parseOpts 依次应用 opts，遇到 accepted 之外类别的选项时返回错误；
// 过滤条件与排序中的字段名会被转换为 bson 路径
func (s *GenericStore[T]) parseOpts(op string, opts []Opt, accepted optKind) (*query, error) {
	q := &query{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if k := opt.kind() &^ accepted; k != 0 {
			return nil, errors.Wrapf(errUnsupportedOpt, "%s %s: %s", op, s.coll.Name(), k)
		}
		opt.apply(q)
	}
	resolve := func(path string) (string, error) { return resolveKey(s.schema.typ, path) }
	var err error
	if q.conds, err = mapFilterKeys(q.conds, resolve); err != nil {
		return nil, errors.Wrapf(err, "%s %s", op, s.coll.Name())
	}
	for i := range q.sort {
		if q.sort[i].Key, err = resolve(q.sort[i].Key); err != nil {
			return nil, errors.Wrapf(err, "%s %s", op, s.coll.Name())
		}
	}
	if q.omitted, err = s.resolveOmitted(q.omitted); err != nil {
		return nil, errors.Wrapf(err, "%s %s", op, s.coll.Name())
	}
	return q, nil
}

// resolveOmitted 将 OmitFields 的字段转换为 bson 路径：字段必须存在于模型中，重复的字段只保留一个，
// 互为父子的字段会被 MongoDB 判定为路径冲突，提前拒绝
func (s *GenericStore[T]) resolveOmitted(fields []string) ([]string, error) {
	if len(fields) == 0 {
		return nil, nil
	}
	paths := make([]fieldPath, 0, len(fields))
	for _, f := range fields {
		p, err := resolvePath(s.schema.typ, f)
		if err != nil {
			return nil, err
		}
		if !p.known {
			return nil, errors.Wrapf(errUnknownField, "omitted field %q", f)
		}
		paths = append(paths, p)
	}
	paths, err := dedupePaths(paths)
	if err != nil {
		return nil, err
	}
	return lo.Map(paths, func(p fieldPath, _ int) string { return p.String() }), nil
}

// requireFilter 要求写操作必须带过滤条件（或显式 All）
func (s *GenericStore[T]) requireFilter(op string, q *query) error {
	if len(q.conds) == 0 && !q.matchAll {
		return errors.Wrapf(errNoFilter, "%s %s", op, s.coll.Name())
	}
	return nil
}

// writeQuery 用于写操作：在 parseOpts 的基础上要求必须带过滤条件（或显式 All）
func (s *GenericStore[T]) writeQuery(op string, opts []Opt, accepted optKind) (*query, error) {
	q, err := s.parseOpts(op, opts, accepted)
	if err != nil {
		return nil, err
	}
	if err = s.requireFilter(op, q); err != nil {
		return nil, err
	}
	return q, nil
}
