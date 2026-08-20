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

package chartbuildpoll

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
	helmchartbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

func newRunningRecord(workspaceID, appID, buildID string) *helmchartbuild.Record {
	return &helmchartbuild.Record{
		WorkspaceID:  workspaceID,
		AppID:        appID,
		BuildID:      buildID,
		PipelineID:   "p-test",
		ChartVersion: "0.0.1",
		Status:       helmchartbuild.StatusRunning,
		Operator:     "alice",
		Params:       map[string]string{},
	}
}

func stubFetch(status helmchartbuild.Status) {
	mockey.Mock(fetchAndUpdateChartBuildRecord).To(
		func(
			_ context.Context,
			_ bkciapi.Client,
			_ *bkci.Pipeline,
			record *helmchartbuild.Record,
			_ string,
		) error {
			record.Status = status
			if record.IsTerminated() {
				endedAt := time.Now()
				record.EndedAt = &endedAt
			}
			return nil
		},
	).Build()
}

var _ = Describe("Poller RunTick", func() {
	var (
		ctx           context.Context
		args          Args
		rec           *helmchartbuild.Record
		recordStore   helmchartbuild.RecordStore
		pipelineStore bkci.PipelineStore
		enqCount      int
		enqDelay      time.Duration
		plr           *poller
	)

	createRecord := func() {
		Expect(recordStore.Create(ctx, rec)).To(Succeed())
		args.BuildID = rec.BuildID
	}

	reloadRecord := func() *helmchartbuild.Record {
		got, err := recordStore.Get(ctx, rec.AppID, rec.BuildID)
		Expect(err).NotTo(HaveOccurred())
		return got
	}

	// Create 会把 StartedAt 写成 now，超时窗口用例需要把库里的时间拨回去
	backdateStartedAt := func(startedAt time.Time) {
		coll := database.Client().Database(database.Name()).Collection("helm_chart_build_records")
		_, err := coll.UpdateOne(ctx, bson.M{
			"appID": rec.AppID, "buildID": rec.BuildID,
		}, bson.M{"$set": bson.M{"startedAt": startedAt}})
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		var err error
		ctx = auth.WithUser(context.Background(), auth.User{ID: "alice"})
		recordStore, err = helmchartbuild.NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		pipelineStore, err = bkci.NewPipelineStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		workspaceID := "ws-" + stringx.Random(8)
		appID := "app-" + stringx.Random(8)
		buildID := "b-" + stringx.Random(8)
		rec = newRunningRecord(workspaceID, appID, buildID)
		args = Args{WorkspaceID: workspaceID, AppID: appID, BuildID: buildID}
		enqCount = 0
		enqDelay = 0
		plr = newPoller(recordStore, pipelineStore)

		Expect(pipelineStore.Create(ctx, &bkci.Pipeline{
			ID:          "p-" + stringx.Random(8),
			Type:        string(bkci.PipelineTypeHelmGitBuild),
			WorkspaceID: workspaceID,
			ProjectCode: "proj",
			Name:        "helm-git-build",
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
	})

	AfterEach(func() {
		time.Sleep(50 * time.Millisecond)
		mockey.UnPatchAll()
	})

	DescribeTable("advances status from fetch helper",
		func(want helmchartbuild.Status, wantEnq int) {
			createRecord()
			stubFetch(want)
			Expect(plr.runTick(ctx, args)).To(Succeed())
			Expect(reloadRecord().Status).To(Equal(want))
			Expect(enqCount).To(Equal(wantEnq))
			if wantEnq > 0 {
				Expect(enqDelay).To(Equal(5 * time.Second))
			}
		},
		Entry("still running, enqueue next tick", helmchartbuild.StatusRunning, 1),
		Entry("success", helmchartbuild.StatusSuccess, 0),
		Entry("failed", helmchartbuild.StatusFailed, 0),
		Entry("canceled", helmchartbuild.StatusCanceled, 0),
	)

	DescribeTable("stops without polling when tick should finish immediately",
		func(setup func(), want helmchartbuild.Status) {
			setup()
			Expect(plr.runTick(ctx, args)).To(Succeed())
			Expect(reloadRecord().Status).To(Equal(want))
			Expect(enqCount).To(Equal(0))
		},
		Entry("already terminated", func() {
			rec.Status = helmchartbuild.StatusSuccess
			createRecord()
		}, helmchartbuild.StatusSuccess),
		Entry("window exceeded", func() {
			createRecord()
			backdateStartedAt(time.Now().Add(-pollingTimeout - time.Minute))
		}, helmchartbuild.StatusPollingTimeout),
	)

	DescribeTable("query failure uses remaining retry budget",
		func(remain int, want helmchartbuild.Status, wantEnq int) {
			createRecord()
			args.FailureRetryRemain = remain
			mockey.Mock(fetchAndUpdateChartBuildRecord).Return(errors.New("bkci down")).Build()
			Expect(plr.runTick(ctx, args)).To(Succeed())
			Expect(reloadRecord().Status).To(Equal(want))
			Expect(enqCount).To(Equal(wantEnq))
		},
		Entry("reschedules when retries remain", 3, helmchartbuild.StatusRunning, 1),
		Entry("marks pollingBroken when retries are exhausted", 1, helmchartbuild.StatusPollingBroken, 0),
	)
})
