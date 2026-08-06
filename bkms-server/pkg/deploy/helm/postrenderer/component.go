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

package postrenderer

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/postrender"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	helmcomp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/helm"
)

// ComponentPostRenderer 组件 PostRenderer
// 根据组件配置对 Helm 渲染后的 manifest 进行 patch 和资源追加
type ComponentPostRenderer struct {
	// components 按优先级排序的组件 patch 列表
	components []ComponentPatch
}

// ComponentPatch 单个组件的 patch 信息
type ComponentPatch struct {
	// Name 组件名称（用于错误信息）
	Name string
	// Target 目标资源选择器
	Target helmcomp.TargetResourceSelector
	// Patchers 是按顺序执行的根节点 Patch。
	Patchers []map[string]any
	// Specs 是要追加的额外资源。
	Specs []map[string]any
}

// 编译期接口实现检查
var _ postrender.PostRenderer = (*ComponentPostRenderer)(nil)

// NewComponentPostRenderer 创建组件 PostRenderer
// 如果 items 为空，返回 nil（表示不需要组件 PostRenderer）
func NewComponentPostRenderer(items []ComponentPatch) *ComponentPostRenderer {
	if len(items) == 0 {
		return nil
	}
	return &ComponentPostRenderer{components: items}
}

// Run 实现 PostRenderer 接口
// 1. 解析 multi-doc YAML 为文档列表
// 2. 对每个组件：匹配目标资源并应用 patcher（JSON Merge Patch）
// 3. 收集所有组件的 Spec（额外资源），追加到文档列表末尾
// 4. 重新组合为 multi-doc YAML 返回
func (r *ComponentPostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	if r == nil || len(r.components) == 0 {
		return renderedManifests, nil
	}

	// 1. 解析 multi-doc YAML
	docs, err := parseMultiDocYAML(renderedManifests)
	if err != nil {
		return nil, errors.Wrap(err, "parse multi-doc YAML for component post renderer")
	}

	// 2. 对每个组件应用 patcher
	matchedItems := make(map[int]bool, len(r.components))
	for idx, item := range r.components {
		if len(item.Patchers) == 0 {
			continue
		}

		matched := false
		for i, doc := range docs {
			if item.matches(doc) {
				matched = true
				patched := doc
				for patcherIndex, patcher := range item.Patchers {
					patched, err = applyPatcher(patched, patcher)
					if err != nil {
						return nil, errors.Wrapf(
							err, "component %q patcher[%d] failed", item.Name, patcherIndex,
						)
					}
				}
				docs[i] = patched
			}
		}
		if !matched {
			log.WarnNoContextf(
				"component %q target not found: %s/%s",
				item.Name,
				item.Target.Kind,
				item.Target.Name,
			)
		}
		matchedItems[idx] = matched
	}

	// 3. 追加 spec（额外资源）
	for i, item := range r.components {
		if len(item.Patchers) > 0 && !matchedItems[i] {
			continue
		}
		docs = append(docs, item.Specs...)
	}

	// 4. 重新组合为 multi-doc YAML
	return assembleMultiDocYAML(docs)
}

func (item ComponentPatch) matches(doc map[string]any) bool {
	return item.Target.Matches(doc)
}

// parseMultiDocYAML 解析 multi-doc YAML 为文档列表
func parseMultiDocYAML(buf *bytes.Buffer) ([]map[string]any, error) {
	decoder := yaml.NewDecoder(buf)
	var docs []map[string]any

	for {
		var doc map[string]any
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "decode YAML document")
		}
		// 跳过空文档
		if doc == nil {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// assembleMultiDocYAML 将文档列表重新组合为 multi-doc YAML
func assembleMultiDocYAML(docs []map[string]any) (*bytes.Buffer, error) {
	var outputDocs []string
	for _, doc := range docs {
		var buf bytes.Buffer
		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(2)
		if err := encoder.Encode(doc); err != nil {
			kind, _ := doc["kind"].(string)
			return nil, errors.Wrapf(err, "encode manifest document kind=%s", kind)
		}
		encoder.Close()
		outputDocs = append(outputDocs, strings.TrimSpace(buf.String()))
	}

	result := strings.Join(outputDocs, yamlDocumentSeparator)
	return bytes.NewBufferString(result + "\n"), nil
}

// applyPatcher 将根节点 patcher 应用到目标文档
// 使用 JSON Merge Patch 语义（RFC 7396）
func applyPatcher(doc, patcher map[string]any) (map[string]any, error) {
	// 将原始文档转为 JSON
	docJSON, err := json.Marshal(doc)
	if err != nil {
		return nil, errors.Wrap(err, "marshal document to JSON")
	}

	patchJSON, err := json.Marshal(patcher)
	if err != nil {
		return nil, errors.Wrap(err, "marshal patcher")
	}
	docJSON, err = jsonMergePatch(docJSON, patchJSON)
	if err != nil {
		return nil, errors.Wrap(err, "apply JSON merge patch")
	}

	// 将结果转回 map
	var result map[string]any
	if err := json.Unmarshal(docJSON, &result); err != nil {
		return nil, errors.Wrap(err, "unmarshal patched document")
	}
	return result, nil
}

// jsonMergePatch 执行 JSON Merge Patch（RFC 7396）
// 将 patch 合并到 original 中，返回合并后的 JSON
func jsonMergePatch(original, patch []byte) ([]byte, error) {
	var originalMap map[string]any
	if err := json.Unmarshal(original, &originalMap); err != nil {
		return nil, errors.Wrap(err, "unmarshal original for merge")
	}

	var patchMap map[string]any
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return nil, errors.Wrap(err, "unmarshal patch for merge")
	}

	merged := deepMerge(originalMap, patchMap)
	return json.Marshal(merged)
}

// deepMerge 深度合并两个 map，patch 中的值覆盖 base 中的值
// 当两个值都是 map 时递归合并，否则 patch 值覆盖 base 值
// null 值表示删除该字段（JSON Merge Patch 语义）
func deepMerge(base, patch map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}

	for k, patchVal := range patch {
		// null 值表示删除
		if patchVal == nil {
			delete(result, k)
			continue
		}

		baseVal, exists := result[k]
		if exists {
			// 如果两个值都是 map，递归合并
			baseMap, baseIsMap := baseVal.(map[string]any)
			patchMap, patchIsMap := patchVal.(map[string]any)
			if baseIsMap && patchIsMap {
				result[k] = deepMerge(baseMap, patchMap)
				continue
			}
		}
		// 否则直接覆盖
		result[k] = patchVal
	}
	return result
}
