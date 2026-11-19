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

package audit

import (
	"reflect"
	"strings"
	"sync"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// 写入操作记录前，按字段上的 audit tag 递归脱敏，结果为拷贝，不修改原对象：
//
//	`audit:"mask"`                非空即打码
//	`audit:"mask,if=IsSensitive"` 同一结构体中该 bool 字段为 true 时打码

// SensitiveValueMask 是操作记录中替换敏感值所用的掩码，与 envvars/types.SensitiveValueMask 保持一致。
const SensitiveValueMask = "******"

const (
	tagName = "audit"
	// maxSanitizeDepth 限制递归深度，防止循环引用导致栈溢出
	maxSanitizeDepth = 64
)

// sanitize 返回可写入操作记录的数据。tag 声明有误或嵌套过深时丢弃数据（返回 nil），避免写入明文。
func sanitize(data any) any {
	if data == nil {
		return nil
	}
	masked, changed, err := maskValue(reflect.ValueOf(data), 0)
	if err != nil {
		log.ErrorNoContextf("sanitize audit data %T failed, data dropped: %v", data, err)
		return nil
	}
	if !changed {
		return data
	}
	return masked.Interface()
}

// maskValue 返回 v 脱敏后的值。只有确实发生变化的路径才会复制，其余部分与原值共享。
func maskValue(v reflect.Value, depth int) (reflect.Value, bool, error) {
	if depth > maxSanitizeDepth {
		return v, false, errors.Errorf("nested deeper than %d", maxSanitizeDepth)
	}
	p := planOf(v.Type())
	if !p.walk {
		return v, false, nil
	}

	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return v, false, nil
		}
		return maskValue(v.Elem(), depth+1)
	case reflect.Pointer:
		if v.IsNil() {
			return v, false, nil
		}
		elem, changed, err := maskValue(v.Elem(), depth+1)
		if err != nil || !changed {
			return v, false, err
		}
		cp := reflect.New(v.Type().Elem())
		cp.Elem().Set(elem)
		return cp, true, nil
	case reflect.Struct:
		return maskStruct(v, p, depth)
	case reflect.Slice, reflect.Array:
		return maskList(v, depth)
	case reflect.Map:
		return maskMap(v, depth)
	default:
		return v, false, nil
	}
}

func maskStruct(v reflect.Value, p *typePlan, depth int) (reflect.Value, bool, error) {
	if p.err != nil {
		return v, false, p.err
	}
	var cp reflect.Value
	set := func(index int, nv reflect.Value) {
		if !cp.IsValid() {
			cp = reflect.New(v.Type()).Elem()
			cp.Set(v)
		}
		cp.Field(index).Set(nv)
	}

	for _, f := range p.fields {
		fv := v.Field(f.index)
		switch f.op {
		case opMask:
			if f.cond >= 0 && !v.Field(f.cond).Bool() {
				continue
			}
			if masked := maskString(fv.String()); masked != fv.String() {
				set(f.index, reflect.ValueOf(masked).Convert(fv.Type()))
			}
		case opWalk:
			nv, changed, err := maskValue(fv, depth+1)
			if err != nil {
				return v, false, errors.Wrapf(err, "field %s", v.Type().Field(f.index).Name)
			}
			if changed {
				set(f.index, nv)
			}
		}
	}
	if !cp.IsValid() {
		return v, false, nil
	}
	return cp, true, nil
}

func maskList(v reflect.Value, depth int) (reflect.Value, bool, error) {
	if v.Kind() == reflect.Slice && v.IsNil() {
		return v, false, nil
	}
	var cp reflect.Value
	for i := range v.Len() {
		elem, changed, err := maskValue(v.Index(i), depth+1)
		if err != nil {
			return v, false, err
		}
		if !changed {
			continue
		}
		if !cp.IsValid() {
			if v.Kind() == reflect.Slice {
				cp = reflect.MakeSlice(v.Type(), v.Len(), v.Len())
				reflect.Copy(cp, v)
			} else {
				cp = reflect.New(v.Type()).Elem()
				cp.Set(v)
			}
		}
		cp.Index(i).Set(elem)
	}
	if !cp.IsValid() {
		return v, false, nil
	}
	return cp, true, nil
}

