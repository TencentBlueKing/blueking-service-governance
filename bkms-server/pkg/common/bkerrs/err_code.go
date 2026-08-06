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

import "net/http"

// ErrCode 错误码
type ErrCode string

// 4xx 错误
const (
	// 400 错误

	// ErrCodeInvalidArgument 参数不符合格式
	ErrCodeInvalidArgument ErrCode = "INVALID_ARGUMENT"
	// ErrCodeInvalidRequest 参数符合格式，但不符合业务规则
	ErrCodeInvalidRequest ErrCode = "INVALID_REQUEST"

	// 401 错误

	// ErrCodeUnauthenticated 当前访问用户未认证
	ErrCodeUnauthenticated ErrCode = "UNAUTHENTICATED"

	// 403 错误

	// ErrCodeIAMNoPermission 权限中心没有相关权限（会返回特定的响应体）
	ErrCodeIAMNoPermission ErrCode = "IAM_NO_PERMISSION"
	// ErrCodeNoPermission 没有相关权限（非权限中心）
	ErrCodeNoPermission ErrCode = "NO_PERMISSION"

	// 404 错误

	// ErrCodeNotFound 资源不存在
	ErrCodeNotFound ErrCode = "NOT_FOUND"

	// 409 错误

	// ErrCodeAlreadyExists 资源已存在
	ErrCodeAlreadyExists ErrCode = "ALREADY_EXISTS"
	// ErrCodeAborted 并发冲突，如读取 / 修改 / 写入冲突
	ErrCodeAborted ErrCode = "ABORTED"

	// 429 错误

	// ErrCodeRateLimitExceeded 超过频率限制
	ErrCodeRateLimitExceeded ErrCode = "RATE_LIMIT_EXCEEDED"
	// ErrCodeResourceExhausted 资源配额不足
	ErrCodeResourceExhausted ErrCode = "RESOURCE_EXHAUSTED"
)

// 5xx 错误
const (
	// 500 错误

	// ErrCodeInternalError 内部错误（用于非显式指定 ErrCode 的默认情况，不推荐直接使用）
	ErrCodeInternalError ErrCode = "INTERNAL_ERROR"

	// ErrCodeInternalServerError 系统内部错误
	ErrCodeInternalServerError ErrCode = "INTERNAL_SERVER_ERROR"

	// 501 错误

	// ErrCodeNotImplemented 未实现
	ErrCodeNotImplemented ErrCode = "NOT_IMPLEMENTED"

	// NOTE：502 / 503 / 504 错误不由服务本身处理，可能在网关 / 接入层就返回掉了
)

// AsHTTPStatusCode 将错误码转换为 HTTP 状态码
func (c ErrCode) AsHTTPStatusCode() int {
	switch c {
	case ErrCodeInvalidArgument, ErrCodeInvalidRequest:
		return http.StatusBadRequest
	case ErrCodeUnauthenticated:
		return http.StatusUnauthorized
	case ErrCodeIAMNoPermission, ErrCodeNoPermission:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeAlreadyExists, ErrCodeAborted:
		return http.StatusConflict
	case ErrCodeRateLimitExceeded, ErrCodeResourceExhausted:
		return http.StatusTooManyRequests
	case ErrCodeInternalError, ErrCodeInternalServerError:
		return http.StatusInternalServerError
	case ErrCodeNotImplemented:
		return http.StatusNotImplemented
	default:
		// 都匹配不上时，返回 500 错误
		return http.StatusInternalServerError
	}
}
