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

package appmodeldeploypoll

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

func newDeployingRecord(appID string) *appmodeldeploy.Record {
	return &appmodeldeploy.Record{
		ID:          bson.NewObjectID(),
		WorkspaceID: "ws-1",
		AppID:       appID,
		EnvName:     "dev",
		ClusterID:   "BCS-K8S-1",
		Namespace:   "default",
		Status:      appmodeldeploy.StatusDeploying,
		StartedAt:   time.Now(),
		Creator:     "alice",
	}
}

func stubFetch(status appmodeldeploy.Status) {
	mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(
		&appmodeldeploy.DeployState{Status: status, Message: string(status)}, nil,
	).Build()
}

var _ = Describe("Poller Handle", func() {
	var (
		ctx       context.Context
		args      Args
		rec       *appmodeldeploy.Record
		db        *mongo.Database
		records   *appmodeldeploy.RecordStore
		enqCount  int
		topoCount atomic.Int32
		hookCount atomic.Int32
		plr       *Poller
	)

	insert := func() {
		Expect(records.Create(ctx, rec)).To(Succeed())
		args.DeployID = rec.ID.Hex()
	}

	reload := func() *appmodeldeploy.Record {
		got, err := appmodeldeploy.GetRecord(ctx, db, rec.AppID, args.DeployID)
		Expect(err).NotTo(HaveOccurred())
		return got
	}

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		records = store.Of(db, appmodeldeploy.RecordMeta)

		appID := "app-" + stringx.Random(8)
		rec = newDeployingRecord(appID)
		args = Args{WorkspaceID: rec.WorkspaceID, AppID: rec.AppID, EnvName: rec.EnvName, DeployID: rec.ID.Hex()}
		enqCount = 0
		topoCount.Store(0)
		hookCount.Store(0)
		plr = NewPoller(db)

		mockey.Mock(taskq.Enqueue).To(func(context.Context, *taskq.Task, ...asynq.Option) error {
			enqCount++
			return nil
		}).Build()
		mockey.Mock(audit.AddOperationRecordAsync).Return().Build()
		mockey.Mock(triggerTopologyRefresh).To(func(context.Context, Args, *appmodeldeploy.Record) {
			topoCount.Add(1)
		}).Build()
		mockey.Mock(handleDeploySucceeded).To(func(context.Context, Args, *appmodeldeploy.Record) {
			hookCount.Add(1)
		}).Build()
	})

	AfterEach(func() {
		time.Sleep(50 * time.Millisecond)
		mockey.UnPatchAll()
	})

	It("enqueues the next tick when deploy is still running", func() {
		insert()
		stubFetch(appmodeldeploy.StatusDeploying)
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(1))
		Expect(reload().Status).To(Equal(appmodeldeploy.StatusDeploying))
	})

	It("finishes already stable records without polling", func() {
		rec.Status = appmodeldeploy.StatusDeployed
		insert()
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(0))
		Expect(hookCount.Load()).To(Equal(int32(1)))
	})

	It("skips uninstalling records without overwriting", func() {
		rec.Status = appmodeldeploy.StatusUninstalling
		insert()
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(0))
		Expect(reload().Status).To(Equal(appmodeldeploy.StatusUninstalling))
	})

	It("marks pollingTimeout when the window is exceeded and deploy is still running", func() {
		rec.StartedAt = time.Now().Add(-pollingTimeout() - time.Minute)
		insert()
		stubFetch(appmodeldeploy.StatusDeploying)
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(reload().Status).To(Equal(appmodeldeploy.StatusPollingTimeout))
		Expect(enqCount).To(Equal(0))
	})

	// worker 积压时首个 tick 就可能落在窗口外，此时仍应采信集群里的真实终态
	It("honors an observed stable status even when the window is exceeded", func() {
		rec.StartedAt = time.Now().Add(-pollingTimeout() - time.Minute)
		insert()
		stubFetch(appmodeldeploy.StatusDeployed)
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(reload().Status).To(Equal(appmodeldeploy.StatusDeployed))
		Expect(enqCount).To(Equal(0))
		Expect(hookCount.Load()).To(Equal(int32(1)))
	})

	It("marks pollingBroken after remaining retries are exhausted", func() {
		insert()
		args.FailureRetryRemain = 1
		mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(nil, errors.New("cluster down")).Build()
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(reload().Status).To(Equal(appmodeldeploy.StatusPollingBroken))
		Expect(enqCount).To(Equal(0))
	})

	It("reschedules when query fails but retries remain", func() {
		insert()
		args.FailureRetryRemain = 3
		mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(nil, errors.New("cluster down")).Build()
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(reload().Status).To(Equal(appmodeldeploy.StatusDeploying))
		Expect(enqCount).To(Equal(1))
	})

	It("triggers topology refresh on the first tick only", func() {
		insert()
		stubFetch(appmodeldeploy.StatusDeploying)
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Eventually(topoCount.Load).Should(Equal(int32(1)))

		args.TopologyRefreshed = true
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Consistently(topoCount.Load).Should(Equal(int32(1)))
	})

	It("returns a retryable error when saving a stable status fails", func() {
		insert()
		stubFetch(appmodeldeploy.StatusDeployed)
		// 只在本例模拟落库失败，其余用例走真实 Mongo Update
		mockey.Mock((*mongo.Collection).UpdateOne).Return(nil, errors.New("db down")).Build()
		err := plr.Handle(ctx, args)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeFalse())
		Expect(enqCount).To(Equal(0))
		Expect(hookCount.Load()).To(Equal(int32(0)))
		Expect(reload().Status).To(Equal(appmodeldeploy.StatusDeploying))
	})

	It("marks canceled when a newer deploy exists", func() {
		insert()
		stubFetch(appmodeldeploy.StatusDeploying)
		time.Sleep(5 * time.Millisecond)
		Expect(records.Create(ctx, newDeployingRecord(rec.AppID))).To(Succeed())
		Expect(plr.Handle(ctx, args)).To(Succeed())
		got := reload()
		Expect(got.Status).To(Equal(appmodeldeploy.StatusCanceled))
		Expect(got.Message).To(ContainSubstring("superseded"))
		Expect(enqCount).To(Equal(0))
	})

	Context("with a batch release", func() {
		pausedProgress := &appmodeldeploy.BatchProgress{
			CurrentBatchIndex: lo.ToPtr(int64(0)),
			BatchPhase:        batchrelease.BatchPhaseManualPaused,
			RolloutPhase:      "Running",
			PauseStartTime:    lo.ToPtr(time.Date(2026, 10, 22, 2, 0, 0, 0, time.UTC)),
			UpdatedReplicas:   1,
			Replicas:          4,
			OldImageTag:       "v1",
			NewImageTag:       "v2",
		}

		stubBatchFetch := func(status appmodeldeploy.Status, progress *appmodeldeploy.BatchProgress) {
			mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(
				&appmodeldeploy.DeployState{Status: status, BatchProgress: progress}, nil,
			).Build()
		}

		BeforeEach(func() {
			rec.ImageTag = "v2"
			rec.ReleaseMode = appmodeldeploy.ReleaseModeCanary
			rec.BatchProgress = &appmodeldeploy.BatchProgress{
				BatchPhase: batchrelease.BatchPhasePending, Replicas: 4, OldImageTag: "v1", NewImageTag: "v2",
			}
		})

		It("writes the progress and keeps polling", func() {
			insert()
			stubBatchFetch(appmodeldeploy.StatusDeploying, pausedProgress)
			Expect(plr.Handle(ctx, args)).To(Succeed())
			got := reload()
			Expect(got.Status).To(Equal(appmodeldeploy.StatusDeploying))
			Expect(got.BatchProgress).To(Equal(pausedProgress))
			Expect(enqCount).To(Equal(1))
		})

		It("does not mark pollingTimeout when the window is exceeded", func() {
			rec.StartedAt = time.Now().Add(-pollingTimeout() - time.Minute)
			insert()
			stubBatchFetch(appmodeldeploy.StatusDeploying, pausedProgress)
			Expect(plr.Handle(ctx, args)).To(Succeed())
			Expect(reload().Status).To(Equal(appmodeldeploy.StatusDeploying))
			Expect(enqCount).To(Equal(1))
		})

		It("keeps the last progress when fetching fails", func() {
			rec.StartedAt = time.Now().Add(-pollingTimeout() - time.Minute)
			insert()
			args.FailureRetryRemain = 3
			mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(nil, errors.New("cluster down")).Build()
			Expect(plr.Handle(ctx, args)).To(Succeed())
			got := reload()
			Expect(got.Status).To(Equal(appmodeldeploy.StatusDeploying))
			Expect(got.BatchProgress).To(Equal(rec.BatchProgress))
			Expect(enqCount).To(Equal(1))
		})

		It("marks pollingBroken after remaining retries are exhausted", func() {
			insert()
			args.FailureRetryRemain = 1
			mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(nil, errors.New("cluster down")).Build()
			Expect(plr.Handle(ctx, args)).To(Succeed())
			Expect(reload().Status).To(Equal(appmodeldeploy.StatusPollingBroken))
			Expect(enqCount).To(Equal(0))
		})

		DescribeTable("stops polling at the terminal status",
			func(status appmodeldeploy.Status, wantHook int32) {
				insert()
				progress := *pausedProgress
				progress.BatchPhase = batchrelease.BatchPhaseUpdating
				stubBatchFetch(status, &progress)
				Expect(plr.Handle(ctx, args)).To(Succeed())
				got := reload()
				Expect(got.Status).To(Equal(status))
				Expect(got.EndedAt).NotTo(BeZero())
				Expect(got.ImageTag).To(Equal(rec.ImageTag))
				Expect(got.BatchProgress).To(Equal(&progress))
				Expect(enqCount).To(Equal(0))
				Expect(hookCount.Load()).To(Equal(wantHook))
			},
			Entry("deployed", appmodeldeploy.StatusDeployed, int32(1)),
			Entry("rolled back", appmodeldeploy.StatusRolledBack, int32(0)),
		)

		It("returns a retryable error when saving the terminal progress fails", func() {
			insert()
			stubBatchFetch(appmodeldeploy.StatusDeployed, pausedProgress)
			mockey.Mock(appmodeldeploy.UpdateBatchProgress).Return(errors.New("db down")).Build()
			err := plr.Handle(ctx, args)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeFalse())
			Expect(reload().Status).To(Equal(appmodeldeploy.StatusDeploying))
			Expect(hookCount.Load()).To(Equal(int32(0)))
		})
	})

	DescribeTable("maps terminal status from fetch helper",
		func(want appmodeldeploy.Status, wantHook int32) {
			insert()
			stubFetch(want)
			Expect(plr.Handle(ctx, args)).To(Succeed())
			Expect(reload().Status).To(Equal(want))
			Expect(enqCount).To(Equal(0))
			Expect(hookCount.Load()).To(Equal(wantHook))
		},
		Entry("deployed", appmodeldeploy.StatusDeployed, int32(1)),
		Entry("failed", appmodeldeploy.StatusFailed, int32(0)),
		Entry("canceled", appmodeldeploy.StatusCanceled, int32(0)),
	)
})

