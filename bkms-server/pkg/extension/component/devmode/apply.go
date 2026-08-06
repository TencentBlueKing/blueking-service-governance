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

// Package devmode 提供开发模式组件支持
package devmode

import (
	"encoding/json"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// PatchGameDeployment 将开发模式组件通过 Strategic Merge Patch 应用到 GameDeployment。
// 返回修改后的 GameDeployment、额外资源对象（ConfigMap）和错误信息。
func PatchGameDeployment(
	gd tkex.GameDeployment,
	config *Config,
) (tkex.GameDeployment, []unstructured.Unstructured, error) {
	extraObjs := make([]unstructured.Unstructured, 0)

	if !config.Enabled {
		return gd, extraObjs, nil
	}

	// 避免与 builder 同名
	build := New(config)
	if err := build.Validate(); err != nil {
		return gd, nil, err
	}
	if !build.IsAllowed() {
		return gd, nil, errors.Errorf("dev mode is not allowed in %s environment", config.EnvType)
	}

	output, err := build.Build()
	if err != nil {
		return gd, nil, errors.Wrap(err, "failed to build dev mode component")
	}
	if output == nil {
		return gd, nil, nil
	}

	// 构建并应用 patch
	patch := buildDevModePatch(output)

	gdBytes, err := json.Marshal(gd)
	if err != nil {
		return gd, nil, errors.Wrap(err, "failed to marshal GameDeployment")
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return gd, nil, errors.Wrap(err, "failed to marshal patch")
	}

	merged, err := strategicpatch.StrategicMergePatch(gdBytes, patchBytes, gd)
	if err != nil {
		return gd, nil, errors.Wrap(err, "failed to apply patch")
	}

	var result tkex.GameDeployment
	if err = json.Unmarshal(merged, &result); err != nil {
		return gd, nil, errors.Wrap(err, "failed to unmarshal GameDeployment")
	}

	if output.ConfigMap != nil {
		configMapObj, err := toUnstructured(output.ConfigMap)
		if err != nil {
			return gd, nil, errors.Wrap(err, "failed to convert ConfigMap to unstructured")
		}
		extraObjs = append(extraObjs, configMapObj)
	}

	return result, extraObjs, nil
}

// buildDevModePatch 构建 patch：添加 Volume、VolumeMount，替换主容器启动命令
func buildDevModePatch(output *Output) map[string]interface{} {
	return map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"volumes": []map[string]interface{}{
						{
							"name": output.Volume.Name,
							"configMap": map[string]any{
								"name": output.ConfigMap.Name,
								// 0755
								"defaultMode": 493,
							},
						},
					},
					"containers": []map[string]any{
						{
							"name":    defaults.WorkloadMainContainerName,
							"command": output.Command,
							// 手动置空 args
							"args": []string{},
							"volumeMounts": []map[string]any{
								{
									"name":      output.VolumeMount.Name,
									"mountPath": output.VolumeMount.MountPath,
								},
							},
						},
					},
				},
			},
		},
	}
}

// toUnstructured 将对象转换为 unstructured.Unstructured
func toUnstructured(obj any) (unstructured.Unstructured, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	var result map[string]any
	if err = json.Unmarshal(data, &result); err != nil {
		return unstructured.Unstructured{}, err
	}

	return unstructured.Unstructured{Object: result}, nil
}
