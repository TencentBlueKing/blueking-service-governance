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

// Package urlmask 脱敏 URL 中的敏感查询参数
package urlmask

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	// Masked 敏感值替换后的内容
	Masked = "***"
	// maxQueryLen 脱敏后 query 最大长度
	maxQueryLen = 1024
)

// sensitiveKeyParts 参数名含以下片段时视为敏感
var sensitiveKeyParts = []string{"token", "ticket", "secret", "password", "passwd", "credential"}

// Query 将敏感参数的值替换为 Masked，其余部分保持原样（顺序与编码不变），超长时截断
func Query(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	parts := strings.Split(rawQuery, "&")
	for i, part := range parts {
		key, _, hasValue := strings.Cut(part, "=")
		if !hasValue {
			continue
		}
		name, err := url.QueryUnescape(key)
		if err != nil {
			name = key
		}
		if isSensitiveKey(name) {
			parts[i] = key + "=" + Masked
		}
	}
	masked := strings.Join(parts, "&")
	return TruncateUTF8(masked, maxQueryLen)
}

// URL 脱敏完整 URL 中的查询参数，无法解析时返回 Masked
func URL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return Masked
	}
	u.RawQuery = Query(u.RawQuery)
	return u.String()
}

// isSensitiveKey 参数名含敏感片段，或以 key 结尾（如 api_key，不含 keyword）
func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	if strings.HasSuffix(key, "key") {
		return true
	}
	for _, part := range sensitiveKeyParts {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

// TruncateUTF8 按字节数截断，保证不切断多字节 UTF-8 字符。
func TruncateUTF8(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	end := maxLen
	// s[end] 是截断点处(被切掉的第一个)字节；若是 continuation byte，说明落在字符中间，回退到起始边界
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end]
}
