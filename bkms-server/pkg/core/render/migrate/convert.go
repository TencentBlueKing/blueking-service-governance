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

// Package migrate converts legacy template syntax to Gonja namespaced syntax.
package migrate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"text/template/parse"

	gonjaConfig "github.com/nikolalohinski/gonja/v2/config"
	gonjaNodes "github.com/nikolalohinski/gonja/v2/nodes"
	gonjaParser "github.com/nikolalohinski/gonja/v2/parser"
	gonjaTokens "github.com/nikolalohinski/gonja/v2/tokens"
)

// ErrNeedsManual indicates the text contains constructs that cannot be auto-converted.
var ErrNeedsManual = errors.New("template contains manual-migration-only constructs")

// builtinCamelToSnake 定义了内置变量从 camelCase 到 SNAKE_CASE 的映射关系，用于在转换过程中识别和重写内置变量。
// 这个映射用于迁移旧 env-scope Go template 写法。
// 历史上 env-scope 除了 {{ .BKMS.ENV.X }}，还兼容过 {{ .bkmsAppName }}
// 这 5 个 camelCase 内置变量。新 runtime 只保留 env namespace，因此迁移时
// 把这些 camelCase 内置变量统一改写成对应的 env.SNAKE_CASE。
var builtinCamelToSnake = map[string]string{
	"bkmsAppName":       "BKMS_APP_NAME",
	"bkmsContainerName": "BKMS_CONTAINER_NAME",
	"bkmsEnvName":       "BKMS_ENV_NAME",
	"bkmsEnvNamespace":  "BKMS_ENV_NAMESPACE",
	"bkmsEnvCluster":    "BKMS_ENV_CLUSTER",
}

// Convert rewrites a string rendered in env-scope (TAF/config/property values).
// Accepts {{ .BKMS.ENV.X }}, {{ .<camelBuiltin> }}, {{ raw "literal" }} and ${{...}} pure variables.
// Bare ${{X}} is normalized to ${{env.X}} because render.Context no longer expands variables to root.
func Convert(text string) (string, error) {
	if text == "" {
		return text, nil
	}

	masked, spans := protect(text)

	// 参考 RenderGoTemplate 内的实现，需要先替换 \" 为 "
	masked = strings.ReplaceAll(masked, `\"`, `"`)

	out, err := walkGoTemplate(masked)
	if err != nil {
		return "", err
	}

	converted := restoreSpans(out, spans)

	if err := validateGonjaAST(converted); err != nil {
		return "", err
	}
	return converted, nil
}

// protect replaces every ${{...}} run in text with a sentinel "\x00<idx>\x00".
// The sentinel survives text/template parsing as plain text.
func protect(text string) (string, []string) {
	var (
		out   strings.Builder
		spans []string
	)
	i := 0
	for i < len(text) {
		if i+3 <= len(text) && text[i:i+3] == "${{" {
			end := strings.Index(text[i+3:], "}}")
			if end < 0 {
				out.WriteString(text[i:])
				break
			}
			rawEnd := i + 3 + end + 2
			spans = append(spans, text[i:rawEnd])
			fmt.Fprintf(&out, "\x00%d\x00", len(spans)-1)
			i = rawEnd
			continue
		}
		out.WriteByte(text[i])
		i++
	}
	return out.String(), spans
}

var sentinelRe = regexp.MustCompile(`\x00(\d+)\x00`)

// restoreSpans replaces sentinels with the original ${{...}} spans,
// rewriting bare ${{X}} to ${{env.X}}.
func restoreSpans(text string, spans []string) string {
	return sentinelRe.ReplaceAllStringFunc(text, func(m string) string {
		sub := sentinelRe.FindStringSubmatch(m)
		idx := mustAtoi(sub[1])
		return normalizeGonjaSpan(spans[idx])
	})
}

func mustAtoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// normalizeGonjaSpan rewrites bare ${{X}} into ${{env.X}}; namespaced spans are left as-is.
func normalizeGonjaSpan(span string) string {
	inner := strings.TrimSpace(span[3 : len(span)-2])
	for _, ns := range []string{"env."} {
		if strings.HasPrefix(inner, ns) {
			return span
		}
	}
	return fmt.Sprintf("${{env.%s}}", inner)
}

// walkGoTemplate parses masked text with text/template and walks the top-level AST nodes.
// Only TextNode and the strict ActionNode shapes (single FieldNode, or raw "literal") are accepted.
func walkGoTemplate(masked string) (string, error) {
	if !strings.Contains(masked, "{{") {
		return masked, nil
	}
	funcs := template.FuncMap{"raw": func(s string) string { return s }}
	tmpl, err := template.New("conv").Funcs(funcs).Parse(masked)
	if err != nil {
		return "", fmt.Errorf("%w: parse go template: %v", ErrNeedsManual, err)
	}
	if len(tmpl.Templates()) > 1 {
		return "", fmt.Errorf("%w: associated templates (define) not supported", ErrNeedsManual)
	}
	var out strings.Builder
	for _, n := range tmpl.Root.Nodes {
		emitted, err := emitGoNode(n)
		if err != nil {
			return "", err
		}
		out.WriteString(emitted)
	}
	return out.String(), nil
}

