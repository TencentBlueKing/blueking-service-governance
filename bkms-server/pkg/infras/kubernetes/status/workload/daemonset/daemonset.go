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

// Package daemonset 提供 DaemonSet 资源的状态解析能力
package daemonset

import (
	"github.com/TencentBlueKing/gopkg/mapx"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload"
)

// Parse 解析 DaemonSet 的综合状态
//
// 判定规则：
//  1. manifest == nil -> Unknown
//  2. status.conditions 中存在 Degraded=True -> Degraded
//  3. status.observedGeneration 为空或小于 metadata.generation -> Progressing
//  4. status.desiredNumberScheduled 与 currentNumberScheduled/updatedNumberScheduled/numberReady 不一致 -> Progressing
//  5. status.numberUnavailable 大于 0 -> Progressing
//  6. 稳定性检查通过（Pod 数一致 且 generation 追上 且 不可用数为 0）-> Available
//  7. 字段缺失无法可靠判断时 -> Deployed 或 Unknown
func Parse(manifest map[string]any) *k8sstatus.Result {
	if manifest == nil {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}
	}

	// 优先检查 conditions 中的 Degraded 状态
	if result := workload.CheckDegraded(manifest); result != nil {
		return result
	}

	// 检查 observedGeneration 是否追上 metadata.generation
	if !workload.IsGenerationObserved(manifest) {
		return &k8sstatus.Result{
			Code:    k8sstatus.Progressing,
			Message: "observedGeneration has not caught up with metadata.generation",
		}
	}

	// 检查 Pod 数一致性
	if consistent, reason := arePodsConsistent(manifest); !consistent {
		return &k8sstatus.Result{Code: k8sstatus.Progressing, Message: "pods are not consistent: " + reason}
	}

	// 检查不可用数
	numberUnavailable := mapx.GetInt64(manifest, "status.numberUnavailable")
	if numberUnavailable > 0 {
		return &k8sstatus.Result{Code: k8sstatus.Progressing, Message: "some pods are unavailable"}
	}

	return &k8sstatus.Result{Code: k8sstatus.Available}
}

// arePodsConsistent 检查 Pod 数一致性
// 返回值：是否一致，以及不一致时的具体原因
func arePodsConsistent(manifest map[string]any) (bool, string) {
	desired := mapx.GetInt64(manifest, "status.desiredNumberScheduled")
	current := mapx.GetInt64(manifest, "status.currentNumberScheduled")
	updated := mapx.GetInt64(manifest, "status.updatedNumberScheduled")
	ready := mapx.GetInt64(manifest, "status.numberReady")

	if desired != current {
		return false, "desiredNumberScheduled != currentNumberScheduled"
	}
	if desired != updated {
		return false, "desiredNumberScheduled != updatedNumberScheduled"
	}
	if desired != ready {
		return false, "desiredNumberScheduled != numberReady"
	}

	return true, ""
}
