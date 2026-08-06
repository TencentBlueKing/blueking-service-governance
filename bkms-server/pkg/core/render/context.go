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

// Package render provides Gonja-based ${{ }} variable rendering with explicit contexts.
package render

import "maps"

type contextKey string

// 这些常量是渲染数据允许的顶层 namespace。当前只开放 env/bkms 的构造入口；
// input/builtin 会在组件 output 切到 Gonja 后再开放。
const (
	ContextEnv     contextKey = "env"
	ContextBkms    contextKey = "bkms"
	ContextInput   contextKey = "input"
	ContextBuiltin contextKey = "builtin"
)

// Context 负责持有渲染数据
// 例如 Context{"env": map[string]string{"BKMS_APP_NAME": "app"}}
type Context map[string]any

// ContextOption configures Context during construction.
type ContextOption func(Context)

// NewContext builds render context from options.
func NewContext(opts ...ContextOption) Context {
	d := Context(make(map[string]any))
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// SetEnvContext registers variables under env namespace.
func SetEnvContext(vars map[string]string) ContextOption {
	return setContext(ContextEnv, vars)
}

// SetBkmsContext registers variables under bkms namespace.
func SetBkmsContext(vars map[string]string) ContextOption {
	return setContext(ContextBkms, vars)
}

func setContext[T any](key contextKey, vars map[string]T) ContextOption {
	return func(d Context) {
		if vars == nil {
			vars = make(map[string]T)
		}
		d[string(key)] = maps.Clone(vars)
	}
}
