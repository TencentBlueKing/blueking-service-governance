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
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"k8s.io/client-go/rest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// batchReleaseFixture 一个处于分批发布中的环境：测试库中的部署记录与模拟集群中的同名 Rollout
type batchReleaseFixture struct {
	ctx    context.Context
	db     *mongo.Database
	fake   *fakeCluster
	appID  string
	record *Record
}

// newBatchReleaseFixture 准备分批发布中的部署记录（尚未写库，由 createRecord 写入）与模拟集群；
// 部署记录使用 2 批的策略，主工作负载为 GameDeployment demo
func newBatchReleaseFixture() *batchReleaseFixture {
	f := &batchReleaseFixture{
		ctx:   auth.WithUser(context.Background(), auth.User{ID: "tester"}),
		db:    testutil.NewDB(GinkgoT()),
		fake:  newFakeCluster(),
		appID: "batch-release-op-" + stringx.Random(6),
	}
	server := httptest.NewServer(f.fake)
	cfg := &cluster.Config{Rest: &rest.Config{Host: server.URL}, ClusterID: "BCS-K8S-00000"}
	mocker := mockey.Mock(cluster.NewConfig).Return(cfg).Build()
	DeferCleanup(func() {
		mocker.UnPatch()
		server.Close()
		_, _ = store.Of(f.db, RecordMeta).DeleteMany(context.Background(), store.Where("appID", f.appID))
	})

	f.record = &Record{
		AppID:        f.appID,
		EnvName:      "prod",
		ClusterID:    "BCS-K8S-00000",
		Namespace:    "demo-ns",
		ImageTag:     "v2",
		Replicas:     4,
		WorkloadKind: k8skind.GameDeploy,
		ResourceKeys: ResourceKeys{{Kind: k8skind.GameDeploy, Name: "demo"}},
		ReleaseMode:  ReleaseModeCanary,
		ReleaseStrategy: &ReleaseStrategySnapshot{
			ID:   "6717a0e1c2a4b3e5d6f70801",
			Name: "careful",
			Batches: []releasestrategy.Batch{
				{Target: "1", PauseMode: releasestrategy.PauseModePermanent},
				{Target: "50%", PauseMode: releasestrategy.PauseModeNone},
			},
		},
		Status: StatusDeploying,
	}
	return f
}

func (f *batchReleaseFixture) createRecord() {
	Expect(store.Of(f.db, RecordMeta).Create(f.ctx, f.record)).To(Succeed())
}

// setRolloutStatus 设置集群中 Rollout 的 status：paused 为 status.paused，stepPhase 为当前批次阶段，
// 暂停中的批次带上暂停开始时间。Rollout 标记为由当前部署记录下发
func (f *batchReleaseFixture) setRolloutStatus(paused, stepPhase string, stepIndex int64) {
	stepStatus := map[string]any{"currentStepIndex": stepIndex, "currentStepPhase": stepPhase}
	if paused != batchrelease.PausedFalse {
		stepStatus["pauseStartTime"] = "2026-10-22T02:00:00Z"
	}
	f.fake.rollouts["demo"] = map[string]any{
		"apiVersion": batchrelease.APIVersion, "kind": batchrelease.Kind,
		"metadata": map[string]any{
			"name": "demo", "namespace": "demo-ns", "resourceVersion": "1",
			"annotations": map[string]any{AnnotationKeyDeployID: f.record.ID.Hex()},
		},
		"status": map[string]any{
			"phase": batchrelease.RolloutPhaseRunning, "paused": paused, "stepStatus": stepStatus,
		},
	}
}

func (f *batchReleaseFixture) rolloutStatus() map[string]any {
	return f.fake.rollouts["demo"]["status"].(map[string]any)
}

