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

package batchrelease

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// WorkloadSnapshot Rollout 所控制的主工作负载（GameDeployment）的状态快照。
// Rollout 的 status 只反映控制器的进度，实例是否已真正更新或回滚需以工作负载为准。
type WorkloadSnapshot struct {
	// Generation metadata.generation
	Generation int64
	// ObservedGeneration status.observedGeneration
	ObservedGeneration int64
	// UpdateRevision status.updateRevision，即工作负载当前 spec 对应的版本
	UpdateRevision string
	// CurrentRevision status.currentRevision，即旧版本实例所在的版本
	CurrentRevision string
	// DesiredReplicas spec.replicas
	DesiredReplicas int64
	// Replicas status.replicas
	Replicas int64
	// ReadyReplicas status.readyReplicas
	ReadyReplicas int64
	// UpdatedReplicas status.updatedReplicas，已是 UpdateRevision 的实例数
	UpdatedReplicas int64
	// UpdatedReadyReplicas status.updatedReadyReplicas
	UpdatedReadyReplicas int64
}

// ParseWorkloadSnapshot 从 GameDeployment 中解析状态快照；字段缺失时为零值，字段类型不符时返回错误
func ParseWorkloadSnapshot(workload map[string]any) (WorkloadSnapshot, error) {
	var snapshot WorkloadSnapshot
	var err error
	strFields := []struct {
		target *string
		path   []string
	}{
		{&snapshot.UpdateRevision, []string{"status", "updateRevision"}},
		{&snapshot.CurrentRevision, []string{"status", "currentRevision"}},
	}
	for _, field := range strFields {
		if *field.target, _, err = unstructured.NestedString(workload, field.path...); err != nil {
			return WorkloadSnapshot{}, errors.Wrap(err, "parse workload status")
		}
	}

	intFields := []struct {
		target *int64
		path   []string
	}{
		{&snapshot.Generation, []string{"metadata", "generation"}},
		{&snapshot.ObservedGeneration, []string{"status", "observedGeneration"}},
		{&snapshot.DesiredReplicas, []string{"spec", "replicas"}},
		{&snapshot.Replicas, []string{"status", "replicas"}},
		{&snapshot.ReadyReplicas, []string{"status", "readyReplicas"}},
		{&snapshot.UpdatedReplicas, []string{"status", "updatedReplicas"}},
		{&snapshot.UpdatedReadyReplicas, []string{"status", "updatedReadyReplicas"}},
	}
	for _, field := range intFields {
		if *field.target, _, err = unstructured.NestedInt64(workload, field.path...); err != nil {
			return WorkloadSnapshot{}, errors.Wrap(err, "parse workload status")
		}
	}
	return snapshot, nil
}

// IsUpdated 判断 workload 是否已经更新完毕：所有实例都已是 UpdateRevision 且就绪
func (w WorkloadSnapshot) IsUpdated() bool {
	return w.isGenerationObserved() &&
		w.CurrentRevision == w.UpdateRevision &&
		w.Replicas == w.DesiredReplicas &&
		w.ReadyReplicas == w.DesiredReplicas &&
		w.UpdatedReplicas == w.DesiredReplicas &&
		w.UpdatedReadyReplicas == w.DesiredReplicas
}

// IsRolledBack 判断 workload 是否已经回滚完毕：没有 UpdateRevision 的实例，且所有实例就绪
func (w WorkloadSnapshot) IsRolledBack() bool {
	return w.isGenerationObserved() &&
		w.UpdatedReplicas == 0 &&
		w.Replicas == w.DesiredReplicas &&
		w.ReadyReplicas == w.DesiredReplicas
}

// isGenerationObserved 工作负载控制器已处理最新 spec，否则 status 中的版本与副本数仍是旧 spec 的结果
func (w WorkloadSnapshot) isGenerationObserved() bool {
	return w.ObservedGeneration > 0 && w.ObservedGeneration >= w.Generation
}
