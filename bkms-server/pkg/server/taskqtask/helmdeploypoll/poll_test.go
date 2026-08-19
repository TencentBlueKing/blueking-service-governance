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

package helmdeploypoll

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	helmrelease "helm.sh/helm/v3/pkg/release"

	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

func newPendingRecord() *helmdeploy.Record {
	return &helmdeploy.Record{
		ID:          bson.NewObjectID(),
		AppID:       "app-1",
		EnvName:     "dev",
		ReleaseName: "dev-app-1",
		Status:      helm.StatusPendingUpgrade,
		StartedAt:   time.Now(),
		Operator:    "alice",
	}
}

func stubFetch(status helmrelease.Status, revision string) {
	mockey.Mock(fetchReleaseStatus).Return(
		&helm.Release{
			Version: revision,
			DeployResult: helm.DeployResult{
				Status:      status,
				Description: string(status),
			},
		},
		nil,
	).Build()
}

var _ = Describe("Manager Handle", func() {
	var (
		ctx        context.Context
		args       Args
		rec        *helmdeploy.Record
		latest     *helmdeploy.Record
		enqCount   int
		releaseCnt int
		topoCount  atomic.Int32
		hookCount  atomic.Int32
		updateErr  error
		mgr        *Manager
	)

	BeforeEach(func() {
		ctx = context.Background()
		rec = newPendingRecord()
		latest = rec
		args = Args{WorkspaceID: "ws-1", AppID: "app-1", EnvName: "dev", DeployID: rec.ID.Hex()}
		enqCount = 0
		releaseCnt = 0
		topoCount.Store(0)
		hookCount.Store(0)
		updateErr = nil
		mgr = NewManager(&helmdeploy.RecordStoreMongo{})

		mockey.Mock((*helmdeploy.RecordStoreMongo).Get).Return(rec, nil).Build()
		mockey.Mock((*helmdeploy.RecordStoreMongo).GetLatest).To(
			func(context.Context, string, string, string) (*helmdeploy.Record, error) { return latest, nil },
		).Build()
		mockey.Mock((*helmdeploy.RecordStoreMongo).Update).To(
			func(context.Context, *helmdeploy.Record) error { return updateErr },
		).Build()
		mockey.Mock(taskq.Enqueue).To(func(context.Context, *taskq.Task, ...asynq.Option) error {
			enqCount++
			return nil
		}).Build()
		mockey.Mock(audit.AddOperationRecordAsync).Return().Build()
		mockey.Mock(triggerTopologyRefresh).To(func(context.Context, Args, *helmdeploy.Record) {
			topoCount.Add(1)
		}).Build()
		mockey.Mock(handleDeploySucceeded).To(func(context.Context, Args, *helmdeploy.Record) {
			hookCount.Add(1)
		}).Build()
		mockey.Mock(releaseDeployLock).To(func(context.Context, Args) { releaseCnt++ }).Build()
	})

	AfterEach(func() {
		time.Sleep(50 * time.Millisecond)
		mockey.UnPatchAll()
	})

	It("enqueues the next tick and refreshes topology only on the first tick", func() {
		stubFetch(helm.StatusPendingUpgrade, "2")
		Expect(mgr.Handle(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(1))
		Expect(releaseCnt).To(Equal(0))
		Expect(rec.Status).To(Equal(helm.StatusPendingUpgrade))
		Eventually(topoCount.Load).Should(Equal(int32(1)))

		args.TopologyRefreshed = true
		Expect(mgr.Handle(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(2))
		Consistently(topoCount.Load).Should(Equal(int32(1)))
	})

	DescribeTable("already stable records skip polling",
		func(newerLatest bool, wantRelease int) {
			rec.Status = helm.StatusDeployed
			if newerLatest {
				latest = newPendingRecord()
			}
			Expect(mgr.Handle(ctx, args)).To(Succeed())
			Expect(enqCount).To(Equal(0))
			Expect(releaseCnt).To(Equal(wantRelease))
			Expect(hookCount.Load()).To(Equal(int32(1)))
		},
		Entry("releases lock when this record is still latest", false, 1),
		Entry("keeps lock when a newer deploy is latest", true, 0),
	)

	It("marks pollingTimeout when StartedAt exceeds configured window", func() {
		rec.StartedAt = time.Now().Add(-pollingTimeout() - time.Minute)
		Expect(mgr.Handle(ctx, args)).To(Succeed())
		Expect(rec.Status).To(Equal(helm.StatusPollingTimeout))
		Expect(enqCount).To(Equal(0))
		Expect(releaseCnt).To(Equal(1))
	})

	DescribeTable("query failure uses remaining retry budget",
		func(remain int, wantStatus helmrelease.Status, wantEnq, wantRelease int) {
			args.FailureRetryRemain = remain
			mockey.Mock(fetchReleaseStatus).Return(nil, errors.New("cluster down")).Build()
			Expect(mgr.Handle(ctx, args)).To(Succeed())
			Expect(rec.Status).To(Equal(wantStatus))
			Expect(enqCount).To(Equal(wantEnq))
			Expect(releaseCnt).To(Equal(wantRelease))
		},
		Entry("reschedules when retries remain", 3, helm.StatusPendingUpgrade, 1, 0),
		Entry("marks pollingBroken when retries are exhausted", 1, helm.StatusPollingBroken, 0, 1),
	)

	DescribeTable("reaches deployed from fetch",
		func(saveErr error, wantRetryableErr bool, wantRelease, wantHook int) {
			stubFetch(helm.StatusDeployed, "5")
			updateErr = saveErr
			err := mgr.Handle(ctx, args)
			if wantRetryableErr {
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeFalse())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(rec.Status).To(Equal(helm.StatusDeployed))
				Expect(rec.Revision).To(Equal("5"))
				Expect(rec.Message).To(Equal(string(helm.StatusDeployed)))
			}
			Expect(enqCount).To(Equal(0))
			Expect(releaseCnt).To(Equal(wantRelease))
			Expect(hookCount.Load()).To(Equal(int32(wantHook)))
		},
		Entry("writes revision and stops", nil, false, 1, 1),
		Entry("returns a retryable error when save fails", errors.New("db down"), true, 0, 0),
	)
})
