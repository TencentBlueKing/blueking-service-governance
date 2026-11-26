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
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// internalErrorMessage 内部错误统一返回给用户的文案，原始错误链放入错误详情
var internalErrorMessage = i18n.Msg("internal server error, please try again later or contact the administrator")

// codeMessages 错误码的默认文案，最外层 *Error 的原文没有译文时使用；
// 内部错误码固定使用 internalErrorMessage，不在此列
var codeMessages = map[ErrCode]string{
	ErrCodeInvalidArgument:   i18n.Msg("invalid request parameters"),
	ErrCodeInvalidRequest:    i18n.Msg("request does not meet business rules"),
	ErrCodeUnauthenticated:   i18n.Msg("user is not authenticated, please log in again"),
	ErrCodeIAMNoPermission:   i18n.Msg("no permission to perform this operation"),
	ErrCodeNoPermission:      i18n.Msg("no permission to perform this operation"),
	ErrCodeNotFound:          i18n.Msg("resource not found"),
	ErrCodeAlreadyExists:     i18n.Msg("resource already exists"),
	ErrCodeAborted:           i18n.Msg("resource was modified concurrently, please refresh and try again"),
	ErrCodeRateLimitExceeded: i18n.Msg("too many requests, please try again later"),
	ErrCodeResourceExhausted: i18n.Msg("resource quota exceeded"),
	ErrCodeNotImplemented:    i18n.Msg("not implemented yet"),
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
		err = Wrap(err, ErrCodeUnauthenticated, i18n.Msg("get auth user from request failed"))
	}

	code := ErrCodeInternalError
	bkErr, categorized := classify(err)
	if bkErr != nil {
		code = bkErr.Code()
	}
	logError(c, code, categorized, err)

	message, details := userMessage(c.Request.Context(), bkErr, code, err)
	c.JSON(code.AsHTTPStatusCode(), GinErrorOutput{
		Error: GinError{
			Code:    code,
			Message: message,
			System:  SystemName,
			Module:  ModuleName,
			Details: details.AsMaps(),
		},
	})
}

// userMessage 生成给用户看的文案与错误详情
func userMessage(ctx context.Context, bkErr *Error, code ErrCode, err error) (string, Details) {
	var details Details
	if bkErr != nil {
		details = bkErr.Details()
	}

	// 1. 内部错误统一使用通用文案，其他错误翻译最外层 *Error 的原文；内层错误面向开发者，不参与翻译
	var message string
	var translated bool
	switch {
	case code.isInternal():
		// 通用文案没有译文时使用英文原文，不能回退到错误链：错误链可能包含内部实现细节
		message, _ = i18n.Translate(ctx, internalErrorMessage)
		translated = true
	case bkErr != nil:
		message, translated = bkErr.translate(ctx)
	}
	// 2. 原文没有译文时使用错误码的默认文案；仍没有时沿用完整的错误信息，详情保持原样返回
	if !translated {
		message, translated = codeMessage(ctx, code)
	}
	if !translated {
		return err.Error(), details
	}
	// 3. 翻译后的文案不再包含错误链，将完整错误信息追加到详情末尾以便排查；已有详情保持在前，不影响前端按 details[0] 取值
	return message, append(slices.Clone(details), NewDetail(ErrDetailCode(code), err.Error(), WithSystem(SystemName)))
}

// codeMessage 返回 code 的默认文案在 ctx 语言下的译文；
// 源语言下完整的错误信息本身可读且比默认文案具体，不使用默认文案
func codeMessage(ctx context.Context, code ErrCode) (string, bool) {
	msgid, ok := codeMessages[code]
	if !ok || i18n.LangFromContext(ctx) == i18n.SourceLang {
		return "", false
	}
	return i18n.Translate(ctx, msgid)
}

// translate 按 ctx 中的语言翻译 msg 的原文并用原参数格式化，原文没有译文时返回 false
func (e *Error) translate(ctx context.Context) (string, bool) {
	msg, ok := i18n.Translate(ctx, e.format)
	if !ok {
		return "", false
	}
	if len(e.args) == 0 {
		return msg, true
	}
	return fmt.Sprintf(msg, e.args...), true
}

// logError 记录需要排查的错误：内部错误可能是服务端故障或遗漏了映射；
// 按存储类别自动转换的 404 可能是内部查询未改判，handler 显式映射的 404 属于正常响应，不记录
func logError(c *gin.Context, code ErrCode, categorized bool, err error) {
	route := c.FullPath()
	if route == "" {
		route = c.Request.URL.Path
	}

	ctx := c.Request.Context()
	switch {
	case code.isInternal():
		log.Errorf(ctx, "%s %s responded %s: %+v", c.Request.Method, route, code, err)
	case categorized && code.AsHTTPStatusCode() == http.StatusNotFound:
		log.Warnf(ctx, "%s %s responded %s: %v", c.Request.Method, route, code, err)
	}
}
