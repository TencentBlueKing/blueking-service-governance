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
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// BatchPhase 当前批次状态
type BatchPhase string

const (
	// BatchPhasePending 未开始
	BatchPhasePending BatchPhase = "pending"
	// BatchPhaseUpdating 更新中
	BatchPhaseUpdating BatchPhase = "updating"
	// BatchPhaseTimedPaused 定时暂停中
	BatchPhaseTimedPaused BatchPhase = "timed-paused"
	// BatchPhaseManualPaused 手动暂停中（含永久暂停与人工暂停）
	BatchPhaseManualPaused BatchPhase = "manual-paused"
	// BatchPhaseCompleted 已完成
	BatchPhaseCompleted BatchPhase = "completed"
)

// status.phase 的取值。CRD 未定义枚举，以下取值及流转依据 bcs-kourse-rollout 源码与实测，
// 见 design_notes/batch_release_kourse.md；仅列出不配置 hook 时会出现的取值。
const (
	// RolloutPhaseRunning 正在执行发布步骤
	RolloutPhaseRunning = "Running"
	// RolloutPhaseUpdatingPrimary 所有步骤已完成，正在更新剩余的旧版本实例
	RolloutPhaseUpdatingPrimary = "UpdatingPrimary"
	// RolloutPhaseCleaningCanary 正在清理灰度资源。发布开始（工作负载版本变化后，workloadRevision 仍为旧版本）
	// 与发布结束（所有实例更新完成后）都会出现
	RolloutPhaseCleaningCanary = "CleaningCanary"
	// RolloutPhaseHealthy 没有进行中的发布。Rollout 刚创建、工作负载尚未更新时同样为该值，
	// 判断分批发布完成时还需确认 workloadRevision 即工作负载的目标版本
	RolloutPhaseHealthy = "Healthy"
	// RolloutPhaseRollbackCompleted 批次序号置为 -1 后立即进入该值并保持到下一次发布，
	// 不代表 Pod 已恢复为旧版本
	RolloutPhaseRollbackCompleted = "RollbackCompleted"
)

// status.stepStatus.currentStepPhase 的取值；两个步骤之间（上一步完成、下一步尚未开始）该字段为空。
// 回滚后该字段不会被清理，保留回滚前的值。
const (
	// StepPhaseScalingReplicas 正在更新本批实例并等待就绪
	StepPhaseScalingReplicas = "ScalingReplicas"
	// StepPhaseHookRunning 正在执行 hook 检查（平台不配置 hook，保留以便识别）
	StepPhaseHookRunning = "HookRunning"
	// StepPhasePausedTemporarily 定时暂停中
	StepPhasePausedTemporarily = "PausedTemporarily"
	// StepPhasePausedPermanently 永久暂停中，需人工继续
	StepPhasePausedPermanently = "PausedPermanently"
	// StepPhasePausedManually 人工冻结中（status.paused 为 manually）
	StepPhasePausedManually = "PausedManually"
	// StepPhaseError 本批执行出错，Kourse 会持续重试，原因见 status.stepStatus.message
	StepPhaseError = "Error"
)

// status.paused 的取值（字符串类型）
const (
	// PausedTrue 暂停在当前步骤，原负载扩缩容时灰度副本数随之调整
	PausedTrue = "true"
	// PausedFalse 未暂停；暂停中改为该值即继续发布
	PausedFalse = "false"
	// PausedTemporary 定时暂停中
	PausedTemporary = "temporary"
	// PausedManually 立即冻结所有动作，保持当前状态不变
	PausedManually = "manually"
)

// Status 从 Rollout CR 中解析出的状态；CR 没有 status 或字段缺失时对应字段为零值
type Status struct {
	// Phase Rollout 整体阶段（status.phase）
	Phase string
	// Message Rollout 整体提示（status.message）
	Message string
	// Paused 暂停标记（status.paused）
	Paused string
	// WorkloadRevision Rollout 正在跟踪的工作负载版本（status.workloadRevision），
	// 与 GameDeployment 的 status.updateRevision 格式相同；发布开始时即切换为新版本
	WorkloadRevision string
	// RolloutRevision 控制器已观察到的 Rollout spec 的哈希（status.rolloutRevision），spec 相同则取值相同。
	// Rollout 没有 observedGeneration，修改 spec 后以该值变化判断控制器已看到新 spec
	RolloutRevision string
	// CurrentStepIndex 当前步骤序号，从 0 开始，-1 表示回滚（status.stepStatus.currentStepIndex）。
	// 所有步骤完成后 stepStatus 被清空，此时为 nil
	CurrentStepIndex *int64
	// CurrentStepPhase 当前步骤阶段（status.stepStatus.currentStepPhase）
	CurrentStepPhase string
	// StepMessage 当前步骤提示（status.stepStatus.message）
	StepMessage string
	// PauseStartTime 当前步骤暂停开始时间（status.stepStatus.pauseStartTime）
	PauseStartTime *time.Time
}

