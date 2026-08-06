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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Test EnvironmentStoreMongo", func() {
	var ctx context.Context

	var diApp *fxtest.App
	var mongoClient *mongo.Client
	var store envmodel.EnvironmentStore
	var appStore bkmsapp.ApplicationStore
	var envSvc *bkmsenv.EnvService
	var dbName string

	BeforeEach(func() {
		ctx = context.Background()

		diApp = fxtest.New(
			GinkgoT(),
			bkmsenv.FxModule,
			bkmsapp.FxModule,
			database.PrivateFxModule,
			fx.Populate(&store, &appStore, &envSvc, &mongoClient, &dbName),
		)
		diApp.RequireStart()
	})
	AfterEach(func() {
		err := store.DeleteAll(ctx)
		Expect(err).NotTo(HaveOccurred())
		_, err = mongoClient.Database(dbName).Collection("applications").DeleteMany(ctx, bson.M{})
		Expect(err).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	Context("Test EnvironmentStoreMongo methods", func() {
		var envID1 bson.ObjectID
		var envID2 bson.ObjectID

		workspaceID1 := stringx.Random(10)
		workspaceID2 := stringx.Random(10)

		createdAt := time.Now()

		env1 := &envmodel.Environment{
			WorkspaceID: workspaceID1, Name: stringx.Random(10), DisplayName: stringx.Random(10),
			Cluster: envmodel.BizCluster{
				ProjectCode: stringx.Random(10),
				ClusterID:   stringx.Random(10),
				ClusterType: stringx.Random(10),
				Namespace:   stringx.Random(10),
			},
			CreatedAt: createdAt, Type: "development",
		}
		env2 := &envmodel.Environment{
			WorkspaceID: workspaceID2, Name: stringx.Random(10), DisplayName: stringx.Random(10),
			Cluster: envmodel.BizCluster{
				ProjectCode: stringx.Random(10),
				ClusterID:   stringx.Random(10),
				ClusterType: stringx.Random(10),
				Namespace:   stringx.Random(10),
			},
			CreatedAt: createdAt, Type: "test",
		}

		BeforeEach(func() {
			var err error
			// test: create environment
			envID1, err = store.Create(ctx, env1)
			Expect(err).NotTo(HaveOccurred())

			envID2, err = store.Create(ctx, env2)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			// test: delete environment
			err := store.Delete(ctx, envID1)
			Expect(err).NotTo(HaveOccurred())
			err = store.Delete(ctx, envID2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("test list environments", func() {
			// 查询 workspace1 下的环境
			environments, err := store.ListStdEnvs(ctx, workspaceID1)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(environments)).To(Equal(1))

			expectedEnv1 := *env1
			expectedEnv1.ID = environments[0].ID
			// 去除时间精度差异. 不影响测试效果
			expectedEnv1.CreatedAt = environments[0].CreatedAt
			expectedEnv1.UpdatedAt = environments[0].UpdatedAt
			expectedEnv1.Status = envmodel.EnvStatusReady
			Expect(environments[0]).To(Equal(expectedEnv1))

			// 查询 workspace2 下的环境
			environments, err = store.ListStdEnvs(ctx, workspaceID2)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(environments)).To(Equal(1))

			expectedEnv2 := *env2
			expectedEnv2.ID = environments[0].ID
			// 去除时间精度差异. 不影响测试效果
			expectedEnv2.CreatedAt = environments[0].CreatedAt
			expectedEnv2.UpdatedAt = environments[0].UpdatedAt
			expectedEnv2.Status = envmodel.EnvStatusReady
			Expect(environments[0]).To(Equal(expectedEnv2))

			// 随机查询一个不存在的 workspace 下的环境
			environments, err = store.ListStdEnvs(ctx, stringx.Random(10))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(environments)).To(Equal(0))
		})

		It("test count environments by workspace IDs", func() {
			counts, err := store.CountByWorkspaceIDs(ctx, []string{workspaceID1, workspaceID2, stringx.Random(10)})
			Expect(err).NotTo(HaveOccurred())
			Expect(counts[workspaceID1]).To(Equal(1))
			Expect(counts[workspaceID2]).To(Equal(1))
		})

		It("lists app environments with standard and owned feature envs only", func() {
			ownerApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "owner-app-id-" + stringx.Random(6),
				Name:        "owner-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})
			otherApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "other-app-id-" + stringx.Random(6),
				Name:        "other-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})

			ownedFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
			otherFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, otherApp, env1)

			workspaceEnvs, err := store.ListStdEnvs(ctx, workspaceID1)
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaceEnvs).To(HaveLen(1))
			Expect(workspaceEnvs[0].GetKind()).To(Equal(envmodel.EnvironmentKindStandard))

			appEnvs, err := store.ListAppEnvs(ctx, workspaceID1, ownerApp.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appEnvs).To(HaveLen(2))
			envNames := lo.Map(appEnvs, func(env envmodel.Environment, _ int) string { return env.Name })
			Expect(envNames).To(ContainElements(
				env1.Name,
				ownedFeatureEnv.Name,
			))
			Expect(envNames).NotTo(ContainElement(otherFeatureEnv.Name))

			counts, err := store.CountByWorkspaceIDs(ctx, []string{workspaceID1})
			Expect(err).NotTo(HaveOccurred())
			Expect(counts[workspaceID1]).To(Equal(1))
		})

		It("lists specified app environments by ids", func() {
			ownerApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "owner-app-id-" + stringx.Random(6),
				Name:        "owner-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})
			otherApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "other-app-id-" + stringx.Random(6),
				Name:        "other-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})

			ownedFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
			otherFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, otherApp, env1)
			Expect(store.AddApp(ctx, envID1, ownerApp.ID)).To(Succeed())

			envs, err := store.ListAppEnvsByIDs(ctx, workspaceID1, ownerApp.ID, []bson.ObjectID{
				envID1,
				ownedFeatureEnv.ID,
				otherFeatureEnv.ID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(envs, func(env envmodel.Environment, _ int) string {
				return env.Name
			})).To(ConsistOf(env1.Name, ownedFeatureEnv.Name))
		})

		It("lists specified app environments by types", func() {
			ownerApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "owner-app-id-" + stringx.Random(6),
				Name:        "owner-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})
			otherApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "other-app-id-" + stringx.Random(6),
				Name:        "other-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})

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

			envs, err := store.ListAppEnvsByTypes(ctx, workspaceID1, ownerApp.ID, []string{"production"})
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(envs, func(env envmodel.Environment, _ int) string {
				return env.Name
			})).To(ConsistOf(prodStdEnv.Name, prodFeatureEnv.Name))
			Expect(lo.Map(envs, func(env envmodel.Environment, _ int) string {
				return env.Type
			})).To(ConsistOf("production", "production"))
			Expect(lo.Map(envs, func(env envmodel.Environment, _ int) string {
				return env.Name
			})).NotTo(ContainElement(devStdEnv.Name))
		})

		It("lists only feature environments owned by the requested app", func() {
			ownerApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "owner-app-id-" + stringx.Random(6),
				Name:        "owner-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})
			otherApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "other-app-id-" + stringx.Random(6),
				Name:        "other-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})

			firstOwnedEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
			secondOwnedEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
			_ = dbfactory.FeatEnv(ctx, envSvc, otherApp, env1)

			featureEnvs, err := store.ListAppFeatEnvs(ctx, ownerApp.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(featureEnvs).To(HaveLen(2))
			Expect(lo.Map(featureEnvs, func(env envmodel.Environment, _ int) string {
				return env.Name
			})).To(ConsistOf(firstOwnedEnv.Name, secondOwnedEnv.Name))
			for _, featureEnv := range featureEnvs {
				Expect(featureEnv.GetKind()).To(Equal(envmodel.EnvironmentKindFeature))
				Expect(featureEnv.OwnerAppID).To(Equal(ownerApp.ID))
			}

			empty, err := store.ListAppFeatEnvs(ctx, "missing-app")
			Expect(err).NotTo(HaveOccurred())
			Expect(empty).To(BeEmpty())
		})

		It("lists batch app environments with shared standard envs and owned feature envs", func() {
			firstApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "first-app-id-" + stringx.Random(6),
				Name:        "first-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})
			secondApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "second-app-id-" + stringx.Random(6),
				Name:        "second-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})
			otherWorkspaceApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "other-workspace-app-id-" + stringx.Random(6),
				Name:        "other-workspace-app-" + stringx.Random(6),
				WorkspaceID: workspaceID2,
			})

			firstFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, firstApp, env1)
			secondFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, secondApp, env1)
			_ = dbfactory.FeatEnv(ctx, envSvc, otherWorkspaceApp, env2)

			envs, err := store.ListBatchAppEnvs(ctx, workspaceID1, []string{
				firstApp.ID,
				secondApp.ID,
				firstApp.ID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(envs, func(env envmodel.Environment, _ int) string {
				return env.Name
			})).To(ConsistOf(env1.Name, firstFeatureEnv.Name, secondFeatureEnv.Name))

			empty, err := store.ListBatchAppEnvs(ctx, workspaceID1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(empty).To(BeEmpty())
		})

		It("gets standard or owned feature env by name", func() {
			ownerApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "owner-app-id-" + stringx.Random(6),
				Name:        "owner-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})
			otherApp := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
				ID:          "other-app-id-" + stringx.Random(6),
				Name:        "other-app-" + stringx.Random(6),
				WorkspaceID: workspaceID1,
			})

			ownedFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, ownerApp, env1)
			_ = dbfactory.FeatEnv(ctx, envSvc, otherApp, env1)

			standardEnv, err := store.GetByName(ctx, workspaceID1, ownerApp.ID, env1.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(standardEnv.ID).To(Equal(envID1))
			Expect(standardEnv.GetKind()).To(Equal(envmodel.EnvironmentKindStandard))

			stdEnvByName, err := store.GetStdEnvByName(ctx, workspaceID1, env1.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdEnvByName.ID).To(Equal(envID1))
			Expect(stdEnvByName.GetKind()).To(Equal(envmodel.EnvironmentKindStandard))

			featureEnv, err := store.GetByName(ctx, workspaceID1, ownerApp.ID, ownedFeatureEnv.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(featureEnv.OwnerAppID).To(Equal(ownerApp.ID))
			Expect(featureEnv.GetKind()).To(Equal(envmodel.EnvironmentKindFeature))

			_, err = store.GetStdEnvByName(ctx, workspaceID1, ownedFeatureEnv.Name)
			Expect(err).To(MatchError(envmodel.ErrEnvNotFound))

			_, err = store.GetByName(ctx, workspaceID1, otherApp.ID, ownedFeatureEnv.Name)
			Expect(err).To(MatchError(envmodel.ErrEnvNotFound))
		})

		It("test get environment", func() {
			env, _ := store.Get(ctx, envID1)
			Expect(env.ID).To(Equal(envID1))
			Expect(env.Name).To(Equal(env1.Name))
			Expect(env.Type).To(Equal(env.Type))
			Expect(env.Status).To(Equal(envmodel.EnvStatusReady))

			_, err := store.Get(ctx, bson.NewObjectID())
			Expect(err).To(HaveOccurred())
		})

		Context("test update environment", func() {
			It("update display name", func() {
				displayName := stringx.Random(50)
				_ = store.Update(ctx, envID1, &envmodel.EnvironmentUpdateData{DisplayName: &displayName})
				env, _ := store.Get(ctx, envID1)
				Expect(env.DisplayName).To(Equal(displayName))
			})

			It("update description", func() {
				desc := stringx.Random(50)
				_ = store.Update(ctx, envID1, &envmodel.EnvironmentUpdateData{Description: &desc})
				env, _ := store.Get(ctx, envID1)
				Expect(env.Description).To(Equal(desc))
			})

			It("update type", func() {
				envType := stringx.Random(10)
				_ = store.Update(ctx, envID1, &envmodel.EnvironmentUpdateData{Type: &envType})
				env, _ := store.Get(ctx, envID1)
				Expect(env.Type).To(Equal(envType))
			})

			It("update cluster and namespace", func() {
				newClusterID := stringx.Random(10)
				newClusterType := stringx.Random(10)
				newNamespace := stringx.Random(15)
				_ = store.Update(
					ctx,
					envID1,
					&envmodel.EnvironmentUpdateData{
						ClusterID:   &newClusterID,
						ClusterType: &newClusterType,
						Namespace:   &newNamespace,
					},
				)
				env, _ := store.Get(ctx, envID1)
				Expect(env.Cluster.ClusterID).To(Equal(newClusterID))
				Expect(env.Cluster.ClusterType).To(Equal(newClusterType))
				Expect(env.Cluster.Namespace).To(Equal(newNamespace))
			})
		})

		It("test add and remove app", func() {
			env, _ := store.Get(ctx, envID1)
			Expect(len(env.AppIDs)).To(Equal(0))

			appID1 := stringx.Random(10)
			_ = store.AddApp(ctx, envID1, appID1)
			env, _ = store.Get(ctx, envID1)
			Expect(len(env.AppIDs)).To(Equal(1))
			// 验证重复添加
			_ = store.AddApp(ctx, envID1, appID1)
			env, _ = store.Get(ctx, envID1)
			Expect(len(env.AppIDs)).To(Equal(1))
			Expect(env.AppIDs[0]).To(Equal(appID1))
			// 验证添加第二个 app
			appID2 := stringx.Random(10)
			_ = store.AddApp(ctx, envID1, appID2)
			env, _ = store.Get(ctx, envID1)
			Expect(len(env.AppIDs)).To(Equal(2))
			Expect(env.AppIDs[0]).To(Equal(appID1))
			Expect(env.AppIDs[1]).To(Equal(appID2))

			// 验证删除 appID1
			_ = store.RemoveApp(ctx, envID1, appID1)
			env, _ = store.Get(ctx, envID1)
			Expect(len(env.AppIDs)).To(Equal(1))
			Expect(env.AppIDs[0]).To(Equal(appID2))
			// 验证继续删除 appID2
			_ = store.RemoveApp(ctx, envID1, appID2)
			env, _ = store.Get(ctx, envID1)
			Expect(len(env.AppIDs)).To(Equal(0))
			// 验证删除不存在的 appID, 表现安全
			err := store.RemoveApp(ctx, envID1, stringx.Random(10))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
