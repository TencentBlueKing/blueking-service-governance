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
	"maps"
	"slices"
	"strings"
	"sync"
)

var (
	definitionsMu sync.RWMutex
	definitions   = map[ErrDetailCode]*Error{}
)

// Define 定义一类已分类的业务错误，应在错误产生的包内以包级变量声明
//
// 错误详情码 detailCode 全局唯一，同时作为 i18n key，重复定义会 panic；返回的定义可直接返回，也可通过 With / WithCause 派生实例，
// 派生出的实例与定义满足 errors.Is
//
// 使用示例:
//
//	var ErrEnvHasDeployedAppsCannotUpdateCluster = bkerrs.Define(
//		bkerrs.ErrCodeInvalidRequest, "CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER", "environment has deployed apps")
//
//	return ErrEnvHasDeployedAppsCannotUpdateCluster.With("appCount", strconv.Itoa(appCount))
func Define(code ErrCode, detailCode ErrDetailCode, msg string) *Error {
	if detailCode == "" {
		panic("bkerrs: Define requires a non-empty detail code")
	}

	definitionsMu.Lock()
	defer definitionsMu.Unlock()
	if _, exists := definitions[detailCode]; exists {
		panic(fmt.Sprintf("bkerrs: detail code %s is already defined", detailCode))
	}
	e := &Error{code: code, detailCode: detailCode, msg: msg, defined: true}
	definitions[detailCode] = e
	return e
}

// Definitions 返回所有通过 Define 定义的错误（按 detailCode 排序）
func Definitions() []*Error {
	definitionsMu.RLock()
	defer definitionsMu.RUnlock()
	return slices.SortedFunc(maps.Values(definitions), func(a, b *Error) int {
		return strings.Compare(string(a.detailCode), string(b.detailCode))
	})
}

// FromError 返回错误链上第一个 *Error，不存在时返回 nil
//
// 外层通过 WithCause 改判的错误排在链的前面，因此会优先于内层的错误被取到
func FromError(err error) *Error {
	var bkErr *Error
	if errors.As(err, &bkErr) {
		return bkErr
	}
	return nil
}

// DetailCode 返回错误详情码，非 Define 定义的错误返回空字符串
func (e *Error) DetailCode() ErrDetailCode {
	return e.detailCode
}

// Params 返回错误原因参数的副本
func (e *Error) Params() map[string]string {
	return maps.Clone(e.params)
}

// With 返回附加了参数的副本，参数用于生成错误详情与用户文案
func (e *Error) With(key, value string) *Error {
	ne := e.clone()
	ne.params = maps.Clone(e.params)
	if ne.params == nil {
		ne.params = make(map[string]string, 1)
	}
	ne.params[key] = value
	return ne
}

// WithCause 返回挂上原因的副本，用于在调用边界上将下层错误显式改判为当前错误
//
// 使用示例:
//
//	if errors.Is(err, envmodel.ErrEnvNotFound) {
//		return ErrAppEnvInconsistent.With("envID", app.EnvID).WithCause(err)
//	}
func (e *Error) WithCause(cause error) *Error {
	ne := e.clone()
	ne.cause = cause
	return ne
}

// Is 判断两个错误是否为同一类已定义错误（code 与 detailCode 均相同）
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && e.detailCode != "" && e.code == t.code && e.detailCode == t.detailCode
}

// clone 复制错误并记录调用方的调用栈，需由导出方法直接调用以保证栈深度正确
func (e *Error) clone() *Error {
	return &Error{
		cause:      e.cause,
		code:       e.code,
		detailCode: e.detailCode,
		params:     e.params,
		msg:        e.msg,
		details:    e.details,
		stack:      callers(),
	}
}

// formatParams 按 key 排序输出 k=v 列表，保证错误信息稳定
func formatParams(params map[string]string) string {
	keys := slices.Sorted(maps.Keys(params))
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+params[k])
	}
	return strings.Join(pairs, ", ")
}
