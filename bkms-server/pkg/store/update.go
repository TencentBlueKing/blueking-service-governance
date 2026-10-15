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
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrNoFieldsToUpdate Update / Upsert 没有任何待写入的字段，如传入空 map、UpdateNonZero 时对象的字段全为零值。
// 允许「没有变更」的调用方可以用 errors.Is 判断后忽略
var ErrNoFieldsToUpdate = errors.New("no fields to update")

// selectAll 在 UpdateSelected 中表示模型的全部顶层字段
const selectAll = "*"

// fieldSelection 汇总传入的 UpdateOpt，见 UpdateOpt
type fieldSelection struct {
	selecting bool
	selected  []string
	omitted   []string
	nonZero   bool
}

// Update 更新匹配的一个文档，不存在时返回 NotFound 错误。target 表示更新后的目标状态，支持两种形式：
//
//   - map[string]any（或 bson.M）：key 即待更新的字段，可以写 bson 路径或 Go 字段名（见 resolvePath），
//     值会被赋给一个新的 T 对象中的对应字段（类型不同时按 assignValue 的规则转换），再与 *T 一样经过 Encode。
//     此时不接受 UpdateOpt
//   - *T：必须通过 UpdateSelected / UpdateOmit / UpdateNonZero 指定写入哪些字段，避免意外覆盖
//
// 在 Encode 的结果中，待更新字段存在的被 $set，不存在的（如 omitempty 的零值、map 中值为 nil 的指针字段）
// 被 $unset。_id 与 createdAt 不能更新；updatedAt 总会刷新为当前时间（显式指定时以指定的值为准），
// 传入 *T 时会回填到对象中。没有任何待更新字段时返回 ErrNoFieldsToUpdate。
//
// Update 不执行 BeforeSave（对象可能只填写了部分字段），Encode 钩子需要能处理只填写了部分字段的对象。
// 例如：
//
//	// 按 map 局部更新，false 也会被写入
//	err := widgets.Update(ctx, map[string]any{"enabled": false, "spec.color": "red"}, ByName(name))
//	// 更新对象中除 ownerID 外的全部字段，obj 中为空的 omitempty 字段会从库中删除
//	err := widgets.Update(ctx, obj, store.UpdateOmit("ownerID"), ByName(name))
//
// 支持的选项：CondOpt、UpdateOpt
func (s *GenericStore[T]) Update(ctx context.Context, target any, opts ...Opt) error {
	q, err := s.writeQuery("update", opts, updateOpts)
	if err != nil {
		return err
	}
	obj, paths, err := s.updateTarget(target, q.fields, false)
	if err != nil {
		return errors.Wrapf(err, "update %s", s.coll.Name())
	}
	_, set, unset, err := s.encodeUpdate(obj, paths, time.Now())
	if err != nil {
		return err
	}
	return s.updateOne(ctx, q, updateDoc(set, unset, nil))
}