var _ = Describe("stableAuditOptions", func() {
	args := Args{WorkspaceID: "ws", AppID: "app", EnvName: "prod", TrafficLaneName: "lane", DeployID: "deploy-1"}

	apply := func(record *appmodeldeploy.Record) (*audit.OperationRecord, map[string]string) {
		r := &audit.OperationRecord{}
		for _, opt := range stableAuditOptions(record, args) {
			opt(r)
		}
		var extras map[string]string
		if len(r.Data.Extras) > 0 {
			Expect(json.Unmarshal(r.Data.Extras, &extras)).To(Succeed())
		}
		return r, extras
	}

	DescribeTable("releasing all at once keeps the plain deploy audit",
		func(status appmodeldeploy.Status, result audit.Result) {
			r, extras := apply(&appmodeldeploy.Record{Status: status})
			Expect(r.Result).To(Equal(result))
			Expect(r.Attribute).To(BeEmpty())
			Expect(extras).To(BeNil())
		},
		Entry("deployed", appmodeldeploy.StatusDeployed, audit.ResultSuccess),
		Entry("failed", appmodeldeploy.StatusFailed, audit.ResultFailed),
	)

	DescribeTable("canary release pairs with the dispatch audit by deployID",
		func(status appmodeldeploy.Status, result audit.Result) {
			r, extras := apply(&appmodeldeploy.Record{
				Status:          status,
				ReleaseMode:     appmodeldeploy.ReleaseModeCanary,
				ReleaseStrategy: &appmodeldeploy.ReleaseStrategySnapshot{ID: "strategy-1", Name: "careful"},
			})
			Expect(r.Result).To(Equal(result))
			Expect(r.Attribute).To(Equal(audit.AttributeReleaseStrategy))
			Expect(r.Group.EnvName).To(Equal("prod"))
			Expect(extras).To(Equal(map[string]string{
				"trafficLaneName":     "lane",
				"deployID":            "deploy-1",
				"releaseMode":         "canary",
				"releaseStrategyID":   "strategy-1",
				"releaseStrategyName": "careful",
				"status":              string(status),
			}))
		},
		Entry("deployed", appmodeldeploy.StatusDeployed, audit.ResultSuccess),
		Entry("rolled back by the user", appmodeldeploy.StatusRolledBack, audit.ResultSuccess),
		Entry("polling broken", appmodeldeploy.StatusPollingBroken, audit.ResultFailed),
		Entry("superseded by a new deploy", appmodeldeploy.StatusCanceled, audit.ResultFailed),
	)
})
