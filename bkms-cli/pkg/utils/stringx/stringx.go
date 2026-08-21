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
	"unicode"
)

// ToWords 将标识符拆分成以单个空格分隔的小写单词，用于生成人类可读的展示文本。
// 支持驼峰、下划线、连字符等常见命名风格，例如：
//
//	appName        -> "app name"
//	appID          -> "app id"
//	HTTPServer     -> "http server"
//	scope_env_name -> "scope env name"
func ToWords(s string) string {
	var (
		words []string
		cur   []rune
	)
	// ToLower 会复制内容，因此可以安全地复用 cur 的底层数组
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}

	runes := []rune(s)
	for i, r := range runes {
		if isWordSeparator(r) {
			flush()
			continue
		}
		if i > 0 && startsNewWord(runes, i) {
			flush()
		}
		cur = append(cur, r)
	}
	flush()

	return strings.Join(words, " ")
}

// isWordSeparator 判断字符是否为单词分隔符，这类字符只用于断词，不进入结果
func isWordSeparator(r rune) bool {
	return r == '_' || r == '-' || r == '.' || unicode.IsSpace(r)
}

// startsNewWord 判断 runes[i] 是否为一个新单词的起始位置，只处理驼峰边界
func startsNewWord(runes []rune, i int) bool {
	if !unicode.IsUpper(runes[i]) {
		return false
	}
	// 小写字母或数字后紧跟大写字母，如 appName 中的 N、v1Alpha 中的 A
	if !unicode.IsUpper(runes[i-1]) {
		return true
	}
	// 连续大写视为缩写词，仅当其后跟着一个完整的小写单词时才断开，
	// 如 HTTPServer 断为 http + server；而 appIDs 结尾的 s 只是复数后缀，
	// 不应断开成 app + i + ds
	return countLowerFrom(runes, i+1) > 1
}

// countLowerFrom 统计从下标 i 开始连续出现的小写字母数量
func countLowerFrom(runes []rune, i int) int {
	count := 0
	for ; i < len(runes) && unicode.IsLower(runes[i]); i++ {
		count++
	}
	return count
}

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
