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
	"time"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

// newGameDeploy 创建用于测试的 GameDeployment，支持预设两个层级的 Annotations
func newGameDeploy(objAnnotations, tplAnnotations map[string]string) *tkex.GameDeployment {
	return &tkex.GameDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app", Annotations: objAnnotations},
		Spec: tkex.GameDeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: tplAnnotations},
			},
		},
	}
}

var _ = Describe("injectDeployID", func() {
	deployer := &Deployer{}

	newResult := func(objAnnotations, tplAnnotations map[string]string) (*workload.BuildResult, *tkex.GameDeployment) {
		gd := newGameDeploy(objAnnotations, tplAnnotations)
		return &workload.BuildResult{MainWorkload: gd}, gd
	}

	// expectBothLevels 断言两个层级的 Annotations 都包含预期的 deployID
	expectBothLevels := func(gd *tkex.GameDeployment, deployID string) {
		Expect(gd.Annotations[AnnotationKeyDeployID]).To(Equal(deployID))
		Expect(gd.Spec.Template.Annotations[AnnotationKeyDeployID]).To(Equal(deployID))
	}

	It("should initialize nil Annotations and inject deployID at both levels", func() {
		result, gd := newResult(nil, nil)
		deployer.injectDeployID(result, "deploy-001")
		expectBothLevels(gd, "deploy-001")
	})

	It("should append deployID without overwriting existing Annotations", func() {
		result, gd := newResult(
			map[string]string{"existing-key": "existing-value"},
			map[string]string{"pod-existing-key": "pod-existing-value"},
		)
		deployer.injectDeployID(result, "deploy-002")
		expectBothLevels(gd, "deploy-002")
		Expect(gd.Annotations).To(HaveKeyWithValue("existing-key", "existing-value"))
		Expect(gd.Spec.Template.Annotations).To(HaveKeyWithValue("pod-existing-key", "pod-existing-value"))
	})

	It("should overwrite the previous deployID with the new one", func() {
		result, gd := newResult(nil, nil)
		deployer.injectDeployID(result, "deploy-first")
		deployer.injectDeployID(result, "deploy-second")
		expectBothLevels(gd, "deploy-second")
	})
})

