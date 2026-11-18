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
)

// SensitiveValueMask 是操作记录中替换敏感值所用的掩码，与 envvars/types.SensitiveValueMask 保持一致。
const SensitiveValueMask = "******"

// sanitizers 按数据类型登记的脱敏函数，只在 init 阶段写入，之后只读，无需加锁。
var sanitizers = map[reflect.Type]func(any) any{}

// RegisterSanitizer 登记类型 T 写入操作记录前的脱敏函数，只能在 init 中调用。
//
// WithDataBefore / WithDataAfter / WithExtras 遇到 T 或 *T 类型的数据时，序列化 fn 的返回值。
// fn 必须返回脱敏后的拷贝，不得修改入参：调用方往往还会用原对象构造接口响应。
//
// 业务模型的脱敏函数统一登记在子包 audit/sanitizers 中，由程序入口以 blank import 引入。
func RegisterSanitizer[T any](fn func(T) any) {
	sanitizers[reflect.TypeFor[T]()] = func(v any) any { return fn(v.(T)) }
}

// sanitize 返回可写入操作记录的数据。nil 指针原样返回，保持序列化为 null。
func sanitize(data any) any {
	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return data
	}
	if fn, ok := sanitizers[v.Type()]; ok {
		return fn(data)
	}
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return data
	}
	if fn, ok := sanitizers[v.Type().Elem()]; ok {
		return fn(v.Elem().Interface())
	}
	return data
}

// MaskString 对非空字符串打码；空串保持为空，以便看出该字段是否设置过。
func MaskString(s string) string {
	if s == "" {
		return ""
	}
	return SensitiveValueMask
}

// sensitiveKeySuffixes 是组件属性等无结构数据中视为敏感的 key 后缀（不区分大小写）。
// 用后缀而不是包含关系，避免误伤 secretName 这类只是引用名称的字段。
var sensitiveKeySuffixes = []string{"token", "password", "passwd", "secret"}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, suffix := range sensitiveKeySuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// MaskSensitiveProperties 返回 props 的深拷贝，其中 key 以 token / password / passwd / secret 结尾
// （不区分大小写）的非空字符串值被打码，嵌套的 map 与 slice 递归处理。props 为 nil 时返回 nil。
func MaskSensitiveProperties(props map[string]any) map[string]any {
	if props == nil {
		return nil
	}
	masked := make(map[string]any, len(props))
	for k, v := range props {
		if s, ok := v.(string); ok && isSensitiveKey(k) {
			masked[k] = MaskString(s)
			continue
		}
		masked[k] = maskNestedValue(v)
	}
	return masked
}

func maskNestedValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return MaskSensitiveProperties(val)
	case []any:
		items := make([]any, len(val))
		for i, item := range val {
			items[i] = maskNestedValue(item)
		}
		return items
	default:
		return v
	}
}
