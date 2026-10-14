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

package path

import (
	"reflect"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Set 将 val 赋给结构体 v（必须可寻址，且是解析 p 的模型类型）中 p 所指的位置，途经的 nil 指针与 nil map
// 会被初始化。p 指向 map 条目且 val 为 nil 时删除该条目，使它在编码结果中不存在，从而被 Update 视为 $unset。
// 值的转换规则见 assignValue；p 必须是 Assignable 的
func (p Path) Set(v reflect.Value, val any) error {
	if !p.Assignable {
		return errors.Errorf("field path %q is not assignable", p)
	}
	return setSteps(v, p.Steps, val)
}

func setSteps(v reflect.Value, steps []Step, val any) error {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	st, rest := steps[0], steps[1:]
	if st.Kind == StepField {
		f := v.FieldByIndex(st.Index)
		if len(rest) == 0 {
			return assignValue(f, val)
		}
		return setSteps(f, rest, val)
	}

	// map 中的条目不可寻址，在副本上修改后再写回
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}
	key := reflect.ValueOf(st.Key).Convert(v.Type().Key())
	if len(rest) == 0 && val == nil {
		v.SetMapIndex(key, reflect.Value{})
		return nil
	}
	elem := reflect.New(v.Type().Elem()).Elem()
	var err error
	if len(rest) == 0 {
		err = assignValue(elem, val)
	} else {
		if old := v.MapIndex(key); old.IsValid() {
			elem.Set(old)
		}
		err = setSteps(elem, rest, val)
	}
	if err != nil {
		return err
	}
	v.SetMapIndex(key, elem)
	return nil
}

// assignValue 将 val 赋给 dst（必须可寻址）：
//   - 类型可以直接赋值时直接赋值，nil 表示零值（map 条目除外，见 Path.Set）
//   - 指针字段可以直接传入元素值，如 *bool 字段传入 true
//   - 底层类型相同的标量直接转换，如 string 写入 type SourceType string
//   - 其余情况借助 bson 编解码转换为字段类型，如 int 写入 int64、[]any 写入 []string；
//     注意经过 bson 转换后，interface 中的嵌套文档会变成 bson.D
func assignValue(dst reflect.Value, val any) error {
	if val == nil {
		dst.SetZero()
		return nil
	}
	src, dt := reflect.ValueOf(val), dst.Type()
	switch {
	case src.Type().AssignableTo(dt):
		dst.Set(src)
		return nil
	case dt.Kind() == reflect.Pointer && src.Type().AssignableTo(dt.Elem()):
		ptr := reflect.New(dt.Elem())
		ptr.Elem().Set(src)
		dst.Set(ptr)
		return nil
	case src.Kind() == dt.Kind() && isScalarKind(src.Kind()) && src.Type().ConvertibleTo(dt):
		dst.Set(src.Convert(dt))
		return nil
	}
	typ, data, err := bson.MarshalValue(val)
	if err != nil {
		return errors.Wrapf(err, "marshal %T", val)
	}
	if err = (bson.RawValue{Type: typ, Value: data}).Unmarshal(dst.Addr().Interface()); err != nil {
		return errors.Wrapf(err, "convert %T to %s", val, dt)
	}
	return nil
}

func isScalarKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
