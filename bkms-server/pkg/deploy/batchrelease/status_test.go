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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
)

const (
	oldRevision = "test-gamedeploy-7b697647cb"
	newRevision = "test-gamedeploy-d8db7cf84"
)

// 以下工作负载快照取自测试集群实测（4 副本，见 design_notes/batch_release_kourse.md）
var (
	// 新版本已全部更新就绪
	updatedWorkload = batchrelease.WorkloadSnapshot{
		Generation: 14, ObservedGeneration: 14, UpdateRevision: newRevision, CurrentRevision: newRevision,
		DesiredReplicas: 4, Replicas: 4, ReadyReplicas: 4, UpdatedReplicas: 4, UpdatedReadyReplicas: 4,
	}
	// 第 0 批暂停中：1 个实例为新版本
	firstBatchWorkload = batchrelease.WorkloadSnapshot{
		Generation: 11, ObservedGeneration: 11, UpdateRevision: newRevision, CurrentRevision: oldRevision,
		DesiredReplicas: 4, Replicas: 4, ReadyReplicas: 4, UpdatedReplicas: 1, UpdatedReadyReplicas: 1,
	}
	// 回滚后新版本实例已清空，但被回滚的实例尚未就绪
	rollingBackWorkload = batchrelease.WorkloadSnapshot{
		Generation: 12, ObservedGeneration: 12, UpdateRevision: newRevision, CurrentRevision: oldRevision,
		DesiredReplicas: 4, Replicas: 4, ReadyReplicas: 3,
	}
	// 回滚完成
	rolledBackWorkload = batchrelease.WorkloadSnapshot{
		Generation: 12, ObservedGeneration: 12, UpdateRevision: newRevision, CurrentRevision: oldRevision,
		DesiredReplicas: 4, Replicas: 4, ReadyReplicas: 4,
	}
)

