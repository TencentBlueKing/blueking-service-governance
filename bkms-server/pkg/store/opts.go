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
	"regexp"
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
//	类型         创建方式                                          支持的操作
//	CondOpt     Where / Filter / ByID / ContainsIgnoreCase / All  全部操作
//	SortOpt     Asc / Desc                                       Get、List、ListPage、GetAndDelete
//	LimitOpt    Limit                                            List
//	ProjectOpt  OmitFields                                       Get、List、ListPage
//	PageOpt     WithoutTotal                                     ListPage
//	UpdateOpt   UpdateSelected / UpdateOmit / UpdateNonZero      Update、Upsert
//
// 只有 CondOpt 影响匹配哪些文档，其余选项只决定结果的顺序、数量、读出哪些字段或写入哪些字段。
//
// 过滤条件、排序中的字段名既可以写 bson 路径，也可以写 Go 字段名（见 resolvePath），
// 模型中不存在的字段原样保留；Update / Upsert 的字段则必须存在于模型中。
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
	kindUpdate
	kindProject
	kindPage
)

// 各操作支持的选项类别
const (
	getOpts          = kindCond | kindSort | kindProject
	listOpts         = kindCond | kindSort | kindLimit | kindProject
	pageOpts         = kindCond | kindSort | kindProject | kindPage
	getAndDeleteOpts = kindCond | kindSort
	condOpts         = kindCond
	updateOpts       = kindCond | kindUpdate
)

// kindNames 按位序排列，与上面的 kindXxx 一一对应
var kindNames = []string{"condition", "sort", "limit", "update fields", "omitted fields", "page"}

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

// CondOpt 过滤条件，由 Where / Filter / ByID / ContainsIgnoreCase / All 创建，所有操作都支持
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
	return CondOpt{conds: sortedDoc(m)}
}

// ByID 按 _id 匹配文档，适用于所有操作，也可与其它条件 Combine 以限定范围（如只在某个工作空间内查找）。
// id 的类型须与模型 _id 字段一致，例如 bson.ObjectID 与它的 hex 字符串不能混用，否则只会表现为查不到
func ByID(id any) CondOpt {
	return Where(keyID, id)
}

// ContainsIgnoreCase 忽略大小写的模糊匹配：任一字段包含 keyword 即命中，多个字段之间是 OR 关系。
// keyword 去掉首尾空白并转义正则元字符后按字面量匹配；为空白时不产生任何条件，读操作因此不做过滤，
// 写操作仍须另带条件。正则匹配用不上普通索引，只适合数据量有限的列表筛选。例如：
//
//	// 名称或描述中包含 "go"（不区分大小写）的镜像
//	images.List(ctx, store.ContainsIgnoreCase(keyword, "name", "description"))
func ContainsIgnoreCase(keyword, field string, moreFields ...string) CondOpt {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return CondOpt{}
	}
	match := bson.M{"$regex": regexp.QuoteMeta(keyword), "$options": "i"}
	if len(moreFields) == 0 {
		return Where(field, match)
	}
	fields := append([]string{field}, moreFields...)
	return Where("$or", lo.Map(fields, func(f string, _ int) bson.D { return bson.D{{Key: f, Value: match}} }))
}

// All 显式声明匹配全部文档，写操作在没有任何过滤条件时必须使用它
func All() CondOpt {
	return CondOpt{matchAll: true}
}

// -----------------------------------------------------------------------------
// SortOpt：排序
// -----------------------------------------------------------------------------

// SortOpt 排序条件，由 Asc / Desc 创建，仅 Get / List / GetAndDelete 支持；多个排序条件按传入顺序生效
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
// ProjectOpt：读出的字段
// -----------------------------------------------------------------------------

// ProjectOpt 指定读操作不读出哪些字段，由 OmitFields 创建，仅 Get / List / ListPage 支持；多个 ProjectOpt 会合并
type ProjectOpt struct {
	omitted []string
}

func (ProjectOpt) kind() optKind { return kindProject }

func (o ProjectOpt) apply(q *query) { q.omitted = append(q.omitted, o.omitted...) }

