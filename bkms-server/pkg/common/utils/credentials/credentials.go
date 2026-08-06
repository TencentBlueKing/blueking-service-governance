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

// Package credentials 提供用户名/密码类凭据的通用校验工具
package credentials

import (
	"strings"

	"github.com/pkg/errors"
)

// ErrInvalidUserPass indicates username and password are not provided as a valid pair.
var ErrInvalidUserPass = errors.New("username and password must both be empty or both have values")

// ValidateOptionalUserPass validates optional username/password credentials.
func ValidateOptionalUserPass(username, password string) error {
	if username == "" && password == "" {
		return nil
	}
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return ErrInvalidUserPass
	}
	return nil
}

// HasUserPass reports whether username/password contain a usable credential pair.
func HasUserPass(username, password string) bool {
	return strings.TrimSpace(username) != "" && strings.TrimSpace(password) != ""
}
