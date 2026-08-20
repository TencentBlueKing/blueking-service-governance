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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// Manifest 相关常量
const (
	// maxManifestSize YAML 序列化后的最大字节数（5MB）
	maxManifestSize = 5 << 20

	// manifestFormatYAML YAML 格式标识
	manifestFormatYAML = "yaml"

	// lastAppliedConfigAnnotation kubectl last-applied-configuration 注解 key
	lastAppliedConfigAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

	// bkmsLastAppliedAnnotation 联邦 Merge Patch 使用的 last-applied 注解，体积大且对用户无意义
	bkmsLastAppliedAnnotation = "bkms.tencent.com/last-applied-configuration"
)

// BuildNodeManifest 从非结构化对象构建节点 Manifest
// 流程：删除 managedFields → 删除 last-applied-configuration → Secret 脱敏 → YAML marshal → 超大截断
func BuildNodeManifest(obj *unstructured.Unstructured) (*NodeManifest, error) {
	// 清理 metadata 中的噪声字段（managedFields、last-applied-configuration 注解）
	sanitizeMetadata(obj)

	// YAML 序列化
	yamlBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return nil, errors.Wrap(err, "marshal manifest to YAML")
	}

	manifest := &NodeManifest{
		Format:    manifestFormatYAML,
		Truncated: false,
	}

	// 超大截断检查
	if len(yamlBytes) > maxManifestSize {
		manifest.Content = "# Manifest too large to display (exceeds 5MB limit)\n" +
			"# Please use kubectl to view this resource directly."
		manifest.Truncated = true
	} else {
		manifest.Content = string(yamlBytes)
	}

	return manifest, nil
}

// sanitizeMetadata 清理 metadata 中的噪声字段
// 1. 删除 managedFields
// 2. 删除 annotations 中的 last-applied-configuration，若删除后注解为空则移除整个 annotations
func sanitizeMetadata(obj *unstructured.Unstructured) {
	metadata, found, _ := unstructured.NestedMap(obj.Object, "metadata")
	if !found {
		return
	}

	// 删除 managedFields
	delete(metadata, "managedFields")

	// 删除 last-applied-configuration 注解（kubectl / bkms 联邦 upsert）
	if annotations, ok := metadata["annotations"].(map[string]any); ok {
		delete(annotations, lastAppliedConfigAnnotation)
		delete(annotations, bkmsLastAppliedAnnotation)
		if len(annotations) == 0 {
			delete(metadata, "annotations")
		}
	}

	obj.Object["metadata"] = metadata
}
