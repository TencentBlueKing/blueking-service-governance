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
	"errors"
	"time"

	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

func newDeployingRecord() *appmodeldeploy.Record {
	return &appmodeldeploy.Record{
		ID:        bson.NewObjectID(),
		AppID:     "app-1",
		EnvName:   "dev",
		Status:    appmodeldeploy.StatusDeploying,
		StartedAt: time.Now(),
		Creator:   "alice",
	}
}

func stubFetch(status appmodeldeploy.Status) {
	mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(
		&appmodeldeploy.DeployState{Status: status, Message: string(status)}, nil,
	).Build()
}

var _ = Describe("Manager Handle", func() {
	var (
		ctx      context.Context
		args     Args
		rec      *appmodeldeploy.Record
		latest   *appmodeldeploy.Record
		enqCount int
		mgr      *Manager
	)

	BeforeEach(func() {
		ctx = context.Background()
		args = Args{WorkspaceID: "ws-1", AppID: "app-1", EnvName: "dev", DeployID: "dep-1"}
		rec = newDeployingRecord()
		latest = rec
		enqCount = 0
		mgr = NewManager(&appmodeldeploy.RecordStoreMongo{}, nil)

		mockey.Mock((*appmodeldeploy.RecordStoreMongo).Get).To(
			func(context.Context, string, string) (*appmodeldeploy.Record, error) { return rec, nil },
		).Build()
		mockey.Mock((*appmodeldeploy.RecordStoreMongo).GetLatest).To(
			func(context.Context, string, string, string) (*appmodeldeploy.Record, error) { return latest, nil },
		).Build()
		mockey.Mock((*appmodeldeploy.RecordStoreMongo).Update).Return(nil).Build()
		mockey.Mock(taskq.Enqueue).To(func(context.Context, *taskq.Task, ...asynq.Option) error {
			enqCount++
			return nil
		}).Build()
		mockey.Mock(audit.AddOperationRecordAsync).Return().Build()
		mockey.Mock(triggerTopologyRefresh).Return().Build()
		mockey.Mock(handleDeploySucceeded).Return().Build()
	})

	AfterEach(func() {
		time.Sleep(50 * time.Millisecond)
		mockey.UnPatchAll()
	})

	It("enqueues the next tick when deploy is still running", func() {
		stubFetch(appmodeldeploy.StatusDeploying)
		err := mgr.Handle(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(enqCount).To(Equal(1))
		Expect(rec.Status).To(Equal(appmodeldeploy.StatusDeploying))
	})

	It("skips already stable records without polling", func() {
		rec.Status = appmodeldeploy.StatusDeployed
		err := mgr.Handle(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(enqCount).To(Equal(0))
	})

	It("skips uninstalling records without overwriting", func() {
		rec.Status = appmodeldeploy.StatusUninstalling
		err := mgr.Handle(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(enqCount).To(Equal(0))
		Expect(rec.Status).To(Equal(appmodeldeploy.StatusUninstalling))
	})

	It("marks pollingTimeout when StartedAt exceeds configured window", func() {
		rec.StartedAt = time.Now().Add(-pollingTimeout() - time.Minute)
		err := mgr.Handle(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(appmodeldeploy.StatusPollingTimeout))
		Expect(enqCount).To(Equal(0))
	})

	It("marks pollingBroken after remaining retries are exhausted", func() {
		args.FailureRetryRemain = 1
		mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(nil, errors.New("cluster down")).Build()
		err := mgr.Handle(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(appmodeldeploy.StatusPollingBroken))
		Expect(enqCount).To(Equal(0))
	})

	It("reschedules when query fails but retries remain", func() {
		args.FailureRetryRemain = 3
		mockey.Mock((*appmodeldeploy.DeployStateGetter).Get).Return(nil, errors.New("cluster down")).Build()
		err := mgr.Handle(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(appmodeldeploy.StatusDeploying))
		Expect(enqCount).To(Equal(1))
	})

	It("marks canceled when a newer deploy exists", func() {
		stubFetch(appmodeldeploy.StatusDeploying)
		latest = newDeployingRecord()
		err := mgr.Handle(ctx, args)
		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Status).To(Equal(appmodeldeploy.StatusCanceled))
		Expect(rec.Message).To(ContainSubstring("superseded"))
		Expect(enqCount).To(Equal(0))
	})

	DescribeTable("maps terminal status from fetch helper",
		func(want appmodeldeploy.Status) {
			stubFetch(want)
			err := mgr.Handle(ctx, args)
			Expect(err).NotTo(HaveOccurred())
			Expect(rec.Status).To(Equal(want))
			Expect(enqCount).To(Equal(0))
		},
		Entry("deployed", appmodeldeploy.StatusDeployed),
		Entry("failed", appmodeldeploy.StatusFailed),
		Entry("canceled", appmodeldeploy.StatusCanceled),
	)
})
