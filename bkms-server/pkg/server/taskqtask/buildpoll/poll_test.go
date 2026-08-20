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

package buildpoll

import (
	"context"
	"errors"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

func newRunningRecord(workspaceID, appID, buildID string) *build.Record {
	return &build.Record{
		WorkspaceID: workspaceID,
		AppID:       appID,
		BuildID:     buildID,
		PipelineID:  "p-test",
		Status:      build.StatusRunning,
		StartedAt:   time.Now(),
		Operator:    "alice",
		Params:      map[string]string{},
	}
}

func stubFetch(status build.Status) {
	mockey.Mock(fetchAndUpdateBuildRecord).To(
		func(_ context.Context, _ bkciapi.Client, _ *bkci.Pipeline, record *build.Record, _ string) error {
			record.Status = status
			if status.IsTerminated() {
				record.EndedAt = time.Now()
			}
			return nil
		},
	).Build()
}

var _ = Describe("Poller RunTick", func() {
	var (
		ctx             context.Context
		args            Args
		rec             *build.Record
		recordStore     build.RecordStore
		pipelineStore   bkci.PipelineStore
		autoDeployStore autodeploy.RecordStore
		enqCount        int
		enqDelay        time.Duration
		plr             *poller
		deployHook      func() error
		snapshotHook    func(context.Context, string, string) error
	)

	createRecord := func() {
		Expect(recordStore.Create(ctx, rec)).To(Succeed())
		args.BuildID = rec.BuildID
	}

	reloadRecord := func() *build.Record {
		got, err := recordStore.Get(ctx, rec.AppID, rec.BuildID)
		Expect(err).NotTo(HaveOccurred())
		return got
	}

	BeforeEach(func() {
		var err error
		ctx = auth.WithUser(context.Background(), auth.User{ID: "alice"})
		recordStore, err = build.NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		pipelineStore, err = bkci.NewPipelineStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		autoDeployStore, err = autodeploy.NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		workspaceID := "ws-" + stringx.Random(8)
		appID := "app-" + stringx.Random(8)
		buildID := "b-" + stringx.Random(8)
		rec = newRunningRecord(workspaceID, appID, buildID)
		args = Args{WorkspaceID: workspaceID, PipelineType: "build", AppID: appID, BuildID: buildID}
		enqCount = 0
		enqDelay = 0
		plr = newPoller(recordStore, pipelineStore, autoDeployStore)
		deployHook = nil
		snapshotHook = nil

		Expect(pipelineStore.Create(ctx, &bkci.Pipeline{
			ID:          "p-" + stringx.Random(8),
			Type:        "build",
			WorkspaceID: workspaceID,
			ProjectCode: "proj",
			Name:        "build",
			Creator:     "alice",
		})).To(Succeed())

		mockey.Mock(bkciapi.New).Return(nil, nil).Build()
		mockey.Mock(taskq.Enqueue).To(func(_ context.Context, _ *taskq.Task, opts ...asynq.Option) error {
			enqCount++
			for _, opt := range opts {
				if opt.Type() == asynq.ProcessInOpt {
					enqDelay = opt.Value().(time.Duration)
				}
			}
			return nil
		}).Build()
		mockey.Mock(audit.AddOperationRecordAsync).Return().Build()
		mockey.Mock(triggerDeployAfterBuild).To(
			func(_ context.Context, _ *autodeploy.Operator, _ *build.Record, _ Args) error {
				if deployHook != nil {
					return deployHook()
				}
				return nil
			},
		).Build()
		mockey.Mock(refreshSnapshots).To(func(ctx context.Context, appID, imageTag string) error {
			if snapshotHook != nil {
				return snapshotHook(ctx, appID, imageTag)
			}
			return nil
		}).Build()
	})

	AfterEach(func() {
		time.Sleep(50 * time.Millisecond)
		mockey.UnPatchAll()
	})

	It("enqueues the next tick with ProcessIn when build is still running", func() {
		createRecord()
		stubFetch(build.StatusRunning)
		Expect(plr.runTick(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(1))
		Expect(enqDelay).To(Equal(10 * time.Second))
		Expect(reloadRecord().Status).To(Equal(build.StatusRunning))
	})

	It("skips already terminated records without polling", func() {
		rec.Status = build.StatusSuccess
		createRecord()
		Expect(plr.runTick(ctx, args)).To(Succeed())
		Expect(enqCount).To(Equal(0))
		Expect(reloadRecord().Status).To(Equal(build.StatusSuccess))
	})

	It("marks pollingTimeout when StartedAt exceeds 24h", func() {
		rec.StartedAt = time.Now().Add(-25 * time.Hour)
		createRecord()
		Expect(plr.runTick(ctx, args)).To(Succeed())
		Expect(reloadRecord().Status).To(Equal(build.StatusPollingTimeout))
		Expect(enqCount).To(Equal(0))
	})

	It("marks pollingBroken after remaining retries are exhausted", func() {
		createRecord()
		args.FailureRetryRemain = 1
		mockey.Mock(fetchAndUpdateBuildRecord).Return(errors.New("bkci down")).Build()
		Expect(plr.runTick(ctx, args)).To(Succeed())
		Expect(reloadRecord().Status).To(Equal(build.StatusPollingBroken))
		Expect(enqCount).To(Equal(0))
	})

	It("reschedules when bkci fails but retries remain", func() {
		createRecord()
		args.FailureRetryRemain = 3
		mockey.Mock(fetchAndUpdateBuildRecord).Return(errors.New("bkci down")).Build()
		Expect(plr.runTick(ctx, args)).To(Succeed())
		Expect(reloadRecord().Status).To(Equal(build.StatusRunning))
		Expect(enqCount).To(Equal(1))
	})

	DescribeTable("maps terminal status from fetch helper",
		func(want build.Status) {
			createRecord()
			stubFetch(want)
			Expect(plr.runTick(ctx, args)).To(Succeed())
			Expect(reloadRecord().Status).To(Equal(want))
			Expect(enqCount).To(Equal(0))
		},
		Entry("success", build.StatusSuccess),
		Entry("failed", build.StatusFailed),
		Entry("canceled", build.StatusCanceled),
	)

	It("triggers auto deploy on success and still succeeds if deploy fails", func() {
		createRecord()
		Expect(autoDeployStore.Create(ctx, &autodeploy.Record{
			ID:          bson.NewObjectID(),
			WorkspaceID: rec.WorkspaceID,
			AppID:       rec.AppID,
			EnvName:     "dev",
			BuildID:     rec.BuildID,
			Stage:       autodeploy.StageBuild,
			Status:      string(build.StatusRunning),
			Operator:    "alice",
			StartedAt:   rec.StartedAt,
		})).To(Succeed())

		stubFetch(build.StatusSuccess)
		args.AutoDeploy = &AutoDeployArgs{EnvName: "dev", Replicas: 1}
		deployCalled := false
		deployHook = func() error {
			deployCalled = true
			return errors.New("deploy unavailable")
		}

		Expect(plr.runTick(ctx, args)).To(Succeed())
		Expect(reloadRecord().Status).To(Equal(build.StatusSuccess))
		Expect(deployCalled).To(BeTrue())
	})

	It("keeps build success when snapshot refresh fails", func() {
		createRecord()
		stubFetch(build.StatusSuccess)
		done := make(chan struct{})
		snapshotHook = func(_ context.Context, appID, _ string) error {
			defer close(done)
			Expect(appID).To(Equal(rec.AppID))
			return errors.New("refresh failed")
		}

		Expect(plr.runTick(ctx, args)).To(Succeed())
		Expect(reloadRecord().Status).To(Equal(build.StatusSuccess))
		Eventually(done).Should(BeClosed())
	})
})
