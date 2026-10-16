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
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

var (
	// errUnknownField 字段路径在模型中不存在
	errUnknownField = errors.New("unknown field")
	// errAmbiguousField 字段路径中的某一段既是一个字段的 bson key，又是另一个字段的 Go 字段名
	errAmbiguousField = errors.New("ambiguous field")
)

// structField 结构体中参与 bson 编码的一个字段
type structField struct {
	key  string
	name string
	// index 字段在结构体中的 index 路径，inline 展开的字段包含嵌入结构体自身的下标
	index []int
	typ   reflect.Type
}

// structFieldsCache 缓存结构体的字段列表（reflect.Type -> []structField）
var structFieldsCache sync.Map

// structFields 返回结构体参与编码的字段，inline 嵌入的值类型结构体会被展开；
// inline map 中的字段是动态的，不在其列
func structFields(typ reflect.Type) []structField {
	if cached, ok := structFieldsCache.Load(typ); ok {
		return cached.([]structField)
	}
	var fields []structField
	var scan func(t reflect.Type, index []int)
	scan = func(t reflect.Type, index []int) {
		for i := range t.NumField() {
			sf := t.Field(i)
			key, inline, ok := bsonKey(sf)
			if !ok {
				continue
			}
			idx := append(slices.Clone(index), i)
			if inline {
				if sf.Type.Kind() == reflect.Struct {
					scan(sf.Type, idx)
				}
				continue
			}
			fields = append(fields, structField{key: key, name: sf.Name, index: idx, typ: sf.Type})
		}
	}
	scan(typ, nil)
	actual, _ := structFieldsCache.LoadOrStore(typ, fields)
	return actual.([]structField)
}

// lookupField 按 bson key 或 Go 字段名查找字段。seg 同时是两个不同字段的 bson key 与 Go 字段名时，
// 无法判断调用方指的是哪一个，返回 errAmbiguousField
func lookupField(typ reflect.Type, seg string) (structField, error) {
	fields := structFields(typ)
	byKey, okKey := lo.Find(fields, func(f structField) bool { return f.key == seg })
	byName, okName := lo.Find(fields, func(f structField) bool { return f.name == seg })
	switch {
	case okKey && okName && !slices.Equal(byKey.index, byName.index):
		return structField{}, errors.Wrapf(errAmbiguousField,
			"%q is the bson key of %s.%s and the name of %s.%s", seg, typ, byKey.name, typ, byName.name)
	case okKey:
		return byKey, nil
	case okName:
		return byName, nil
	}
	return structField{}, errors.Wrapf(errUnknownField, "%s has no field %q", typ, seg)
}

// stepKind 字段路径中一段的类型
type stepKind uint8

const (
	// stepField 结构体字段（含 inline 展开的字段）
	stepField stepKind = iota
	// stepMapKey key 为字符串的 map 中的条目
	stepMapKey
	// stepOpaque 无法静态解析的段，如数组下标、interface 中的字段、模型中不存在的字段，原样保留
	stepOpaque
)

type pathStep struct {
	key  string
	kind stepKind
	// index 仅 stepField 使用，为字段在所在结构体中的 index 路径
	index []int
}

// fieldPath 解析后的字段路径
type fieldPath struct {
	steps []pathStep
	// known 每一段都能在模型中找到；为 false 时，第一个找不到的段及之后的段原样保留
	known bool
	// assignable 路径只经过结构体字段与 map 条目，可以直接在对象上定位、赋值
	assignable bool
}

// String 返回 bson 路径
func (p fieldPath) String() string {
	return strings.Join(lo.Map(p.steps, func(st pathStep, _ int) string { return st.key }), ".")
}

// isAncestorOf 判断 p 是否是 other 的上级字段，如 "spec" 是 "spec.color" 的上级，但不是 "specs" 的
func (p fieldPath) isAncestorOf(other fieldPath) bool {
	return strings.HasPrefix(other.String(), p.String()+".")
}

// resolveKey 将过滤条件、排序中的字段名转换为 bson 路径，模型中不存在的字段原样保留
func resolveKey(typ reflect.Type, path string) (string, error) {
	p, err := resolvePath(typ, path)
	if err != nil {
		return "", err
	}
	return p.String(), nil
}

// dedupePaths 去掉重复的字段（如同一字段分别以 bson key 与 Go 字段名指定），并拒绝互为父子的字段，
// 后者会被 MongoDB 判定为更新冲突
func dedupePaths(paths []fieldPath) ([]fieldPath, error) {
	paths = lo.UniqBy(paths, fieldPath.String)
	for _, a := range paths {
		for _, b := range paths {
			if a.isAncestorOf(b) {
				return nil, errors.Errorf("conflicting fields %q and %q, keep only one of them", a, b)
			}
		}
	}
	return paths, nil
}

// resolvePath 将以 "." 分隔的字段路径转换为 bson 路径。每一段既可以写 bson key，也可以写 Go 字段名，
// inline 嵌入结构体中的字段直接写字段本身的名字，例如 Widget 的 "Spec.Color" 与 "spec.color" 等价。
//
// 数组按 MongoDB 点号路径的规则穿透：数字段是下标，其余段作用于数组元素。
// 进入 interface 或 key 不是字符串的 map 后无法静态解析，剩余的段原样保留。
// 遇到模型中不存在的段时 known 为 false，不返回错误，由调用方决定是否接受；只有歧义时返回错误
func resolvePath(typ reflect.Type, path string) (fieldPath, error) {
	segs := strings.Split(path, ".")
	p := fieldPath{steps: make([]pathStep, 0, len(segs)), known: true, assignable: true}
	keepRest := func(rest []string) fieldPath {
		for _, seg := range rest {
			p.steps = append(p.steps, pathStep{key: seg, kind: stepOpaque})
		}
		p.assignable = false
		return p
	}
	for i, seg := range segs {
		typ = derefType(typ)
		for isArrayType(typ) && !isArrayIndex(seg) {
			typ = derefType(typ.Elem())
			p.assignable = false
		}
		switch {
		case isArrayType(typ):
			p.steps = append(p.steps, pathStep{key: seg, kind: stepOpaque})
			p.assignable = false
			typ = typ.Elem()
		case typ.Kind() == reflect.Struct && typ != timeType:
			f, err := lookupField(typ, seg)
			if errors.Is(err, errUnknownField) {
				p.known = false
				return keepRest(segs[i:]), nil
			}
			if err != nil {
				return fieldPath{}, errors.Wrapf(err, "resolve field path %q", path)
			}
			p.steps = append(p.steps, pathStep{key: f.key, kind: stepField, index: f.index})
			typ = f.typ
		case typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String:
			p.steps = append(p.steps, pathStep{key: seg, kind: stepMapKey})
			typ = typ.Elem()
		case typ.Kind() == reflect.Interface || typ.Kind() == reflect.Map:
			return keepRest(segs[i:]), nil
		default:
			// 路径进入了标量字段，如 "name.x"
			p.known = false
			return keepRest(segs[i:]), nil
		}
	}
	return p, nil
}

func derefType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func isArrayType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array
}

func isArrayIndex(seg string) bool {
	_, err := strconv.ParseUint(seg, 10, 64)
	return err == nil
}
