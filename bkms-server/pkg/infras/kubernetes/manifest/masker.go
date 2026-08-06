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

package manifest

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

var workloadPodSpecPaths = map[string][]string{
	k8skind.Po:         {"spec"},
	k8skind.Deploy:     {"spec", "template", "spec"},
	k8skind.RS:         {"spec", "template", "spec"},
	k8skind.STS:        {"spec", "template", "spec"},
	k8skind.DS:         {"spec", "template", "spec"},
	k8skind.Job:        {"spec", "template", "spec"},
	k8skind.GameDeploy: {"spec", "template", "spec"},
	k8skind.GameSTS:    {"spec", "template", "spec"},
	k8skind.CJ:         {"spec", "jobTemplate", "spec", "template", "spec"},
}

// Masker 在 Kubernetes manifest 序列化前屏蔽敏感字段
type Masker struct {
	sensitiveEnvVarValues map[string]string
	maskValue             string
}

// NewMasker 根据敏感环境变量值和 mask 值创建 manifest masker
func NewMasker(sensitiveEnvVarValues map[string]string, maskValue string) *Masker {
	return &Masker{
		sensitiveEnvVarValues: sensitiveEnvVarValues,
		maskValue:             maskValue,
	}
}

// Mask 屏蔽 unstructured object 中的敏感字段。
func (m *Masker) Mask(obj *unstructured.Unstructured) {
	if obj == nil {
		return
	}

	switch obj.GetKind() {
	case k8skind.CM:
		m.maskConfigMapData(obj)
	case k8skind.Secret:
		m.maskSecretData(obj)
	default:
		m.maskEnvVarValues(obj)
	}
}

// maskEnvVarValues 屏蔽 unstructured object 中的敏感环境变量值。
// 匹配规则：key 匹配时进行屏蔽，避免敏感环境变量值暴露
func (m *Masker) maskEnvVarValues(obj *unstructured.Unstructured) {
	if obj == nil {
		return
	}
	if len(m.sensitiveEnvVarValues) == 0 {
		return
	}

	podSpecPath, ok := workloadPodSpecPaths[obj.GetKind()]
	if !ok {
		return
	}
	podSpec, found, _ := unstructured.NestedMap(obj.Object, podSpecPath...)
	if !found {
		return
	}
	m.maskPodSpec(podSpec)
	_ = unstructured.SetNestedMap(obj.Object, podSpec, podSpecPath...)
}

// maskPodSpec 屏蔽 Pod 规范中所有容器（containers、initContainers、ephemeralContainers）的敏感环境变量值。
func (m *Masker) maskPodSpec(podSpec map[string]any) {
	for _, field := range []string{"containers", "initContainers", "ephemeralContainers"} {
		containers, found, _ := unstructured.NestedSlice(podSpec, field)
		if !found {
			continue
		}
		for i, item := range containers {
			container, ok := item.(map[string]any)
			if !ok {
				continue
			}
			m.maskContainer(container)
			containers[i] = container
		}
		_ = unstructured.SetNestedSlice(podSpec, containers, field)
	}
}

// maskContainer 屏蔽单个容器定义中环境变量（env）的敏感值。
// 遍历容器的 env 列表，若环境变量的 name 匹配敏感变量且存在 value 字段，则将 value 替换为 mask 值。
func (m *Masker) maskContainer(container map[string]any) {
	envList, found, _ := unstructured.NestedSlice(container, "env")
	if !found {
		return
	}

	for i, item := range envList {
		envVar, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, ok := envVar["name"].(string)
		if !ok {
			continue
		}
		if _, ok = m.sensitiveEnvVarValues[name]; !ok {
			continue
		}
		if _, ok = envVar["value"]; !ok {
			continue
		}
		envVar["value"] = m.maskValue
		envList[i] = envVar
	}
	_ = unstructured.SetNestedSlice(container, envList, "env")
}

// maskConfigMapData 屏蔽 ConfigMap 中 data 和 binaryData 字段的敏感值。
// 遍历 ConfigMap 的 data/binaryData，若 key 匹配敏感环境变量名，则将其值替换为 mask 值。
func (m *Masker) maskConfigMapData(obj *unstructured.Unstructured) {
	if len(m.sensitiveEnvVarValues) == 0 {
		return
	}
	for _, field := range []string{"data", "binaryData"} {
		data, found, _ := unstructured.NestedMap(obj.Object, field)
		if !found {
			continue
		}
		for key := range data {
			if _, ok := m.sensitiveEnvVarValues[key]; ok {
				data[key] = m.maskValue
			}
		}
		_ = unstructured.SetNestedMap(obj.Object, data, field)
	}
}

// maskSecretData 屏蔽 Secret 中 data 和 stringData 字段的所有值。
// 无论 key 是否匹配敏感变量，Secret 中的所有条目值都会被替换为 mask 值。
func (m *Masker) maskSecretData(obj *unstructured.Unstructured) {
	for _, field := range []string{"data", "stringData"} {
		data, found, _ := unstructured.NestedMap(obj.Object, field)
		if !found {
			continue
		}
		for key := range data {
			data[key] = m.maskValue
		}
		_ = unstructured.SetNestedMap(obj.Object, data, field)
	}
}