var _ = Describe("getExpiredResources", func() {
	const envName, laneName = "stag", ""
	var (
		ctx      context.Context
		db       *mongo.Database
		deployer *Deployer
		appID    string
		baseTime time.Time
	)
	gdKey := ResourceKey{Kind: k8skind.GameDeploy, Name: "demo"}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		appID = "app-" + stringx.Random(6)
		deployer = &Deployer{db: db, app: &bkmsapp.Application{ID: appID}}
		baseTime = time.Now().Add(-time.Hour)
		records := store.Of(db, RecordMeta)
		DeferCleanup(func() { _, _ = records.DeleteMany(ctx, store.Where("appID", appID)) })
	})

	// createRecords 按传入顺序（由旧到新）创建部署记录
	createRecords := func(records ...Record) {
		for i := range records {
			records[i].AppID = appID
			records[i].EnvName = envName
			records[i].TrafficLaneName = laneName
			records[i].CreatedAt = baseTime.Add(time.Duration(i) * time.Minute)
			Expect(store.Of(db, RecordMeta).Create(ctx, &records[i])).To(Succeed())
		}
	}

	It("should return nothing for the first deployment", func() {
		expired, err := deployer.getExpiredResources(
			ctx, envName, laneName, ResourceKeys{gdKey, configMapKey("cm-a")},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(expired).To(BeEmpty())
	})

	It("should delete removed resources immediately but keep the previous ConfigMap", func() {
		svcKey := ResourceKey{Kind: k8skind.SVC, Name: "demo-svc"}
		createRecords(Record{
			Status: StatusDeployed, ResourceKeys: ResourceKeys{gdKey, svcKey, configMapKey("cm-a")},
		})

		expired, err := deployer.getExpiredResources(
			ctx, envName, laneName, ResourceKeys{gdKey, configMapKey("cm-b")},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(expired).To(Equal(ResourceKeys{svcKey}))
	})

	It("should clean up ConfigMaps older than the latest deployed record", func() {
		createRecords(
			Record{Status: StatusDeployed, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-z")}},
			Record{Status: StatusDeployed, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-a")}},
			Record{Status: StatusCanceled, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-b")}},
		)

		expired, err := deployer.getExpiredResources(
			ctx, envName, laneName, ResourceKeys{gdKey, configMapKey("cm-c")},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(expired).To(Equal(ResourceKeys{configMapKey("cm-z")}))
	})

	It("should keep an old ConfigMap reused by the current deployment", func() {
		createRecords(
			Record{Status: StatusDeployed, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-z")}},
			Record{Status: StatusDeployed, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-a")}},
		)

		// 配置改回 cm-z 对应的内容，ConfigMap 名称与旧版本相同
		expired, err := deployer.getExpiredResources(
			ctx, envName, laneName, ResourceKeys{gdKey, configMapKey("cm-z")},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(expired).To(BeEmpty())
	})

	It("should only look at records of the same env and traffic lane", func() {
		createRecords(Record{Status: StatusDeployed, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-z")}})
		other := Record{
			AppID: appID, EnvName: envName, TrafficLaneName: "lane-x", Status: StatusDeployed,
			ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-x")}, CreatedAt: time.Now(),
		}
		Expect(store.Of(db, RecordMeta).Create(ctx, &other)).To(Succeed())

		expired, err := deployer.getExpiredResources(
			ctx, envName, laneName, ResourceKeys{gdKey, configMapKey("cm-a")},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(expired).To(BeEmpty())
	})
})

var _ = Describe("syncBuildAutoDeployStatus", func() {
	It("should update linked build auto deploy record by deploy ID", func() {
		ctx := testutil.NewDefaultTenantContext()
		db := testutil.NewDB(GinkgoT())
		records := store.Of(db, autodeploy.RecordMeta)
		appID := "app-" + stringx.Random(6)
		DeferCleanup(func() { _, _ = records.DeleteMany(ctx, store.Where("appID", appID)) })

		deployID := bson.NewObjectID()
		record := &autodeploy.Record{
			AppID:    appID,
			BuildID:  "build-" + stringx.Random(6),
			DeployID: deployID.Hex(),
			Stage:    autodeploy.StageDeploy,
			Status:   "deploying",
		}
		Expect(records.Create(ctx, record)).To(Succeed())
		deployer := &Deployer{
			db:  db,
			app: &bkmsapp.Application{ID: appID},
		}
		deployRecord := &Record{
			ID:      deployID,
			Status:  StatusUninstalled,
			Message: "done",
			EndedAt: time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC),
		}

		err := deployer.syncBuildAutoDeployStatus(ctx, deployRecord)

		Expect(err).NotTo(HaveOccurred())
		got, err := records.Get(ctx, autodeploy.ByDeploy(appID, deployID.Hex()))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Status).To(Equal(string(StatusUninstalled)))
		Expect(got.Stage).To(Equal(autodeploy.StageDeploy))
		Expect(got.Message).To(Equal("done"))
		Expect(got.EndedAt).To(BeTemporally("==", deployRecord.EndedAt))
	})
})

var _ = Describe("Deploy tracks app in env", func() {
	var (
		ctx      context.Context
		db       *mongo.Database
		env      *bkmsenv.Environment
		deployer *Deployer
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		envs := store.Of(db, bkmsenv.EnvMeta)
		env = &bkmsenv.Environment{WorkspaceID: "ws-" + stringx.Random(6), Name: "feat-" + stringx.Random(6)}
		Expect(envs.Create(ctx, env)).To(Succeed())
		DeferCleanup(func() { _ = envs.Delete(ctx, store.ByID(env.ID)) })
		deployer = &Deployer{db: db, app: &bkmsapp.Application{ID: "app-" + stringx.Random(6)}}
	})

	deploy := func() error {
		_, err := deployer.Deploy(ctx, &appmodel.AppModel{}, env, nil, "", "v1", "", 1, nil, nil)
		return err
	}
	trackedAppIDs := func() []string {
		got, err := store.Of(db, bkmsenv.EnvMeta).Get(ctx, store.ByID(env.ID))
		Expect(err).NotTo(HaveOccurred())
		return got.AppIDs
	}

	It("does not track the app when deploy fails before creating the deploy record", func() {
		mockey.PatchConvey("build fails", GinkgoT(), func() {
			mockey.Mock((*Deployer).buildWorkload).Return(nil, errors.New("components not supported")).Build()

			Expect(deploy()).To(MatchError(ContainSubstring("components not supported")))
			Expect(trackedAppIDs()).NotTo(ContainElement(deployer.app.ID))
		})
	})

	It("tracks the app once the deploy record is created, even if applying resources fails", func() {
		mockey.PatchConvey("apply fails", GinkgoT(), func() {
			mockey.Mock((*Deployer).buildWorkload).Return(&workload.BuildResult{
				MainWorkload: newGameDeploy(nil, nil), WorkloadKind: k8skind.GameDeploy,
			}, nil).Build()
			// 单测没有 kubeconfig。集群配置在创建部署记录之前获取，不 mock 会在这里 panic
			mockey.Mock((*Deployer).clusterConfig).Return(&cluster.Config{}, nil).Build()
			mockey.Mock(newWorkloadRollout).Return(&workloadRollout{}, nil).Build()
			mockey.Mock((*Deployer).claimOwnership).Return(nil).Build()
			mockey.Mock((*Deployer).getExpiredResources).Return(nil, nil).Build()
			mockey.Mock((*Deployer).createDeployRecord).Return(bson.NewObjectID().Hex(), nil).Build()
			mockey.Mock((*Deployer).markRecordFailed).To(func(*Deployer, context.Context, string, error) {}).Build()
			mockey.Mock((*Deployer).syncImagePullSecret).Return(errors.New("sync secret failed")).Build()

			Expect(deploy()).To(MatchError(ContainSubstring("sync secret failed")))
			Expect(trackedAppIDs()).To(ContainElement(deployer.app.ID))
		})
	})
})
