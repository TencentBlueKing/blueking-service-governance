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

package render

import (
	"fmt"
	"strings"

	gonjaConfig "github.com/nikolalohinski/gonja/v2/config"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/loaders"
	gonjaNodes "github.com/nikolalohinski/gonja/v2/nodes"
	gonjaParser "github.com/nikolalohinski/gonja/v2/parser"
	gonjaTokens "github.com/nikolalohinski/gonja/v2/tokens"
)

// templateEnvironment 是 ${{context.KEY}} 渲染使用的 gonja 环境，仅初始化一次。
//
// 当前未注册 Filters / Tests / ControlStructures / Methods，
// 因此模板引擎仅支持简单的变量替换（如 ${{env.BKMS_APP_NAME}}）。
// 条件判断（if/elif/else）、循环（for）、过滤器（| upper）等均不可用。
// 如需扩展能力，可按需启用下方注释掉的字段。
var templateEnvironment = &exec.Environment{
	Context: exec.EmptyContext(),

	// Filters: 注册模板过滤器（如 {{ value | upper }}、{{ text | truncate(100) }}）。
	// 启用后可使用 gonja 内置的 52 个过滤器（upper/lower/default/join/sort/tojson 等）。
	// 适用方向：需要在模板中对变量值做格式化、截断、编码等后处理时启用。
	// Filters: builtins.Filters,

	// Tests: 注册模板测试函数（如 {% if value is defined %}、{% if num is even %}）。
	// 启用后可使用 gonja 内置的 33 个测试（defined/undefined/none/even/odd/eq/gt 等）。
	// 适用方向：需要在条件语句中对变量做类型判断或比较测试时启用（需同时启用 ControlStructures）。
	// Tests: builtins.Tests,

	// ControlStructures: 注册控制结构（if/for/block/macro/set/extends/include 等共 12 个）。
	// 启用后模板支持条件分支、循环迭代、模板继承等完整的模板逻辑能力。
	// 适用方向：需要根据变量值做条件渲染或遍历列表生成重复内容时启用。
	// ControlStructures: builtins.ControlStructures,

	// Methods: 为基础类型（string/int/float/bool/dict/list）注册实例方法。
	// 启用后可在模板中调用如 {{ name.upper() }}、{{ items.keys() }} 等方法。
	// 适用方向：需要在模板中直接调用值的方法做转换或查询时启用。
	// Methods: builtins.Methods,
}

// templateLoader 作为 ShiftedLoader 的 sub-loader，用于支持 include/import 等指令。
// 当前场景仅做变量替换，不会实际调用此 loader。
var templateLoader = loaders.MustNewMemoryLoader(map[string]string{})

// templateConfig 返回 ${{...}} 渲染所使用的 gonja 配置。
func templateConfig() *gonjaConfig.Config {
	cfg := gonjaConfig.New()
	cfg.VariableStartString = "${{"
	cfg.VariableEndString = "}}"
	// 禁用 block 和 comment 语法：设为不会出现在正常模板中的标记
	cfg.BlockStartString = "<##BLOCK_DISABLED##"
	cfg.BlockEndString = "##BLOCK_DISABLED##>"
	cfg.CommentStartString = "<##COMMENT_DISABLED##"
	cfg.CommentEndString = "##COMMENT_DISABLED##>"
	return cfg
}

// renderGonja 使用 ${{context.KEY}} 语法渲染字符串。
// 当前仅支持简单变量替换，控制流（{% if %} / {% for %}）和注释（{# #}）语法
// 已在配置层面禁用，模板中的 {% %} 和 {# #} 会作为普通文本原样保留。
func renderGonja(value string, data map[string]any) (string, error) {
	cfg := templateConfig()
	if !strings.Contains(value, cfg.VariableStartString) {
		return value, nil
	}

	const rootID = "render-template"

	loader, err := loaders.NewShiftedLoader(rootID, strings.NewReader(value), templateLoader)
	if err != nil {
		return "", fmt.Errorf("creating shifted loader: %w", err)
	}

	tmpl, err := exec.NewTemplate(rootID, cfg, loader, templateEnvironment)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	ctx := exec.NewContext(data)

	result, err := tmpl.ExecuteToString(ctx)
	if err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return result, nil
}

