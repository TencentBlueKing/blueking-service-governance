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
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Upsert 更新匹配的一个文档，不存在时插入，返回是否发生了插入。支持的选项：CondOpt、UpdateOpt。
//
// target 的形式与字段规则同 Update，区别在于：
//   - 传入 *T 时会先执行 BeforeSave，未传入 UpdateOpt 时更新全部顶层字段（相当于 UpdateSelected("*")），
//     因此 obj 中为空的 omitempty 字段会从库中删除；模型中没有定义的字段（如模型删除字段后库中残留的旧数据）
//     不会被删除，需要整体替换文档时请使用 Replace
//   - 传入 map 时不执行 BeforeSave，因为无法预知本次是插入还是更新，只填写了部分字段的对象可能通不过校验
//
// 与 Update 不同，Upsert 可能插入，obj 须是完整的目标状态，因此默认写入全部字段；
// 由其他路径维护的字段（如 Modify 的 $inc 计数）须用 UpdateOmit 排除。
//
// 未被选中的字段只在插入时写入（$setOnInsert），例如传入 UpdateSelected("priority") 时，已存在的文档只更新
// priority，插入的文档则包含 obj 的全部字段；传入 map 时插入的文档中其余字段为零值。过滤条件中以等值方式
// 出现的字段在插入时取过滤条件中的值，不会被零值覆盖。这里的等值只识别直接写值的条件（含 $and 中的），
// 不识别 {"$eq": v} 等操作符形式，以免复刻 MongoDB 的等值提取规则：这类字段在插入时仍取 obj 中的值，
// 可能导致插入的文档不满足过滤条件，需要插入时保留条件值请直接写值。
//
// 待写入的字段可以是嵌套字段（如 "envStates.prod"），此时它的上级字段不会在插入时整体写入（否则与 $set 的路径冲突），
// 插入的文档中上级字段只包含被写入的子字段。
//
// _id 与 createdAt 仅在插入时写入。过滤条件中的 _id 只参与匹配，不会成为待写入的值：
// obj 已有 _id 时，该值会自动追加到过滤条件并写入 $setOnInsert，因此可不额外传入 CondOpt；
// 此时若调用方又显式传入 _id 条件，Upsert 会在发送请求前报错；
// obj 的 _id 为零值时，插入所需的 _id 会按模型类型生成，但不会加入过滤条件。
// createdAt 取 obj 的值，零值时取当前时间。obj 的 updatedAt 总会被更新；
// 插入时 _id 与 createdAt 会回填到 obj，更新时库中已有的值未知，obj 中的这两个字段保持不变
func (s *GenericStore[T]) Upsert(ctx context.Context, target any, opts ...Opt) (bool, error) {
	q, err := s.parseOpts("upsert into", opts, upsertOpts)
	if err != nil {
		return false, err
	}
	op, err := s.prepareUpsert(target, q, time.Now())
	if err != nil {
		return false, err
	}
	res, err := s.coll.UpdateOne(ctx, op.filter, op.update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return false, s.wrapWriteErr(err, "upsert into")
	}
	if res.UpsertedCount == 0 {
		return false, nil
	}
	op.inserted(res.UpsertedID)
	return true, nil
}

// UpsertMany 批量 Upsert：每个对象的语义同 Upsert(ctx, obj, opts..., key(obj))，key 返回该对象的匹配条件，
// 只能由 CondOpt 组成，opts 则对所有对象生效。例如：
//
//	// 按 key 写入变量，已存在的只更新 value 与 description
//	vars.UpsertMany(ctx, objs, func(v *EnvVar) store.Opt { return ByKey(v.Key) },
//		InScope(scope), store.UpdateSelected("value", "description"))
//
// 全部写入在一次 BulkWrite 中按顺序执行，遇到错误即停止（之前的对象已写入），唯一键冲突时返回 Duplicate 错误。
// 返回插入的文档数量，插入的对象会回填 _id 与 createdAt。
// 出错时不做回填且返回的数量为 0：出错前已插入的对象也拿不到 _id 与 createdAt，
// 需要它们时请按 key 重新查询。支持的选项：CondOpt、UpdateOpt
func (s *GenericStore[T]) UpsertMany(ctx context.Context, objs []*T, key func(*T) Opt, opts ...Opt) (int64, error) {
	if len(objs) == 0 {
		return 0, nil
	}
	now := time.Now()
	ops := make([]*upsertOp, 0, len(objs))
	models := make([]mongo.WriteModel, 0, len(objs))
	for i, obj := range objs {
		if obj == nil {
			return 0, errors.Errorf("upsert many into %s: object %d is nil", s.coll.Name(), i)
		}
		k := key(obj)
		if k == nil || k.kind()&^kindCond != 0 {
			return 0, errors.Errorf("upsert many into %s: key of object %d must be a condition", s.coll.Name(), i)
		}
		q, err := s.parseOpts("upsert many into", append(slices.Clone(opts), k), upsertOpts)
		if err != nil {
			return 0, err
		}
		op, err := s.prepareUpsert(obj, q, now)
		if err != nil {
			return 0, errors.Wrapf(err, "object %d", i)
		}
		ops = append(ops, op)
		models = append(models, mongo.NewUpdateOneModel().SetFilter(op.filter).SetUpdate(op.update).SetUpsert(true))
	}
	res, err := s.coll.BulkWrite(ctx, models)
	if err != nil {
		return 0, s.wrapWriteErr(err, "upsert many into")
	}
	for i, id := range res.UpsertedIDs {
		ops[i].inserted(id)
	}
	return res.UpsertedCount, nil
}

