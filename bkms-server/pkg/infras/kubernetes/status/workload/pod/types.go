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

package pod

import corev1 "k8s.io/api/core/v1"

// PartialPodCondition ...
type PartialPodCondition struct {
	Type   corev1.PodConditionType
	Status corev1.ConditionStatus
}

// PartialContainerStateWaiting ...
type PartialContainerStateWaiting struct {
	Reason string
}

// PartialContainerStateRunning ...
type PartialContainerStateRunning struct {
	StartedAt string
}

// PartialContainerStateTerminated ...
type PartialContainerStateTerminated struct {
	ExitCode int32
	Signal   int32
	Reason   string
}

// PartialContainerState ...
type PartialContainerState struct {
	Waiting    *PartialContainerStateWaiting
	Running    *PartialContainerStateRunning
	Terminated *PartialContainerStateTerminated
}

// PartialContainerStatus ...
type PartialContainerStatus struct {
	State PartialContainerState
	Ready bool
	Name  string
}

// PartialPodStatus 轻量化的 PodStatus，用于解析 Pod Status 信息
type PartialPodStatus struct {
	Phase                 corev1.PodPhase
	Conditions            []PartialPodCondition
	Reason                string
	InitContainerStatuses []PartialContainerStatus
	ContainerStatuses     []PartialContainerStatus
}
