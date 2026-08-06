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

// Package render provides template rendering with variables and functions
// scoped by a rendering context.
package render

import (
	"bytes"
	"os"
	"strings"
	"text/template"
)

// Renderer renders text with variables and functions from its context.
type Renderer struct {
	context Context
}

// New creates a Renderer with the supplied context options.
func New(opts ...ContextOption) *Renderer {
	return &Renderer{
		context: NewContext(opts...),
	}
}

// Render renders text using ${{ }} variable syntax with the given data.
// Supports root-level e.g. ${{env.VAR}} when configured via SetEnvContext.
func (r *Renderer) Render(text string) (string, error) {
	return renderGonja(text, r.context)
}

// RenderGoTemplate 使用标准 Go template 语法渲染字符串。
// 当前仅组件 patcher/spec 模板渲染还在使用，考虑后期迁移并移除。
func RenderGoTemplate(value string, data map[string]any) (string, error) {
	if !strings.Contains(value, "{{") {
		return value, nil
	}

	// FIXME: 后续移除 raw 语法后，这里可以去掉
	// Workaround: Replace escaped quotes with actual quotes to handle JSON-encoded strings.
	// This allows templates like `{{ raw \"{{.bcs.pod_name}}\" }}` to be parsed correctly.
	normalizedVal := strings.ReplaceAll(value, `\"`, `"`)

	tmpl, err := template.New("value").Funcs(goTmplFuncMap).Parse(normalizedVal)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderShellVars renders shell-style variable references like "${VAR_NAME}" or "$VAR_NAME"
// in the content string using the provided variable map. Variables not found in the map
// will be left unchanged.
//
// NOTE: os.Expand 无法区分原始写法是 "$VAR" 还是 "${VAR}"，因此未匹配的变量统一
// 恢复为 "${VAR}" 格式。例如输入 "$UNKNOWN" 未匹配时会变为 "${UNKNOWN}"。 参考单元测试 render_test.go
func RenderShellVars(content string, vars map[string]string) string {
	return os.Expand(content, func(key string) string {
		if val, ok := vars[key]; ok {
			return val
		}
		return "${" + key + "}"
	})
}

// goTmplFuncMap 导出自定义模板函数映射，供外部（如 evaluate.go 中 output 渲染）使用。
//
// FIXME: 为了支持存量数据，暂时保留 raw 写法，后续考虑如何迁移或兼容
//   - raw: 返回原始字符串，不做处理。用于在模板中保留嵌套的模板语法。
//     例如: {{ raw "{{.bcs.pod_name}}" }} 输出 {{.bcs.pod_name}}
var goTmplFuncMap = template.FuncMap{
	"raw": func(s string) string {
		return s
	},
}
