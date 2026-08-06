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

// Package stringx 提供通用的字符串处理工具函数
package stringx

import (
	"reflect"
	"strings"
)

// TrimSpaceRecursive 递归对结构体中所有 string 字段执行 TrimSpace
func TrimSpaceRecursive(v reflect.Value) {
	switch v.Kind() {
	case reflect.Ptr:
		if !v.IsNil() {
			TrimSpaceRecursive(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			TrimSpaceRecursive(v.Field(i))
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(strings.TrimSpace(v.String()))
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			TrimSpaceRecursive(v.Index(i))
		}
	case reflect.Map:
		if v.IsNil() {
			return
		}
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			// map 的 value 不可直接 Set，需要替换
			if val.Kind() == reflect.String {
				v.SetMapIndex(key, reflect.ValueOf(strings.TrimSpace(val.String())))
			}
			// key 也 trim
			if key.Kind() == reflect.String {
				trimmedKey := strings.TrimSpace(key.String())
				if trimmedKey != key.String() {
					v.SetMapIndex(key, reflect.Value{}) // 删除旧 key
					v.SetMapIndex(reflect.ValueOf(trimmedKey), reflect.ValueOf(strings.TrimSpace(val.String())))
				}
			}
		}
	}
}
