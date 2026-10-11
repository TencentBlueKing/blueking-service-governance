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

package tenant

// Scopable 由领域 Store 接口嵌入，T 是该 Store 接口自身。
type Scopable[T any] interface {
	// Scopes 返回按给定范围派生的 Store，不修改当前 Store。后传入的范围覆盖先传入的范围。
	Scopes(scopes ...Scope) T
}

// Base 由 Store 实现嵌入，持有 Collection 并实现 Scopes。
type Base[T any] struct {
	collection *Collection
	// wrap 把换过范围的 Base 组装成 Store
	wrap func(Base[T]) T
}

// NewBase 构造 Base。wrap 只负责把传入的 Base 和 Store 自己的字段组装成新 Store。
func NewBase[T any](coll *Collection, wrap func(Base[T]) T) Base[T] {
	if wrap == nil {
		panic("tenant: NewBase requires wrap")
	}
	return Base[T]{collection: coll, wrap: wrap}
}

// Scopes 实现 Scopable。
func (b Base[T]) Scopes(scopes ...Scope) T {
	next := b
	for _, scope := range scopes {
		if scope == nil {
			continue
		}
		next.collection = scope(next.collection)
	}
	return b.wrap(next)
}

// Collection 返回当前范围下的 Collection。
func (b Base[T]) Collection() *Collection {
	return b.collection
}