var _ = Describe("ParseStatus", func() {
	It("parses all progress related fields", func() {
		status, err := batchrelease.ParseStatus(map[string]any{
			"status": map[string]any{
				"phase":            "Running",
				"message":          "Running canary step 1",
				"paused":           "true",
				"workloadRevision": newRevision,
				"rolloutRevision":  "565bf4d5c7",
				"stepStatus": map[string]any{
					"currentStepIndex": int64(1),
					"currentStepPhase": "PausedPermanently",
					"message":          "Paused permanently",
					"pauseStartTime":   "2026-09-28T08:00:00Z",
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		pauseStartTime := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
		Expect(status).To(Equal(batchrelease.Status{
			Phase:            "Running",
			Message:          "Running canary step 1",
			Paused:           "true",
			WorkloadRevision: newRevision,
			RolloutRevision:  "565bf4d5c7",
			CurrentStepIndex: lo.ToPtr(int64(1)),
			CurrentStepPhase: "PausedPermanently",
			StepMessage:      "Paused permanently",
			PauseStartTime:   &pauseStartTime,
		}))
		Expect(status.BatchPhase(firstBatchWorkload)).To(Equal(batchrelease.BatchPhaseManualPaused))
	})

	It("distinguishes step index 0 from a missing step index", func() {
		status, err := batchrelease.ParseStatus(map[string]any{
			"status": map[string]any{"stepStatus": map[string]any{"currentStepIndex": int64(0)}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(status.CurrentStepIndex).To(Equal(lo.ToPtr(int64(0))))

		// 所有步骤完成后 Kourse 把 stepStatus 清成 {}
		status, err = batchrelease.ParseStatus(map[string]any{
			"status": map[string]any{"phase": "Healthy", "stepStatus": map[string]any{}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(status.CurrentStepIndex).To(BeNil())
	})

	DescribeTable("returns zero values without error when fields are missing",
		func(rollout map[string]any) {
			status, err := batchrelease.ParseStatus(rollout)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(batchrelease.Status{}))
			Expect(status.BatchPhase(batchrelease.WorkloadSnapshot{})).To(Equal(batchrelease.BatchPhasePending))
		},
		Entry("no status", map[string]any{"spec": map[string]any{}}),
		Entry("empty status", map[string]any{"status": map[string]any{}}),
		Entry("empty step status", map[string]any{"status": map[string]any{"stepStatus": map[string]any{}}}),
	)

	DescribeTable("returns an error when field types mismatch",
		func(status map[string]any) {
			_, err := batchrelease.ParseStatus(map[string]any{"status": status})
			Expect(err).To(HaveOccurred())
		},
		Entry("non string phase", map[string]any{"phase": int64(1)}),
		Entry("non string workload revision", map[string]any{"workloadRevision": int64(1)}),
		Entry("non string rollout revision", map[string]any{"rolloutRevision": int64(1)}),
		Entry("invalid pause start time", map[string]any{"stepStatus": map[string]any{"pauseStartTime": "yesterday"}}),
		Entry("float step index", map[string]any{"stepStatus": map[string]any{"currentStepIndex": float64(-1)}}),
		Entry("null step index", map[string]any{"stepStatus": map[string]any{"currentStepIndex": nil}}),
	)
})

var _ = Describe("Status.BatchPhase", func() {
	DescribeTable("maps rollout status to batch phase",
		func(status batchrelease.Status, workload batchrelease.WorkloadSnapshot, expected batchrelease.BatchPhase) {
			Expect(status.BatchPhase(workload)).To(Equal(expected))
		},
		Entry("scaling replicas",
			batchrelease.Status{
				Phase: "Running", Paused: "false", CurrentStepIndex: lo.ToPtr(int64(0)),
				CurrentStepPhase: "ScalingReplicas",
			},
			firstBatchWorkload, batchrelease.BatchPhaseUpdating),
		Entry(
			"running hook",
			batchrelease.Status{
				Phase:            "Running",
				CurrentStepIndex: lo.ToPtr(int64(0)),
				CurrentStepPhase: "HookRunning",
			},
			firstBatchWorkload,
			batchrelease.BatchPhaseUpdating,
		),
		Entry("step error",
			batchrelease.Status{Phase: "Running", CurrentStepIndex: lo.ToPtr(int64(0)), CurrentStepPhase: "Error"},
			firstBatchWorkload, batchrelease.BatchPhaseUpdating),
		Entry("paused temporarily",
			batchrelease.Status{
				Phase: "Running", CurrentStepIndex: lo.ToPtr(int64(0)), CurrentStepPhase: "PausedTemporarily",
			},
			firstBatchWorkload, batchrelease.BatchPhaseTimedPaused),
		Entry("temporary paused flag",
			batchrelease.Status{Phase: "Running", CurrentStepIndex: lo.ToPtr(int64(0)), Paused: "temporary"},
			firstBatchWorkload, batchrelease.BatchPhaseTimedPaused),
		Entry("paused permanently",
			batchrelease.Status{
				Phase: "Running", CurrentStepIndex: lo.ToPtr(int64(0)), CurrentStepPhase: "PausedPermanently",
			},
			firstBatchWorkload, batchrelease.BatchPhaseManualPaused),
		Entry("paused by user while scaling",
			batchrelease.Status{
				Phase: "Running", Paused: "true", CurrentStepIndex: lo.ToPtr(int64(0)),
				CurrentStepPhase: "ScalingReplicas",
			},
			firstBatchWorkload, batchrelease.BatchPhaseManualPaused),
		Entry("paused by user during a timed pause",
			batchrelease.Status{
				Phase: "Running", Paused: "true", CurrentStepIndex: lo.ToPtr(int64(0)),
				CurrentStepPhase: "PausedTemporarily",
			},
			firstBatchWorkload, batchrelease.BatchPhaseManualPaused),
		Entry("frozen manually",
			batchrelease.Status{
				Phase: "Running", Paused: "manually", CurrentStepIndex: lo.ToPtr(int64(0)),
				CurrentStepPhase: "PausedManually",
			},
			firstBatchWorkload, batchrelease.BatchPhaseManualPaused),
		Entry("between two steps",
			batchrelease.Status{Phase: "Running", CurrentStepIndex: lo.ToPtr(int64(1))},
			firstBatchWorkload, batchrelease.BatchPhasePending),
		Entry("rolling back with the step phase left over from the pause",
			batchrelease.Status{
				Phase: "RollbackCompleted", Paused: "false", WorkloadRevision: newRevision,
				CurrentStepIndex: lo.ToPtr(int64(-1)), CurrentStepPhase: "PausedPermanently",
			},
			rollingBackWorkload, batchrelease.BatchPhaseUpdating),
		Entry("rolled back",
			batchrelease.Status{
				Phase: "RollbackCompleted", Paused: "false", WorkloadRevision: newRevision,
				CurrentStepIndex: lo.ToPtr(int64(-1)), CurrentStepPhase: "PausedPermanently",
			},
			rolledBackWorkload, batchrelease.BatchPhaseUpdating),
		Entry("cleaning previous canary resources when the release starts",
			batchrelease.Status{Phase: "CleaningCanary", WorkloadRevision: oldRevision},
			firstBatchWorkload, batchrelease.BatchPhaseUpdating),
		Entry("updating primary",
			batchrelease.Status{Phase: "UpdatingPrimary", WorkloadRevision: newRevision},
			firstBatchWorkload, batchrelease.BatchPhaseUpdating),
		Entry("cleaning canary resources when the release ends",
			batchrelease.Status{Phase: "CleaningCanary", WorkloadRevision: newRevision},
			updatedWorkload, batchrelease.BatchPhaseUpdating),
		Entry("healthy after all instances are updated",
			batchrelease.Status{Phase: "Healthy", Paused: "false", WorkloadRevision: newRevision},
			updatedWorkload, batchrelease.BatchPhaseCompleted),
		Entry("healthy before the rollout notices the workload change",
			batchrelease.Status{Phase: "Healthy", WorkloadRevision: oldRevision},
			firstBatchWorkload, batchrelease.BatchPhasePending),
		Entry("healthy while the workload has not observed the latest spec",
			batchrelease.Status{Phase: "Healthy", WorkloadRevision: newRevision},
			func() batchrelease.WorkloadSnapshot {
				w := updatedWorkload
				w.Generation++
				return w
			}(),
			batchrelease.BatchPhasePending),
		Entry("healthy while instances are not all updated",
			batchrelease.Status{Phase: "Healthy", WorkloadRevision: newRevision},
			firstBatchWorkload, batchrelease.BatchPhasePending),
		Entry("healthy without workload revision",
			batchrelease.Status{Phase: "Healthy"},
			batchrelease.WorkloadSnapshot{}, batchrelease.BatchPhasePending),
	)

	It("reports paused phases", func() {
		Expect(batchrelease.BatchPhaseManualPaused.IsPaused()).To(BeTrue())
		Expect(batchrelease.BatchPhaseTimedPaused.IsPaused()).To(BeTrue())
		Expect(batchrelease.BatchPhaseUpdating.IsPaused()).To(BeFalse())
		Expect(batchrelease.BatchPhasePending.IsPaused()).To(BeFalse())
		Expect(batchrelease.BatchPhaseCompleted.IsPaused()).To(BeFalse())
	})
})

var _ = Describe("Status.IsRolledBack", func() {
	rollbackStatus := batchrelease.Status{
		Phase: "RollbackCompleted", Paused: "false", WorkloadRevision: newRevision,
		CurrentStepIndex: lo.ToPtr(int64(-1)), CurrentStepPhase: "PausedPermanently",
		Message: "Rollout has been rolled back successfully",
	}

	DescribeTable("confirms the rollback with the workload",
		func(status batchrelease.Status, workload batchrelease.WorkloadSnapshot, expected bool) {
			Expect(status.IsRolledBack(workload)).To(Equal(expected))
		},
		Entry("all instances are back to the old revision and ready", rollbackStatus, rolledBackWorkload, true),
		Entry("rolled back instances are not ready yet", rollbackStatus, rollingBackWorkload, false),
		Entry("new revision instances remain", rollbackStatus, firstBatchWorkload, false),
		Entry("workload has not observed the latest spec", rollbackStatus,
			func() batchrelease.WorkloadSnapshot {
				w := rolledBackWorkload
				w.Generation++
				return w
			}(),
			false),
		Entry("rollout has not handled the rollback yet",
			batchrelease.Status{Phase: "Running", CurrentStepIndex: lo.ToPtr(int64(-1))},
			rolledBackWorkload, false),
		Entry("not rolling back",
			batchrelease.Status{Phase: "RollbackCompleted"},
			rolledBackWorkload, false),
	)
})

var _ = Describe("ParseWorkloadSnapshot", func() {
	It("parses generation, revisions and replicas", func() {
		snapshot, err := batchrelease.ParseWorkloadSnapshot(map[string]any{
			"metadata": map[string]any{"generation": int64(12)},
			"spec":     map[string]any{"replicas": int64(4)},
			"status": map[string]any{
				"observedGeneration":   int64(12),
				"currentRevision":      oldRevision,
				"updateRevision":       newRevision,
				"replicas":             int64(4),
				"readyReplicas":        int64(3),
				"updatedReplicas":      int64(0),
				"updatedReadyReplicas": int64(0),
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(snapshot).To(Equal(rollingBackWorkload))
	})

	It("returns zero values when status is missing", func() {
		snapshot, err := batchrelease.ParseWorkloadSnapshot(map[string]any{
			"metadata": map[string]any{"generation": int64(1)},
			"spec":     map[string]any{"replicas": int64(2)},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(snapshot).To(Equal(batchrelease.WorkloadSnapshot{Generation: 1, DesiredReplicas: 2}))
		Expect(snapshot.IsUpdated()).To(BeFalse())
		Expect(snapshot.IsRolledBack()).To(BeFalse())
	})

	DescribeTable("returns an error when field types mismatch",
		func(workload map[string]any) {
			_, err := batchrelease.ParseWorkloadSnapshot(workload)
			Expect(err).To(HaveOccurred())
		},
		Entry("non string revision", map[string]any{"status": map[string]any{"updateRevision": int64(1)}}),
		Entry("float replicas", map[string]any{"spec": map[string]any{"replicas": float64(4)}}),
	)
})
