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

package build

import (
	"context"
	"time"

	"github.com/bytedance/mockey"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

var _ = Describe("StartAndScheduleBuild", func() {
	var (
		ctx          context.Context
		app          *bkmsapp.Application
		buildService *Service
	)

	BeforeEach(func() {
		ctx = context.Background()
		app = &bkmsapp.Application{ID: "app-1", WorkspaceID: "ws-1"}
		buildService = &Service{}
	})

	AfterEach(func() {
		mockey.UnPatchAll()
	})

	It("enqueues polling with the initial interval", func() {
		startedAt := time.Now()
		mockey.Mock((*Service).Build).Return(&BuildResult{
			PipelineType: "build",
			Record: &imagebuild.Record{
				BuildID:   "b-1",
				Status:    imagebuild.StatusRunning,
				StartedAt: startedAt,
				CreatedAt: startedAt,
			},
		}, nil).Build()

		var enqueued bool
		var delay time.Duration
		mockey.Mock(taskq.Enqueue).To(func(_ context.Context, _ *taskq.Task, opts ...asynq.Option) error {
			enqueued = true
			for _, opt := range opts {
				if opt.Type() == asynq.ProcessInOpt {
					delay = opt.Value().(time.Duration)
				}
			}
			return nil
		}).Build()

		record, err := StartAndScheduleBuild(ctx, buildService, app, "master", "v1", StartOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(record.BuildID).To(Equal("b-1"))
		Expect(enqueued).To(BeTrue())
		Expect(delay).To(Equal(10 * time.Second))
	})

	It("returns error when enqueue fails", func() {
		mockey.Mock((*Service).Build).Return(&BuildResult{
			PipelineType: "build",
			Record: &imagebuild.Record{
				BuildID:   "b-1",
				Status:    imagebuild.StatusRunning,
				StartedAt: time.Now(),
			},
		}, nil).Build()
		mockey.Mock(taskq.Enqueue).Return(errors.New("redis down")).Build()

		_, err := StartAndScheduleBuild(ctx, buildService, app, "master", "v1", StartOptions{})
		Expect(err).To(MatchError(ContainSubstring("enqueue polling build status task")))
	})
})
