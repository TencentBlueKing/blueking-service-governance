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

// Package gamestatefulset 提供 GameStatefulSet 资源的状态解析能力
package gamestatefulset

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// PartialGameStatefulSetSpec 轻量化的 GameStatefulSet Spec
type PartialGameStatefulSetSpec struct {
	Replicas       *int32 `json:"replicas"`
	UpdateStrategy struct {
		Paused bool `json:"paused"`
	} `json:"updateStrategy"`
}

// PartialGameStatefulSetStatus 轻量化的 GameStatefulSet Status
type PartialGameStatefulSetStatus struct {
	Replicas             int32  `json:"replicas"`
	ReadyReplicas        int32  `json:"readyReplicas"`
	CurrentReplicas      int32  `json:"currentReplicas"`
	UpdatedReplicas      int32  `json:"updatedReplicas"`
	UpdatedReadyReplicas int32  `json:"updatedReadyReplicas"`
	ObservedGeneration   int64  `json:"observedGeneration"`
	CurrentRevision      string `json:"currentRevision"`
	UpdateRevision       string `json:"updateRevision"`
}

// PartialGameStatefulSet 轻量化的 GameStatefulSet
type PartialGameStatefulSet struct {
	Generation int64                        `json:"generation"`
	Spec       PartialGameStatefulSetSpec   `json:"spec"`
	Status     PartialGameStatefulSetStatus `json:"status"`
}

// Parse 解析 GameStatefulSet 的综合状态
//
// 判定规则：
//  1. manifest == nil -> Unknown
//  2. spec.updateStrategy.paused == true -> Suspended
//  3. replicas == 0 且 status.replicas == 0 -> Available（缩容到 0 稳定态）
//  4. observedGeneration == 0 -> Progressing
//  5. observedGeneration < generation -> Progressing
//  6. replicas != status.replicas -> Progressing
//  7. replicas != status.readyReplicas -> Progressing
//  8. replicas != status.updatedReplicas -> Progressing
//  9. RollingUpdate 场景下 currentRevision 与 updateRevision 不一致且 updatedReplicas 未达期望 -> Progressing
//  10. 以上均通过 -> Available
func Parse(manifest map[string]any) (*k8sstatus.Result, error) {
	if manifest == nil {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}, nil
	}

	// 转换为 GameStatefulSet 类型
	gsts := new(PartialGameStatefulSet)
	if err := mapstructure.Decode(manifest, &gsts); err != nil {
		return nil, errors.Wrap(err, "convert unstructured to GameStatefulSet")
	}

	// 更新策略设置为暂停 -> Suspended
	if gsts.Spec.UpdateStrategy.Paused {
		return &k8sstatus.Result{Code: k8sstatus.Suspended, Message: "GameStatefulSet is Suspended"}, nil
	}

	replicas := int32(0)
	if gsts.Spec.Replicas != nil {
		replicas = *gsts.Spec.Replicas
	}

	// 副本数为 0 且实际副本数为 0 -> Available（缩容到 0 稳定态）
	if replicas == 0 && gsts.Status.Replicas == 0 {
		return &k8sstatus.Result{Code: k8sstatus.Available}, nil
	}

	// 使用 switch 判断各种 Progressing 状态
	var progressMsg string
	switch {
	// 观察的生成版本为空
	case gsts.Status.ObservedGeneration == 0:
		progressMsg = "observed generation is empty"
	// 观察的生成版本小于期望的生成版本
	case gsts.Status.ObservedGeneration < gsts.Generation:
		progressMsg = "observed generation less than desired"
	// 副本数不等于实际副本数
	case replicas != gsts.Status.Replicas:
		progressMsg = "replicas hasn't finished updating..."
	// 副本数不等于就绪副本数
	case replicas != gsts.Status.ReadyReplicas:
		progressMsg = "readyReplicas hasn't finished updating..."
	// 副本数不等于更新副本数
	case replicas != gsts.Status.UpdatedReplicas:
		progressMsg = "updatedReplicas hasn't finished updating..."
	// RollingUpdate 场景下 revision 不一致且 updatedReplicas 未达期望
	case gsts.Status.CurrentRevision != gsts.Status.UpdateRevision &&
		replicas != gsts.Status.UpdatedReplicas:
		progressMsg = "rolling update in progress..."
	}

	// 如果有 Progressing 消息，返回 Progressing 状态
	if progressMsg != "" {
		return &k8sstatus.Result{
			Code:    k8sstatus.Progressing,
			Message: "Waiting for GameStatefulSet to finish: " + progressMsg,
		}, nil
	}

	return &k8sstatus.Result{Code: k8sstatus.Available}, nil
}