// Upsert 更新匹配的一个文档，不存在时插入，返回是否发生了插入。支持的选项：CondOpt、UpdateOpt。
//
// target 的形式与字段规则同 Update，区别在于：
//   - 传入 *T 时会先执行 BeforeSave，未传入 UpdateOpt 时更新全部顶层字段（相当于 UpdateSelected("*")），
//     因此 obj 中为空的 omitempty 字段会从库中删除；模型中没有定义的字段（如模型删除字段后库中残留的旧数据）
//     不会被删除，需要整体替换文档时请使用 Replace
//   - 传入 map 时不执行 BeforeSave，因为无法预知本次是插入还是更新，只填写了部分字段的对象可能通不过校验
//   - 待写入的字段只能是顶层字段，避免与插入时写入的字段冲突
//
// 未被选中的字段只在插入时写入（$setOnInsert），例如传入 UpdateSelected("priority") 时，已存在的文档只更新
// priority，插入的文档则包含 obj 的全部字段；传入 map 时插入的文档中其余字段为零值。过滤条件中以等值方式
// 出现的字段在插入时取过滤条件中的值，不会被零值覆盖。
//
// _id 与 createdAt 仅在插入时写入。过滤条件中的 _id 只参与匹配，不会成为待写入的值：
// obj 已有 _id 时，该值会自动追加到过滤条件并写入 $setOnInsert，因此可不额外传入 CondOpt；
// 此时若调用方又显式传入 _id 条件，Upsert 会在发送请求前报错；
// obj 的 _id 为零值时，插入所需的 _id 会按模型类型生成，但不会加入过滤条件。
// createdAt 取 obj 的值，零值时取当前时间。obj 的 updatedAt 总会被更新；
// 插入时 _id 与 createdAt 会回填到 obj，更新时库中已有的值未知，obj 中的这两个字段保持不变
func (s *GenericStore[T]) Upsert(ctx context.Context, target any, opts ...Opt) (bool, error) {
	q, err := s.parseOpts("upsert into", opts, updateOpts)
	if err != nil {
		return false, err
	}
	obj, paths, err := s.updateTarget(target, q.fields, true)
	if err != nil {
		return false, errors.Wrapf(err, "upsert into %s", s.coll.Name())
	}
	if p, found := lo.Find(paths, func(p fieldPath) bool { return len(p.steps) > 1 }); found {
		return false, errors.Errorf(
			"upsert into %s: nested field %q is not supported, select its top-level field instead",
			s.coll.Name(), p,
		)
	}

	now := time.Now()
	v := reflect.ValueOf(obj).Elem()
	objectID, hasObjectID := s.schema.idOf(v)
	if hasObjectID {
		if q.hasKey(keyID) {
			return false, errors.Errorf(
				"upsert into %s: %s is specified in both object and filter",
				s.coll.Name(),
				keyID,
			)
		}
		q.conds = append(q.conds, bson.E{Key: keyID, Value: objectID})
	}
	if err = s.requireFilter("upsert into", q); err != nil {
		return false, err
	}
	raw, set, unset, err := s.encodeUpdate(obj, paths, now)
	if err != nil {
		return false, err
	}

	onInsert, err := s.insertOnlyFields(raw, set, unset, q)
	if err != nil {
		return false, err
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

	update := updateDoc(set, unset, onInsert)
	res, err := s.coll.UpdateOne(ctx, q.filter(), update, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return false, s.wrapWriteErr(err, "upsert into")
	}
	if res.UpsertedCount == 0 {
		return false, nil
	}
	s.schema.setID(v, res.UpsertedID)
	if s.schema.createdAt != nil {
		v.FieldByIndex(s.schema.createdAt).Set(reflect.ValueOf(createdAt))
	}
	return true, nil
}

// updateTarget 解析 Update / Upsert 的 target，返回待编码的对象与待写入的字段。
// upsert 为 true 时，*T 未传入 UpdateOpt 则选中全部顶层字段，并在计算字段之前执行 BeforeSave
// （BeforeSave 可能补全默认值，影响 UpdateNonZero 的结果）
func (s *GenericStore[T]) updateTarget(target any, sel *fieldSelection, upsert bool) (*T, []fieldPath, error) {
	switch t := target.(type) {
	case *T:
		if t == nil {
			return nil, nil, errors.New("target object is nil")
		}
		if sel == nil {
			if !upsert {
				return nil, nil, errors.New(
					"specify the fields to update with UpdateSelected, UpdateOmit or UpdateNonZero",
				)
			}
			sel = &fieldSelection{}
		}
		if upsert {
			if err := s.beforeSave(t); err != nil {
				return nil, nil, err
			}
		}
		paths, err := sel.paths(s.schema, reflect.ValueOf(t).Elem())
		return t, paths, err
	case bson.M:
		return s.objectFromMap(t, sel)
	case map[string]any:
		return s.objectFromMap(t, sel)
	}
	return nil, nil, errors.Errorf("unsupported target type %T, expected *%s or map[string]any", target, s.schema.typ)
}

// objectFromMap 将 m 中的值赋给一个新的 T 对象，返回该对象与 m 中的字段
func (s *GenericStore[T]) objectFromMap(m map[string]any, sel *fieldSelection) (*T, []fieldPath, error) {
	if sel != nil {
		return nil, nil, errors.New("UpdateOpt is not accepted when the target is a map, its keys are the fields")
	}
	if len(m) == 0 {
		return nil, nil, ErrNoFieldsToUpdate
	}
	obj := new(T)
	v := reflect.ValueOf(obj).Elem()
	keys := lo.Keys(m)
	slices.Sort(keys)
	paths := make([]fieldPath, 0, len(keys))
	for _, key := range keys {
		p, err := resolveUpdatePath(s.schema.typ, key)
		if err != nil {
			return nil, nil, err
		}
		if !p.assignable {
			return nil, nil, errors.Errorf(
				"field %q cannot be set from a map, only struct fields and map entries are supported", key,
			)
		}
		if err = setPath(v, p.steps, m[key]); err != nil {
			return nil, nil, errors.Wrapf(err, "set field %q", key)
		}
		paths = append(paths, p)
	}
	paths, err := dedupePaths(paths)
	return obj, paths, err
}

// paths 按 sel 计算对象 v（模型 sc 的结构体值）中待写入的字段，规则见 UpdateOpt
func (sel *fieldSelection) paths(sc *schema, v reflect.Value) ([]fieldPath, error) {
	all := !sel.selecting || slices.Equal(sel.selected, []string{selectAll})
	switch {
	case sel.selecting && len(sel.selected) == 0:
		return nil, errors.Wrap(ErrNoFieldsToUpdate, "UpdateSelected is given without any field")
	case !all && slices.Contains(sel.selected, selectAll):
		return nil, errors.Errorf("%q cannot be combined with other fields in UpdateSelected", selectAll)
	}

	var candidates []fieldPath
	if all {
		for _, key := range sc.keys {
			if key == keyID || key == keyCreatedAt || key == keyUpdatedAt {
				continue
			}
			p, err := resolvePath(sc.typ, key)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, p)
		}
	} else {
		for _, f := range sel.selected {
			p, err := resolveUpdatePath(sc.typ, f)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, p)
		}
	}

	for _, f := range sel.omitted {
		p, err := resolvePath(sc.typ, f)
		if err != nil {
			return nil, err
		}
		if !p.known {
			return nil, errors.Wrapf(errUnknownField, "omitted field %q", f)
		}
		if parent, found := lo.Find(candidates, func(c fieldPath) bool { return c.isAncestorOf(p) }); found {
			return nil, errors.Errorf("cannot omit %q, its parent %q is selected as a whole", f, parent)
		}
		candidates = lo.Reject(candidates, func(c fieldPath, _ int) bool { return c.String() == p.String() })
	}

	if sel.nonZero {
		candidates = lo.Filter(candidates, func(c fieldPath, _ int) bool {
			if !c.assignable {
				return true
			}
			val, ok := valueAt(v, c.steps)
			return ok && !val.IsZero()
		})
	}
	if len(candidates) == 0 {
		return nil, ErrNoFieldsToUpdate
	}
	return dedupePaths(candidates)
}