func maskMap(v reflect.Value, depth int) (reflect.Value, bool, error) {
	if v.IsNil() {
		return v, false, nil
	}
	type entry struct{ key, value reflect.Value }
	var changes []entry
	iter := v.MapRange()
	for iter.Next() {
		value, changed, err := maskValue(iter.Value(), depth+1)
		if err != nil {
			return v, false, err
		}
		if changed {
			changes = append(changes, entry{iter.Key(), value})
		}
	}
	if len(changes) == 0 {
		return v, false, nil
	}
	cp := reflect.MakeMapWithSize(v.Type(), v.Len())
	iter = v.MapRange()
	for iter.Next() {
		cp.SetMapIndex(iter.Key(), iter.Value())
	}
	for _, c := range changes {
		cp.SetMapIndex(c.key, c.value)
	}
	return cp, true, nil
}

// fieldOp 结构体字段的处理方式
type fieldOp int

const (
	// opWalk 字段本身无 tag，但其类型可能包含需脱敏的数据，递归处理
	opWalk fieldOp = iota
	// opMask 对应 audit:"mask"
	opMask
)

type fieldPlan struct {
	index int
	op    fieldOp
	// cond 为 if= 指向的 bool 字段下标，-1 表示无条件
	cond int
}

// typePlan 描述某个类型的脱敏方式，按类型缓存
type typePlan struct {
	// walk 为 false 时该类型不可能包含需脱敏的数据，直接跳过
	walk bool
	// fields 仅结构体使用
	fields []fieldPlan
	// err 结构体 tag 声明有误
	err error
}

var (
	planCache sync.Map // reflect.Type -> *typePlan
	planMu    sync.Mutex
)

func planOf(t reflect.Type) *typePlan {
	if p, ok := planCache.Load(t); ok {
		return p.(*typePlan)
	}
	planMu.Lock()
	defer planMu.Unlock()
	// 先在局部构建，完整后再发布，避免并发读到未完成的 plan
	building := map[reflect.Type]*typePlan{}
	p := buildPlan(t, building)
	for typ, plan := range building {
		planCache.Store(typ, plan)
	}
	return p
}

func buildPlan(t reflect.Type, building map[reflect.Type]*typePlan) *typePlan {
	if p, ok := planCache.Load(t); ok {
		return p.(*typePlan)
	}
	if p, ok := building[t]; ok {
		return p
	}
	// 递归类型在构建完成前按"需要遍历"处理，偏保守
	p := &typePlan{walk: true}
	building[t] = p

	switch t.Kind() {
	case reflect.Interface:
		p.walk = true
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		p.walk = buildPlan(t.Elem(), building).walk
	case reflect.Struct:
		p.fields, p.err = buildFieldPlans(t, building)
		p.walk = len(p.fields) > 0 || p.err != nil
	default:
		p.walk = false
	}
	return p
}

func buildFieldPlans(t reflect.Type, building map[reflect.Type]*typePlan) ([]fieldPlan, error) {
	var plans []fieldPlan
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag, ok := f.Tag.Lookup(tagName)
		if !ok {
			if buildPlan(f.Type, building).walk {
				plans = append(plans, fieldPlan{index: i, op: opWalk, cond: -1})
			}
			continue
		}
		plan, err := parseFieldTag(t, f, tag)
		if err != nil {
			return nil, errors.Wrapf(err, "%s.%s", t, f.Name)
		}
		plan.index = i
		plans = append(plans, plan)
	}
	return plans, nil
}

func parseFieldTag(t reflect.Type, f reflect.StructField, tag string) (fieldPlan, error) {
	opts := strings.Split(tag, ",")
	switch opts[0] {
	case "mask":
		if f.Type.Kind() != reflect.String {
			return fieldPlan{}, errors.New(`audit:"mask" requires a string field`)
		}
		plan := fieldPlan{op: opMask, cond: -1}
		for _, opt := range opts[1:] {
			name, ok := strings.CutPrefix(opt, "if=")
			if !ok {
				return fieldPlan{}, errors.Errorf("unknown audit tag option %q", opt)
			}
			cond, found := t.FieldByName(name)
			if !found || len(cond.Index) != 1 || !cond.IsExported() || cond.Type.Kind() != reflect.Bool {
				return fieldPlan{}, errors.Errorf(
					"audit tag if=%s requires an exported bool field in the same struct", name,
				)
			}
			plan.cond = cond.Index[0]
		}
		return plan, nil
	default:
		return fieldPlan{}, errors.Errorf("unknown audit tag %q", tag)
	}
}

// maskString 对非空字符串打码；空串保持为空，以便看出该字段是否设置过。
func maskString(s string) string {
	if s == "" {
		return ""
	}
	return SensitiveValueMask
}
