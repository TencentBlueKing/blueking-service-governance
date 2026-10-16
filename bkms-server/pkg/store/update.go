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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store/path"
)

// ErrNoFieldsToUpdate Update / Upsert 没有任何待写入的字段，如传入空 map、UpdateOmit 去掉了全部字段。
// 允许「没有变更」的调用方可以用 errors.Is 判断后忽略
var ErrNoFieldsToUpdate = errors.New("no fields to update")

// selectAll 在 UpdateSelected 中表示模型的全部顶层字段
const selectAll = "*"

// fieldSelection 汇总传入的 UpdateOpt，见 UpdateOpt
type fieldSelection struct {
	selecting bool
	selected  []string
	omitted   []string
}

// Update 更新匹配的一个文档，不存在时返回 NotFound 错误。target 表示更新后的目标状态，支持两种形式：
//
//   - map[string]any（或 bson.M）：key 即待更新的字段，可以写 bson 路径或 Go 字段名（见 path.SchemaFields.Resolve），
//     值会被赋给一个新的 T 对象中的对应字段（类型不同时按 path.Path.Set 的规则转换），再与 *T 一样经过 Encode。
//     此时不接受 UpdateOpt
//   - *T：必须通过 UpdateSelected / UpdateOmit 指定写入哪些字段，避免意外覆盖
//
// 在 Encode 的结果中，待更新字段存在的被 $set，不存在的（如 omitempty 的零值、map 中值为 nil 的指针字段）
// 被 $unset；map 中以 nil 为值的 map 条目（如 "envStates.prod": nil）会被删除。_id 与 createdAt 不能更新；
// updatedAt 默认刷新为当前时间（显式指定时以指定的值为准，传入 KeepUpdatedAt 时不刷新），
// 传入 *T 时会在写库之前回填到对象中（写入失败时同样如此，见 GenericStore）。
// 没有任何待更新字段时返回 ErrNoFieldsToUpdate。
//
// Update 不执行 BeforeSave（对象可能只填写了部分字段），Encode 钩子需要能处理只填写了部分字段的对象。
// 例如：
//
//	// 按 map 局部更新，false 也会被写入
//	err := widgets.Update(ctx, map[string]any{"enabled": false, "spec.color": "red"}, ByName(name))
//	// 更新对象中除 ownerID 外的全部字段，obj 中为空的 omitempty 字段会从库中删除
//	err := widgets.Update(ctx, obj, store.UpdateOmit("ownerID"), ByName(name))
//
// 支持的选项：CondOpt、UpdateOpt、TimestampOpt
func (s *GenericStore[T]) Update(ctx context.Context, target any, opts ...Opt) error {
	q, update, err := s.prepareUpdate("update", target, opts)
	if err != nil {
		return err
	}
	return s.updateOne(ctx, q, update)
}

// UpdateMany 更新全部匹配的文档，返回匹配的文档数量，没有匹配时不返回错误。
// target 与字段规则同 Update，每个文档写入相同的值。支持的选项：CondOpt、UpdateOpt、TimestampOpt
func (s *GenericStore[T]) UpdateMany(ctx context.Context, target any, opts ...Opt) (int64, error) {
	q, update, err := s.prepareUpdate("update many in", target, opts)
	if err != nil {
		return 0, err
	}
	res, err := s.coll.UpdateMany(ctx, q.filter(), update)
	if err != nil {
		return 0, s.wrapWriteErr(err, "update many in")
	}
	return res.MatchedCount, nil
}

// prepareUpdate 解析 Update / UpdateMany 的选项与 target，返回查询条件与更新文档
func (s *GenericStore[T]) prepareUpdate(op string, target any, opts []Opt) (*query, bson.M, error) {
	q, err := s.writeQuery(op, opts, updateOpts)
	if err != nil {
		return nil, nil, err
	}
	obj, paths, err := s.updateTarget(target, q.fields, false)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "%s %s", op, s.coll.Name())
	}
	_, set, unset, err := s.encodeUpdate(obj, paths, time.Now(), q.keepUpdatedAt)
	if err != nil {
		return nil, nil, err
	}
	return q, updateDoc(set, unset, nil), nil
}

