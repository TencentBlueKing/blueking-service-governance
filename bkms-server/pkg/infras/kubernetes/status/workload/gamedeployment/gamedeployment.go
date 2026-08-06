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

// Package gamedeployment 提供 GameDeployment 资源的状态解析能力
package gamedeployment

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// PartialGameDeploymentSpec 轻量化的 GameDeployment Spec
type PartialGameDeploymentSpec struct {
	Replicas       *int32 `json:"replicas"`
	UpdateStrategy struct {
		Paused bool `json:"paused"`
	} `json:"updateStrategy"`
}

// PartialGameDeploymentStatus 轻量化的 GameDeployment Status
type PartialGameDeploymentStatus struct {
	Replicas             int32 `json:"replicas"`
	ReadyReplicas        int32 `json:"readyReplicas"`
	UpdatedReplicas      int32 `json:"updatedReplicas"`
	UpdatedReadyReplicas int32 `json:"updatedReadyReplicas"`
	ObservedGeneration   int64 `json:"observedGeneration"`
}

// PartialGameDeployment 轻量化的 GameDeployment
type PartialGameDeployment struct {
	Generation int64                       `json:"generation"`
	Spec       PartialGameDeploymentSpec   `json:"spec"`
	Status     PartialGameDeploymentStatus `json:"status"`
}

// Parse 解析 GameDeployment 的综合状态
//
// 判定规则：
//  1. manifest == nil -> Unknown
//  2. GVK 不匹配（不是 GameDeployment）-> 返回错误
//  3. 转换为 GameDeployment 类型失败 -> 返回错误
//  4. spec.updateStrategy.paused == true -> Suspended
//  5. replicas == 0 且 status.replicas == 0 -> Available（缩容到 0 稳定态）
//  6. observedGeneration == 0 -> Progressing
//  7. observedGeneration < generation -> Progressing
//  8. replicas != status.replicas -> Progressing
//  9. replicas != status.readyReplicas -> Progressing
//  10. replicas != status.updatedReadyReplicas -> Progressing
//  11. replicas != status.updatedReplicas -> Progressing
//  12. 以上均通过 -> Healthy
func Parse(manifest map[string]any) (*k8sstatus.Result, error) {
	if manifest == nil {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}, nil
	}

	// 转换为 GameDeployment 类型
	gd := new(PartialGameDeployment)
	if err := mapstructure.Decode(manifest, &gd); err != nil {
		return nil, errors.Wrap(err, "convert unstructured to GameDeployment")
	}

	// 更新策略设置为暂停 -> Suspended
	if gd.Spec.UpdateStrategy.Paused {
		return &k8sstatus.Result{Code: k8sstatus.Suspended, Message: "GameDeployment is Suspended"}, nil
	}

	replicas := int32(0)
	if gd.Spec.Replicas != nil {
		replicas = *gd.Spec.Replicas
	}

	// 副本数为 0 且实际副本数为 0 -> Available（缩容到 0 稳定态）
	if replicas == 0 && gd.Status.Replicas == 0 {
		return &k8sstatus.Result{Code: k8sstatus.Available}, nil
	}

	// 使用 switch 判断各种 Progressing 状态
	var progressMsg string
	switch {
	// 观察的生成版本为空
	case gd.Status.ObservedGeneration == 0:
		progressMsg = "observed generation is empty"
	// 观察的生成版本小于期望的生成版本
	case gd.Status.ObservedGeneration < gd.Generation:
		progressMsg = "observed generation less than desired"
	// 副本数不等于实际副本数
	case replicas != gd.Status.Replicas:
		progressMsg = "replicas hasn't finished updating..."
	// 副本数不等于就绪副本数
	case replicas != gd.Status.ReadyReplicas:
		progressMsg = "readyReplicas hasn't finished updating..."
	// 副本数不等于更新就绪副本数
	case replicas != gd.Status.UpdatedReadyReplicas:
		progressMsg = "updatedReadyReplicas hasn't finished updating..."
	// 副本数不等于更新副本数
	case replicas != gd.Status.UpdatedReplicas:
		progressMsg = "updatedReplicas hasn't finished updating..."
	}

	// 如果有 Progressing 消息，返回 Progressing 状态
	if progressMsg != "" {
		return &k8sstatus.Result{
			Code:    k8sstatus.Progressing,
			Message: "Waiting for GameDeployment to finish: " + progressMsg,
		}, nil
	}

	return &k8sstatus.Result{Code: k8sstatus.Healthy}, nil
}
