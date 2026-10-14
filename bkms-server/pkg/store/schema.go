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
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store/path"
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

// schema 是对模型结构体的反射分析结果，用于定位 _id 与时间戳字段，以及解析字段路径
type schema struct {
	typ    reflect.Type
	fields *path.SchemaFields
	// 以下为字段在结构体中的 index 路径，nil 表示模型没有该字段
	id        []int
	createdAt []int
	updatedAt []int
	// keys 模型编码后的全部顶层字段名，含 inline 结构体展开的字段；inline map 中的字段是动态的，不在其列
	keys []string
}

// newSchema 扫描顶层与 inline 的值类型结构体字段，因为这些是 _id、时间戳等顶层字段可能出现的位置
func newSchema(typ reflect.Type) (*schema, error) {
	fields, err := path.New(typ)
	if err != nil {
		return nil, err
	}
	s := &schema{typ: typ, fields: fields}
	for _, f := range fields.Fields() {
		s.keys = append(s.keys, f.Key)
		switch f.Key {
		case keyID:
			s.id = f.Index
		case keyCreatedAt, keyUpdatedAt:
			if f.Type != timeType {
				return nil, errors.Errorf(
					"scan model %s: field %s (%s) must be time.Time, got %s", typ, f.Name, f.Key, f.Type,
				)
			}
			if f.Key == keyCreatedAt {
				s.createdAt = f.Index
			} else {
				s.updatedAt = f.Index
			}
		}
	}
	return s, nil
}

// resolveKey 将过滤条件、排序中的字段名转换为 bson 路径，模型中不存在的字段原样保留
func (s *schema) resolveKey(field string) (string, error) {
	p, err := s.fields.Resolve(field)
	if err != nil {
		return "", err
	}
	return p.String(), nil
}

// resolveProjected 将 SelectFields / OmitFields 的字段转换为 bson 路径，label 用于错误信息：
// 字段必须存在于模型中，重复的字段只保留一个，互为父子的字段会被 MongoDB 判定为路径冲突，提前拒绝
func (s *schema) resolveProjected(fields []string, label string) ([]string, error) {
	if len(fields) == 0 {
		return nil, nil
	}
	paths := make([]path.Path, 0, len(fields))
	for _, f := range fields {
		p, err := s.fields.ResolveKnown(f)
		if err != nil {
			return nil, errors.Wrap(err, label)
		}
		paths = append(paths, p)
	}
	paths, err := path.Dedupe(paths)
	if err != nil {
		return nil, err
	}
	return lo.Map(paths, func(p path.Path, _ int) string { return p.String() }), nil
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
