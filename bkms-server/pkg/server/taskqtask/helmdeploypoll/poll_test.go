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

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	helmrelease "helm.sh/helm/v3/pkg/release"

	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

func newPendingRecord(appID string) *helmdeploy.Record {
	return &helmdeploy.Record{
		ID:          bson.NewObjectID(),
		WorkspaceID: "ws-1",
		AppID:       appID,
		EnvName:     "dev",
		ReleaseName: "dev-" + appID,
		ClusterID:   "BCS-K8S-1",
		Namespace:   "default",
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

var _ = Describe("Poller Handle", func() {
	var (
		ctx        context.Context
		args       Args
		rec        *helmdeploy.Record
		store      helmdeploy.RecordStore
		enqCount   int
		releaseCnt int
		topoCount  atomic.Int32
		hookCount  atomic.Int32
		plr        *Poller
	)

	insert := func() {
		id, err := store.Create(ctx, rec)
		Expect(err).NotTo(HaveOccurred())
		args.DeployID = id
	}

	reload := func() *helmdeploy.Record {
		got, err := store.Get(ctx, rec.AppID, args.DeployID)
		Expect(err).NotTo(HaveOccurred())
		return got
	}

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		store, err = helmdeploy.NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		appID := "app-" + stringx.Random(8)
		rec = newPendingRecord(appID)
		args = Args{WorkspaceID: rec.WorkspaceID, AppID: rec.AppID, EnvName: rec.EnvName, DeployID: rec.ID.Hex()}
		enqCount = 0
		releaseCnt = 0
		topoCount.Store(0)
		hookCount.Store(0)
		plr = NewPoller(store)

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
		insert()
		stubFetch(helm.StatusPendingUpgrade, "2")
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(1))
		Expect(releaseCnt).To(Equal(0))
		Expect(reload().Status).To(Equal(helm.StatusPendingUpgrade))
		Eventually(topoCount.Load).Should(Equal(int32(1)))

		args.TopologyRefreshed = true
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(2))
		Consistently(topoCount.Load).Should(Equal(int32(1)))
	})

	DescribeTable("already stable records skip polling",
		func(newerLatest bool, wantRelease int) {
			rec.Status = helm.StatusDeployed
			insert()
			if newerLatest {
				time.Sleep(5 * time.Millisecond)
				_, err := store.Create(ctx, newPendingRecord(rec.AppID))
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(plr.Handle(ctx, args)).To(Succeed())
			Expect(enqCount).To(Equal(0))
			Expect(releaseCnt).To(Equal(wantRelease))
			Expect(hookCount.Load()).To(Equal(int32(1)))
		},
		Entry("releases lock when this record is still latest", false, 1),
		Entry("keeps lock when a newer deploy is latest", true, 0),
	)

	It("marks pollingTimeout when the window is exceeded and the release is still pending", func() {
		rec.StartedAt = time.Now().Add(-pollingTimeout() - time.Minute)
		insert()
		stubFetch(helm.StatusPendingUpgrade, "2")
		Expect(plr.Handle(ctx, args)).To(Succeed())
		Expect(reload().Status).To(Equal(helm.StatusPollingTimeout))
		Expect(enqCount).To(Equal(0))
		Expect(releaseCnt).To(Equal(1))
	})

	// worker 积压时首个 tick 就可能落在窗口外，此时仍应采信集群里的真实终态
	It("honors an observed stable status even when the window is exceeded", func() {
		rec.StartedAt = time.Now().Add(-pollingTimeout() - time.Minute)
		insert()
		stubFetch(helm.StatusDeployed, "1")
		Expect(plr.Handle(ctx, args)).To(Succeed())
		got := reload()
		Expect(got.Status).To(Equal(helm.StatusDeployed))
		Expect(got.Revision).To(Equal("1"))
		Expect(enqCount).To(Equal(0))
		Expect(hookCount.Load()).To(Equal(int32(1)))
	})

	DescribeTable("query failure uses remaining retry budget",
		func(remain int, wantStatus helmrelease.Status, wantEnq, wantRelease int) {
			insert()
			args.FailureRetryRemain = remain
			mockey.Mock(fetchReleaseStatus).Return(nil, errors.New("cluster down")).Build()
			Expect(plr.Handle(ctx, args)).To(Succeed())
			Expect(reload().Status).To(Equal(wantStatus))
			Expect(enqCount).To(Equal(wantEnq))
			Expect(releaseCnt).To(Equal(wantRelease))
		},
		Entry("reschedules when retries remain", 3, helm.StatusPendingUpgrade, 1, 0),
		Entry("marks pollingBroken when retries are exhausted", 1, helm.StatusPollingBroken, 0, 1),
	)

	It("writes revision and stops when status becomes deployed", func() {
		insert()
		stubFetch(helm.StatusDeployed, "5")
		Expect(plr.Handle(ctx, args)).To(Succeed())
		got := reload()
		Expect(got.Status).To(Equal(helm.StatusDeployed))
		Expect(got.Revision).To(Equal("5"))
		Expect(got.Message).To(Equal(string(helm.StatusDeployed)))
		Expect(enqCount).To(Equal(0))
		Expect(releaseCnt).To(Equal(1))
		Expect(hookCount.Load()).To(Equal(int32(1)))
	})

	It("returns a retryable error when saving a stable status fails", func() {
		insert()
		stubFetch(helm.StatusDeployed, "5")
		// 只在本例模拟落库失败，其余用例走真实 Mongo Update
		mockey.Mock((*helmdeploy.RecordStoreMongo).Update).Return(errors.New("db down")).Build()
		err := plr.Handle(ctx, args)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeFalse())
		Expect(enqCount).To(Equal(0))
		Expect(releaseCnt).To(Equal(0))
		Expect(hookCount.Load()).To(Equal(int32(0)))
		Expect(reload().Status).To(Equal(helm.StatusPendingUpgrade))
	})
})
