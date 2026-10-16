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

// Package path 处理模型结构体中参与 bson 编码的字段：扫描字段，并在 bson key 与 Go 字段名之间转换字段路径
package path

import (
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

var (
	// ErrUnknownField 字段路径在模型中不存在
	ErrUnknownField = errors.New("unknown field")
	// ErrAmbiguousField 字段路径中的某一段既是一个字段的 bson key，又是另一个字段的 Go 字段名
	ErrAmbiguousField = errors.New("ambiguous field")
)

var timeType = reflect.TypeFor[time.Time]()

// Field 结构体中参与 bson 编码的一个字段
type Field struct {
	// Key 字段的 bson key
	Key string
	// Name 字段的 Go 字段名
	Name string
	// Index 字段在结构体中的 index 路径，inline 展开的字段包含嵌入结构体自身的下标
	Index []int
	Type  reflect.Type
}

// SchemaFields 模型结构体的字段表。创建时扫描模型，以及能通过字段路径到达的全部结构体
// （嵌套字段、指针、数组元素、key 为字符串的 map 的值），之后只读，可以并发使用
type SchemaFields struct {
	typ reflect.Type
	// fields 结构体类型 -> 该结构体参与编码的字段
	fields map[reflect.Type][]Field
}

// New 扫描模型结构体 typ 的字段
func New(typ reflect.Type) (*SchemaFields, error) {
	if typ.Kind() != reflect.Struct {
		return nil, errors.Errorf("model must be a struct, got %s", typ)
	}
	s := &SchemaFields{typ: typ, fields: map[reflect.Type][]Field{}}
	s.scan(typ)
	return s, nil
}

// Type 返回模型结构体的类型
func (s *SchemaFields) Type() reflect.Type {
	return s.typ
}

// Fields 返回模型的顶层字段，inline 嵌入的值类型结构体会被展开；inline map 中的字段是动态的，不在其列
func (s *SchemaFields) Fields() []Field {
	return s.fields[s.typ]
}

// Lookup 按 bson key 或 Go 字段名查找模型的顶层字段，规则见 Resolve
func (s *SchemaFields) Lookup(name string) (Field, error) {
	return s.lookup(s.typ, name)
}

// scan 收集 typ 及能从它到达的结构体的字段，遍历类型的方式须与 Resolve 保持一致，
// 保证 Resolve 途经的每个结构体都已扫描
func (s *SchemaFields) scan(typ reflect.Type) {
	typ = derefType(typ)
	switch {
	case isArrayType(typ):
		s.scan(typ.Elem())
	case typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String:
		s.scan(typ.Elem())
	case isDocument(typ):
		// 先登记再递归，自引用的类型（如树形结构）才不会无限递归
		if _, ok := s.fields[typ]; ok {
			return
		}
		fields := structFields(typ)
		s.fields[typ] = fields
		for _, f := range fields {
			s.scan(f.Type)
		}
	}
}

// lookup 在结构体 typ 中按 bson key 或 Go 字段名查找字段。seg 同时是两个不同字段的 bson key 与 Go 字段名时，
// 无法判断调用方指的是哪一个，返回 ErrAmbiguousField
func (s *SchemaFields) lookup(typ reflect.Type, seg string) (Field, error) {
	fields := s.fields[typ]
	byKey, okKey := lo.Find(fields, func(f Field) bool { return f.Key == seg })
	byName, okName := lo.Find(fields, func(f Field) bool { return f.Name == seg })
	switch {
	case okKey && okName && !slices.Equal(byKey.Index, byName.Index):
		return Field{}, errors.Wrapf(ErrAmbiguousField,
			"%q is the bson key of %s.%s and the name of %s.%s", seg, typ, byKey.Name, typ, byName.Name)
	case okKey:
		return byKey, nil
	case okName:
		return byName, nil
	}
	return Field{}, errors.Wrapf(ErrUnknownField, "%s has no field %q", typ, seg)
}

// Resolve 将以 "." 分隔的字段路径转换为 bson 路径。每一段既可以写 bson key，也可以写 Go 字段名，
// inline 嵌入结构体中的字段直接写字段本身的名字，例如 Widget 的 "Spec.Color" 与 "spec.color" 等价。
//
// 数组按 MongoDB 点号路径的规则穿透：数字段与位置操作符（$、$[]、$[id]）指向元素，其余段作用于数组元素。
// 进入 interface 或 key 不是字符串的 map 后无法静态解析，剩余的段原样保留。
// 遇到模型中不存在的段时 Known 为 false，不返回错误，由调用方决定是否接受；只有歧义时返回错误。
// 字段必须存在于模型中时使用 ResolveKnown
func (s *SchemaFields) Resolve(path string) (Path, error) {
	p, _, err := s.resolve(path)
	return p, err
}

// ResolveKnown 同 Resolve，但字段路径必须能在模型中找到，否则返回 ErrUnknownField，并指明是哪一段。
// 例如 Widget 的 "spec.Missing.x" 返回的错误中包含 `WidgetSpec has no field "Missing"`
func (s *SchemaFields) ResolveKnown(path string) (Path, error) {
	p, unknown, err := s.resolve(path)
	if err != nil {
		return Path{}, err
	}
	if unknown != nil {
		return Path{}, errors.Wrapf(unknown, "resolve field path %q", path)
	}
	return p, nil
}

// resolve 实现 Resolve。遇到模型中不存在的段时，unknown 为说明是哪一段的 ErrUnknownField，err 仍为 nil
func (s *SchemaFields) resolve(path string) (p Path, unknown, err error) {
	segs := strings.Split(path, ".")
	p = Path{Steps: make([]Step, 0, len(segs)), Known: true, Assignable: true}
	keepRest := func(rest []string) Path {
		for _, seg := range rest {
			p.Steps = append(p.Steps, Step{Key: seg, Kind: StepOpaque})
		}
		p.Assignable = false
		return p
	}
	typ := s.typ
	for i, seg := range segs {
		typ = derefType(typ)
		for isArrayType(typ) && !isArrayIndex(seg) {
			typ = derefType(typ.Elem())
			p.Assignable = false
		}
		switch {
		case isArrayType(typ):
			p.Steps = append(p.Steps, Step{Key: seg, Kind: StepOpaque})
			p.Assignable = false
			typ = typ.Elem()
		case isDocument(typ):
			f, lookupErr := s.lookup(typ, seg)
			if errors.Is(lookupErr, ErrUnknownField) {
				p.Known = false
				return keepRest(segs[i:]), lookupErr, nil
			}
			if lookupErr != nil {
				return Path{}, nil, errors.Wrapf(lookupErr, "resolve field path %q", path)
			}
			p.Steps = append(p.Steps, Step{Key: f.Key, Kind: StepField, Index: f.Index})
			typ = f.Type
		case typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String:
			p.Steps = append(p.Steps, Step{Key: seg, Kind: StepMapKey})
			typ = typ.Elem()
		case typ.Kind() == reflect.Interface || typ.Kind() == reflect.Map:
			return keepRest(segs[i:]), nil, nil
		default:
			// 路径进入了标量字段，如 "name.x"
			p.Known = false
			return keepRest(segs[i:]), errors.Wrapf(ErrUnknownField, "%s has no field %q", typ, seg), nil
		}
	}
	return p, nil, nil
}

// structFields 返回结构体 typ 参与编码的字段，inline 嵌入的值类型结构体会被展开；
// inline map 中的字段是动态的，不在其列
func structFields(typ reflect.Type) []Field {
	var fields []Field
	var collect func(t reflect.Type, index []int)
	collect = func(t reflect.Type, index []int) {
		for i := range t.NumField() {
			sf := t.Field(i)
			key, inline, ok := bsonKey(sf)
			if !ok {
				continue
			}
			idx := append(slices.Clone(index), i)
			if inline {
				if sf.Type.Kind() == reflect.Struct {
					collect(sf.Type, idx)
				}
				continue
			}
			fields = append(fields, Field{Key: key, Name: sf.Name, Index: idx, Type: sf.Type})
		}
	}
	collect(typ, nil)
	return fields
}

// bsonKey 解析结构体字段的 bson key，规则与 mongo-driver 默认的 struct tag 解析（parseStructTags）
// 保持一致：未设置 key 时使用小写的字段名；"-" 与未导出字段不参与编码，此时 ok 为 false
func bsonKey(sf reflect.StructField) (key string, inline, ok bool) {
	if !sf.IsExported() {
		return "", false, false
	}
	tag, found := sf.Tag.Lookup("bson")
	// 兼容驱动支持的旧式写法：整个 struct tag 即 bson tag，如 `myb,omitempty`
	if !found && !strings.Contains(string(sf.Tag), ":") {
		tag = string(sf.Tag)
	}
	if tag == "-" {
		return "", false, false
	}
	key, opts, _ := strings.Cut(tag, ",")
	inline = slices.Contains(strings.Split(opts, ","), "inline")
	if key == "" {
		key = strings.ToLower(sf.Name)
	}
	return key, inline, true
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

// isDocument 判断 typ 是否是编码为 bson 文档、可以按字段继续解析的结构体；time.Time 编码为日期，不在其列
func isDocument(typ reflect.Type) bool {
	return typ.Kind() == reflect.Struct && typ != timeType
}
