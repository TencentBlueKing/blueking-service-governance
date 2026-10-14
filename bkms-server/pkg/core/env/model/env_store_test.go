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

package model_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// envNames 提取环境名称，便于断言列表内容
func envNames(envs []envmodel.Environment) []string {
	return lo.Map(envs, func(env envmodel.Environment, _ int) string { return env.Name })
}

func newTestEnv(workspaceID, envType string) *envmodel.Environment {
	return &envmodel.Environment{
		WorkspaceID: workspaceID, Name: stringx.Random(10), DisplayName: stringx.Random(10),
		Cluster: envmodel.BizCluster{
			ProjectCode: stringx.Random(10),
			ClusterID:   stringx.Random(10),
			ClusterType: stringx.Random(10),
			Namespace:   stringx.Random(10),
		},
		CreatedAt: time.Now(), Type: envType,
	}
}

var _ = Describe("Environment store", func() {
	var ctx context.Context

	var diApp *fxtest.App
	var db *mongo.Database
	var envs *envmodel.EnvironmentStore
	var appStore *bkmsapp.ApplicationStore
	var envSvc *bkmsenv.EnvService

	var workspaceID1, workspaceID2 string
	var env1, env2 *envmodel.Environment

	BeforeEach(func() {
		ctx = context.Background()

		db = testutil.NewDB(GinkgoT())
		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		envs = testutil.NewStore(GinkgoT(), envmodel.EnvMeta)
		diApp = fxtest.New(GinkgoT(), bkmsenv.FxModule, fx.Populate(&envSvc))
		diApp.RequireStart()

		workspaceID1 = stringx.Random(10)
		workspaceID2 = stringx.Random(10)
		env1 = newTestEnv(workspaceID1, "development")
		env2 = newTestEnv(workspaceID2, "test")
		Expect(envs.Create(ctx, env1)).To(Succeed())
		Expect(envs.Create(ctx, env2)).To(Succeed())
	})
	AfterEach(func() {
		Expect(testutil.CleanupCollection("environments")).To(Succeed())
		Expect(testutil.CleanupCollection("applications")).To(Succeed())
		diApp.RequireStop()
	})

	newApp := func(prefix, workspaceID string) *bkmsapp.Application {
		return dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
			ID:          prefix + "-app-id-" + stringx.Random(6),
			Name:        prefix + "-app-" + stringx.Random(6),
			WorkspaceID: workspaceID,
		})
	}

	It("fills defaults on create and rejects duplicate names", func() {
		Expect(env1.Kind).To(Equal(envmodel.EnvironmentKindStandard))
		Expect(env1.AppIDs).To(BeEmpty())

		dup := newTestEnv(workspaceID1, "development")
		dup.Name = env1.Name
		Expect(envs.Create(ctx, dup)).To(MatchError(envmodel.ErrEnvAlreadyExists))
	})

	It("lists standard environments in a workspace", func() {
		// 查询 workspace1 下的环境
		environments, err := envs.List(ctx, envmodel.StdEnvsIn(workspaceID1), envmodel.NewestFirst())
		Expect(err).NotTo(HaveOccurred())
		Expect(environments).To(HaveLen(1))

		expectedEnv1 := *env1
		// 去除时间精度差异. 不影响测试效果
		expectedEnv1.CreatedAt = environments[0].CreatedAt
		expectedEnv1.UpdatedAt = environments[0].UpdatedAt
		expectedEnv1.Status = envmodel.EnvStatusReady
		Expect(environments[0]).To(Equal(expectedEnv1))

		// 查询 workspace2 下的环境
		environments, err = envs.List(ctx, envmodel.StdEnvsIn(workspaceID2), envmodel.NewestFirst())
		Expect(err).NotTo(HaveOccurred())
		Expect(environments).To(HaveLen(1))

		expectedEnv2 := *env2
		// 去除时间精度差异. 不影响测试效果
		expectedEnv2.CreatedAt = environments[0].CreatedAt
		expectedEnv2.UpdatedAt = environments[0].UpdatedAt
		expectedEnv2.Status = envmodel.EnvStatusReady
		Expect(environments[0]).To(Equal(expectedEnv2))
	})

	It("counts standard environments by workspace IDs", func() {
		ownerApp := newApp("owner", workspaceID1)
		_ = dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)

		counts, err := envmodel.CountByWorkspaceIDs(ctx, db, []string{workspaceID1, workspaceID2, stringx.Random(10)})
		Expect(err).NotTo(HaveOccurred())
		Expect(counts).To(Equal(map[string]int{workspaceID1: 1, workspaceID2: 1}))
	})

	It("lists app environments with standard and owned feature envs only", func() {
		ownerApp := newApp("owner", workspaceID1)
		otherApp := newApp("other", workspaceID1)

		ownedFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
		otherFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, otherApp, env1)

		appEnvs, err := envs.List(ctx,
			envmodel.InWorkspace(workspaceID1), envmodel.VisibleToApp(ownerApp.ID), envmodel.NewestFirst(),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames(appEnvs)).To(ConsistOf(env1.Name, ownedFeatureEnv.Name))
		Expect(envNames(appEnvs)).NotTo(ContainElement(otherFeatureEnv.Name))
	})

	It("lists environments bound to an app by ids", func() {
		ownerApp := newApp("owner", workspaceID1)
		otherApp := newApp("other", workspaceID1)

		ownedFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
		otherFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, otherApp, env1)
		Expect(envmodel.AddApp(ctx, db, env1.ID, ownerApp.ID)).To(Succeed())

		ids := []bson.ObjectID{env1.ID, ownedFeatureEnv.ID, otherFeatureEnv.ID}
		result, err := envs.List(ctx,
			envmodel.InWorkspace(workspaceID1),
			store.Where("_id", bson.M{"$in": ids}),
			envmodel.BoundToApp(ownerApp.ID),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames(result)).To(ConsistOf(env1.Name, ownedFeatureEnv.Name))
	})

	It("lists environments bound to an app by types", func() {
		ownerApp := newApp("owner", workspaceID1)
		otherApp := newApp("other", workspaceID1)

		prodStdEnv := dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
			WorkspaceID: workspaceID1,
			Type:        "production",
			AppIDs:      []string{ownerApp.ID},
		})
		devStdEnv := dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
			WorkspaceID: workspaceID1,
			Type:        "development",
			AppIDs:      []string{ownerApp.ID},
		})
		prodFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, prodStdEnv)
		_ = dbfactory.FeatEnv(ctx, envSvc, otherApp, prodStdEnv)

		result, err := envs.List(ctx,
			envmodel.InWorkspace(workspaceID1),
			store.Where("type", bson.M{"$in": []string{"production"}}),
			envmodel.BoundToApp(ownerApp.ID),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames(result)).To(ConsistOf(prodStdEnv.Name, prodFeatureEnv.Name))
		Expect(envNames(result)).NotTo(ContainElement(devStdEnv.Name))
	})

	It("lists only feature environments owned by the requested app", func() {
		ownerApp := newApp("owner", workspaceID1)
		otherApp := newApp("other", workspaceID1)

		firstOwnedEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
		secondOwnedEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
		_ = dbfactory.FeatEnv(ctx, envSvc, otherApp, env1)

		featureEnvs, err := envs.List(ctx, envmodel.FeatEnvsOf(ownerApp.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames(featureEnvs)).To(ConsistOf(firstOwnedEnv.Name, secondOwnedEnv.Name))
		for _, featureEnv := range featureEnvs {
			Expect(featureEnv.GetKind()).To(Equal(envmodel.EnvironmentKindFeature))
			Expect(featureEnv.OwnerAppID).To(Equal(ownerApp.ID))
		}

		empty, err := envs.List(ctx, envmodel.FeatEnvsOf("missing-app"))
		Expect(err).NotTo(HaveOccurred())
		Expect(empty).To(BeEmpty())
	})

	It("lists batch app environments with shared standard envs and owned feature envs", func() {
		firstApp := newApp("first", workspaceID1)
		secondApp := newApp("second", workspaceID1)
		otherWorkspaceApp := newApp("other-workspace", workspaceID2)

		firstFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, firstApp, env1)
		secondFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, secondApp, env1)
		_ = dbfactory.FeatEnv(ctx, envSvc, otherWorkspaceApp, env2)

		result, err := envmodel.ListBatchAppEnvs(
			ctx,
			db,
			workspaceID1,
			[]string{firstApp.ID, secondApp.ID, firstApp.ID},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(envNames(result)).To(ConsistOf(env1.Name, firstFeatureEnv.Name, secondFeatureEnv.Name))

		empty, err := envmodel.ListBatchAppEnvs(ctx, db, workspaceID1, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(empty).To(BeEmpty())
	})

	It("gets standard or owned feature env by name", func() {
		ownerApp := newApp("owner", workspaceID1)
		otherApp := newApp("other", workspaceID1)

		ownedFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
		_ = dbfactory.FeatEnv(ctx, envSvc, otherApp, env1)

		standardEnv, err := envs.Get(ctx, envmodel.AppEnvByName(workspaceID1, ownerApp.ID, env1.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(standardEnv.ID).To(Equal(env1.ID))
		Expect(standardEnv.GetKind()).To(Equal(envmodel.EnvironmentKindStandard))

		stdEnvByName, err := envs.Get(ctx, envmodel.ByName(workspaceID1, env1.Name), envmodel.StdEnvs())
		Expect(err).NotTo(HaveOccurred())
		Expect(stdEnvByName.ID).To(Equal(env1.ID))
		Expect(stdEnvByName.GetKind()).To(Equal(envmodel.EnvironmentKindStandard))

		featureEnv, err := envs.Get(ctx, envmodel.AppEnvByName(workspaceID1, ownerApp.ID, ownedFeatureEnv.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(featureEnv.OwnerAppID).To(Equal(ownerApp.ID))
		Expect(featureEnv.GetKind()).To(Equal(envmodel.EnvironmentKindFeature))
		workspaceEnv, err := envs.Get(ctx, envmodel.ByName(workspaceID1, ownedFeatureEnv.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(workspaceEnv.ID).To(Equal(ownedFeatureEnv.ID))
		Expect(workspaceEnv.Status).To(Equal(envmodel.EnvStatusReady))
		_, err = envs.Get(ctx, envmodel.ByName(workspaceID2, ownedFeatureEnv.Name))
		Expect(err).To(MatchError(envmodel.ErrEnvNotFound))

		_, err = envs.Get(ctx, envmodel.ByName(workspaceID1, ownedFeatureEnv.Name), envmodel.StdEnvs())
		Expect(err).To(MatchError(envmodel.ErrEnvNotFound))

		_, err = envs.Get(ctx, envmodel.AppEnvByName(workspaceID1, otherApp.ID, ownedFeatureEnv.Name))
		Expect(err).To(MatchError(envmodel.ErrEnvNotFound))
	})

	It("finds the environment bound to a cluster namespace", func() {
		found, err := envmodel.FindByClusterNamespace(
			ctx,
			db,
			env1.Cluster.ClusterID,
			env1.Cluster.Namespace,
			bson.ObjectID{},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).NotTo(BeNil())
		Expect(found.ID).To(Equal(env1.ID))

		// 排除自身后，没有其他环境占用该命名空间
		found, err = envmodel.FindByClusterNamespace(ctx, db, env1.Cluster.ClusterID, env1.Cluster.Namespace, env1.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeNil())
	})

	Context("UpdateEnv", func() {
		It("updates display name", func() {
			displayName := stringx.Random(50)
			Expect(envmodel.UpdateEnv(ctx, db, env1.ID, &envmodel.EnvironmentUpdateData{DisplayName: &displayName})).
				To(Succeed())
			env, _ := envs.GetByID(ctx, env1.ID)
			Expect(env.DisplayName).To(Equal(displayName))
		})

		It("updates description", func() {
			desc := stringx.Random(50)
			Expect(envmodel.UpdateEnv(ctx, db, env1.ID, &envmodel.EnvironmentUpdateData{Description: &desc})).
				To(Succeed())
			env, _ := envs.GetByID(ctx, env1.ID)
			Expect(env.Description).To(Equal(desc))
		})

		It("updates type", func() {
			envType := stringx.Random(10)
			Expect(envmodel.UpdateEnv(ctx, db, env1.ID, &envmodel.EnvironmentUpdateData{Type: &envType})).
				To(Succeed())
			env, _ := envs.GetByID(ctx, env1.ID)
			Expect(env.Type).To(Equal(envType))
		})

		It("updates cluster and namespace", func() {
			newClusterID := stringx.Random(10)
			newClusterType := stringx.Random(10)
			newNamespace := stringx.Random(15)
			Expect(envmodel.UpdateEnv(ctx, db, env1.ID, &envmodel.EnvironmentUpdateData{
				ClusterID:   &newClusterID,
				ClusterType: &newClusterType,
				Namespace:   &newNamespace,
			})).To(Succeed())
			env, _ := envs.GetByID(ctx, env1.ID)
			Expect(env.Cluster.ClusterID).To(Equal(newClusterID))
			Expect(env.Cluster.ClusterType).To(Equal(newClusterType))
			Expect(env.Cluster.Namespace).To(Equal(newNamespace))
		})

		It("updates cluster isFederation", func() {
			isFederation := true
			Expect(envmodel.UpdateEnv(ctx, db, env1.ID, &envmodel.EnvironmentUpdateData{IsFederation: &isFederation})).
				To(Succeed())
			env, _ := envs.GetByID(ctx, env1.ID)
			Expect(env.Cluster.IsFederation).To(BeTrue())
		})

		It("is a no-op when no field is given", func() {
			before, err := envs.GetByID(ctx, env1.ID)
			Expect(err).NotTo(HaveOccurred())

			Expect(envmodel.UpdateEnv(ctx, db, env1.ID, &envmodel.EnvironmentUpdateData{})).To(Succeed())
			after, err := envs.GetByID(ctx, env1.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(after.UpdatedAt).To(Equal(before.UpdatedAt))
		})
	})

	It("adds and removes apps without touching updatedAt", func() {
		env, _ := envs.GetByID(ctx, env1.ID)
		Expect(env.AppIDs).To(BeEmpty())
		updatedAt := env.UpdatedAt

		appID1 := stringx.Random(10)
		Expect(envmodel.AddApp(ctx, db, env1.ID, appID1)).To(Succeed())
		env, _ = envs.GetByID(ctx, env1.ID)
		Expect(env.AppIDs).To(Equal([]string{appID1}))
		// 验证重复添加
		Expect(envmodel.AddApp(ctx, db, env1.ID, appID1)).To(Succeed())
		env, _ = envs.GetByID(ctx, env1.ID)
		Expect(env.AppIDs).To(Equal([]string{appID1}))
		// 验证添加第二个 app
		appID2 := stringx.Random(10)
		Expect(envmodel.AddApp(ctx, db, env1.ID, appID2)).To(Succeed())
		env, _ = envs.GetByID(ctx, env1.ID)
		Expect(env.AppIDs).To(Equal([]string{appID1, appID2}))

		// 验证删除 appID1
		Expect(envmodel.RemoveApp(ctx, db, env1.ID, appID1)).To(Succeed())
		env, _ = envs.GetByID(ctx, env1.ID)
		Expect(env.AppIDs).To(Equal([]string{appID2}))
		// 验证继续删除 appID2
		Expect(envmodel.RemoveApp(ctx, db, env1.ID, appID2)).To(Succeed())
		env, _ = envs.GetByID(ctx, env1.ID)
		Expect(env.AppIDs).To(BeEmpty())
		// 验证删除不存在的 appID, 表现安全
		Expect(envmodel.RemoveApp(ctx, db, env1.ID, stringx.Random(10))).To(Succeed())

		Expect(env.UpdatedAt).To(Equal(updatedAt))
	})
})