var _ = Describe("OperateBatchRelease", func() {
	var f *batchReleaseFixture

	BeforeEach(func() {
		f = newBatchReleaseFixture()
	})

	operate := func(action batchrelease.Action) (*Record, error) {
		return OperateBatchRelease(f.ctx, f.db, f.appID, "prod", "", action)
	}

	DescribeTable("patches the rollout status when the action is allowed",
		func(paused, stepPhase string, stepIndex int64, action batchrelease.Action, expected map[string]any) {
			f.createRecord()
			f.setRolloutStatus(paused, stepPhase, stepIndex)

			record, err := operate(action)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.ID).To(Equal(f.record.ID))

			status := f.rolloutStatus()
			stepStatus := status["stepStatus"].(map[string]any)
			Expect(status).To(HaveKeyWithValue("paused", expected["paused"]))
			Expect(stepStatus).To(HaveKeyWithValue("currentStepIndex", BeNumerically("==", expected["stepIndex"])))
			if action == batchrelease.ActionBackward {
				// 回退时清空暂停开始时间，否则 Kourse 会当成本批已暂停过而直接跳回下一批
				Expect(stepStatus).NotTo(HaveKey("pauseStartTime"))
			}
		},
		Entry("continue a permanent pause",
			batchrelease.PausedTrue, batchrelease.StepPhasePausedPermanently, int64(0), batchrelease.ActionContinue,
			map[string]any{"paused": batchrelease.PausedFalse, "stepIndex": 0}),
		Entry("continue a timed pause ahead of time",
			batchrelease.PausedTemporary, batchrelease.StepPhasePausedTemporarily, int64(0),
			batchrelease.ActionContinue, map[string]any{"paused": batchrelease.PausedFalse, "stepIndex": 0}),
		Entry("pause a batch being executed",
			batchrelease.PausedFalse, batchrelease.StepPhaseScalingReplicas, int64(0), batchrelease.ActionPause,
			map[string]any{"paused": batchrelease.PausedTrue, "stepIndex": 0}),
		Entry("go back to the previous batch",
			batchrelease.PausedTrue, batchrelease.StepPhasePausedPermanently, int64(1), batchrelease.ActionBackward,
			map[string]any{"paused": batchrelease.PausedFalse, "stepIndex": 0}),
		Entry("skip the remaining batches",
			batchrelease.PausedTrue, batchrelease.StepPhasePausedPermanently, int64(0), batchrelease.ActionSkip,
			map[string]any{"paused": batchrelease.PausedFalse, "stepIndex": 2}),
		Entry("roll back",
			batchrelease.PausedFalse, batchrelease.StepPhaseScalingReplicas, int64(1), batchrelease.ActionRollback,
			map[string]any{"paused": batchrelease.PausedFalse, "stepIndex": -1}),
	)

	DescribeTable("rejects the action without patching the rollout when the precondition is not met",
		func(paused, stepPhase string, stepIndex int64, action batchrelease.Action, reason string) {
			f.createRecord()
			f.setRolloutStatus(paused, stepPhase, stepIndex)

			_, err := operate(action)
			Expect(errors.Is(err, batchrelease.ErrActionNotAllowed)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring(reason))
			Expect(f.fake.requestsWith(http.MethodPatch)).To(BeEmpty())
		},
		Entry("continue when not paused",
			batchrelease.PausedFalse, batchrelease.StepPhaseScalingReplicas, int64(0), batchrelease.ActionContinue,
			"current batch is not paused"),
		Entry("pause when already paused",
			batchrelease.PausedTrue, batchrelease.StepPhasePausedPermanently, int64(0), batchrelease.ActionPause,
			"current batch is already paused"),
		Entry("go back from the first batch",
			batchrelease.PausedTrue, batchrelease.StepPhasePausedPermanently, int64(0), batchrelease.ActionBackward,
			"current batch is the first batch"),
	)

	DescribeTable("rejects the action without requesting the cluster when no batch release is in progress",
		func(mutate func(*Record)) {
			if mutate != nil {
				mutate(f.record)
				f.createRecord()
			}

			_, err := operate(batchrelease.ActionRollback)
			Expect(errors.Is(err, batchrelease.ErrNoBatchReleaseInProgress)).To(BeTrue())
			Expect(f.fake.requestsWith("/")).To(BeEmpty())
		},
		Entry("without any deploy record", (func(*Record))(nil)),
		Entry("when the batch release is completed", func(r *Record) { r.Status = StatusDeployed }),
		Entry("when the batch release is rolled back", func(r *Record) { r.Status = StatusRolledBack }),
		Entry("when the latest deploy released all at once", func(r *Record) { r.ReleaseMode = ReleaseModeAll }),
	)

	It("only takes the latest deploy record into account", func() {
		f.createRecord()
		// 分批发布之后又发起了一次更新全部的部署
		newer := *f.record
		newer.ID = bson.ObjectID{}
		newer.ReleaseMode = ReleaseModeAll
		newer.ReleaseStrategy = nil
		newer.Status = StatusDeployed
		Expect(store.Of(f.db, RecordMeta).Create(f.ctx, &newer)).To(Succeed())

		_, err := operate(batchrelease.ActionContinue)
		Expect(errors.Is(err, batchrelease.ErrNoBatchReleaseInProgress)).To(BeTrue())
	})

	DescribeTable("rejects the action without patching when the rollout does not belong to the latest deploy record",
		func(annotations map[string]any) {
			f.createRecord()
			f.setRolloutStatus(batchrelease.PausedTrue, batchrelease.StepPhasePausedPermanently, 0)
			metadata := f.fake.rollouts["demo"]["metadata"].(map[string]any)
			if annotations == nil {
				delete(metadata, "annotations")
			} else {
				metadata["annotations"] = annotations
			}

			_, err := operate(batchrelease.ActionSkip)
			Expect(errors.Is(err, batchrelease.ErrReleaseSwitching)).To(BeTrue())
			Expect(f.fake.requestsWith(http.MethodPatch)).To(BeEmpty())
		},
		// 再次部署已创建新记录，集群中仍是上一次发布的 Rollout
		Entry("left by the previous deploy", map[string]any{AnnotationKeyDeployID: bson.NewObjectID().Hex()}),
		Entry("without the deploy ID annotation", (map[string]any)(nil)),
	)

	It("rejects the action without retrying when the rollout is modified after being read", func() {
		f.createRecord()
		f.setRolloutStatus(batchrelease.PausedTemporary, batchrelease.StepPhasePausedTemporarily, 0)
		// 读取之后定时暂停恰好到期，控制器进入下一批
		f.fake.afterRolloutGet = func(rollout map[string]any) {
			rollout["status"] = map[string]any{
				"phase": batchrelease.RolloutPhaseRunning, "paused": batchrelease.PausedFalse,
				"stepStatus": map[string]any{
					"currentStepIndex": int64(1), "currentStepPhase": batchrelease.StepPhaseScalingReplicas,
				},
			}
			bumpFakeResourceVersion(rollout)
		}

		_, err := operate(batchrelease.ActionContinue)
		Expect(errors.Is(err, batchrelease.ErrRolloutChanged)).To(BeTrue())
		Expect(f.fake.requestsWith(http.MethodPatch)).To(HaveLen(1))
		Expect(f.rolloutStatus()["stepStatus"]).To(HaveKeyWithValue("currentStepIndex", int64(1)))
	})

	It("returns the cluster error when the rollout does not exist", func() {
		f.createRecord()

		_, err := operate(batchrelease.ActionContinue)
		Expect(errors.Is(err, batchrelease.ErrRolloutNotFound)).To(BeTrue())
	})
})

