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

	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

func newRunningRecord() *build.Record {
	return &build.Record{
		AppID:     "app-1",
		BuildID:   "build-1",
		Status:    build.StatusRunning,
		StartedAt: time.Now(),
		Params:    map[string]string{},
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
		ctx          context.Context
		args         Args
		rec          *build.Record
		enqCount     int
		enqDelay     time.Duration
		plr          *poller
		deployHook   func() error
		snapshotHook func(context.Context, string, string) error
	)

	BeforeEach(func() {
		ctx = auth.WithUser(context.Background(), auth.User{ID: "alice"})
		args = Args{WorkspaceID: "ws-1", PipelineType: "build", AppID: "app-1", BuildID: "build-1"}
		rec = newRunningRecord()
		enqCount = 0
		enqDelay = 0
		plr = newPoller(&build.RecordStoreMongo{}, &bkci.PipelineStoreMongo{}, nil)
		deployHook = nil
		snapshotHook = nil

		mockey.Mock((*build.RecordStoreMongo).Get).To(
			func(_ context.Context, appID, buildID string) (*build.Record, error) {
				if rec == nil || rec.AppID != appID || rec.BuildID != buildID {
					return nil, errors.New("build record not found")
				}
				cp := *rec
				return &cp, nil
			},
		).Build()
		mockey.Mock((*build.RecordStoreMongo).Update).To(
			func(_ context.Context, record *build.Record) error {
				cp := *record
				*rec = cp
				return nil
			},
		).Build()
		mockey.Mock((*bkci.PipelineStoreMongo).GetByWorkspaceAndType).Return(&bkci.Pipeline{
			ID: "p-1", ProjectCode: "proj", Type: "build", WorkspaceID: "ws-1",
		}, nil).Build()
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
		mockey.Mock(syncBuildStatus).Return(nil).Build()
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
		stubFetch(build.StatusRunning)
		err := plr.runTick(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(enqCount).To(Equal(1))
		Expect(enqDelay).To(Equal(10 * time.Second))
		Expect(rec.Status).To(Equal(build.StatusRunning))
	})

	It("skips already terminated records without polling", func() {
		rec.Status = build.StatusSuccess
		err := plr.runTick(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(enqCount).To(Equal(0))
	})

	It("marks pollingTimeout when StartedAt exceeds 24h", func() {
		rec.StartedAt = time.Now().Add(-25 * time.Hour)
		err := plr.runTick(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(build.StatusPollingTimeout))
		Expect(enqCount).To(Equal(0))
	})

	It("marks pollingBroken after remaining retries are exhausted", func() {
		args.FailureRetryRemain = 1
		mockey.Mock(fetchAndUpdateBuildRecord).Return(errors.New("bkci down")).Build()
		err := plr.runTick(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(build.StatusPollingBroken))
		Expect(enqCount).To(Equal(0))
	})

	It("reschedules when bkci fails but retries remain", func() {
		args.FailureRetryRemain = 3
		mockey.Mock(fetchAndUpdateBuildRecord).Return(errors.New("bkci down")).Build()
		err := plr.runTick(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(build.StatusRunning))
		Expect(enqCount).To(Equal(1))
	})

	DescribeTable("maps terminal status from fetch helper",
		func(want build.Status) {
			stubFetch(want)
			err := plr.runTick(ctx, args)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Status).To(Equal(want))
			Expect(enqCount).To(Equal(0))
		},
		Entry("success", build.StatusSuccess),
		Entry("failed", build.StatusFailed),
		Entry("canceled", build.StatusCanceled),
	)

	It("triggers auto deploy on success and still succeeds if deploy fails", func() {
		stubFetch(build.StatusSuccess)
		args.AutoDeploy = &AutoDeployArgs{EnvName: "dev", Replicas: 1}
		plr.autoDeployStore = &autodeploy.RecordStoreMongo{}
		deployCalled := false
		deployHook = func() error {
			deployCalled = true
			return errors.New("deploy unavailable")
		}

		err := plr.runTick(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(build.StatusSuccess))
		Expect(deployCalled).To(BeTrue())
	})

	It("keeps build success when snapshot refresh fails", func() {
		stubFetch(build.StatusSuccess)
		done := make(chan struct{})
		snapshotHook = func(_ context.Context, appID, _ string) error {
			defer close(done)
			Expect(appID).To(Equal("app-1"))
			return errors.New("refresh failed")
		}

		err := plr.runTick(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(build.StatusSuccess))
		Eventually(done).Should(BeClosed())
	})
})
