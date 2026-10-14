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
	"strconv"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

var (
	// ErrBatchReleaseInProgress 分批发布进行中，拒绝新部署、下架、实例灰度更新、删除实例、手动扩缩容等冲突操作
	ErrBatchReleaseInProgress = errors.New("batch release is in progress, complete, skip or roll it back first")
	// ErrNoBatchReleaseInProgress 当前没有进行中的分批发布，无法执行手动操作
	ErrNoBatchReleaseInProgress = errors.New("no batch release is in progress")
	// ErrActionNotAllowed 当前批次状态不满足手动操作的前置条件
	ErrActionNotAllowed = errors.New("batch release action is not allowed in current state")
)

// Action 分批发布手动操作类型
type Action string

const (
	// ActionContinue 继续发布：暂停中（手动 / 永久 / 定时）时 status.paused 置为 "false"
	ActionContinue Action = "continue"
	// ActionPause 暂停：批次执行中且未暂停时 status.paused 置为 "true"
	ActionPause Action = "pause"
	// ActionBackward 回退上一批：当前批次序号 ≥ 1 时 currentStepIndex 置为当前值 - 1，同时将 status.paused 置为 "false"
	ActionBackward Action = "backward"
	// ActionSkip 跳过剩余批次：currentStepIndex 置为批次总数、status.paused 置为 "false"，所有实例更新为新版本
	ActionSkip Action = "skip"
	// ActionRollback 回滚：currentStepIndex 置为 -1、status.paused 置为 "false"，Kourse 完全回滚。
	// 注意工作负载 spec 仍为新版本，只有 Pod 恢复为旧版本。
	ActionRollback Action = "rollback"
)

// ActionPatch 校验手动操作的前置条件，返回执行该操作需要对 Rollout status 做的修改；
// batchCount 为本次发布的批次总数。不满足前置条件时返回 ErrActionNotAllowed。
//
// 所有操作都要求 Rollout 处于 Running 且批次序号在 [0, batchCount) 内：进入 UpdatingPrimary 后
// Kourse 不再处理批次序号，此时修改会被静默忽略。
// backward / skip / rollback 同时将 status.paused 置为 "false"(用户执行这种操作时，认为需要忽略暂停)
func (s Status) ActionPatch(action Action, batchCount int) (StatusPatch, error) {
	if s.Phase != RolloutPhaseRunning || s.CurrentStepIndex == nil ||
		*s.CurrentStepIndex < 0 || *s.CurrentStepIndex >= int64(batchCount) {
		return StatusPatch{}, s.actionNotAllowed(action)
	}
	stepIndex := *s.CurrentStepIndex
	paused := s.stepBatchPhase().IsPaused()

	switch action {
	case ActionContinue:
		if paused {
			return StatusPatch{Paused: lo.ToPtr(PausedFalse)}, nil
		}
	case ActionPause:
		if !paused {
			return StatusPatch{Paused: lo.ToPtr(PausedTrue)}, nil
		}
	case ActionBackward:
		if stepIndex >= 1 {
			return StatusPatch{
				Paused:           lo.ToPtr(PausedFalse),
				CurrentStepIndex: lo.ToPtr(stepIndex - 1),
			}, nil
		}
	case ActionSkip:
		return StatusPatch{
			Paused:           lo.ToPtr(PausedFalse),
			CurrentStepIndex: lo.ToPtr(int64(batchCount)),
		}, nil
	case ActionRollback:
		return StatusPatch{
			Paused:           lo.ToPtr(PausedFalse),
			CurrentStepIndex: lo.ToPtr(int64(-1)),
		}, nil
	default:
		return StatusPatch{}, errors.Wrapf(ErrActionNotAllowed, "unknown action %q", action)
	}
	return StatusPatch{}, s.actionNotAllowed(action)
}

func (s Status) actionNotAllowed(action Action) error {
	stepIndex := "none"
	if s.CurrentStepIndex != nil {
		stepIndex = strconv.FormatInt(*s.CurrentStepIndex, 10)
	}
	return errors.Wrapf(ErrActionNotAllowed, "action %s: rollout phase %q, step index %s, step phase %q, paused %q",
		action, s.Phase, stepIndex, s.CurrentStepPhase, s.Paused)
}
