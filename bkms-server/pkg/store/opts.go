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
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// =============================================================================
// Opt 接口与通用选项
// =============================================================================

// Opt 是各操作可选参数的公共接口，多个过滤条件按 AND 合并。
//
// 选项按用途分为不同的类型，每个操作只接受其中一部分。传入不支持的选项会返回错误而不是被静默忽略，
// 否则可能违背调用方预期，例如 DeleteMany(..., Limit(10)) 实际会删除全部匹配文档：
//
//	类型        创建方式                 支持的操作
//	CondOpt    Where / Filter / All    全部操作
//	SortOpt    Asc / Desc              Get、List
//	LimitOpt   Limit                   List
//	UpsertOpt  UnsetOmitted            Upsert
//
// 领域包可基于 Where / Filter 定义具名的 Opt，让字段名只出现在一处，例如：
//
//	// 包 bkci 中
//	func ByWorkspace(workspaceID string) store.Opt { return store.Where("workspaceID", workspaceID) }
//
//	// 调用方
//	projects.Get(ctx, bkci.ByWorkspace(wsID))
//	pipelines.List(ctx, bkci.ByWorkspace(wsID), store.Desc("createdAt"), store.Limit(10))
type Opt interface {
	// kind 返回选项所属的类别，各操作据此判断自己是否支持该选项
	kind() optKind
	apply(q *query)
}

// optKind 是选项类别的位集合，Combine 组合出的选项可能同时属于多个类别
type optKind uint8

const (
	kindCond optKind = 1 << iota
	kindSort
	kindLimit
	kindUpsert
)

// 各操作支持的选项类别
const (
	getOpts    = kindCond | kindSort
	listOpts   = kindCond | kindSort | kindLimit
	condOpts   = kindCond
	upsertOpts = kindCond | kindUpsert
)

// kindNames 按位序排列，与上面的 kindXxx 一一对应
var kindNames = []string{"condition", "sort", "limit", "upsert"}

func (k optKind) String() string {
	var names []string
	for i, name := range kindNames {
		if k&(1<<i) != 0 {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

// -----------------------------------------------------------------------------
// CondOpt：过滤条件
// -----------------------------------------------------------------------------

// CondOpt 过滤条件，由 Where / Filter / All 创建，所有操作都支持
type CondOpt struct {
	conds    bson.D
	matchAll bool
}

func (CondOpt) kind() optKind { return kindCond }

func (o CondOpt) apply(q *query) {
	q.conds = append(q.conds, o.conds...)
	q.matchAll = q.matchAll || o.matchAll
}

// Where 增加一个字段条件，value 可以是普通值，也可以是 bson.M{"$in": ...} 等操作符表达式
func Where(field string, value any) CondOpt {
	return CondOpt{conds: bson.D{{Key: field, Value: value}}}
}

// Filter 将 m 中的每个条件加入过滤条件
func Filter(m bson.M) CondOpt {
	// 按 key 排序，保证生成的过滤条件与错误信息稳定
	keys := lo.Keys(m)
	slices.Sort(keys)
	return CondOpt{conds: lo.Map(keys, func(k string, _ int) bson.E { return bson.E{Key: k, Value: m[k]} })}
}

// All 显式声明匹配全部文档，写操作在没有任何过滤条件时必须使用它
func All() CondOpt {
	return CondOpt{matchAll: true}
}

// -----------------------------------------------------------------------------
// SortOpt：排序
// -----------------------------------------------------------------------------

// SortOpt 排序条件，由 Asc / Desc 创建，仅 Get / List 支持；多个排序条件按传入顺序生效
type SortOpt struct {
	elem bson.E
}

func (SortOpt) kind() optKind { return kindSort }

func (o SortOpt) apply(q *query) { q.sort = append(q.sort, o.elem) }

// Asc 按字段升序排序
func Asc(field string) SortOpt {
	return SortOpt{elem: bson.E{Key: field, Value: 1}}
}

// Desc 按字段降序排序
func Desc(field string) SortOpt {
	return SortOpt{elem: bson.E{Key: field, Value: -1}}
}

// -----------------------------------------------------------------------------
// LimitOpt：数量限制
// -----------------------------------------------------------------------------

// LimitOpt 数量限制，由 Limit 创建，仅 List 支持
type LimitOpt struct {
	n int64
}

func (LimitOpt) kind() optKind { return kindLimit }

func (o LimitOpt) apply(q *query) { q.limit = o.n }

// Limit 限制返回的文档数量，n <= 0 表示不限制
func Limit(n int64) LimitOpt {
	return LimitOpt{n: n}
}

// -----------------------------------------------------------------------------
// UpsertOpt：Upsert 写入方式
// -----------------------------------------------------------------------------

// UpsertOpt 调整 Upsert 的写入方式，由 UnsetOmitted 创建，仅 Upsert 支持
type UpsertOpt struct {
	unsetOmitted bool
}

func (UpsertOpt) kind() optKind { return kindUpsert }

func (o UpsertOpt) apply(q *query) { q.unsetOmitted = q.unsetOmitted || o.unsetOmitted }

// UnsetOmitted 让 Upsert 删除模型中定义、但 obj 编码结果中没有的顶层字段（如 omitempty 的零值），
// 使 obj 中被清空的字段在库中同样被清除。用法与适用场景见 Upsert
func UnsetOmitted() UpsertOpt {
	return UpsertOpt{unsetOmitted: true}
}

// -----------------------------------------------------------------------------
// Opt 组合
// -----------------------------------------------------------------------------

// Combine 将多个 Opt 组合为一个，便于领域包在已有的具名 Opt 上构造新的；
// 组合出的选项仍按其中每个选项的类别校验。例如：
//
//	func ByWorkspaceAndType(workspaceID string, typ PipelineType) store.Opt {
//		return store.Combine(ByWorkspace(workspaceID), store.Where("type", typ))
//	}
func Combine(opts ...Opt) Opt {
	return combinedOpt(opts)
}

type combinedOpt []Opt

func (c combinedOpt) kind() optKind {
	var k optKind
	for _, opt := range c {
		if opt != nil {
			k |= opt.kind()
		}
	}
	return k
}

func (c combinedOpt) apply(q *query) {
	for _, opt := range c {
		if opt != nil {
			opt.apply(q)
		}
	}
}

// =============================================================================
// Query 对象与解析
// =============================================================================

// query 汇总从 Opt 中解析出的操作参数
type query struct {
	conds    bson.D
	matchAll bool
	sort     bson.D
	limit    int64
	// unsetOmitted 仅 Upsert 使用，见 UnsetOmitted
	unsetOmitted bool
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

// parseOpts 依次应用 opts，遇到 accepted 之外类别的选项时返回错误
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
	return q, nil
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
