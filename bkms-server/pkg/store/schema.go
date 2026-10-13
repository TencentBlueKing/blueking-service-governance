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
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	keyID        = "_id"
	keyCreatedAt = "createdAt"
	keyUpdatedAt = "updatedAt"
)

var (
	timeType     = reflect.TypeFor[time.Time]()
	objectIDType = reflect.TypeFor[bson.ObjectID]()
)

// schema 是对模型结构体的反射分析结果，用于定位 _id 与时间戳字段
type schema struct {
	typ reflect.Type
	// 以下为字段在结构体中的 index 路径，nil 表示模型没有该字段
	id        []int
	createdAt []int
	updatedAt []int
}

func newSchema(typ reflect.Type) (*schema, error) {
	if typ.Kind() != reflect.Struct {
		return nil, errors.Errorf("model must be a struct, got %s", typ)
	}
	s := &schema{typ: typ}
	if err := s.scan(typ, nil); err != nil {
		return nil, errors.Wrapf(err, "scan model %s", typ)
	}
	return s, nil
}

// scan 只扫描顶层与 inline 的值类型结构体字段，因为这些是 _id 与时间戳可能出现的位置
func (s *schema) scan(typ reflect.Type, index []int) error {
	for i := range typ.NumField() {
		sf := typ.Field(i)
		key, inline, ok := bsonKey(sf)
		if !ok {
			continue
		}
		idx := append(slices.Clone(index), i)
		if inline {
			if sf.Type.Kind() == reflect.Struct {
				if err := s.scan(sf.Type, idx); err != nil {
					return err
				}
			}
			continue
		}
		switch key {
		case keyID:
			s.id = idx
		case keyCreatedAt, keyUpdatedAt:
			if sf.Type != timeType {
				return errors.Errorf("field %s (%s) must be time.Time, got %s", sf.Name, key, sf.Type)
			}
			if key == keyCreatedAt {
				s.createdAt = idx
			} else {
				s.updatedAt = idx
			}
		}
	}
	return nil
}

// fillID 当 _id 为零值时生成新的 ID：ObjectID 类型生成 ObjectID，string 类型生成其 hex 形式。
// 在写入前生成 ID，调用方无需再从 InsertedID 中回读
func (s *schema) fillID(v reflect.Value) {
	if s.id == nil {
		return
	}
	if f := v.FieldByIndex(s.id); f.IsZero() {
		if gen, ok := s.newID(); ok {
			f.Set(reflect.ValueOf(gen).Convert(f.Type()))
		}
	}
}

// newID 根据 _id 字段的类型生成新的 ID，不支持的类型返回 false，交由 MongoDB 自行生成
func (s *schema) newID() (any, bool) {
	if s.id == nil {
		return nil, false
	}
	switch t := s.typ.FieldByIndex(s.id).Type; {
	case t == objectIDType:
		return bson.NewObjectID(), true
	case t.Kind() == reflect.String:
		return bson.NewObjectID().Hex(), true
	default:
		return nil, false
	}
}

// fillCreated 为新文档补全时间戳：未设置的 createdAt 取当前时间，updatedAt 与 createdAt 保持一致
func (s *schema) fillCreated(v reflect.Value, now time.Time) {
	created := now
	if s.createdAt != nil {
		f := v.FieldByIndex(s.createdAt)
		if f.IsZero() {
			f.Set(reflect.ValueOf(now))
		}
		created = f.Interface().(time.Time)
	}
	if s.updatedAt != nil {
		v.FieldByIndex(s.updatedAt).Set(reflect.ValueOf(created))
	}
}

func (s *schema) setUpdated(v reflect.Value, now time.Time) {
	if s.updatedAt != nil {
		v.FieldByIndex(s.updatedAt).Set(reflect.ValueOf(now))
	}
}

func (s *schema) createdAtOf(v reflect.Value) time.Time {
	return v.FieldByIndex(s.createdAt).Interface().(time.Time)
}

// setID 将数据库返回的 _id（如 UpsertedID）写回对象；类型不匹配时（如模型未声明 _id 字段的具体类型）忽略
func (s *schema) setID(v reflect.Value, id any) {
	if s.id == nil || id == nil {
		return
	}
	f, rv := v.FieldByIndex(s.id), reflect.ValueOf(id)
	switch {
	case rv.Type().AssignableTo(f.Type()):
		f.Set(rv)
	case rv.Kind() == f.Kind() && rv.Type().ConvertibleTo(f.Type()):
		f.Set(rv.Convert(f.Type()))
	}
}

// idOf 返回对象的 _id，零值时返回 false
func (s *schema) idOf(v reflect.Value) (any, bool) {
	if s.id == nil {
		return nil, false
	}
	f := v.FieldByIndex(s.id)
	return f.Interface(), !f.IsZero()
}

// hasPath 判断以 "." 分隔的 bson 路径是否是模型中合法的字段，用于在更新前拦截拼写错误。
// 路径进入 map、interface、slice、array 后无法静态校验，视为合法
func (s *schema) hasPath(path string) bool {
	return typeHasPath(s.typ, strings.Split(path, "."))
}

func typeHasPath(typ reflect.Type, segs []string) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if len(segs) == 0 {
		return true
	}
	switch typ.Kind() {
	case reflect.Map, reflect.Interface, reflect.Slice, reflect.Array:
		return true
	case reflect.Struct:
		if typ == timeType {
			return false
		}
		for i := range typ.NumField() {
			sf := typ.Field(i)
			key, inline, ok := bsonKey(sf)
			if !ok {
				continue
			}
			if inline {
				if typeHasPath(sf.Type, segs) {
					return true
				}
				continue
			}
			if key == segs[0] {
				return typeHasPath(sf.Type, segs[1:])
			}
		}
	}
	return false
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
