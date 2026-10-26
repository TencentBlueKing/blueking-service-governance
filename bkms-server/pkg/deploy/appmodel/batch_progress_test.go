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

package appmodel

import (
	"context"
	"net/http/httptest"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"k8s.io/client-go/rest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

const (
	fakeOldRevision = "demo-7b697647cb"
	fakeNewRevision = "demo-d8db7cf84"
)

// fakeGameDeployWithStatus 生成带 status 的 GameDeployment，4 副本均就绪，其中 updated 个为新版本
func fakeGameDeployWithStatus(updated int64) map[string]any {
	currentRevision := lo.Ternary(updated == 4, fakeNewRevision, fakeOldRevision)
	return map[string]any{
		"apiVersion": "tkex.tencent.com/v1alpha1",
		"kind":       "GameDeployment",
		"metadata":   map[string]any{"name": "demo", "namespace": "demo-ns", "generation": 3},
		"spec":       map[string]any{"replicas": 4, "updateStrategy": map[string]any{"partition": 4 - updated}},
		"status": map[string]any{
			"observedGeneration":   3,
			"updateRevision":       fakeNewRevision,
			"currentRevision":      currentRevision,
			"replicas":             4,
			"readyReplicas":        4,
			"updatedReplicas":      updated,
			"updatedReadyReplicas": updated,
		},
	}
}

func fakeRolloutWithStatus(status map[string]any) map[string]any {
	return map[string]any{
		"apiVersion": batchrelease.APIVersion, "kind": batchrelease.Kind,
		"metadata": map[string]any{"name": "demo", "namespace": "demo-ns"},
		"status":   status,
	}
}

var _ = Describe("DeployStateGetter for batch release", func() {
	var (
		ctx    context.Context
		fake   *fakeCluster
		server *httptest.Server
		mocker *mockey.Mocker
		record *Record
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake = newFakeCluster()
		server = httptest.NewServer(fake)
		cfg := &cluster.Config{Rest: &rest.Config{Host: server.URL}, ClusterID: "BCS-K8S-00000"}
		mocker = mockey.Mock(cluster.NewConfig).Return(cfg).Build()

		record = &Record{
			ClusterID:    "BCS-K8S-00000",
			Namespace:    "demo-ns",
			ImageTag:     "v2",
			Replicas:     4,
			WorkloadKind: k8skind.GameDeploy,
			ResourceKeys: ResourceKeys{{Kind: k8skind.GameDeploy, Name: "demo"}},
			ReleaseMode:  ReleaseModeCanary,
			Status:       StatusDeploying,
			BatchProgress: &BatchProgress{
				BatchPhase: batchrelease.BatchPhasePending, Replicas: 4, OldImageTag: "v1", NewImageTag: "v2",
			},
		}
	})

	AfterEach(func() {
		mocker.UnPatch()
		server.Close()
	})

	It("reports the progress of a permanently paused batch", func() {
		fake.gameDeploys["demo"] = fakeGameDeployWithStatus(1)
		fake.rollouts["demo"] = fakeRolloutWithStatus(map[string]any{
			"phase":            "Running",
			"message":          "Rollout is running canary step 0",
			"paused":           "false",
			"workloadRevision": fakeNewRevision,
			"stepStatus": map[string]any{
				"currentStepIndex": 0,
				"currentStepPhase": "PausedPermanently",
				"pauseStartTime":   "2026-10-22T02:00:00Z",
			},
		})

		state, err := NewDeployStateGetter(record).Get(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(Equal(&DeployState{
			Status:  StatusDeploying,
			Message: "Rollout is running canary step 0",
			BatchProgress: &BatchProgress{
				CurrentBatchIndex: lo.ToPtr(int64(0)),
				BatchPhase:        batchrelease.BatchPhaseManualPaused,
				RolloutPhase:      "Running",
				Message:           "Rollout is running canary step 0",
				PauseStartTime:    lo.ToPtr(time.Date(2026, 10, 22, 2, 0, 0, 0, time.UTC)),
				UpdatedReplicas:   1,
				Replicas:          4,
				OldImageTag:       "v1",
				NewImageTag:       "v2",
			},
		}))
	})

	It("marks deployed once the rollout is healthy and all instances are updated", func() {
		fake.gameDeploys["demo"] = fakeGameDeployWithStatus(4)
		fake.rollouts["demo"] = fakeRolloutWithStatus(map[string]any{
			"phase": "Healthy", "paused": "false", "workloadRevision": fakeNewRevision, "stepStatus": map[string]any{},
		})

		state, err := NewDeployStateGetter(record).Get(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Status).To(Equal(StatusDeployed))
		Expect(state.BatchProgress.BatchPhase).To(Equal(batchrelease.BatchPhaseCompleted))
		Expect(state.BatchProgress.CurrentBatchIndex).To(BeNil())
		Expect(state.BatchProgress.UpdatedReplicas).To(Equal(int32(4)))
	})

	It("keeps deploying while the healthy rollout still tracks the old revision", func() {
		fake.gameDeploys["demo"] = fakeGameDeployWithStatus(0)
		fake.rollouts["demo"] = fakeRolloutWithStatus(map[string]any{
			"phase": "Healthy", "workloadRevision": fakeOldRevision,
		})

		state, err := NewDeployStateGetter(record).Get(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Status).To(Equal(StatusDeploying))
		Expect(state.BatchProgress.BatchPhase).To(Equal(batchrelease.BatchPhasePending))
	})

	It("marks rolled back once all instances are back to the old revision", func() {
		fake.gameDeploys["demo"] = fakeGameDeployWithStatus(0)
		fake.rollouts["demo"] = fakeRolloutWithStatus(map[string]any{
			"phase":            "RollbackCompleted",
			"message":          "Rollout has been rolled back successfully",
			"workloadRevision": fakeNewRevision,
			"stepStatus":       map[string]any{"currentStepIndex": -1, "currentStepPhase": "PausedPermanently"},
		})

		state, err := NewDeployStateGetter(record).Get(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Status).To(Equal(StatusRolledBack))
		Expect(state.BatchProgress.CurrentBatchIndex).To(Equal(lo.ToPtr(int64(-1))))
		Expect(state.BatchProgress.NewImageTag).To(Equal("v2"))
	})

	It("keeps deploying while rolled back instances are still on the new revision", func() {
		fake.gameDeploys["demo"] = fakeGameDeployWithStatus(1)
		fake.rollouts["demo"] = fakeRolloutWithStatus(map[string]any{
			"phase":            "RollbackCompleted",
			"workloadRevision": fakeNewRevision,
			"stepStatus":       map[string]any{"currentStepIndex": -1},
		})

		state, err := NewDeployStateGetter(record).Get(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Status).To(Equal(StatusDeploying))
		Expect(state.BatchProgress.BatchPhase).To(Equal(batchrelease.BatchPhaseUpdating))
	})

	It("shows the step message when the batch fails", func() {
		fake.gameDeploys["demo"] = fakeGameDeployWithStatus(1)
		fake.rollouts["demo"] = fakeRolloutWithStatus(map[string]any{
			"phase":   "Running",
			"message": "Rollout is running canary step 1",
			"stepStatus": map[string]any{
				"currentStepIndex": 1, "currentStepPhase": "Error", "message": "failed to scale canary replicas",
			},
		})

		state, err := NewDeployStateGetter(record).Get(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Status).To(Equal(StatusDeploying))
		Expect(state.Message).To(Equal("failed to scale canary replicas"))
		Expect(state.BatchProgress.Message).To(Equal("failed to scale canary replicas"))
	})

	It("returns an error when the rollout cannot be read", func() {
		fake.gameDeploys["demo"] = fakeGameDeployWithStatus(1)

		_, err := NewDeployStateGetter(record).Get(ctx)
		Expect(err).To(MatchError(batchrelease.ErrRolloutNotFound))
	})

	It("marks failed when the main workload is missing", func() {
		state, err := NewDeployStateGetter(record).Get(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(state.Status).To(Equal(StatusFailed))
		Expect(state.BatchProgress).To(BeNil())
	})
})

var _ = Describe("createDeployRecord for batch release", func() {
	var (
		ctx      context.Context
		deployer *Deployer
		env      *bkmsenv.Environment
		records  *RecordStore
		appID    string
	)

	strategy := &releasestrategy.ReleaseStrategy{
		Name: "careful",
		Batches: []releasestrategy.Batch{
			{Target: "1", PauseMode: releasestrategy.PauseModePermanent},
			{Target: "50%", PauseMode: releasestrategy.PauseModeNone},
		},
	}

	createRecord := func(strategy *releasestrategy.ReleaseStrategy) *Record {
		deployID, err := deployer.createDeployRecord(
			ctx, env, "", "v3", "", 4, nil, &workload.BuildResult{WorkloadKind: k8skind.GameDeploy},
			ResourceKeys{{Kind: k8skind.GameDeploy, Name: "demo"}}, strategy,
		)
		Expect(err).NotTo(HaveOccurred())
		record, err := GetRecord(ctx, deployer.db, appID, deployID)
		Expect(err).NotTo(HaveOccurred())
		return record
	}

	BeforeEach(func() {
		ctx = auth.WithUser(context.Background(), auth.User{ID: "tester"})
		db := testutil.NewDB(GinkgoT())
		records = store.Of(db, RecordMeta)
		appID = "batch-progress-" + stringx.Random(6)
		DeferCleanup(func() {
			_, _ = records.DeleteMany(context.Background(), store.Where("appID", appID))
		})
		deployer = &Deployer{db: db, app: &bkmsapp.Application{ID: appID}}
		env = &bkmsenv.Environment{Name: "prod"}
	})

	It("writes the initial progress with the image of the latest deployed record", func() {
		Expect(records.Create(ctx, &Record{AppID: appID, EnvName: "prod", ImageTag: "v1", Status: StatusDeployed})).
			To(Succeed())
		// 最新记录已回滚，Pod 仍运行最近一次已部署的镜像
		Expect(records.Create(ctx, &Record{
			AppID: appID, EnvName: "prod", ImageTag: "v2", Status: StatusRolledBack, CreatedAt: time.Now(),
		})).To(Succeed())

		record := createRecord(strategy)
		Expect(record.ReleaseMode).To(Equal(ReleaseModeCanary))
		Expect(record.BatchProgress).To(Equal(&BatchProgress{
			BatchPhase: batchrelease.BatchPhasePending, Replicas: 4, OldImageTag: "v1", NewImageTag: "v3",
		}))
	})

	It("leaves the old image empty on the first deploy", func() {
		record := createRecord(strategy)
		Expect(record.BatchProgress.OldImageTag).To(BeEmpty())
		Expect(record.BatchProgress.NewImageTag).To(Equal("v3"))
	})
})