// OmitFields 读取时不从数据库取回指定的字段，用于跳过体积大、列表页用不到的字段（如完整的资源清单）。
// 它不影响匹配哪些文档：被跳过的字段在返回的对象中为零值，Decode 钩子照常执行，须能处理这些零值。
// 字段可以写 bson 路径或 Go 字段名，必须存在于模型中，避免拼错时静默读出全部字段。例如：
//
//	// 列出部署记录的资源快照，但不读取 manifest
//	snapshots.List(ctx, ByDeployRecord(recordID), store.OmitFields("manifest"))
func OmitFields(fields ...string) ProjectOpt {
	return ProjectOpt{omitted: fields}
}

// -----------------------------------------------------------------------------
// PageOpt：分页行为
// -----------------------------------------------------------------------------

// PageOpt 调整 ListPage 的行为，由 WithoutTotal 创建，仅 ListPage 支持
type PageOpt struct {
	withoutTotal bool
}

func (PageOpt) kind() optKind { return kindPage }

func (o PageOpt) apply(q *query) { q.withoutTotal = q.withoutTotal || o.withoutTotal }

// WithoutTotal 让 ListPage 不统计匹配的总数，只判断是否还有下一页，适合「加载更多」式的分页，
// 省去一次 Count。此时 Page.Total 恒为 0，不代表没有数据
func WithoutTotal() PageOpt {
	return PageOpt{withoutTotal: true}
}

// -----------------------------------------------------------------------------
// UpdateOpt：Update / Upsert 写入的字段
// -----------------------------------------------------------------------------

// UpdateOpt 指定 Update / Upsert 传入对象（*T）时写入哪些字段，由 UpdateSelected / UpdateOmit / UpdateNonZero
// 创建，仅 Update / Upsert 支持；传入 map 时字段已由 key 确定，不接受 UpdateOpt。字段集合按以下顺序计算：
//
//  1. 候选字段：UpdateSelected 指定的字段；未指定或指定 "*" 时为模型的全部顶层字段
//  2. 去掉 UpdateOmit 指定的字段
//  3. 传入了 UpdateNonZero 时，只保留对象中值非零的字段
//
// 多个 UpdateOpt 会合并。_id、createdAt 不会被更新，updatedAt 总会被刷新，详见 Update
type UpdateOpt struct {
	selecting bool
	selected  []string
	omitted   []string
	nonZero   bool
}

func (UpdateOpt) kind() optKind { return kindUpdate }

func (o UpdateOpt) apply(q *query) {
	if q.fields == nil {
		q.fields = &fieldSelection{}
	}
	q.fields.selecting = q.fields.selecting || o.selecting
	q.fields.selected = append(q.fields.selected, o.selected...)
	q.fields.omitted = append(q.fields.omitted, o.omitted...)
	q.fields.nonZero = q.fields.nonZero || o.nonZero
}

// UpdateSelected 指定写入的字段，"*" 表示模型的全部顶层字段（不能与其它字段同时指定）。
// 字段可以写 bson 路径（如 "spec.color"），也可以写 Go 字段名（如 "Spec.Color"）
func UpdateSelected(fields ...string) UpdateOpt {
	return UpdateOpt{selecting: true, selected: fields}
}

// UpdateOmit 从候选字段中去掉指定的字段，适合「除不可变字段外全部更新」，模型新增的字段会自动参与更新
func UpdateOmit(fields ...string) UpdateOpt {
	return UpdateOpt{omitted: fields}
}

// UpdateNonZero 只写入对象中值非零（reflect.Value.IsZero 为 false）的字段，因此无法用它把字段改为
// false、0 或空串；空切片、空 map 不是零值，会被写入。嵌套结构体按整体判断，非零时整体写入
func UpdateNonZero() UpdateOpt {
	return UpdateOpt{nonZero: true}
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
	// fields 仅 Update / Upsert 使用，nil 表示没有传入任何 UpdateOpt
	fields *fieldSelection
	// omitted 读操作不读出的字段，parseOpts 之后为 bson 路径
	omitted []string
	// withoutTotal 仅 ListPage 使用
	withoutTotal bool
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
