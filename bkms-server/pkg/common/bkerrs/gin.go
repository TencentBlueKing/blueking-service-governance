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
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

type unauthenticatedError interface {
	error
	IsUnauthenticated() bool
}

// APIDetailError 声明该错误返回给前端时使用的错误码、详情码和模块。
// bkerrs.Wrap 会覆盖错误码和详情，WriteError 在输出时按该声明还原。
type APIDetailError interface {
	error
	APIErrorDetail() (code ErrCode, detail ErrDetailCode, module string)
}

// GinErrorOutput is the standard bkms Gin error response body.
type GinErrorOutput struct {
	Error GinError `json:"error"`
}

// GinError is the standard bkms Gin error object.
type GinError struct {
	Code    ErrCode          `json:"code"`
	Message string           `json:"message"`
	System  string           `json:"system"`
	Module  string           `json:"module"`
	Details []map[string]any `json:"details"`
}

// ErrorHandler renders errors collected on Gin context using bkms API error format.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		WriteError(c, c.Errors.Last().Err)
	}
}

// AbortWithErr records err and stops the current Gin handler chain.
func AbortWithErr(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

// WriteError writes err using the same response shape as the trpc restful error handler.
func WriteError(c *gin.Context, err error) {
	span := trace.SpanFromContext(c.Request.Context())
	if span != nil && span.SpanContext().HasTraceID() {
		c.Header("X-Trace-Id", span.SpanContext().TraceID().String())
	}
	if err == nil {
		return
	}

	var unauthErr unauthenticatedError
	if errors.As(err, &unauthErr) && unauthErr.IsUnauthenticated() {
		err = Wrap(err, ErrCodeUnauthenticated, "get auth user from request failed")
	}
	// 上层 Wrap 会覆盖错误码与 details，需在输出时还原 BSCP 未启用的错误明细
	if errors.Is(err, ErrBSCPDisabled) {
		err = WrapBSCPDisabled(err)
	}
	// 领域错误声明了应返回的详情码时，同样在输出时还原，避免被包成内部错误
	var declared APIDetailError
	if errors.As(err, &declared) {
		err = restoreAPIDetail(err, declared)
	}

	var bkErr *Error
	if errors.As(err, &bkErr) {
		details := bkErr.Details()
		c.JSON(bkErr.Code().AsHTTPStatusCode(), GinErrorOutput{
			Error: GinError{
				Code:    bkErr.Code(),
				Message: err.Error(),
				System:  SystemName,
				Module:  ModuleName,
				Details: details.AsMaps(),
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, GinErrorOutput{
		Error: GinError{
			Code:    ErrCodeInternalError,
			Message: err.Error(),
			System:  SystemName,
			Module:  ModuleName,
		},
	})
}

// restoreAPIDetail 按错误链中的声明还原错误码和详情。
// 已经是目标错误码且带有该详情时保持原样，避免重复包装。
func restoreAPIDetail(err error, declared APIDetailError) error {
	code, detailCode, module := declared.APIErrorDetail()
	var bkErr *Error
	if errors.As(err, &bkErr) && bkErr.Code() == code && hasDetail(bkErr, detailCode) {
		return err
	}
	var promoted *Error
	if errors.As(err, &bkErr) {
		promoted = newErr(bkErr.cause, code, bkErr.msg)
	} else {
		promoted = Wrap(err, code, declared.Error())
	}
	opts := []Option{WithSystem(SystemName)}
	if module != "" {
		opts = append(opts, WithModule(module))
	}
	return promoted.SetDetails(NewDetail(detailCode, promoted.Error(), opts...))
}

func hasDetail(err *Error, code ErrDetailCode) bool {
	for _, detail := range err.details {
		if detail.code == code {
			return true
		}
	}
	return false
}
