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

// Package bkerrs 提供创建 / 包装错误的方法，目的是返回特定类型的错误（bkerrs.Error）
// 包含更具体的错误信息，便于 ErrHandler 处理成特定格式的响应内容（新版蓝鲸 HTTP API 协议）
package bkerrs

import (
	"fmt"
	"io"
	"runtime"

	pkgerrors "github.com/pkg/errors"
)

const (
	// SystemName 当前服务系统名称
	SystemName = "bkms"
	// ModuleName 当前服务模块名称
	ModuleName = "bkms-server"
)

// Error bkms 错误类型
type Error struct {
	// cause 是原始错误
	cause error
	// code 是自定义错误码
	code ErrCode
	// msg 是错误描述
	msg string
	// details 包含错误详情，主要是面向开发者的错误提示
	details []Detail
	// stack 是调用栈，包含原始错误出现的位置
	stack *stack
}

// New 新建错误
//
// 使用示例:
//
//	err := bkerrs.New(bkerrs.ErrCodeInvalidArgument, "用户名不能为空")
//
// 返回: [INVALID_ARGUMENT] 用户名不能为空
// 注：返回值说明中方括号内容为 Code 字面值，在 err.String() 时不会 Format 进去
func New(code ErrCode, msg string) *Error {
	return newErr(nil, code, msg)
}

// Errorf 新建错误（支持字符串 format）
//
// 使用示例:
//
//	userID := "admin"
//	err := bkerrs.Errorf(bkerrs.ErrCodeNotFound, "用户 %s 不存在", userID)
//
// 返回: [NOT_FOUND] 用户 admin 不存在
func Errorf(code ErrCode, format string, args ...any) *Error {
	return newErr(nil, code, fmt.Sprintf(format, args...))
}

// Wrap 错误包装
// 需注意：bkerrs.Wrap() 会覆盖原 bkerrs.Error 的错误码 & 错误详情，若希望避免该行为，则应使用 pkg/errors.Wrap()
//
// 使用示例:
//
//	dbErr := db.Create(ctx, sql)
//	if dbErr != nil {
//	    return bkerrs.Wrap(dbErr, bkerrs.ErrCodeAlreadyExists, "创建工作空间失败")
//	}
//
// 返回: [ALREADY_EXISTS] 创建工作空间失败: original db error message
func Wrap(cause error, code ErrCode, msg string) *Error {
	return newErr(cause, code, msg)
}

// Wrapf 错误包装（支持字符串 format）
// 需注意：bkerrs.Wrapf() 会覆盖原 bkerrs.Error 的错误码 & 错误详情，若希望避免该行为，则应使用 pkg/errors.Wrapf()
//
// 使用示例:
//
//	workspaceID := "my-workspace"
//	dbErr := db.Delete(ctx, workspaceID)
//	if dbErr != nil {
//	    return bkerrs.Wrapf(dbErr, bkerrs.ErrCodeInternalServerError, "删除工作空间 %s 失败", workspaceID)
//	}
//
// 返回: [INTERNAL_SERVER_ERROR] 删除工作空间 my-workspace 失败: original db error message
func Wrapf(cause error, code ErrCode, format string, args ...any) *Error {
	return newErr(cause, code, fmt.Sprintf(format, args...))
}

// newErr 新建错误
func newErr(cause error, code ErrCode, msg string) *Error {
	return &Error{
		cause:   cause,
		code:    code,
		msg:     msg,
		details: nil,
		stack:   callers(),
	}
}

// Error 返回错误信息
func (e *Error) Error() string {
	msg := e.msg
	if e.cause != nil {
		msg = msg + ": " + e.cause.Error()
	}
	return msg
}

// String 返回错误信息（包含详情）
func (e *Error) String() string {
	msg := e.Error()
	// 拼接错误详情
	for idx, dt := range e.details {
		msg = fmt.Sprintf("%s\n - details[%d]: %s", msg, idx+1, dt.String())
	}
	return msg
}

// Cause 返回错误原因
func (e *Error) Cause() error {
	return e.cause
}

// Unwrap 返回被包装的错误
func (e *Error) Unwrap() error {
	return e.Cause()
}

// Code 返回错误码
func (e *Error) Code() ErrCode {
	return e.code
}

// Details 返回错误详情
func (e *Error) Details() Details {
	return e.details
}

// SetDetails 设置错误详情（全量替换）
func (e *Error) SetDetails(details ...Detail) *Error {
	e.details = details
	return e
}

