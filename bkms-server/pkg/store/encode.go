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
	"slices"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (s *GenericStore[T]) beforeSave(obj *T) error {
	for _, hook := range s.meta.BeforeSave {
		if err := hook(obj); err != nil {
			return errors.Wrapf(err, "before save into %s", s.coll.Name())
		}
	}
	return nil
}

// encode 返回待写入的对象：在深拷贝上依次执行 Encode 钩子，调用方的 obj 保持不变
func (s *GenericStore[T]) encode(obj *T) (*T, error) {
	if len(s.meta.Encode) == 0 {
		return obj, nil
	}
	cp, err := deepCopy(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "copy document for %s", s.coll.Name())
	}
	for _, hook := range s.meta.Encode {
		if err := hook(cp); err != nil {
			return nil, errors.Wrapf(err, "encode document for %s", s.coll.Name())
		}
	}
	return cp, nil
}

// deepCopy 深拷贝对象，供 Encode 钩子在副本上修改。
//
// IgnoreEmpty：目标是全新的零值对象，跳过零值字段不会丢数据；同时避免 copier 把 nil 切片、nil map
// 拷贝为空值，否则未设置 omitempty 的字段编码结果会从 null 变成 []。
// 注意 copier 不会深拷贝 interface 字段中的指针、数组中的切片，Encode 不应通过这两类路径修改数据
func deepCopy[T any](src *T) (*T, error) {
	dst := new(T)
	if err := copier.CopyWithOption(dst, src, copier.Option{DeepCopy: true, IgnoreEmpty: true}); err != nil {
		return nil, errors.Wrap(err, "deep copy")
	}
	return dst, nil
}

func (s *GenericStore[T]) encodeRaw(obj *T) (bson.Raw, error) {
	enc, err := s.encode(obj)
	if err != nil {
		return nil, err
	}
	raw, err := bson.Marshal(enc)
	if err != nil {
		return nil, errors.Wrapf(err, "marshal document for %s", s.coll.Name())
	}
	return raw, nil
}

// encodeDoc 返回编码后的顶层字段，并去掉 omit 中的字段（如不可修改的 _id）
func (s *GenericStore[T]) encodeDoc(obj *T, omit ...string) (bson.D, error) {
	raw, err := s.encodeRaw(obj)
	if err != nil {
		return nil, err
	}
	elems, err := raw.Elements()
	if err != nil {
		return nil, errors.Wrapf(err, "read encoded document for %s", s.coll.Name())
	}
	doc := make(bson.D, 0, len(elems))
	for _, e := range elems {
		if !slices.Contains(omit, e.Key()) {
			doc = append(doc, bson.E{Key: e.Key(), Value: e.Value()})
		}
	}
	return doc, nil
}
