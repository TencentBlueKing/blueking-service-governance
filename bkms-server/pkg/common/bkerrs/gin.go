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
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// notFoundError、alreadyExistsError 供不依赖 bkerrs 的通用组件（如 store）声明错误类别，出口据此定码与翻译
type notFoundError interface {
	error
	IsNotFound() bool
}

type alreadyExistsError interface {
	error
	IsAlreadyExists() bool
}

// declaredCategories 通用组件可声明的错误类别，按顺序匹配；新增类别时声明标记接口并在此追加一项
var declaredCategories = []struct {
	declares func(error) bool
	def      *Error
}{
	{
		declaredBy(notFoundError.IsNotFound),
		Define(ErrCodeNotFound, "RESOURCE_NOT_FOUND", "resource not found"),
	},
	{
		declaredBy(alreadyExistsError.IsAlreadyExists),
		Define(ErrCodeAlreadyExists, "RESOURCE_ALREADY_EXISTS", "resource already exists"),
	},
}

// declaredBy 返回判断函数：错误链上存在 T 类型的错误，且 is 对其返回 true
func declaredBy[T error](is func(T) bool) func(error) bool {
	return func(err error) bool {
		var target T
		return errors.As(err, &target) && is(target)
	}
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

	// 上层 Wrap 会覆盖错误码与 details，需在取码前还原 BSCP 未启用的错误
	if errors.Is(err, ErrBSCPDisabled) {
		err = WrapBSCPDisabled(err)
	}

	// 错误码取错误链上第一个 *Error，其次按通用组件声明的错误类别转换，均没有时视为未分类的内部错误
	code := ErrCodeInternalError
	bkErr := classify(err)
	if bkErr != nil {
		code = bkErr.Code()
	}
	logError(c, code, err)

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

// classify 返回出口用于定码与翻译的错误：优先取错误链上第一个 *Error，其次按 declaredCategories 声明的类别转换，
// 均不满足时返回 nil
func classify(err error) *Error {
	if bkErr := FromError(err); bkErr != nil {
		return bkErr
	}

	for _, c := range declaredCategories {
		if c.declares(err) {
			return c.def.WithCause(err)
		}
	}
	return nil
}

// userMessage 生成给用户看的文案与错误详情
func userMessage(ctx context.Context, bkErr *Error, code ErrCode, err error) (string, Details) {
	// 1. 取出错误自带的错误详情（如 Define 自动生成的详情），供后续组装开发者可见信息使用
	var details Details
	if bkErr != nil {
		details = bkErr.Details()
	}

	var message string
	var localized bool
	// 2. 生成本地化文案：优先按 detailCode 精确翻译，失败则降级到 code 通用文案；5xx 内部错误统一用 code 通用文案避免暴露内部信息
	switch {
	case bkErr != nil && bkErr.detailCode != "":
		message, localized = i18n.Localize(ctx, string(bkErr.detailCode), bkErr.params)
		if !localized {
			message, localized = i18n.Localize(ctx, string(code), nil)
		}
	case code.AsHTTPStatusCode() >= http.StatusInternalServerError:
		message, localized = i18n.Localize(ctx, string(code), nil)
	}
	// 3. 本地化未成功时直接沿用原始错误信息兜底，详情保持原样返回
	if !localized {
		return err.Error(), details
	}

	// 4. 本地化成功后，将已被翻译替换的原始错误保留进错误详情以便排查：
	//    有 detailCode 且无自带详情时生成一条以 detailCode 为码的详情（附带业务参数），否则在已有详情后追加一条以 code 为码的详情
	if bkErr != nil && bkErr.detailCode != "" && len(bkErr.details) == 0 {
		return message, Details{
			NewDetail(bkErr.detailCode, err.Error(), WithSystem(SystemName), WithExtras(bkErr.Params())),
		}
	}
	return message, append(slices.Clone(details), NewDetail(ErrDetailCode(code), err.Error(), WithSystem(SystemName)))
}

// logError 记录需要排查的错误：5xx 可能是服务端故障或遗漏了分类，404 可能是内部查询未改判
func logError(c *gin.Context, code ErrCode, err error) {
	route := c.FullPath()
	if route == "" {
		route = c.Request.URL.Path
	}

	ctx := c.Request.Context()
	switch status := code.AsHTTPStatusCode(); {
	case status >= http.StatusInternalServerError:
		log.Errorf(ctx, "%s %s responded %s: %+v", c.Request.Method, route, code, err)
	case status == http.StatusNotFound:
		log.Warnf(ctx, "%s %s responded %s: %v", c.Request.Method, route, code, err)
	}
}