// updateTarget 解析 Update / Upsert 的 target，返回待编码的对象与待写入的字段。
// upsert 为 true 时，*T 未传入 UpdateOpt 则选中全部顶层字段，并在计算字段之前执行 BeforeSave
func (s *GenericStore[T]) updateTarget(target any, sel *fieldSelection, upsert bool) (*T, []path.Path, error) {
	switch t := target.(type) {
	case *T:
		if t == nil {
			return nil, nil, errors.New("target object is nil")
		}
		if sel == nil {
			if !upsert {
				return nil, nil, errors.New(
					"specify the fields to update with UpdateSelected or UpdateOmit",
				)
			}
			sel = &fieldSelection{}
		}
		if upsert {
			if err := s.beforeSave(t); err != nil {
				return nil, nil, err
			}
		}
		paths, err := sel.paths(s.schema)
		return t, paths, err
	case bson.M:
		return s.objectFromMap(t, sel)
	case map[string]any:
		return s.objectFromMap(t, sel)
	}
	return nil, nil, errors.Errorf("unsupported target type %T, expected *%s or map[string]any", target, s.schema.typ)
}

// objectFromMap 将 m 中的值赋给一个新的 T 对象，返回该对象与 m 中的字段
func (s *GenericStore[T]) objectFromMap(m map[string]any, sel *fieldSelection) (*T, []path.Path, error) {
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
	paths := make([]path.Path, 0, len(keys))
	for _, key := range keys {
		p, err := s.schema.resolveUpdatePath(key)
		if err != nil {
			return nil, nil, err
		}
		if !p.Assignable {
			return nil, nil, errors.Errorf(
				"field %q cannot be set from a map, only struct fields and map entries are supported", key,
			)
		}
		if err = p.Set(v, m[key]); err != nil {
			return nil, nil, errors.Wrapf(err, "set field %q", key)
		}
		paths = append(paths, p)
	}
	paths, err := path.Dedupe(paths)
	return obj, paths, err
}

// paths 按 sel 计算模型 sc 中待写入的字段，规则见 UpdateOpt
func (sel *fieldSelection) paths(sc *schema) ([]path.Path, error) {
	// 合并多个 UpdateSelected("*") 时 selected 中会有重复的 "*"，仍表示全部字段
	all := !sel.selecting || slices.Equal(lo.Uniq(sel.selected), []string{selectAll})
	switch {
	case sel.selecting && len(sel.selected) == 0:
		return nil, errors.Wrap(ErrNoFieldsToUpdate, "UpdateSelected is given without any field")
	case !all && slices.Contains(sel.selected, selectAll):
		return nil, errors.Errorf("%q cannot be combined with other fields in UpdateSelected", selectAll)
	}

	var candidates []path.Path
	if all {
		for _, key := range sc.keys {
			if key == keyID || key == keyCreatedAt || key == keyUpdatedAt {
				continue
			}
			p, err := sc.fields.Resolve(key)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, p)
		}
	} else {
		for _, f := range sel.selected {
			p, err := sc.resolveUpdatePath(f)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, p)
		}
	}

	for _, f := range sel.omitted {
		p, err := sc.fields.ResolveKnown(f)
		if err != nil {
			return nil, errors.Wrap(err, "omitted field")
		}
		if parent, found := lo.Find(candidates, func(c path.Path) bool { return c.IsAncestorOf(p) }); found {
			return nil, errors.Errorf("cannot omit %q, its parent %q is selected as a whole", f, parent)
		}
		candidates = lo.Reject(candidates, func(c path.Path, _ int) bool { return c.String() == p.String() })
	}

	if len(candidates) == 0 {
		return nil, ErrNoFieldsToUpdate
	}
	return path.Dedupe(candidates)
}

// resolveUpdatePath 解析待写入的字段：必须存在于模型中，且不能是 _id 或 createdAt
func (s *schema) resolveUpdatePath(field string) (path.Path, error) {
	if isOperatorKey(field) {
		return path.Path{}, errors.Errorf(
			"update operator %q is not supported, use Modify for operators other than $set / $unset", field,
		)
	}
	p, err := s.fields.ResolveKnown(field)
	if err != nil {
		return path.Path{}, err
	}
	if top := p.Steps[0].Key; top == keyID || top == keyCreatedAt {
		return path.Path{}, errors.Errorf("field %q cannot be updated", field)
	}
	return p, nil
}

// encodeUpdate 刷新 obj 的 updatedAt（paths 中显式包含或 keepUpdatedAt 为 true 时不刷新）并执行 Encode，
// 再从编码结果中取出 paths 的值：存在的放入 set，不存在的放入 unset
func (s *GenericStore[T]) encodeUpdate(
	obj *T, paths []path.Path, now time.Time, keepUpdatedAt bool,
) (raw bson.Raw, set, unset bson.M, err error) {
	keys := lo.Map(paths, func(p path.Path, _ int) string { return p.String() })
	if s.schema.updatedAt != nil && !keepUpdatedAt && !slices.Contains(keys, keyUpdatedAt) {
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
