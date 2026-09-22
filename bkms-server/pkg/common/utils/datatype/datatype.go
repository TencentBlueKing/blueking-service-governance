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

// Package datatype 提供基础数据结构的通用操作
package datatype

import "github.com/jinzhu/copier"

// CloneMap 深拷贝一个 map，使调用方无法通过返回值改到原始数据。
//
// 嵌套容器一并拷贝，包括 bson.D / bson.A —— 它们的底层是切片，只遍历
// map[string]any 和 []any 的浅层拷贝会让副本仍然共享其底层数组。
func CloneMap[K comparable, V any](in map[K]V) map[K]V {
	if in == nil {
		return nil
	}

	out := make(map[K]V, len(in))
	// copier 仅在类型不受支持时报错，这里两侧都是同一个 map 类型。
	_ = copier.CopyWithOption(&out, &in, copier.Option{DeepCopy: true})
	return out
}
