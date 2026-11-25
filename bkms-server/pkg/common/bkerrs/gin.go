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
//
// globalRules 为在任何接口中含义都相同的规则（如 BSCP 未启用），handler 无需逐个指定。
// 它们优先于 handler 自己的映射：即使 handler 已把错误包装为其他 *Error，错误链上命中时仍以全局规则为准；
// 仅当 handler 给出的错误码与全局规则相同时保留 handler 的错误
func ErrorHandler(globalRules ...Rule) gin.HandlerFunc {
	global := Rules(globalRules...)
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		WriteError(c, matchGlobal(c.Errors.Last().Err, global))
	}
}

func matchGlobal(err error, global Rule) error {
	apiErr := global(err)
	if apiErr == nil {
		return err
	}
	var handlerErr *Error
	if errors.As(err, &handlerErr) && handlerErr.Code() == apiErr.Code() {
		return err
	}
	return apiErr
}

// AbortWithErr 记录错误并中止当前 Gin handler 链，由错误出口按错误链定码（未分类的错误为 500）。
// 传入 rules 时先按 Match 匹配，把领域错误转换为 API 错误；都不命中时原样记录 err：
//
//	bkerrs.AbortWithErr(c, errors.Wrap(err, "update platform workspace cfg switch"),
//	    bkerrs.On(model.ErrUnknownSwitch, bkerrs.ErrCodeInvalidArgument, i18n.Msg("unknown cfg switch key")),
//	)
func AbortWithErr(c *gin.Context, err error, rules ...Rule) {
	_ = c.Error(Match(err, rules...))
	c.Abort()
}

// AbortInvalidArg 把 err 包装为参数错误并中止请求，msg 为面向用户的原文。
// 需要按领域错误区分错误码时使用 AbortWithErr 加规则
func AbortInvalidArg(c *gin.Context, err error, msg string) {
	AbortWithErr(c, Wrap(err, ErrCodeInvalidArgument, msg))
}

// AbortInternal 把 err 包装为内部错误并中止请求。
// 内部错误统一返回通用文案，msg 只作为错误链的上下文出现在日志与错误详情中，不需要用 i18n.Msg 标记。
// 错误链上 store 声明的"不存在 / 已存在"不再自动转换为 404 / 409；需要转换时用 AbortWithErr 交给错误出口分类。
// 全局规则（如 BSCP 无权限）仍然生效
func AbortInternal(c *gin.Context, err error, msg string) {
	AbortWithErr(c, Wrap(err, ErrCodeInternalServerError, msg))
}

// AbortNotFound 把 err 包装为资源不存在错误并中止请求，msg 为面向用户的原文
func AbortNotFound(c *gin.Context, err error, msg string) {
	AbortWithErr(c, Wrap(err, ErrCodeNotFound, msg))
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
		message, translated = i18n.Translate(ctx, internalErrorMessage)
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