// StackTrace 返回调用栈信息（兼容 pkg/errors 接口）
func (e *Error) StackTrace() pkgerrors.StackTrace {
	if e.stack == nil {
		return nil
	}
	return e.stack.StackTrace()
}

// Format 实现 fmt.Formatter 接口，支持格式化输出
// 支持的格式：
//
//		%s/%v  - 输出错误信息
//		%+v    - 输出错误信息 + 完整堆栈信息（递归打印整个错误链）
//	    %q     - 返回格式化字符串（带引号）
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		// %+v 需额外输出详细信息（包含堆栈）
		if s.Flag('+') {
			// 1. 递归打印 cause 输出完整的错误链
			if e.cause != nil {
				_, _ = fmt.Fprintf(s, "%+v\n", e.cause)
			}

			// 2. 打印当前层的错误消息
			_, _ = io.WriteString(s, e.String())

			// 3. 打印当前层的堆栈信息
			if e.stack != nil {
				e.stack.Format(s, verb)
			}
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}

var (
	_ error         = &Error{}
	_ fmt.Formatter = &Error{}
)

// Detail bkms 错误详情
// 相较于 Error.Message 这类给用户看的友好错误，
// 详情中会包括更多信息，诸如触发错误的系统、模块、额外信息等
type Detail struct {
	code   ErrDetailCode
	msg    string
	system string
	module string
	extras map[string]string
}

// NewDetail 新建错误详情
func NewDetail(code ErrDetailCode, msg string, opts ...Option) Detail {
	dt := Detail{code: code, msg: msg}
	for _, opt := range opts {
		opt(&dt)
	}
	return dt
}

// String 返回错误详情字符串
// 格式: [code] [system:xxx] [module:xxx] message [extra_key:extra_value]...
func (d *Detail) String() string {
	// 构建前缀: [code] [system] [module]
	prefix := fmt.Sprintf("[%s]", d.code)
	if d.system != "" {
		prefix = prefix + fmt.Sprintf(" [system:%s]", d.system)
	}
	if d.module != "" {
		prefix = prefix + fmt.Sprintf(" [module:%s]", d.module)
	}
	msg := prefix + " " + d.msg

	// 追加 extras
	for k, v := range d.extras {
		msg = msg + fmt.Sprintf(" [%s:%s]", k, v)
	}
	return msg
}

// Details 错误详情列表
type Details []Detail

// AsMaps 转换为 map[string]any 切片
func (ds *Details) AsMaps() []map[string]any {
	var res []map[string]any
	for _, dt := range *ds {
		res = append(res, map[string]any{
			"code":    dt.code,
			"message": dt.msg,
			"system":  dt.system,
			"module":  dt.module,
			"extras":  dt.extras,
		})
	}
	return res
}

// Option 错误详情选项
type Option func(*Detail)

// WithSystem 添加错误详情 - 系统信息
func WithSystem(system string) Option {
	return func(dt *Detail) {
		dt.system = system
	}
}

// WithModule 添加错误详情 - 模块信息
func WithModule(module string) Option {
	return func(dt *Detail) {
		dt.module = module
	}
}

// WithExtras 添加错误详情 - 扩展字段
func WithExtras(extras map[string]string) Option {
	return func(dt *Detail) {
		dt.extras = extras
	}
}

// Stack Trace Implementation (inspired by github.com/pkg/errors)

// stack 是程序计数器的堆栈，可用于还原出完成的函数调用链路
type stack []uintptr

// StackTrace 将 stack 转换为 pkgerrors.StackTrace，兼容格式化处理逻辑
func (s *stack) StackTrace() pkgerrors.StackTrace {
	if s == nil {
		return nil
	}

	f := make([]pkgerrors.Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = pkgerrors.Frame((*s)[i])
	}
	return f
}

// Format 实现 fmt.Formatter 接口，支持自定义格式化输出
func (s *stack) Format(st fmt.State, verb rune) {
	if s == nil {
		return
	}

	switch verb {
	case 'v':
		// 当使用 %+v 时，输出详细的堆栈信息
		if st.Flag('+') {
			// 遍历每个程序计数器
			for _, pc := range *s {
				// 将 uintptr 转换为 Frame
				f := pkgerrors.Frame(pc)
				// 使用 Frame 的 %+v 格式化输出
				// 这会输出函数名、文件路径和行号
				_, _ = fmt.Fprintf(st, "\n%+v", f)
			}
		}
		// 如果是 %v（没有 '+' 标志），则不输出堆栈信息
	}
	// 其他格式化动词（如 %s, %q）不处理，保持默认行为
}

func callers() *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(4, pcs[:])
	var st stack = pcs[0:n]
	return &st
}
