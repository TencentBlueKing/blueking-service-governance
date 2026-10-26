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
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
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
		db              *mongo.Database
		pipelineStore   *bkci.PipelineStore
		autoDeployStore *autodeploy.RecordStore
		enqCount        int
		enqDelay        time.Duration
		plr             *poller
		deployHook      func() error
		snapshotHook    func(context.Context, string, string) error
		digestHook      func(appID, buildID string) error
	)

	createRecord := func() {
		Expect(build.CreateRecord(ctx, db, rec)).To(Succeed())
		args.BuildID = rec.BuildID
	}

	reloadRecord := func() *build.Record {
		got, err := plr.records.Get(ctx, build.ByBuild(rec.AppID, rec.BuildID))
		Expect(err).NotTo(HaveOccurred())
		return got
	}

	BeforeEach(func() {
		ctx = auth.WithUser(context.Background(), auth.User{ID: "alice"})
		db = testutil.NewDB(GinkgoT())
		pipelineStore = store.Of(db, bkci.PipelineMeta)
		autoDeployStore = store.Of(db, autodeploy.RecordMeta)

		workspaceID := "ws-" + stringx.Random(8)
		appID := "app-" + stringx.Random(8)
		buildID := "b-" + stringx.Random(8)
		rec = newRunningRecord(workspaceID, appID, buildID)
		args = Args{WorkspaceID: workspaceID, PipelineType: "build", AppID: appID, BuildID: buildID}
		enqCount = 0
		enqDelay = 0
		plr = newPoller(db)
		deployHook = nil
		snapshotHook = nil
		digestHook = nil

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
		mockey.Mock(enqueueRecordImageDigest).To(func(_ context.Context, appID, buildID string) error {
			if digestHook != nil {
				return digestHook(appID, buildID)
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

	It("still refreshes snapshots when enqueueing image digest task fails", func() {
		createRecord()
		stubFetch(build.StatusSuccess)
		digested := make(chan string, 1)
		digestHook = func(_, buildID string) error {
			digested <- buildID
			return errors.New("redis unavailable")
		}
		refreshed := make(chan struct{})
		snapshotHook = func(context.Context, string, string) error {
			close(refreshed)
			return nil
		}

		Expect(plr.runTick(ctx, args)).To(Succeed())
		Eventually(digested).Should(Receive(Equal(rec.BuildID)))
		Eventually(refreshed).Should(BeClosed())
		Expect(reloadRecord().Status).To(Equal(build.StatusSuccess))
	})

	It("does not record image digest when build fails", func() {
		createRecord()
		stubFetch(build.StatusFailed)
		digestCalled := make(chan struct{}, 1)
		digestHook = func(_, _ string) error {
			digestCalled <- struct{}{}
			return nil
		}

		Expect(plr.runTick(ctx, args)).To(Succeed())
		Consistently(digestCalled, 100*time.Millisecond).ShouldNot(Receive())
	})
})

var _ = Describe("recordImageDigest", func() {
	const repoName = "mirrors.example.com/bkms/demo"

	var (
		ctx        context.Context
		db         *mongo.Database
		records    *build.RecordStore
		rec        *build.Record
		args       ImageDigestArgs
		fetchedAt  []string
		detailErr  error
		resolveErr error
		// onFetch 在读取镜像仓库时执行，用于模拟读取期间的并发写入
		onFetch func()
	)

	createRecord := func() {
		Expect(build.CreateRecord(ctx, db, rec)).To(Succeed())
	}

	reload := func() *build.Record {
		got, err := records.Get(ctx, build.ByBuild(rec.AppID, rec.BuildID))
		Expect(err).NotTo(HaveOccurred())
		return got
	}

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		records = store.Of(db, build.RecordMeta)
		fetchedAt = nil
		detailErr = nil
		resolveErr = nil
		onFetch = nil

		rec = newRunningRecord("ws-"+stringx.Random(8), "app-"+stringx.Random(8), "b-"+stringx.Random(8))
		rec.Status = build.StatusSuccess
		rec.EndedAt = time.Now()
		rec.Params = map[string]string{pipelineparam.ImageTag: "v1.0.0"}
		rec.Artifact = repoName + ":v1.0.0"
		args = ImageDigestArgs{AppID: rec.AppID, BuildID: rec.BuildID, EnqueuedAt: time.Now()}

		mockey.Mock((*snapshot.Service).ResolveRepoKeyForApp).To(
			func(_ *snapshot.Service, _ context.Context, _ string) (*snapshot.RepoKeyInfo, error) {
				if resolveErr != nil {
					return nil, resolveErr
				}
				return &snapshot.RepoKeyInfo{RepoKey: "rk", RepoName: repoName}, nil
			},
		).Build()
		mockey.Mock((*registry.Client).GetTagDetail).To(
			func(_ *registry.Client, _ context.Context, repo, tag string) (registry.ImageDetail, error) {
				fetchedAt = append(fetchedAt, repo+":"+tag)
				if onFetch != nil {
					onFetch()
				}
				if detailErr != nil {
					return registry.ImageDetail{}, detailErr
				}
				return registry.ImageDetail{Tag: tag, Digest: "sha256:abc"}, nil
			},
		).Build()
	})

	AfterEach(func() {
		mockey.UnPatchAll()
	})

	It("records digest of the artifact in app image repo", func() {
		createRecord()
		Expect(recordImageDigest(ctx, db, args)).To(Succeed())
		Expect(fetchedAt).To(Equal([]string{repoName + ":v1.0.0"}))
		Expect(reload().ImageDigest).To(Equal("sha256:abc"))
	})

	It("skips when digest is already recorded", func() {
		rec.ImageDigest = "sha256:old"
		createRecord()
		Expect(recordImageDigest(ctx, db, args)).To(Succeed())
		Expect(fetchedAt).To(BeEmpty())
		Expect(reload().ImageDigest).To(Equal("sha256:old"))
	})

	It("keeps the digest written by a concurrent run", func() {
		createRecord()
		onFetch = func() {
			Expect(build.UpdateRecordImageDigest(ctx, records, rec.AppID, rec.BuildID, "sha256:first")).To(Succeed())
		}
		Expect(recordImageDigest(ctx, db, args)).To(Succeed())
		Expect(reload().ImageDigest).To(Equal("sha256:first"))
	})

	It("skips when build is not succeeded", func() {
		rec.Status = build.StatusFailed
		createRecord()
		Expect(recordImageDigest(ctx, db, args)).To(Succeed())
		Expect(fetchedAt).To(BeEmpty())
	})

	It("records digest of a long build detected late by polling", func() {
		// 长构建轮询间隔可达 5 分钟，发现成功时距构建结束已超过窗口，窗口从投递时起算故仍应记录
		rec.EndedAt = time.Now().Add(-imageDigestRecordWindow - time.Minute)
		createRecord()
		Expect(recordImageDigest(ctx, db, args)).To(Succeed())
		Expect(reload().ImageDigest).To(Equal("sha256:abc"))
	})

	It("stops retrying without reading registry when beyond the record window", func() {
		args.EnqueuedAt = time.Now().Add(-imageDigestRecordWindow - time.Minute)
		createRecord()
		err := recordImageDigest(ctx, db, args)
		Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeTrue())
		Expect(fetchedAt).To(BeEmpty())
		Expect(reload().ImageDigest).To(BeEmpty())
	})

	It("stops retrying when build record is missing", func() {
		err := recordImageDigest(ctx, db, args)
		Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeTrue())
		Expect(fetchedAt).To(BeEmpty())
	})

	It("skips when artifact is not in app image repo", func() {
		rec.Artifact = "other.example.com/bkms/demo:v1.0.0"
		createRecord()
		Expect(recordImageDigest(ctx, db, args)).To(Succeed())
		Expect(fetchedAt).To(BeEmpty())
		Expect(reload().ImageDigest).To(BeEmpty())
	})

	It("skips when image tag is empty", func() {
		rec.Params = map[string]string{}
		createRecord()
		Expect(recordImageDigest(ctx, db, args)).To(Succeed())
		Expect(fetchedAt).To(BeEmpty())
	})

	It("stops retrying when build config of the app is gone", func() {
		createRecord()
		resolveErr = errors.Join(errors.New("get app build config"), build.ErrConfigNotFound)
		err := recordImageDigest(ctx, db, args)
		Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeTrue())
		Expect(fetchedAt).To(BeEmpty())
	})

	It("returns a retryable error and keeps digest empty when registry fails", func() {
		createRecord()
		detailErr = errors.New("boom")
		err := recordImageDigest(ctx, db, args)
		Expect(err).To(MatchError(ContainSubstring("boom")))
		Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeFalse())
		Expect(reload().ImageDigest).To(BeEmpty())
	})
})
