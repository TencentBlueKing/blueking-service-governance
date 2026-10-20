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

// Package store 提供基于泛型的通用 MongoDB Store。
//
// 只需要通用 CRUD 的数据模型无需再定义自己的 Store，只要通过 Meta 描述集合名、
// 业务错误与读写钩子，再通过 Of(db, meta) 即可获得 GenericStore[T]。
// 过滤、排序、更新中的字段名既可以写 bson 字段名，也可以写 Go 字段名，嵌套字段使用 "a.b" 形式；
// 两种写法等价，代码中优先使用 bson 字段名。
package store

import (
	"context"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GenericStore 基于 Meta 描述的通用 Store，提供模型 T 的 CRUD 操作。
//
// 约定：
//   - 过滤条件、排序、数量限制等均通过 Opt 传入，各操作只接受部分类型的 Opt（见 Opt）；
//     读操作不带过滤条件时匹配全部文档
//   - bson tag 为 "_id"、"createdAt"、"updatedAt" 的字段（含 inline 嵌入）会被自动维护
//   - 写操作传入 *T 时，BeforeSave 的修改与时间戳（Create 还包括 _id）在写库之前就会作用到该对象上，
//     写入失败（如 NotFound、Duplicate）时对象同样已被修改，不能据此判断是否写入成功
//   - 返回的错误均已附带集合名等上下文，NotFound / Duplicate 可用 errors.Is 判断
type GenericStore[T any] struct {
	coll   *mongo.Collection
	meta   Meta[T]
	schema *schema
}

// Of 返回基于 db 的模型 T 的 GenericStore，适合在每次用到时调用：模型的反射分析结果会被缓存，构造本身很轻。
// Meta 配置错误（如集合名为空）属于编程错误，直接 panic，应由模型包的单测提前发现
func Of[T any](db *mongo.Database, meta Meta[T]) *GenericStore[T] {
	return lo.Must(New(db, meta))
}

// New 创建 GenericStore，T 必须是结构体，createdAt / updatedAt 字段（若有）必须是 time.Time。
// 需要以 error 而非 panic 处理配置错误时使用，其余场景请用 Of
func New[T any](db *mongo.Database, meta Meta[T]) (*GenericStore[T], error) {
	typ := reflect.TypeFor[T]()
	if meta.Collection == "" {
		return nil, errors.Errorf("meta of %s: collection is required", typ)
	}
	sc, err := schemaOf(typ)
	if err != nil {
		return nil, errors.Wrapf(err, "meta of %s", typ)
	}
	return &GenericStore[T]{coll: db.Collection(meta.Collection), meta: meta, schema: sc}, nil
}

// schemaCache 缓存模型结构体的反射分析结果（reflect.Type -> *schema）。
// schema 只取决于模型类型、创建后不再修改，因此可以在多个 Store 之间共享
var schemaCache sync.Map

func schemaOf(typ reflect.Type) (*schema, error) {
	if sc, ok := schemaCache.Load(typ); ok {
		return sc.(*schema), nil
	}
	sc, err := newSchema(typ)
	if err != nil {
		return nil, err
	}
	actual, _ := schemaCache.LoadOrStore(typ, sc)
	return actual.(*schema), nil
}

// Collection 返回底层集合，用于 GenericStore 未覆盖的操作（如建立索引）。
// 聚合请使用 Aggregate，数组与计数类的修改请使用 Modify，它们会校验字段并维护 updatedAt
func (s *GenericStore[T]) Collection() *mongo.Collection {
	return s.coll
}

// Get 获取一个匹配的文档，可通过 Asc / Desc 指定取哪一个；不存在时返回 NotFound 错误。
// 支持的选项：CondOpt、SortOpt、ProjectOpt
func (s *GenericStore[T]) Get(ctx context.Context, opts ...Opt) (*T, error) {
	q, err := s.parseOpts("find one in", opts, getOpts)
	if err != nil {
		return nil, err
	}
	findOpts := options.FindOne()
	if len(q.sort) > 0 {
		findOpts.SetSort(q.sort)
	}
	if proj := q.projection(); proj != nil {
		findOpts.SetProjection(proj)
	}

	var obj T
	err = s.coll.FindOne(ctx, q.filter(), findOpts).Decode(&obj)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, s.notFound(q)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "find one in %s, filter %v", s.coll.Name(), q.filter())
	}
	if err := s.decode(&obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// GetByID 按 _id 获取文档，等价于 Get(ctx, ByID(id))；不存在时返回 NotFound 错误。
// 需要附加范围条件时（如 _id 须属于某个工作空间）改用 Get 并组合 ByID 与其它条件
func (s *GenericStore[T]) GetByID(ctx context.Context, id any) (*T, error) {
	return s.Get(ctx, ByID(id))
}

// List 获取所有匹配的文档，可通过 Asc / Desc / Limit 指定排序与数量；无结果时返回空切片而非 nil。
// 支持的选项：CondOpt、SortOpt、LimitOpt、ProjectOpt
func (s *GenericStore[T]) List(ctx context.Context, opts ...Opt) ([]T, error) {
	q, err := s.parseOpts("find in", opts, listOpts)
	if err != nil {
		return nil, err
	}
	findOpts := options.Find()
	if q.limit > 0 {
		findOpts.SetLimit(q.limit)
	}
	return s.find(ctx, q, findOpts)
}

// find 按 q 的过滤条件、排序与投影查询并解码文档，findOpts 中的其余设置（如 skip、limit）由调用方指定
func (s *GenericStore[T]) find(ctx context.Context, q *query, findOpts *options.FindOptionsBuilder) ([]T, error) {
	if len(q.sort) > 0 {
		findOpts.SetSort(q.sort)
	}
	if proj := q.projection(); proj != nil {
		findOpts.SetProjection(proj)
	}
	cur, err := s.coll.Find(ctx, q.filter(), findOpts)
	if err != nil {
		return nil, errors.Wrapf(err, "find in %s, filter %v", s.coll.Name(), q.filter())
	}
	return s.decodeAll(ctx, cur)
}

// decodeAll 读出游标中的全部文档并执行 Decode，无结果时返回空切片而非 nil
func (s *GenericStore[T]) decodeAll(ctx context.Context, cur *mongo.Cursor) ([]T, error) {
	objs := []T{}
	if err := cur.All(ctx, &objs); err != nil {
		return nil, errors.Wrapf(err, "decode documents from %s", s.coll.Name())
	}
	for i := range objs {
		if err := s.decode(&objs[i]); err != nil {
			return nil, err
		}
	}
	return objs, nil
}

// Exists 判断是否存在匹配的文档。支持的选项：CondOpt
func (s *GenericStore[T]) Exists(ctx context.Context, opts ...Opt) (bool, error) {
	q, err := s.parseOpts("exists in", opts, condOpts)
	if err != nil {
		return false, err
	}
	n, err := s.coll.CountDocuments(ctx, q.filter(), options.Count().SetLimit(1))
	if err != nil {
		return false, errors.Wrapf(err, "count in %s, filter %v", s.coll.Name(), q.filter())
	}
	return n > 0, nil
}

// Count 统计匹配的文档数量。支持的选项：CondOpt
func (s *GenericStore[T]) Count(ctx context.Context, opts ...Opt) (int64, error) {
	q, err := s.parseOpts("count in", opts, condOpts)
	if err != nil {
		return 0, err
	}
	n, err := s.coll.CountDocuments(ctx, q.filter())
	if err != nil {
		return 0, errors.Wrapf(err, "count in %s, filter %v", s.coll.Name(), q.filter())
	}
	return n, nil
}

func (s *GenericStore[T]) decode(obj *T) error {
	for _, hook := range s.meta.Decode {
		if err := hook(obj); err != nil {
			return errors.Wrapf(err, "decode document from %s", s.coll.Name())
		}
	}
	return nil
}