// upsertOp 一次 Upsert 的请求
type upsertOp struct {
	filter bson.D
	update bson.M
	// inserted 发生插入后，将 _id 与 createdAt 回填到对象中
	inserted func(id any)
}

// prepareUpsert 根据 target 与已解析的选项 q 生成 Upsert 请求，q 中的条件可能被追加对象的 _id
func (s *GenericStore[T]) prepareUpsert(target any, q *query, now time.Time) (*upsertOp, error) {
	obj, paths, err := s.updateTarget(target, q.fields, true)
	if err != nil {
		return nil, errors.Wrapf(err, "upsert into %s", s.coll.Name())
	}
	v := reflect.ValueOf(obj).Elem()
	objectID, hasObjectID := s.schema.idOf(v)
	if hasObjectID {
		if q.hasKey(keyID) {
			return nil, errors.Errorf(
				"upsert into %s: %s is specified in both object and filter",
				s.coll.Name(),
				keyID,
			)
		}
		q.conds = append(q.conds, bson.E{Key: keyID, Value: objectID})
	}
	if err = s.requireFilter("upsert into", q); err != nil {
		return nil, err
	}
	raw, set, unset, err := s.encodeUpdate(obj, paths, now, false)
	if err != nil {
		return nil, err
	}

	onInsert, err := s.insertOnlyFields(raw, set, unset, q)
	if err != nil {
		return nil, err
	}
	var createdAt time.Time
	if s.schema.createdAt != nil {
		createdAt = lo.Ternary(s.schema.createdAtOf(v).IsZero(), now, s.schema.createdAtOf(v))
		onInsert[keyCreatedAt] = createdAt
	}
	if hasObjectID {
		onInsert[keyID] = objectID
	} else if id, ok := s.schema.newID(); ok {
		onInsert[keyID] = id
	}

	inserted := func(id any) {
		s.schema.setID(v, id)
		if s.schema.createdAt != nil {
			v.FieldByIndex(s.schema.createdAt).Set(reflect.ValueOf(createdAt))
		}
	}
	return &upsertOp{filter: q.filter(), update: updateDoc(set, unset, onInsert), inserted: inserted}, nil
}

// insertOnlyFields 返回 Upsert 只在插入时写入的字段：编码结果中的顶层字段，排除 _id、createdAt（另行处理）、
// 已被 $set / $unset 的字段、过滤条件中以等值方式出现的字段（插入时由 MongoDB 取过滤条件中的值），
// 以及后两者的上级字段（同时写入上级字段会被 MongoDB 判定为路径冲突）
func (s *GenericStore[T]) insertOnlyFields(raw bson.Raw, set, unset bson.M, q *query) (bson.M, error) {
	elems, err := raw.Elements()
	if err != nil {
		return nil, errors.Wrapf(err, "read encoded document for %s", s.coll.Name())
	}
	written := slices.Concat(lo.Keys(set), lo.Keys(unset), lo.Keys(equalityKeys(q.conds)))
	covered := func(key string) bool {
		return lo.ContainsBy(written, func(w string) bool { return w == key || strings.HasPrefix(w, key+".") })
	}
	onInsert := bson.M{}
	for _, e := range elems {
		key := e.Key()
		if key == keyID || key == keyCreatedAt || covered(key) {
			continue
		}
		onInsert[key] = e.Value()
	}
	return onInsert, nil
}
