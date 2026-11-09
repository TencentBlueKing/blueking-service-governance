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
)

const (
	// Masked 敏感值替换后的内容
	Masked = "***"
	// maxQueryLen 脱敏后 query 最大长度
	maxQueryLen = 1024
)

// sensitiveKeyParts 参数名含以下片段时视为敏感
var sensitiveKeyParts = []string{"token", "ticket", "secret", "password", "passwd", "credential"}

// Query 将敏感参数的值替换为 Masked，超长时截断
func Query(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Masked
	}
	for key, vals := range values {
		if isSensitiveKey(key) {
			for i := range vals {
				vals[i] = Masked
			}
		}
	}
	encoded := values.Encode()
	if len(encoded) > maxQueryLen {
		return encoded[:maxQueryLen]
	}
	return encoded
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
