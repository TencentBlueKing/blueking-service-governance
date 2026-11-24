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

package bkerrs

import (
	"errors"
	"fmt"
)

// Rule 把领域错误映射为 API 错误的规则，err 不匹配时返回 nil
type Rule func(err error) *Error

// Map 按顺序匹配 rules，返回第一条命中规则生成的 API 错误；都不命中时原样返回 err，
// 由错误出口按存储类别（不存在 / 已存在）转换或视为内部错误
//
// 使用示例:
//
//	bkerrs.AbortWithErr(c, bkerrs.Map(err,
//	    bkerrs.On(model.ErrUnknownSwitch, bkerrs.ErrCodeInvalidArgument, "unknown cfg switch key"),
//	    bkerrs.OnAs(func(err error, e *bkmsenv.EnvHasAppsError) *bkerrs.Error {
//	        return bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidRequest, "environment has %d deployed apps, cannot delete",
//
// e.AppCount)
//
//	    }),
//	))
func Map(err error, rules ...Rule) error {
	if err == nil {
		return nil
	}
	for _, rule := range rules {
		if apiErr := rule(err); apiErr != nil {
			return apiErr
		}
	}
	return err
}

// On 错误链上存在 target 时，以 code 与 format 包装 err；format 须为字符串字面量，`make i18n` 才能提取翻译
func On(target error, code ErrCode, format string, args ...any) Rule {
	msg := fmt.Sprintf(format, args...)
	return func(err error) *Error {
		if !errors.Is(err, target) {
			return nil
		}
		return newErr(err, code, msg, format, args)
	}
}

// Otherwise 匹配任意错误，放在规则末尾作为兜底；不需要兜底时省略，交给错误出口按存储类别转换或视为内部错误
func Otherwise(code ErrCode, format string, args ...any) Rule {
	msg := fmt.Sprintf(format, args...)
	return func(err error) *Error {
		return newErr(err, code, msg, format, args)
	}
}

// OnFunc 错误链上存在 target 时，由 build 生成 API 错误，用于需要附带错误详情等场景
func OnFunc(target error, build func(err error) *Error) Rule {
	return func(err error) *Error {
		if !errors.Is(err, target) {
			return nil
		}
		return build(err)
	}
}

// OnAs 错误链上存在 T 类型的错误时，由 build 生成 API 错误，用于从携带参数的领域错误中取出参数
func OnAs[T error](build func(err error, target T) *Error) Rule {
	return func(err error) *Error {
		var target T
		if !errors.As(err, &target) {
			return nil
		}
		return build(err, target)
	}
}
