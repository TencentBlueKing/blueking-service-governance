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

// Package app 提供应用创建相关的处理逻辑
package app

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// appValidator 全局 validator 实例
var appValidator *validator.Validate

// nameRegexp 应用名称校验正则：小写字母开头，可含小写字母/数字/中划线，不以中划线结尾，1~63字符
var nameRegexp = regexp.MustCompile(`^[a-z](?:[a-z0-9-]*[a-z0-9])?$`)

func init() {
	appValidator = validator.New(validator.WithRequiredStructEnabled())

	appValidator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("yaml"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	// 注册自定义 tag 校验器
	_ = appValidator.RegisterValidation("app_name", validateAppName)
}

// Validate 校验 AppCreateSpec 的合法性
// CLI 端仅做不为空校验和格式校验（名称正则、枚举值），提供快速反馈；
// 完整的业务规则校验（如条件必填等）以后端为准。
func (s *AppCreateSpec) Validate() error {
	// 对所有字符串字段去除前后空格
	s.TrimSpace()
	if err := appValidator.Struct(s); err != nil {
		return formatValidationError(err)
	}
	return nil
}

// validateAppName 自定义 app_name tag 校验器
func validateAppName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	if len(name) < 1 || len(name) > 63 {
		return false
	}
	return nameRegexp.MatchString(name)
}

// formatValidationError 将 validator 错误转换为用户友好的错误信息，只返回第一个错误保持 CLI 输出简洁
func formatValidationError(err error) error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}
	return errors.New(buildFieldErrorMessage(validationErrors[0]))
}

// buildFieldErrorMessage 根据 FieldError 构建用户友好的错误信息
func buildFieldErrorMessage(fe validator.FieldError) string {
	namespace := fe.Namespace()
	field := namespace
	if idx := strings.Index(namespace, "."); idx != -1 {
		field = namespace[idx+1:]
	}

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("'%s' is required", field)
	case "oneof":
		return fmt.Sprintf("'%s' is invalid: must be one of [%s], got %q",
			field, strings.ReplaceAll(fe.Param(), " ", ", "), fe.Value())
	case "app_name":
		return fmt.Sprintf("'%s' is invalid: must start with a lowercase letter, "+
			"contain only lowercase letters, digits, and hyphens, not end with a hyphen, and be 1~63 characters", field)
	default:
		return fmt.Sprintf("'%s' failed validation on '%s' tag", field, fe.Tag())
	}
}
