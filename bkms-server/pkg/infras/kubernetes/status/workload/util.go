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

// Package workload 提供 Kubernetes 工作负载资源的公共工具函数
package workload

import (
	"github.com/TencentBlueKing/gopkg/mapx"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// CombineMessage 将 reason 和 message 合并为一条消息
func CombineMessage(reason, message string) string {
	if reason != "" && message != "" {
		return reason + ": " + message
	}
	if reason != "" {
		return reason
	}
	return message
}

// GetCondition 从 manifest 的 status.conditions 中获取指定类型的 condition
func GetCondition(manifest map[string]any, condType string) map[string]any {
	conditions := mapx.GetList(manifest, "status.conditions")
	for _, c := range conditions {
		cond, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if mapx.GetStr(cond, "type") == condType {
			return cond
		}
	}
	return nil
}

// IsGenerationObserved 检查 observedGeneration 是否追上 metadata.generation
func IsGenerationObserved(manifest map[string]any) bool {
	observedGen := mapx.GetInt64(manifest, "status.observedGeneration")
	metadataGen := mapx.GetInt64(manifest, "metadata.generation")
	if observedGen == 0 || metadataGen == 0 {
		return false
	}
	return observedGen >= metadataGen
}

// CheckDegraded 检查 conditions 中是否存在 Degraded=True 的 condition
// 适用于 DaemonSet、StatefulSet 等简单 Degraded 检查场景
func CheckDegraded(manifest map[string]any) *k8sstatus.Result {
	cond := GetCondition(manifest, "Degraded")
	if cond != nil && mapx.GetStr(cond, "status") == "True" {
		reason := mapx.GetStr(cond, "reason")
		msg := mapx.GetStr(cond, "message")
		return &k8sstatus.Result{Code: k8sstatus.Degraded, Message: CombineMessage(reason, msg)}
	}
	return nil
}

// CheckDeploymentDegraded 检查 Deployment 的 Degraded 状态
// Deployment 的 Degraded 判定逻辑与 DaemonSet/StatefulSet 不同：
// 1. 先检查 ReplicaFailure=True（有失败的 pod 通常意味着 Degraded）
// 2. 再检查 Progressing.Status=False 且 Reason=ProgressDeadlineExceeded
func CheckDeploymentDegraded(manifest map[string]any) *k8sstatus.Result {
	// 先检查 ReplicaFailure（有失败的 pod 通常意味着 Degraded）
	replicaFailureCond := GetCondition(manifest, "ReplicaFailure")
	if replicaFailureCond != nil && mapx.GetStr(replicaFailureCond, "status") == "True" {
		reason := mapx.GetStr(replicaFailureCond, "reason")
		msg := mapx.GetStr(replicaFailureCond, "message")
		return &k8sstatus.Result{Code: k8sstatus.Degraded, Message: CombineMessage(reason, msg)}
	}

	// 检查 Progressing condition 是否为 Degraded（Progressing.Status=False 且 Reason=ProgressDeadlineExceeded）
	progressingCond := GetCondition(manifest, "Progressing")
	if progressingCond != nil && mapx.GetStr(progressingCond, "status") != "True" {
		reason := mapx.GetStr(progressingCond, "reason")
		if reason == "ProgressDeadlineExceeded" {
			msg := mapx.GetStr(progressingCond, "message")
			return &k8sstatus.Result{Code: k8sstatus.Degraded, Message: CombineMessage(reason, msg)}
		}
	}

	return nil
}

// AreReplicasConsistent 检查 spec.replicas 与 status 中的各副本数字段是否一致
// extraStatusFields 为额外需要与 spec.replicas 比较的 status 字段路径（如 "status.availableReplicas"）
// 返回值：是否一致，以及不一致时的具体原因
func AreReplicasConsistent(manifest map[string]any, extraStatusFields ...string) (bool, string) {
	specReplicas := mapx.GetInt64(manifest, "spec.replicas")
	statusReplicas := mapx.GetInt64(manifest, "status.replicas")

	// spec.replicas == 0 且 status.replicas == 0 视为稳定态
	if specReplicas == 0 && statusReplicas == 0 {
		return true, ""
	}

	if specReplicas != statusReplicas {
		return false, "spec.replicas != status.replicas"
	}

	// 比较基础字段
	for _, field := range []string{"status.updatedReplicas", "status.readyReplicas"} {
		if specReplicas != mapx.GetInt64(manifest, field) {
			return false, "spec.replicas != " + field
		}
	}

	// 比较额外字段
	for _, field := range extraStatusFields {
		if specReplicas != mapx.GetInt64(manifest, field) {
			return false, "spec.replicas != " + field
		}
	}

	return true, ""
}
