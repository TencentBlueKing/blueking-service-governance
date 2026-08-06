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

// Package pod 提供 Pod 状态解析能力
package pod

import (
	"fmt"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/mitchellh/mapstructure"
	v1 "k8s.io/api/core/v1"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// Parser Pod 状态解析器
type Parser struct {
	Manifest     map[string]any
	initializing bool
	reason       string
}

// NewParser 创建 Pod 状态解析器
func NewParser(manifest map[string]any) *Parser {
	return &Parser{Manifest: manifest, initializing: false}
}

// Parse 状态解析
// Refs：https://github.com/kubernetes/dashboard/blob/49406263/modules/api/pkg/resource/pod/common.go#L43
func (p *Parser) Parse() *k8sstatus.Result {
	// 构造轻量化的 PodStatus 用于解析 Pod Status（total）字段
	podStatus := PartialPodStatus{}
	if err := mapstructure.Decode(p.Manifest["status"], &podStatus); err != nil {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}
	}

	// 1. 默认使用 Pod.Status.Phase
	p.reason = string(podStatus.Phase)

	// 2. 若有具体的 Pod.Status.Reason 则使用
	if podStatus.Reason != "" {
		p.reason = podStatus.Reason
	}

	// 3. 根据 Pod 容器状态更新状态
	p.updateByInitContainerStatuses(&podStatus)
	if !p.initializing || hasPodInitializedCondition(podStatus.Conditions) {
		p.updateByContainerStatuses(&podStatus)
	}

	// 4. 根据 Pod.Metadata.DeletionTimestamp 更新状态
	deletionTimestamp, _ := mapx.GetItems(p.Manifest, "metadata.deletionTimestamp")
	if deletionTimestamp != nil && podStatus.Reason == "NodeLost" {
		p.reason = string(v1.PodUnknown)
	} else if deletionTimestamp != nil && !isPodPhaseTerminal(podStatus.Phase) {
		p.reason = "Terminating"
	}

	// 5. 若状态未初始化或在转移中丢失，则标记为未知状态
	if p.reason == "" {
		p.reason = string(v1.PodUnknown)
	}
	return &k8sstatus.Result{Code: p.reason}
}

// updateByInitContainerStatuses 根据 pod.Status.InitContainerStatuses 更新状态
func (p *Parser) updateByInitContainerStatuses(podStatus *PartialPodStatus) {
	// 指定名称的 init 容器重启策略是否为 always 映射表
	isInitContainerAlwaysRestartPolicyMap := map[string]bool{}
	for _, c := range mapx.GetList(p.Manifest, "spec.initContainers") {
		if v, ok := c.(map[string]any); ok {
			name := mapx.GetStr(v, "name")
			flag := mapx.GetStr(v, "restartPolicy") == "Always"
			isInitContainerAlwaysRestartPolicyMap[name] = flag
		}
	}

	for i := range podStatus.InitContainerStatuses {
		c := podStatus.InitContainerStatuses[i]
		switch {
		case c.State.Terminated != nil && c.State.Terminated.ExitCode == 0:
			continue
		case isInitContainerAlwaysRestartPolicyMap[c.Name] && c.State.Running != nil:
			continue
		case c.State.Terminated != nil:
			if c.State.Terminated.Reason != "" {
				p.reason = "Init: " + c.State.Terminated.Reason
			} else if c.State.Terminated.Signal != 0 {
				p.reason = fmt.Sprintf("Init: Signal %d", c.State.Terminated.Signal)
			} else {
				p.reason = fmt.Sprintf("Init: ExitCode %d", c.State.Terminated.ExitCode)
			}
			p.initializing = true
		case c.State.Waiting != nil && c.State.Waiting.Reason != "" && c.State.Waiting.Reason != "PodInitializing":
			p.reason = fmt.Sprintf("Init: %s", c.State.Waiting.Reason)
			p.initializing = true
		default:
			initContainers := mapx.GetList(p.Manifest, "spec.initContainers")
			p.reason = fmt.Sprintf("Init: %d/%d", i, len(initContainers))
			p.initializing = true
		}
		break
	}
}

func (p *Parser) updateByContainerStatuses(podStatus *PartialPodStatus) {
	hasRunning := false
	for i := len(podStatus.ContainerStatuses) - 1; i >= 0; i-- {
		c := podStatus.ContainerStatuses[i]
		switch {
		case c.State.Waiting != nil && c.State.Waiting.Reason != "":
			p.reason = c.State.Waiting.Reason
		case c.State.Terminated != nil:
			if c.State.Terminated.Reason != "" {
				p.reason = c.State.Terminated.Reason
			} else if c.State.Terminated.Signal != 0 {
				p.reason = fmt.Sprintf("Signal: %d", c.State.Terminated.Signal)
			} else {
				p.reason = fmt.Sprintf("ExitCode: %d", c.State.Terminated.ExitCode)
			}
		case c.Ready && c.State.Running != nil:
			hasRunning = true
		}
	}
	if p.reason == "Completed" && hasRunning {
		if hasPodReadyCondition(podStatus.Conditions) {
			p.reason = string(v1.PodRunning)
		} else {
			p.reason = "NotReady"
		}
	}
}

// hasPodInitializedCondition 判断 pod 是否 initialized
func hasPodInitializedCondition(conditions []PartialPodCondition) bool {
	for _, condition := range conditions {
		if condition.Type != v1.PodInitialized {
			continue
		}

		return condition.Status == v1.ConditionTrue
	}
	return false
}

// hasPodReadyCondition 判断 pod 是否 ready
func hasPodReadyCondition(conditions []PartialPodCondition) bool {
	for _, condition := range conditions {
		if condition.Type != v1.PodReady {
			continue
		}

		return condition.Status == v1.ConditionTrue
	}
	return false
}

// isPodPhaseTerminal 判断 pod 是否终止
func isPodPhaseTerminal(phase v1.PodPhase) bool {
	return phase == v1.PodFailed || phase == v1.PodSucceeded
}
