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

package types

import (
	"regexp"
	"unicode/utf8"

	pkgerrors "github.com/pkg/errors"
)

const (
	// EnvVarKeyMaxLength matches the CRUD validation limit used by current APIs.
	EnvVarKeyMaxLength = 256
	// EnvVarValueMaxLength matches the CRUD validation limit used by current APIs.
	EnvVarValueMaxLength = 8192
)

// envVarKeyPattern 匹配合法的环境变量 key 名称（以字母或下划线开头，后续可跟字母、数字或下划线）
// TODO: 与 core/app/serializer 和 appmodelcore/appmodel 中的重复定义，后续统一收敛到此位置
var envVarKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// IsValidEnvVarKey 是底层格式判断函数，仅回答“是否匹配 key 命名规则”。
// 若上层需要复用完整共享约束和统一错误文案，应优先使用 ValidateEnvVarKey。
func IsValidEnvVarKey(key string) bool {
	return envVarKeyPattern.MatchString(key)
}

// ValidateEnvVarKey 是环境变量 key 的共享校验入口，统一封装格式、长度和错误文案。
func ValidateEnvVarKey(key string) error {
	if !IsValidEnvVarKeyLength(key) {
		return pkgerrors.Errorf("env var key %q must be at most %d characters", key, EnvVarKeyMaxLength)
	}
	if !IsValidEnvVarKey(key) {
		return pkgerrors.Errorf(
			"invalid env var key %q: must start with a letter or underscore and contain only letters, numbers, and underscores",
			key,
		)
	}
	return nil
}

// IsValidEnvVarKeyLength 是底层长度判断函数，仅回答“是否超出共享长度上限”。
func IsValidEnvVarKeyLength(key string) bool {
	return utf8.RuneCountInString(key) <= EnvVarKeyMaxLength
}

// IsValidEnvVarValueLength 是底层长度判断函数，仅回答“是否超出共享长度上限”。
func IsValidEnvVarValueLength(value string) bool {
	return utf8.RuneCountInString(value) <= EnvVarValueMaxLength
}

// ValidateEnvVarValue 是环境变量 value 的共享校验入口，统一封装长度约束和错误文案。
func ValidateEnvVarValue(value string) error {
	if !IsValidEnvVarValueLength(value) {
		return pkgerrors.Errorf("env var value must be at most %d characters", EnvVarValueMaxLength)
	}
	return nil
}