func emitGoNode(n parse.Node) (string, error) {
	switch t := n.(type) {
	case *parse.TextNode:
		return string(t.Text), nil
	case *parse.ActionNode:
		return emitAction(t)
	default:
		return "", fmt.Errorf("%w: unsupported node %T", ErrNeedsManual, n)
	}
}

func emitAction(a *parse.ActionNode) (string, error) {
	pipe := a.Pipe
	if pipe == nil || len(pipe.Decl) > 0 || len(pipe.Cmds) != 1 {
		return "", fmt.Errorf("%w: complex pipeline", ErrNeedsManual)
	}
	cmd := pipe.Cmds[0]
	if len(cmd.Args) == 0 {
		return "", fmt.Errorf("%w: empty command", ErrNeedsManual)
	}

	if len(cmd.Args) == 1 {
		if field, ok := cmd.Args[0].(*parse.FieldNode); ok {
			return mapEnvField(field.Ident)
		}
	}

	if len(cmd.Args) == 2 {
		ident, ok1 := cmd.Args[0].(*parse.IdentifierNode)
		str, ok2 := cmd.Args[1].(*parse.StringNode)
		if ok1 && ok2 && ident.Ident == "raw" {
			return emitRawLiteral(str.Text)
		}
	}

	return "", fmt.Errorf("%w: action shape not supported", ErrNeedsManual)
}

func mapEnvField(ident []string) (string, error) {
	if len(ident) == 0 {
		return "", fmt.Errorf("%w: empty field", ErrNeedsManual)
	}
	if ident[0] == "BKMS" {
		if len(ident) != 3 || ident[1] != "ENV" {
			return "", fmt.Errorf("%w: unsupported BKMS path %v", ErrNeedsManual, ident)
		}
		return fmt.Sprintf("${{env.%s}}", ident[2]), nil
	}
	if len(ident) != 1 {
		return "", fmt.Errorf("%w: multi-segment field %v", ErrNeedsManual, ident)
	}
	name := ident[0]
	if snake, ok := builtinCamelToSnake[name]; ok {
		return fmt.Sprintf("${{env.%s}}", snake), nil
	}
	return "", fmt.Errorf("%w: unknown env-scope field %q", ErrNeedsManual, name)
}

// emitRawLiteral
// emitRawLiteral emits the raw string literal verbatim. {{...}} inside is fine (canonical
// raw usage); ${{...}} would be re-interpreted by gonja at runtime so we reject by checking
// for sentinel NUL bytes (sentinels indicate the original raw string had a ${{...}} span).
func emitRawLiteral(s string) (string, error) {
	if strings.ContainsRune(s, '\x00') {
		return "", fmt.Errorf("%w: raw content contains gonja marker", ErrNeedsManual)
	}
	return s, nil
}

// validateGonjaAST parses converted text with gonja and ensures every Output node is a pure
// variable expression (Name + GetAttribute chain) — no filters, conditions, tests, calls,
// indexing, or other higher-order constructs.
func validateGonjaAST(converted string) error {
	if !strings.Contains(converted, "${{") {
		return nil
	}
	cfg := gonjaConfig.New()
	cfg.VariableStartString = "${{"
	cfg.VariableEndString = "}}"
	cfg.BlockStartString = "<##BLOCK_DISABLED##"
	cfg.BlockEndString = "##BLOCK_DISABLED##>"
	cfg.CommentStartString = "<##COMMENT_DISABLED##"
	cfg.CommentEndString = "##COMMENT_DISABLED##>"

	stream := gonjaTokens.Lex(converted, cfg)
	p := gonjaParser.NewParser("convert", stream, cfg, nil, nil)
	root, err := p.Parse()
	if err != nil {
		return fmt.Errorf("%w: parse gonja: %v", ErrNeedsManual, err)
	}
	for _, n := range root.Nodes {
		switch x := n.(type) {
		case *gonjaNodes.Data:
		case *gonjaNodes.Output:
			if x.Condition != nil || x.Alternative != nil {
				return fmt.Errorf("%w: gonja conditional output", ErrNeedsManual)
			}
			if err := validateGonjaExpr(x.Expression); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: unsupported gonja top node %T", ErrNeedsManual, n)
		}
	}
	return nil
}

func validateGonjaExpr(expr gonjaNodes.Expression) error {
	switch e := expr.(type) {
	case *gonjaNodes.Name:
		return nil
	case *gonjaNodes.GetAttribute:
		inner, ok := e.Node.(gonjaNodes.Expression)
		if !ok {
			return fmt.Errorf("%w: getattribute on non-expression", ErrNeedsManual)
		}
		return validateGonjaExpr(inner)
	default:
		return fmt.Errorf("%w: gonja expression %T not allowed", ErrNeedsManual, expr)
	}
}

// HasTemplate returns true if s may contain legacy or Gonja template markers.
func HasTemplate(s string) bool {
	return strings.Contains(s, "{{")
}
