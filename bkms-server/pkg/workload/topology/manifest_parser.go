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

package topology

import (
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// manifestDocSeparator YAML 多文档分隔符
const manifestDocSeparator = "---"

// k8sResourceMeta 用于解析 Kubernetes 资源 Manifest 中的元信息
type k8sResourceMeta struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
}

// ParseManifest 解析 Helm Release Manifest（多文档 YAML），提取资源条目列表
// manifest 是以 "---" 分隔的多文档 YAML 字符串
// defaultNamespace 用于填充未指定 namespace 的资源
func ParseManifest(manifest, defaultNamespace string) ([]ResourceEntry, error) {
	if strings.TrimSpace(manifest) == "" {
		return nil, nil
	}

	docs := strings.Split(manifest, manifestDocSeparator)
	var entries []ResourceEntry

	for _, doc := range docs {
		trimmed := strings.TrimSpace(doc)
		if trimmed == "" {
			continue
		}

		// 跳过纯注释文档
		if isCommentOnly(trimmed) {
			continue
		}

		var meta k8sResourceMeta
		if err := yaml.Unmarshal([]byte(trimmed), &meta); err != nil {
			return nil, errors.Wrapf(err, "unmarshal manifest document")
		}

		// 跳过缺失关键字段的文档
		if meta.Kind == "" || meta.Metadata.Name == "" {
			continue
		}

		// 集群级别（Cluster-scoped）资源不需要 namespace，保留为空
		namespace := meta.Metadata.Namespace
		if namespace == "" && !k8skind.IsClusterScoped(meta.Kind) {
			namespace = defaultNamespace
		}

		entries = append(entries, ResourceEntry{
			Kind:         meta.Kind,
			APIVersion:   meta.APIVersion,
			Namespace:    namespace,
			Name:         meta.Metadata.Name,
			IsManaged:    true,
			SourceType:   SourceTypeHelmManifest,
			SourceReason: "parsed from Helm release manifest",
		})
	}

	return entries, nil
}

// isCommentOnly 判断文本是否只包含 YAML 注释行（以 # 开头的行）
func isCommentOnly(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		if !strings.HasPrefix(trimmedLine, "#") {
			return false
		}
	}
	return true
}