// ParseStatus 解析 Rollout CR 的 status；字段缺失时返回零值，字段类型不符时返回错误
func ParseStatus(rollout map[string]any) (Status, error) {
	var status Status
	var err error
	strFields := []struct {
		target *string
		path   []string
	}{
		{&status.Phase, []string{"status", "phase"}},
		{&status.Message, []string{"status", "message"}},
		{&status.Paused, []string{"status", "paused"}},
		{&status.WorkloadRevision, []string{"status", "workloadRevision"}},
		{&status.RolloutRevision, []string{"status", "rolloutRevision"}},
		{&status.CurrentStepPhase, []string{"status", "stepStatus", "currentStepPhase"}},
		{&status.StepMessage, []string{"status", "stepStatus", "message"}},
	}
	for _, field := range strFields {
		if *field.target, _, err = unstructured.NestedString(rollout, field.path...); err != nil {
			return Status{}, errors.Wrap(err, "parse rollout status")
		}
	}

	stepIndex, found, err := unstructured.NestedInt64(rollout, "status", "stepStatus", "currentStepIndex")
	if err != nil {
		return Status{}, errors.Wrap(err, "parse rollout status")
	}
	if found {
		status.CurrentStepIndex = &stepIndex
	}

	pauseStartTime, _, err := unstructured.NestedString(rollout, "status", "stepStatus", "pauseStartTime")
	if err != nil {
		return Status{}, errors.Wrap(err, "parse rollout status")
	}
	if pauseStartTime != "" {
		t, parseErr := time.Parse(time.RFC3339, pauseStartTime)
		if parseErr != nil {
			return Status{}, errors.Wrapf(parseErr, "parse rollout status pauseStartTime %q", pauseStartTime)
		}
		status.PauseStartTime = &t
	}
	return status, nil
}

// BatchPhase 将 Rollout 状态映射为当前批次状态，workload 为 Rollout 所控制的主工作负载的当前状态。
//
// 按以下顺序判断：
//   - 回滚中（批次序号 < 0）视为更新中；回滚后步骤阶段仍保留回滚前的值，必须先于暂停判断
//   - UpdatingPrimary / CleaningCanary 视为更新中
//   - Healthy 仅在 Rollout 跟踪的版本即工作负载目标版本、且所有实例已更新就绪时视为已完成，否则视为未开始
//   - 其余情况人工暂停标记优先于步骤阶段，其次按步骤阶段区分暂停方式；步骤阶段为空表示本批尚未开始
func (s Status) BatchPhase(workload WorkloadSnapshot) BatchPhase {
	if s.isRollingBack() {
		return BatchPhaseUpdating
	}
	switch s.Phase {
	case RolloutPhaseUpdatingPrimary, RolloutPhaseCleaningCanary:
		return BatchPhaseUpdating
	case RolloutPhaseHealthy:
		if s.WorkloadRevision != "" && s.WorkloadRevision == workload.UpdateRevision && workload.IsUpdated() {
			return BatchPhaseCompleted
		}
		return BatchPhasePending
	}
	return s.stepBatchPhase()
}

// IsRolledBack 判断回滚是否已完成：Kourse 在回滚一开始就把 phase 置为 RollbackCompleted，
// 且在 Pod 就绪前就提示回滚成功，因此以工作负载不再有新版本实例、所有实例就绪为准
func (s Status) IsRolledBack(workload WorkloadSnapshot) bool {
	return s.isRollingBack() && s.Phase == RolloutPhaseRollbackCompleted && workload.IsRolledBack()
}

// IsPaused 是否处于暂停中（手动、永久或定时暂停）
func (p BatchPhase) IsPaused() bool {
	return p == BatchPhaseManualPaused || p == BatchPhaseTimedPaused
}

func (s Status) isRollingBack() bool {
	return s.CurrentStepIndex != nil && *s.CurrentStepIndex < 0
}

// stepBatchPhase 执行步骤期间的批次状态
func (s Status) stepBatchPhase() BatchPhase {
	switch {
	case s.Paused == PausedTrue || s.Paused == PausedManually || s.CurrentStepPhase == StepPhasePausedPermanently:
		return BatchPhaseManualPaused
	case s.Paused == PausedTemporary || s.CurrentStepPhase == StepPhasePausedTemporarily:
		return BatchPhaseTimedPaused
	case s.CurrentStepPhase == "":
		return BatchPhasePending
	default:
		return BatchPhaseUpdating
	}
}