// VarsSet 按 context 命名空间分组的 Gonja 变量引用集合。
// 外层 key 为 context（如 "bkms"），内层 key 为变量名（如 "ARTIFACT_IMAGE"）。
// 例如模板 ${{ bkms.ARTIFACT_IMAGE }} + ${{ env.APP_NAME }} 返回
// {"bkms": {"ARTIFACT_IMAGE": {}}, "env": {"APP_NAME": {}}}.
type VarsSet map[string]map[string]struct{}

// ExtractVars 提取文本中的变量引用，按 context 命名空间分组返回。
// 仅收集纯变量表达式（如 ${{ bkms.ARTIFACT_IMAGE }}），过滤器/条件/函数调用等会被忽略。
//
// Args:
//   - text: 包含 Gonja 模板语法的文本，例如 "image: ${{ bkms.ARTIFACT_IMAGE }}"
//
// Returns:
//   - VarsSet: 按 context 分组的变量集合，例如 "image: ${{ bkms.ARTIFACT_IMAGE }}"
//     返回 VarsSet{"bkms": {"ARTIFACT_IMAGE": {}}}；文本不含变量标记时返回 nil
//   - error: 模板解析失败时返回
func ExtractVars(text string) (VarsSet, error) {
	cfg := templateConfig()
	// 无变量起始标记则无需 parse，直接返回。
	if !strings.Contains(text, cfg.VariableStartString) {
		return nil, nil
	}

	// 复用 Gonja 自身的 lexer/parser 将文本解析为 AST，再遍历 top-level 节点：
	// Data/Block/Comment 节点忽略，仅处理 Output（即 ${{ ... }}）节点，
	// 交由 collectVarRef 提取 (context, varName) 对后按 context 去重分组。
	stream := gonjaTokens.Lex(text, cfg)
	p := gonjaParser.NewParser("extract-vars", stream, cfg, nil, nil)
	root, err := p.Parse()
	if err != nil {
		return nil, err
	}

	result := make(VarsSet)
	for _, n := range root.Nodes {
		output, ok := n.(*gonjaNodes.Output)
		if !ok {
			continue
		}
		ctxName, varName := collectVarRef(output.Expression)
		if ctxName == "" || varName == "" {
			continue
		}
		if result[ctxName] == nil {
			result[ctxName] = make(map[string]struct{})
		}
		result[ctxName][varName] = struct{}{}
	}
	return result, nil
}

// collectVarRef 遍历 Gonja 表达式 AST，提取 (context, varName) 对。
//
// 仅支持 ${{ context.varName }} 两层结构（Name + 一级 GetAttribute）：
//   - ${{ X }}          → Name 节点，返回 ("", "X")，无命名空间
//   - ${{ A.B }}        → GetAttribute{Node: Name{"A"}, Attribute: "B"}，返回 ("A", "B")
//
// 嵌套超过一层（如 ${{ A.B.C }}）、过滤器、索引、函数调用等均返回 ("", "")。
func collectVarRef(expr gonjaNodes.Expression) (context, varName string) {
	switch e := expr.(type) {
	case *gonjaNodes.Name:
		return "", e.Name.Val
	case *gonjaNodes.GetAttribute:
		inner, ok := e.Node.(gonjaNodes.Expression)
		if !ok {
			return "", ""
		}
		prefixCtx, prefixVar := collectVarRef(inner)
		// GetAttribute 必须在 Name 之上：prefixCtx 应为空，prefixVar 为前缀名。
		// 例如 ${{ bkms.ARTIFACT_IMAGE }} → prefixCtx="" prefixVar="bkms"，合并 Attr 得 ("bkms", "ARTIFACT_IMAGE")
		if prefixCtx != "" || prefixVar == "" {
			return "", ""
		}
		return prefixVar, e.Attribute
	default:
		return "", ""
	}
}
