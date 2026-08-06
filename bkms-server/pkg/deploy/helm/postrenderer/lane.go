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
	"io"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/postrender"

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// yamlDocumentSeparator YAML 多文档分隔符
const yamlDocumentSeparator = "\n---\n"

// shouldAddTrafficLaneLabelsKinds 需要添加流量泳道标签的 k8s 资源类型
// 该变量被 LanePostRenderer 使用，用于判断哪些资源需要注入泳道标签
var shouldAddTrafficLaneLabelsKinds = []string{
	k8skind.Deploy,
	k8skind.STS,
	k8skind.GameDeploy,
	k8skind.GameSTS,
}

// LanePostRenderer 实现 Helm PostRenderer 接口
// 在 Helm 渲染 Manifest 后、提交到集群前，拦截目标资源并注入泳道标签
type LanePostRenderer struct {
	labels map[string]string
}

// NewLanePostRenderer 创建泳道标签 PostRenderer
// labels 为空时返回 nil（表示不需要 PostRenderer）
func NewLanePostRenderer(labels map[string]string) *LanePostRenderer {
	if len(labels) == 0 {
		return nil
	}
	return &LanePostRenderer{labels: labels}
}

var _ postrender.PostRenderer = (*LanePostRenderer)(nil)

// Run 实现 PostRenderer 接口
// 解析 multi-doc YAML，对工作负载类型资源注入泳道标签，其他资源原样返回
func (r *LanePostRenderer) Run(renderedManifests *bytes.Buffer) (*bytes.Buffer, error) {
	if r == nil || len(r.labels) == 0 {
		return renderedManifests, nil
	}

	// 使用 YAML decoder 逐个读取文档
	decoder := yaml.NewDecoder(renderedManifests)
	var outputDocs []string

	for {
		var doc map[string]any
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "decode manifest document")
		}

		// 空文档跳过
		if doc == nil {
			continue
		}

		// 判断资源类型是否为需要注入泳道标签的工作负载
		kind, _ := doc["kind"].(string)
		if slices.Contains(shouldAddTrafficLaneLabelsKinds, kind) {
			r.injectLabels(doc)
		}

		// 编码回 YAML
		var buf bytes.Buffer
		encoder := yaml.NewEncoder(&buf)
		encoder.SetIndent(2)
		if err := encoder.Encode(doc); err != nil {
			return nil, errors.Wrapf(err, "encode manifest document kind=%s", kind)
		}
		encoder.Close()
		outputDocs = append(outputDocs, strings.TrimSpace(buf.String()))
	}

	// 重新组合为 multi-doc YAML
	result := strings.Join(outputDocs, yamlDocumentSeparator)
	return bytes.NewBufferString(result + "\n"), nil
}

// injectLabels 向工作负载资源注入泳道标签
// 注入位置：spec.selector.matchLabels + spec.template.metadata.labels
func (r *LanePostRenderer) injectLabels(doc map[string]any) {
	// 确保 spec 字段存在
	spec := ensureMap(doc, "spec")

	// 注入到 spec.selector.matchLabels
	selector := ensureMap(spec, "selector")
	matchLabels := ensureMap(selector, "matchLabels")
	for k, v := range r.labels {
		matchLabels[k] = v
	}

	// 注入到 spec.template.metadata.labels
	template := ensureMap(spec, "template")
	metadata := ensureMap(template, "metadata")
	templateLabels := ensureMap(metadata, "labels")
	for k, v := range r.labels {
		templateLabels[k] = v
	}
}

// ensureMap 确保 parent[key] 是 map[string]any 类型，如果不存在则创建
func ensureMap(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key]; ok {
		if m, ok := existing.(map[string]any); ok {
			return m
		}
	}
	m := map[string]any{}
	parent[key] = m
	return m
}
