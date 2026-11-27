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

	"github.com/gin-gonic/gin"
)

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

// AbortInvalidArg 把 err 包装为参数错误并中止请求，msg 为面向用户的原文。
// 需要按领域错误区分错误码时使用 AbortWithErr 加规则
func AbortInvalidArg(c *gin.Context, err error, msg string) {
	AbortWithErr(c, Wrap(err, ErrCodeInvalidArgument, msg))
}

// AbortInternal 把 err 作为内部错误中止请求，响应统一返回通用文案。
// 没有 msg 参数：内部错误没有面向用户的文案，开发者上下文与 AbortWithErr 一样用 errors.Wrap 附加：
//
//	bkerrs.AbortInternal(c, errors.Wrap(err, "list app config file versions"))
//
// 错误链上 store 声明的"不存在 / 已存在"不再自动转换为 404 / 409；需要转换时用 AbortWithErr 交给错误出口分类。
// 全局规则（如 BSCP 未启用）仍然生效
func AbortInternal(c *gin.Context, err error) {
	AbortWithErr(c, Wrap(err, ErrCodeInternalServerError, "internal error"))
}

// AbortNotFound 把 err 包装为资源不存在错误并中止请求，msg 为面向用户的原文
func AbortNotFound(c *gin.Context, err error, msg string) {
	AbortWithErr(c, Wrap(err, ErrCodeNotFound, msg))
}

// AbortWithErr 记录错误并中止当前 Gin handler 链，由错误出口按错误链定码（未分类的错误为 500）。
// 传入 rules 时先按 Match 匹配，把领域错误转换为 API 错误；都不命中时原样记录 err：
//
//	bkerrs.AbortWithErr(c, errors.Wrap(err, "update platform workspace cfg switch"),
//	    bkerrs.WhenIs(model.ErrUnknownSwitch, bkerrs.ErrCodeInvalidArgument, i18n.Msg("unknown cfg switch key")),
//	)
func AbortWithErr(c *gin.Context, err error, rules ...Rule) {
	_ = c.Error(Match(err, rules...))
	c.Abort()
}
