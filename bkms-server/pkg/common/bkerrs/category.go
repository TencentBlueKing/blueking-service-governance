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
	"github.com/pkg/errors"
)

type unauthenticatedError interface {
	error
	IsUnauthenticated() bool
}

// notFoundError、alreadyExistsError 供不依赖 bkerrs 的通用组件（如 store）声明错误类别，出口据此定码
type notFoundError interface {
	error
	IsNotFound() bool
}

type alreadyExistsError interface {
	error
	IsAlreadyExists() bool
}

// domainError 由声明了错误类别的组件实现，提供比类别更具体的业务错误，出口以其文本作为面向用户的原文。
// 例如 store 查询环境不存在时，错误同时满足：
//   - IsNotFound() == true：出口据此返回 404
//   - DomainError() == model.ErrEnvNotFound：出口以 "environment not found" 为原文，中文翻译为"环境不存在"
//
// 未提供业务错误时使用错误码的默认文案（"resource not found"）。
// 业务错误不在 Unwrap 链上（store 只通过 Is 匹配它），errors.As 取不到，因此需要单独的方法暴露
type domainError interface {
	error
	DomainError() error
}

// classify 返回出口用于定码与翻译的错误：优先取错误链上最外层的 *Error，其次按通用组件声明的错误类别转换
// （categorized 为 true），均不满足时返回 nil
func classify(err error) (bkErr *Error, categorized bool) {
	if errors.As(err, &bkErr) {
		return bkErr, false
	}

	switch {
	case declares(err, notFoundError.IsNotFound):
		return categoryError(err, ErrCodeNotFound, codeMessages[ErrCodeNotFound]), true
	case declares(err, alreadyExistsError.IsAlreadyExists):
		return categoryError(err, ErrCodeAlreadyExists, codeMessages[ErrCodeAlreadyExists]), true
	}
	return nil, false
}

// declares 判断错误链上存在 T 类型的错误，且 is 对其返回 true
func declares[T error](err error, is func(T) bool) bool {
	var target T
	return errors.As(err, &target) && is(target)
}

// categoryError 把声明了类别的错误包装为 code；错误带有领域错误时以领域错误的文本为原文，否则使用 fallback
func categoryError(err error, code ErrCode, fallback string) *Error {
	var d domainError
	if errors.As(err, &d) && d.DomainError() != nil {
		return Wrap(err, code, d.DomainError().Error())
	}
	return Wrap(err, code, fallback)
}
