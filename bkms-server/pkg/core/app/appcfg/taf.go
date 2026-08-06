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

package appcfg

import (
	"bytes"
	"strings"

	"github.com/TarsCloud/TarsGo/tars/util/conf"
	"github.com/pkg/errors"
)

// mergeTAFContent merges two TAF contents, with overlay taking precedence over base.
// TAF format only supports simple merge, overlay values replace base values.
//
// The merge logic:
//   - For leaf nodes (key=value pairs): overlay values replace base values
//   - For branch nodes (<section>): recursively merge children
//   - New nodes in overlay are added to the result
func mergeTAFContent(base string, overlay *string) (string, error) {
	if overlay == nil {
		return base, nil
	}
	if *overlay == "" {
		return base, nil
	}

	// Parse base config
	baseConf := conf.New()
	if err := baseConf.InitFromString(base); err != nil {
		return "", errors.Wrap(err, "parsing base TAF content")
	}

	// Parse overlay config
	overlayConf := conf.New()
	if err := overlayConf.InitFromString(*overlay); err != nil {
		return "", errors.Wrap(err, "parsing overlay TAF content")
	}

	// Build merged config structure
	merged := mergeTAFNodes(baseConf, overlayConf, "")

	// Serialize merged config to TAF format string
	return serializeTAFNode(merged, 0), nil
}

// tafNode represents a node in the TAF config tree
type tafNode struct {
	// node name (section name or empty for root)
	name string
	// key=value pairs at this level
	keyvals map[string]string
	// raw lines (for preserving order and comments)
	lines []string
	// child sections
	children map[string]*tafNode
	// names of child sections, keep order of children for consistent output
	childrenNames []string
}

// newTAFNode creates a new tafNode
func newTAFNode(name string) *tafNode {
	return &tafNode{
		name:          name,
		keyvals:       make(map[string]string),
		lines:         make([]string, 0),
		children:      make(map[string]*tafNode),
		childrenNames: make([]string, 0),
	}
}

// mergeTAFNodes merges overlay config into base config recursively
// Returns the merged TAF node tree.
// 拆分为若干辅助函数以降低圈复杂度：键值合并、base 行处理、overlay-only 行处理、子域递归合并各自独立
func mergeTAFNodes(base, overlay *conf.Conf, path string) *tafNode {
	merged := newTAFNode("")

	baseDomains := base.GetDomain(path)
	overlayDomains := overlay.GetDomain(path)
	baseLines := base.GetDomainLine(path)
	overlayLines := overlay.GetDomainLine(path)

	// 合并当前层级的 key=value，overlay 覆盖 base
	mergeKeyvals(merged, base.GetMap(path), overlay.GetMap(path))

	// 按 base 顺序重放行，使用 merged 后的值（可能被 overlay 覆盖）
	appendBaseLines(merged, baseLines)

	// 追加仅在 overlay 出现的键值行与无 = 的原样行
	appendOverlayOnlyLines(merged, baseLines, overlayLines)

	// 递归合并子域：先按 base 顺序，再补充 overlay 独有域
	mergeTAFChildren(base, overlay, path, baseDomains, overlayDomains, merged)

	return merged
}

// mergeKeyvals 合并两个层级 key=value：baseMap 先写入，overlayMap 后写入以实现覆盖优先
func mergeKeyvals(merged *tafNode, baseMap, overlayMap map[string]string) {
	for k, v := range baseMap {
		merged.keyvals[k] = v
	}
	for k, v := range overlayMap {
		merged.keyvals[k] = v
	}
}

// appendBaseLines 按 base 中原有顺序将行追加到 merged.lines
// 空行与注释原样保留；key=value 行使用合并后的值（可能已被 overlay 覆盖）；无 = 的行原样保留
func appendBaseLines(merged *tafNode, baseLines []string) {
	for _, line := range baseLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			merged.lines = append(merged.lines, line)
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			if val, ok := merged.keyvals[key]; ok {
				merged.lines = append(merged.lines, key+"="+val)
			}
			continue
		}
		// 无 = 的整行（如 'allow'）
		merged.lines = append(merged.lines, line)
	}
}

// collectBaseKeys 从 base 行中提取所有 key=value 的 key 集合，用于识别 overlay-only 的键
func collectBaseKeys(baseLines []string) map[string]bool {
	keys := make(map[string]bool)
	for _, line := range baseLines {
		if idx := strings.Index(line, "="); idx > 0 {
			keys[strings.TrimSpace(line[:idx])] = true
		}
	}
	return keys
}

// appendOverlayOnlyLines 处理仅出现在 overlay 中的行：
// - key=value 行：base 中不存在同名 key 时追加合并后的值
// - 无 = 的原样行（非空且非注释）：base 行中未出现过时追加
func appendOverlayOnlyLines(merged *tafNode, baseLines, overlayLines []string) {
	baseKeySet := collectBaseKeys(baseLines)
	for _, line := range overlayLines {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			if baseKeySet[key] {
				continue
			}
			if val, ok := merged.keyvals[key]; ok {
				merged.lines = append(merged.lines, key+"="+val)
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !containsTrimmedLine(baseLines, line) {
			merged.lines = append(merged.lines, line)
		}
	}
}

// containsTrimmedLine 判断 lines 中（按 TrimSpace 后）是否存在与 target 完全相等的行
func containsTrimmedLine(lines []string, target string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) == target {
			return true
		}
	}
	return false
}

// mergeTAFChildren 递归合并子域到 merged：
// 先按 base 顺序合并所有 base 子域，再补充仅出现在 overlay 中的子域，保证输出顺序稳定
func mergeTAFChildren(base, overlay *conf.Conf, path string, baseDomains, overlayDomains []string, merged *tafNode) {
	for _, domain := range baseDomains {
		merged.children[domain] = mergeTAFNodes(base, overlay, joinTAFPath(path, domain))
		merged.childrenNames = append(merged.childrenNames, domain)
	}
	for _, domain := range overlayDomains {
		if _, exists := merged.children[domain]; exists {
			continue
		}
		merged.children[domain] = mergeTAFNodes(base, overlay, joinTAFPath(path, domain))
		merged.childrenNames = append(merged.childrenNames, domain)
	}
}

// joinTAFPath 按 TAF 的路径约定拼接子域路径：根路径拼接前缀 "/"，非根路径以 "/" 分隔
func joinTAFPath(path, domain string) string {
	if path == "" {
		return "/" + domain
	}
	return path + "/" + domain
}

// serializeTAFNode serializes a TAF node tree to string with proper indentation
func serializeTAFNode(node *tafNode, indent int) string {
	var buf bytes.Buffer
	indentStr := strings.Repeat("    ", indent)

	// Write key-value lines
	for _, line := range node.lines {
		buf.WriteString(indentStr)
		buf.WriteString(line)
		buf.WriteString("\n")
	}

	// Write child sections in order
	for _, childName := range node.childrenNames {
		child := node.children[childName]
		buf.WriteString(indentStr)
		buf.WriteString("<")
		buf.WriteString(childName)
		buf.WriteString(">\n")
		buf.WriteString(serializeTAFNode(child, indent+1))
		buf.WriteString(indentStr)
		buf.WriteString("</")
		buf.WriteString(childName)
		buf.WriteString(">\n")
	}

	return buf.String()
}
