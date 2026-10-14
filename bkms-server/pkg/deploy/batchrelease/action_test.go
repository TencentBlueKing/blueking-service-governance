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

package batchrelease_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
)

var _ = Describe("Status.ActionPatch", func() {
	const batchCount = 2

	var (
		pausedAtFirst = batchrelease.Status{
			Phase: "Running", Paused: "true", CurrentStepIndex: lo.ToPtr(int64(0)),
			CurrentStepPhase: "PausedPermanently",
		}
		pausedAtSecond = batchrelease.Status{
			Phase: "Running", Paused: "true", CurrentStepIndex: lo.ToPtr(int64(1)),
			CurrentStepPhase: "PausedPermanently",
		}
		frozenAtSecond = batchrelease.Status{
			Phase: "Running", Paused: "manually", CurrentStepIndex: lo.ToPtr(int64(1)),
			CurrentStepPhase: "PausedManually",
		}
		scaling = batchrelease.Status{
			Phase: "Running", Paused: "false", CurrentStepIndex: lo.ToPtr(int64(0)),
			CurrentStepPhase: "ScalingReplicas",
		}
	)

	DescribeTable("builds the status patch when the action is allowed",
		func(status batchrelease.Status, action batchrelease.Action, expected batchrelease.StatusPatch) {
			patch, err := status.ActionPatch(action, batchCount)
			Expect(err).NotTo(HaveOccurred())
			Expect(patch).To(Equal(expected))
		},
		Entry("continue a permanent pause", pausedAtFirst, batchrelease.ActionContinue,
			batchrelease.StatusPatch{Paused: lo.ToPtr(batchrelease.PausedFalse)}),
		Entry("continue a timed pause",
			batchrelease.Status{
				Phase: "Running", Paused: "temporary", CurrentStepIndex: lo.ToPtr(int64(0)),
				CurrentStepPhase: "PausedTemporarily",
			},
			batchrelease.ActionContinue, batchrelease.StatusPatch{Paused: lo.ToPtr(batchrelease.PausedFalse)}),
		Entry("continue a manual freeze", frozenAtSecond, batchrelease.ActionContinue,
			batchrelease.StatusPatch{Paused: lo.ToPtr(batchrelease.PausedFalse)}),
		Entry("pause while scaling", scaling, batchrelease.ActionPause,
			batchrelease.StatusPatch{Paused: lo.ToPtr(batchrelease.PausedTrue)}),
		Entry("pause between two steps",
			batchrelease.Status{Phase: "Running", Paused: "false", CurrentStepIndex: lo.ToPtr(int64(1))},
			batchrelease.ActionPause, batchrelease.StatusPatch{Paused: lo.ToPtr(batchrelease.PausedTrue)}),
		Entry("go back to the previous batch", pausedAtSecond, batchrelease.ActionBackward,
			batchrelease.StatusPatch{
				Paused:           lo.ToPtr(batchrelease.PausedFalse),
				CurrentStepIndex: lo.ToPtr(int64(0)),
			}),
		Entry("go back to the previous batch while frozen manually", frozenAtSecond, batchrelease.ActionBackward,
			batchrelease.StatusPatch{
				Paused:           lo.ToPtr(batchrelease.PausedFalse),
				CurrentStepIndex: lo.ToPtr(int64(0)),
			}),
		Entry("skip the remaining batches", pausedAtFirst, batchrelease.ActionSkip,
			batchrelease.StatusPatch{
				Paused:           lo.ToPtr(batchrelease.PausedFalse),
				CurrentStepIndex: lo.ToPtr(int64(batchCount)),
			}),
		Entry("skip the remaining batches while frozen manually", frozenAtSecond, batchrelease.ActionSkip,
			batchrelease.StatusPatch{
				Paused:           lo.ToPtr(batchrelease.PausedFalse),
				CurrentStepIndex: lo.ToPtr(int64(batchCount)),
			}),
		Entry("roll back while paused", pausedAtFirst, batchrelease.ActionRollback,
			batchrelease.StatusPatch{
				Paused:           lo.ToPtr(batchrelease.PausedFalse),
				CurrentStepIndex: lo.ToPtr(int64(-1)),
			}),
		Entry("roll back while frozen manually", frozenAtSecond, batchrelease.ActionRollback,
			batchrelease.StatusPatch{
				Paused:           lo.ToPtr(batchrelease.PausedFalse),
				CurrentStepIndex: lo.ToPtr(int64(-1)),
			}),
		Entry("roll back while scaling", scaling, batchrelease.ActionRollback,
			batchrelease.StatusPatch{
				Paused:           lo.ToPtr(batchrelease.PausedFalse),
				CurrentStepIndex: lo.ToPtr(int64(-1)),
			}),
	)

	DescribeTable("rejects the action in the current state",
		func(status batchrelease.Status, action batchrelease.Action) {
			_, err := status.ActionPatch(action, batchCount)
			Expect(err).To(MatchError(batchrelease.ErrActionNotAllowed))
		},
		Entry("continue when not paused", scaling, batchrelease.ActionContinue),
		Entry("pause when already paused", pausedAtFirst, batchrelease.ActionPause),
		Entry("pause when frozen manually", frozenAtSecond, batchrelease.ActionPause),
		Entry("go back from the first batch", pausedAtFirst, batchrelease.ActionBackward),
		Entry("unknown action", pausedAtFirst, batchrelease.Action("restart")),
		// 跳过剩余批次后、Kourse 尚未进入 UpdatingPrimary 前
		Entry("step index reaches the batch count",
			batchrelease.Status{Phase: "Running", CurrentStepIndex: lo.ToPtr(int64(batchCount))},
			batchrelease.ActionRollback),
		Entry("step index is missing",
			batchrelease.Status{Phase: "Running"},
			batchrelease.ActionRollback),
		Entry("updating primary",
			batchrelease.Status{Phase: "UpdatingPrimary"},
			batchrelease.ActionRollback),
		Entry("cleaning canary resources",
			batchrelease.Status{Phase: "CleaningCanary", CurrentStepIndex: lo.ToPtr(int64(0))},
			batchrelease.ActionSkip),
		Entry("healthy",
			batchrelease.Status{Phase: "Healthy", Paused: "true"},
			batchrelease.ActionContinue),
		Entry("rolling back",
			batchrelease.Status{
				Phase: "RollbackCompleted", CurrentStepIndex: lo.ToPtr(int64(-1)),
				CurrentStepPhase: "PausedPermanently",
			},
			batchrelease.ActionContinue),
	)
})