// resolveUpdatePath 解析待写入的字段：必须存在于模型中，且不能是 _id 或 createdAt
func resolveUpdatePath(typ reflect.Type, path string) (fieldPath, error) {
	if isOperatorKey(path) {
		return fieldPath{}, errors.Errorf(
			"update operator %q is not supported, use Collection() for operators other than $set / $unset", path,
		)
	}
	p, err := resolvePath(typ, path)
	if err != nil {
		return fieldPath{}, err
	}
	if !p.known {
		return fieldPath{}, errors.Wrapf(errUnknownField, "field %q", path)
	}
	if top := p.steps[0].key; top == keyID || top == keyCreatedAt {
		return fieldPath{}, errors.Errorf("field %q cannot be updated", path)
	}
	return p, nil
}

// encodeUpdate 刷新 obj 的 updatedAt（paths 中显式包含时以对象中的值为准）并执行 Encode，
// 再从编码结果中取出 paths 的值：存在的放入 set，不存在的放入 unset
func (s *GenericStore[T]) encodeUpdate(
	obj *T, paths []fieldPath, now time.Time,
) (raw bson.Raw, set, unset bson.M, err error) {
	keys := lo.Map(paths, func(p fieldPath, _ int) string { return p.String() })
	if s.schema.updatedAt != nil && !slices.Contains(keys, keyUpdatedAt) {
		s.schema.setUpdated(reflect.ValueOf(obj).Elem(), now)
		keys = append(keys, keyUpdatedAt)
	}
	if raw, err = s.encodeRaw(obj); err != nil {
		return nil, nil, nil, err
	}
	set, unset = bson.M{}, bson.M{}
	for _, key := range keys {
		if v, err := raw.LookupErr(strings.Split(key, ".")...); err == nil {
			set[key] = v
		} else {
			unset[key] = ""
		}
	}
	return raw, set, unset, nil
}

// insertOnlyFields 返回 Upsert 只在插入时写入的字段：编码结果中未被选中的顶层字段，
// 排除 _id、createdAt（另行处理）以及过滤条件中以等值方式出现的字段（插入时由 MongoDB 取过滤条件中的值）
func (s *GenericStore[T]) insertOnlyFields(raw bson.Raw, set, unset bson.M, q *query) (bson.M, error) {
	elems, err := raw.Elements()
	if err != nil {
		return nil, errors.Wrapf(err, "read encoded document for %s", s.coll.Name())
	}
	fromFilter := equalityKeys(q.conds)
	onInsert := bson.M{}
	for _, e := range elems {
		key := e.Key()
		_, inSet := set[key]
		_, inUnset := unset[key]
		if inSet || inUnset || key == keyID || key == keyCreatedAt || fromFilter[key] {
			continue
		}
		onInsert[key] = e.Value()
	}
	return onInsert, nil
}