var _ = Describe("Deployer during a batch release", func() {
	var (
		f        *batchReleaseFixture
		deployer *Deployer
		env      *bkmsenv.Environment
	)

	BeforeEach(func() {
		f = newBatchReleaseFixture()
		deployer = &Deployer{db: f.db, app: &bkmsapp.Application{ID: f.appID}, podManager: &podManager{}}
		env = &bkmsenv.Environment{Name: "prod"}
	})

	DescribeTable("rejects the operations conflicting with the rollout without requesting the cluster",
		func(operate func() error) {
			f.createRecord()

			err := operate()
			Expect(errors.Is(err, batchrelease.ErrBatchReleaseInProgress)).To(BeTrue())
			Expect(f.fake.requestsWith("/")).To(BeEmpty())
			latest, err := GetLatest(f.ctx, f.db, f.appID, "prod", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.ID).To(Equal(f.record.ID))
			Expect(latest.Status).To(Equal(StatusDeploying))
		},
		Entry("grayscale update instances", func() error {
			_, err := deployer.UpdateInstances(f.ctx, "prod", "", "v3", "", []string{"demo-0"})
			return err
		}),
		Entry("delete instances", func() error {
			return deployer.BatchDeleteInstances(f.ctx, "prod", "", []string{"demo-0"})
		}),
		Entry("scale", func() error {
			return deployer.Scale(f.ctx, "prod", "", 2)
		}),
	)

	It("uninstalls and deletes the rollout", func() {
		f.createRecord()
		f.setRolloutStatus(batchrelease.PausedTrue, batchrelease.StepPhasePausedPermanently, 0)
		f.fake.gameDeploys["demo"] = map[string]any{
			"apiVersion": "tkex.tencent.com/v1alpha1", "kind": k8skind.GameDeploy,
			"metadata": map[string]any{"name": "demo", "namespace": "demo-ns"},
		}
		mocker := mockey.Mock(discovery.GetGroupVersionResource).Return(&gvr.GameDeploy, nil).Build()
		DeferCleanup(mocker.UnPatch)
		deployer.polarisEnvStateManager = polaris.NewPolarisEnvStateManager(f.db)
		deployer.hostPortEnvStateManager = hostport.NewEnvStateManager(f.db)

		Expect(deployer.Uninstall(f.ctx, env, "")).To(Succeed())
		Expect(f.fake.rollouts).NotTo(HaveKey("demo"))
		Expect(f.fake.gameDeploys).NotTo(HaveKey("demo"))
		latest, err := GetLatest(f.ctx, f.db, f.appID, "prod", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(latest.Status).To(Equal(StatusUninstalled))
	})
})
