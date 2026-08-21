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

package client

import (
	"encoding/json"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// lastAppliedAnnotation 记录上次联邦 Merge Patch 写入的期望配置。
// 三路合并时据此把「上次写过、本次已省略」的字段置为 null，贴近 SSA 的省略即删除；
// 从未写入过的字段（如 Service clusterIP、status、其他控制器注入的 label）不会被删。
// 不复用 kubectl.kubernetes.io/last-applied-configuration，避免与 kubectl apply 互相覆盖。
const lastAppliedAnnotation = "bkms.tencent.com/last-applied-configuration"

// prepareFederationDesired 复制并清洗 manifest，写入 last-applied 注解。
// 返回的 desired 供 Create / 三路合并使用；不修改调用方传入的 manifest。
func prepareFederationDesired(manifest map[string]any) (map[string]any, error) {
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, errors.Wrap(err, "marshal manifest")
	}
	var desired map[string]any
	if err = json.Unmarshal(data, &desired); err != nil {
		return nil, errors.Wrap(err, "unmarshal manifest")
	}
	sanitizeManifestForSSA(desired)
	stripLastAppliedAnnotation(desired)

	lastApplied, err := json.Marshal(desired)
	if err != nil {
		return nil, errors.Wrap(err, "marshal last-applied configuration")
	}
	setLastAppliedAnnotation(desired, string(lastApplied))
	return desired, nil
}

func stripLastAppliedAnnotation(obj map[string]any) {
	metadata, _ := obj["metadata"].(map[string]any)
	if metadata == nil {
		return
	}
	annotations, _ := metadata["annotations"].(map[string]any)
	if annotations == nil {
		return
	}
	delete(annotations, lastAppliedAnnotation)
	if len(annotations) == 0 {
		delete(metadata, "annotations")
	}
}

func setLastAppliedAnnotation(obj map[string]any, value string) {
	metadata, _ := obj["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
		obj["metadata"] = metadata
	}
	annotations, _ := metadata["annotations"].(map[string]any)
	if annotations == nil {
		annotations = map[string]any{}
		metadata["annotations"] = annotations
	}
	annotations[lastAppliedAnnotation] = value
}

func lastAppliedFromLive(live *unstructured.Unstructured) []byte {
	if live == nil {
		return nil
	}
	return []byte(live.GetAnnotations()[lastAppliedAnnotation])
}
