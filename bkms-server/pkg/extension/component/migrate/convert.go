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

package migrate

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"text/template"
	"text/template/parse"

	"gopkg.in/yaml.v3"
)

const templatePlaceholderPrefix = "__BKMS_COMPONENT_TEMPLATE_"

type protectedTemplate struct {
	text         string
	replacements map[string]string
}

func (p protectedTemplate) restore(value string) string {
	for placeholder, action := range p.replacements {
		value = strings.ReplaceAll(value, placeholder, action)
	}
	return value
}

// ConvertLegacyOutput converts one legacy output document into ordered root fragments.
func ConvertLegacyOutput(output string) ([]string, []string, error) {
	protected, err := protectComponentTemplate(output)
	if err != nil {
		return nil, nil, err
	}

	var document yaml.Node
	if err = yaml.Unmarshal([]byte(protected.text), &document); err != nil {
		return nil, nil, fmt.Errorf("parse output YAML: %w", err)
	}
	root := yamlDocumentRoot(&document)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("output root is not a mapping")
	}

	patchers, err := convertLegacyPatchers(yamlMappingValue(root, "patcher"), protected)
	if err != nil {
		return nil, nil, err
	}
	specs, err := convertLegacySpecs(yamlMappingValue(root, "spec"), protected)
	if err != nil {
		return nil, nil, err
	}
	return patchers, specs, nil
}

func protectComponentTemplate(value string) (protectedTemplate, error) {
	// 与存量渲染逻辑保持一致，兼容 JSON/YAML 双引号字符串中的 raw \"literal\" 写法。
	if strings.Contains(value, "{{") {
		value = strings.ReplaceAll(value, `\"`, `"`)
	}
	tmpl, err := template.New("component-output").Funcs(template.FuncMap{
		"raw": func(s string) string { return s },
	}).Parse(value)
	if err != nil {
		return protectedTemplate{}, fmt.Errorf("parse Go template: %w", err)
	}
	if len(tmpl.Templates()) != 1 {
		return protectedTemplate{}, fmt.Errorf("associated Go templates require manual migration")
	}

	var protected strings.Builder
	replacements := make(map[string]string)
	for index, node := range tmpl.Root.Nodes {
		switch node := node.(type) {
		case *parse.TextNode:
			protected.Write(node.Text)
		case *parse.ActionNode:
			if len(node.Pipe.Decl) != 0 {
				return protectedTemplate{}, fmt.Errorf("go template declarations require manual migration")
			}
			action, err := normalizeRawAction(node)
			if err != nil {
				return protectedTemplate{}, err
			}
			placeholder := fmt.Sprintf("%s%06d__", templatePlaceholderPrefix, index)
			replacements[placeholder] = action
			protected.WriteString(placeholder)
		default:
			return protectedTemplate{}, fmt.Errorf("go template node %T requires manual migration", node)
		}
	}
	return protectedTemplate{text: protected.String(), replacements: replacements}, nil
}

// normalizeRawAction 将旧 raw 调用改写为等价的字符串字面量，避免新格式继续依赖 raw 函数。
func normalizeRawAction(node *parse.ActionNode) (string, error) {
	for _, command := range node.Pipe.Cmds {
		if len(command.Args) == 0 {
			continue
		}
		identifier, ok := command.Args[0].(*parse.IdentifierNode)
		if !ok || identifier.Ident != "raw" {
			continue
		}
		if len(node.Pipe.Cmds) != 1 || len(command.Args) != 2 {
			return "", fmt.Errorf("raw action requires exactly one string literal")
		}
		literal, ok := command.Args[1].(*parse.StringNode)
		if !ok {
			return "", fmt.Errorf("raw action requires exactly one string literal")
		}
		return fmt.Sprintf("{{%s}}", literal.String()), nil
	}
	return node.String(), nil
}

// convertLegacyPatchers 将旧 patcher 的点路径 Map 转换为 patchers[]。
// 每个点路径单独展开成一个根节点 YAML，按路径排序保证迁移结果稳定。
func convertLegacyPatchers(node *yaml.Node, protected protectedTemplate) ([]string, error) {
	if node == nil || node.Tag == "!!null" {
		return []string{}, nil
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("patcher is not a mapping")
	}

	type pathValue struct {
		path  string
		value *yaml.Node
	}
	entries := make([]pathValue, 0, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		path := node.Content[i].Value
		if path == "" || strings.Contains(path, templatePlaceholderPrefix) {
			return nil, fmt.Errorf("patcher path %q is invalid", path)
		}
		entries = append(entries, pathValue{path: path, value: node.Content[i+1]})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })

	patchers := make([]string, 0, len(entries))
	for _, entry := range entries {
		root, err := expandPatcherPath(entry.path, entry.value)
		if err != nil {
			return nil, err
		}
		data, err := yaml.Marshal(root)
		if err != nil {
			return nil, fmt.Errorf("marshal patcher path %q: %w", entry.path, err)
		}
		patchers = append(patchers, protected.restore(string(data)))
	}
	return patchers, nil
}

// convertLegacySpecs 将旧 spec 数组拆成 specs[]，每个元素是一份独立的资源 YAML。
func convertLegacySpecs(node *yaml.Node, protected protectedTemplate) ([]string, error) {
	if node == nil || node.Tag == "!!null" {
		return []string{}, nil
	}
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("spec is not a sequence")
	}
	specs := make([]string, 0, len(node.Content))
	for index, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("spec[%d] is not a mapping", index)
		}
		data, err := yaml.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("marshal spec[%d]: %w", index, err)
		}
		specs = append(specs, protected.restore(string(data)))
	}
	return specs, nil
}

// expandPatcherPath 将 "spec.template.spec" 还原为 spec.template.spec 嵌套 Mapping。
func expandPatcherPath(path string, value *yaml.Node) (*yaml.Node, error) {
	parts := strings.Split(path, ".")
	if slices.Contains(parts, "") {
		return nil, fmt.Errorf("patcher path %q contains an empty segment", path)
	}
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	current := root
	for index, part := range parts {
		current.Content = append(current.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part})
		if index == len(parts)-1 {
			cloned, err := yamlNodeClone(value)
			if err != nil {
				return nil, fmt.Errorf("clone patcher path %q value: %w", path, err)
			}
			current.Content = append(current.Content, cloned)
			break
		}
		nested := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		current.Content = append(current.Content, nested)
		current = nested
	}
	return root, nil
}

func yamlDocumentRoot(document *yaml.Node) *yaml.Node {
	if document == nil {
		return nil
	}
	if document.Kind == yaml.DocumentNode {
		if len(document.Content) == 0 {
			return nil
		}
		return document.Content[0]
	}
	return document
}

func yamlMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

// yamlNodeClone 通过序列化和反序列化复制 YAML 节点，避免新旧语法树共享节点。
func yamlNodeClone(node *yaml.Node) (*yaml.Node, error) {
	data, err := yaml.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("marshal YAML node: %w", err)
	}
	var document yaml.Node
	if err = yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("unmarshal YAML node: %w", err)
	}
	return yamlDocumentRoot(&document), nil
}
