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

package appmodel_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("RecordStore", func() {
	var recordStore *appmodel.RecordStore
	var db *mongo.Database
	var ctx context.Context

	var workspaceID, appID, envName, trafficLaneName string
	var recordA, recordB appmodel.Record

	// listRecords 按 handler 的查询组合分页列出当前应用在指定环境、泳道下的部署记录
	listRecords := func(envName, laneName, keyword string, page, size int64) ([]appmodel.Record, int64, error) {
		p, err := recordStore.ListPage(
			ctx,
			page,
			size,
			deploytypes.ByAppEnvLane(
				appID,
				envName,
				laneName,
			),
			appmodel.KeywordMatches(keyword),
			deploytypes.NewestFirst(),
		)
		if err != nil {
			return nil, 0, err
		}
		return p.Items, p.Total, nil
	}
	// listByImageTag 按 handler 的查询组合分页列出当前应用使用指定镜像标签的部署记录
	listByImageTag := func(imageTag string, page, size int64) ([]appmodel.Record, int64, error) {
		p, err := recordStore.ListPage(
			ctx,
			page,
			size,
			appmodel.ByImageTag(appID, imageTag),
			deploytypes.NewestFirst(),
		)
		if err != nil {
			return nil, 0, err
		}
		return p.Items, p.Total, nil
	}

	BeforeEach(func() {
		db = testutil.NewDB(GinkgoT())
		recordStore = store.Of(db, appmodel.RecordMeta)

		ctx = context.Background()
		workspaceID = "test-workspace-" + stringx.Random(6)
		appID = "test-app-" + stringx.Random(6)
		envName = "staging"
		trafficLaneName = "base"

		recordA = appmodel.Record{
			WorkspaceID:     workspaceID,
			AppID:           appID,
			EnvName:         envName,
			TrafficLaneName: trafficLaneName,
			ClusterID:       "BCS-K8S-12345",
			Namespace:       "default",
			ImageTag:        "v1.0.0",
			Replicas:        3,
			Message:         "deployed successfully",
			Status:          "deployed",
			Creator:         "admin",
			Extras: map[string]string{
				"version": "1.0.0",
			},
		}

		recordB = appmodel.Record{
			WorkspaceID:     workspaceID,
			AppID:           appID,
			EnvName:         envName,
			TrafficLaneName: trafficLaneName,
			ClusterID:       "BCS-K8S-54321",
			Namespace:       "blueking",
			ImageTag:        "v1.0.1",
			Replicas:        5,
			Message:         "deployed with new version",
			Status:          "deployed",
			Creator:         "blueking",
			Extras: map[string]string{
				"version": "1.0.1",
			},
		}
	})

	Context("Update", func() {
		It("should update deploy record successfully", func() {
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 更新记录

			recordA.Message = "updated message"
			recordA.Status = "failed"
			recordA.Extras["error"] = "some error"

			err = appmodel.UpdateRecord(ctx, db, &recordA)
			Expect(err).NotTo(HaveOccurred())

			record, err := appmodel.GetRecord(ctx, db, appID, recordA.ID.Hex())
			Expect(err).NotTo(HaveOccurred())
			Expect(record.Message).To(Equal("updated message"))
			Expect(record.Status).To(Equal(appmodel.Status("failed")))
			Expect(record.Extras["error"]).To(Equal("some error"))
		})

		It("should persist batch release fields", func() {
			strategy := &appmodel.ReleaseStrategySnapshot{
				ID:   bson.NewObjectID().Hex(),
				Name: "生产谨慎发布",
				Batches: []releasestrategy.Batch{
					{Target: "1", PauseMode: releasestrategy.PauseModePermanent},
					{Target: "50%", PauseMode: releasestrategy.PauseModeDuration, PauseSeconds: 1800},
				},
			}
			recordA.Status = appmodel.StatusDeploying
			recordA.ReleaseMode = appmodel.ReleaseModeBatch
			recordA.ReleaseStrategy = strategy
			Expect(recordStore.Create(ctx, &recordA)).To(Succeed())
			recordID := recordA.ID.Hex()

			record, err := appmodel.GetRecord(ctx, db, appID, recordID)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.ReleaseMode).To(Equal(appmodel.ReleaseModeBatch))
			Expect(record.ReleaseStrategy).To(Equal(strategy))
			Expect(record.BatchProgress).To(BeNil())
			Expect(record.IsBatchReleasing()).To(BeTrue())

			// 轮询写入进度，毫秒精度与 Mongo 存储一致
			pauseStartTime := time.Now().UTC().Truncate(time.Millisecond)
			progress := &appmodel.BatchProgress{
				CurrentBatchIndex: lo.ToPtr(int64(0)),
				BatchPhase:        batchrelease.BatchPhaseManualPaused,
				RolloutPhase:      "Running",
				Message:           "Running canary step 0",
				PauseStartTime:    &pauseStartTime,
				UpdatedReplicas:   1,
				Replicas:          4,
				OldImageTag:       "v1.0.0",
				NewImageTag:       "v1.1.0",
			}
			Expect(appmodel.UpdateBatchProgress(ctx, db, recordID, progress)).To(Succeed())

			record, err = appmodel.GetRecord(ctx, db, appID, recordID)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.BatchProgress).To(Equal(progress))
			Expect(record.ReleaseStrategy).To(Equal(strategy))

			// 通用 Update 拿着不含进度的快照写回，不能把刚写入的进度盖掉
			record.BatchProgress = nil
			record.Message = "still deploying"
			Expect(appmodel.UpdateRecord(ctx, db, record)).To(Succeed())

			record, err = appmodel.GetRecord(ctx, db, appID, recordID)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.Message).To(Equal("still deploying"))
			Expect(record.BatchProgress).To(Equal(progress))
		})

		It("rejects a nil batch progress and a missing record", func() {
			Expect(appmodel.UpdateBatchProgress(ctx, db, bson.NewObjectID().Hex(), nil)).NotTo(Succeed())

			err := appmodel.UpdateBatchProgress(ctx, db, bson.NewObjectID().Hex(), &appmodel.BatchProgress{
				BatchPhase: batchrelease.BatchPhasePending,
			})
			Expect(err).To(MatchError(appmodel.ErrDeployRecordNotFound))
		})

		It("should treat records without release mode as releasing all at once", func() {
			recordA.Status = appmodel.StatusDeploying
			Expect(recordStore.Create(ctx, &recordA)).To(Succeed())
			recordID := recordA.ID.Hex()

			raw := bson.M{}
			Expect(recordStore.Collection().FindOne(ctx, bson.M{"_id": recordA.ID}).Decode(&raw)).To(Succeed())
			Expect(raw).NotTo(HaveKey("releaseMode"))
			Expect(raw).NotTo(HaveKey("releaseStrategy"))
			Expect(raw).NotTo(HaveKey("batchProgress"))

			record, err := appmodel.GetRecord(ctx, db, appID, recordID)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.GetReleaseMode()).To(Equal(appmodel.ReleaseModeAll))
			Expect(record.IsBatchReleasing()).To(BeFalse())

			// 状态更新不能给非分批记录补上 batchProgress: null
			record.Status = appmodel.StatusDeployed
			Expect(appmodel.UpdateRecord(ctx, db, record)).To(Succeed())
			raw = bson.M{}
			Expect(recordStore.Collection().FindOne(ctx, bson.M{"_id": recordA.ID}).Decode(&raw)).To(Succeed())
			Expect(raw).NotTo(HaveKey("batchProgress"))
		})
	})

	Context("GetLatest", func() {
		It("should get the latest deploy record", func() {
			// 创建第一条记录
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			// 创建第二条记录（应该是最新的）
			err = recordStore.Create(ctx, &recordB)
			recordID2 := recordB.ID.Hex()
			Expect(err).NotTo(HaveOccurred())

			// 获取最新记录
			latest, err := appmodel.GetLatest(ctx, db, appID, envName, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).NotTo(BeNil())
			Expect(latest.ID.Hex()).To(Equal(recordID2))
			Expect(latest.ImageTag).To(Equal("v1.0.1"))
			Expect(latest.Replicas).To(Equal(int32(5)))
			Expect(latest.Creator).To(Equal("blueking"))
		})

		It("should return ErrDeployRecordNotFound when no record exists", func() {
			_, err := appmodel.GetLatest(ctx, db, "non-existent-app", envName, trafficLaneName)
			Expect(err).To(MatchError(appmodel.ErrDeployRecordNotFound))
		})

		It("should filter by envName correctly", func() {
			// 为 staging 环境创建记录
			recordA.EnvName = "staging"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 为 production 环境创建记录
			recordB.EnvName = "production"
			err = recordStore.Create(ctx, &recordB)
			recordID2 := recordB.ID.Hex()
			Expect(err).NotTo(HaveOccurred())

			// 获取 production 环境的最新记录
			latest, err := appmodel.GetLatest(ctx, db, appID, "production", trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.ID.Hex()).To(Equal(recordID2))
			Expect(latest.EnvName).To(Equal("production"))
		})

		It("should filter by trafficLaneName correctly", func() {
			// 为 base 泳道创建记录
			recordA.TrafficLaneName = "base"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 为 feature 泳道创建记录
			recordB.TrafficLaneName = "feature-lane"
			err = recordStore.Create(ctx, &recordB)
			recordID2 := recordB.ID.Hex()
			Expect(err).NotTo(HaveOccurred())

			// 获取 feature 泳道的最新记录
			latest, err := appmodel.GetLatest(ctx, db, appID, envName, "feature-lane")
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.ID.Hex()).To(Equal(recordID2))
			Expect(latest.TrafficLaneName).To(Equal("feature-lane"))
		})

		It("should return the most recent record when multiple exist", func() {
			// 创建多条记录
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.ImageTag = "v1.0.2"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			recordC := recordB
			recordC.ID = bson.NilObjectID
			recordC.ImageTag = "v1.0.3"
			err = recordStore.Create(ctx, &recordC)
			recordID3 := recordC.ID.Hex()
			Expect(err).NotTo(HaveOccurred())

			// 获取最新记录应该返回最后创建的那条
			latest, err := appmodel.GetLatest(ctx, db, appID, envName, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.ID.Hex()).To(Equal(recordID3))
			Expect(latest.ImageTag).To(Equal("v1.0.3"))
		})

		It("should get the latest record matching statuses", func() {
			recordA.Status = appmodel.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// MongoDB DateTime 精度为毫秒，依赖 createdAt 排序的场景需要等到下一个精度窗口
			time.Sleep(5 * time.Millisecond)

			recordB.Status = appmodel.StatusFailed
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// MongoDB DateTime 精度为毫秒，依赖 createdAt 排序的场景需要等到下一个精度窗口
			time.Sleep(5 * time.Millisecond)

			recordC := recordB
			recordC.ID = bson.NilObjectID
			recordC.ImageTag = "v1.0.2"
			recordC.Status = appmodel.StatusDeployed
			err = recordStore.Create(ctx, &recordC)
			recordID3 := recordC.ID.Hex()
			Expect(err).NotTo(HaveOccurred())

			latest, err := appmodel.GetLatest(
				ctx,
				db,
				appID,
				envName,
				trafficLaneName,
				appmodel.StatusDeployed,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).NotTo(BeNil())
			Expect(latest.ID.Hex()).To(Equal(recordID3))
			Expect(latest.Status).To(Equal(appmodel.StatusDeployed))
			Expect(latest.ImageTag).To(Equal("v1.0.2"))
		})

		It("should return error when no record matches statuses", func() {
			recordA.Status = appmodel.StatusFailed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			_, err = appmodel.GetLatest(
				ctx,
				db,
				appID,
				envName,
				trafficLaneName,
				appmodel.StatusDeployed,
			)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("ListLatestByApps", func() {
		// Create 会把 CreatedAt 写成 time.Now()。BSON DateTime 只有毫秒精度，
		// 连续插入可能落在同一毫秒，导致 $sort + $first 取「最新」不稳定。
		// 创建后再显式回写 createdAt，用明确时间差拉开新旧记录，避免 CI 抖动影响结果。
		persistCreatedAt := func(id string, createdAt time.Time) {
			GinkgoHelper()
			objID, err := bson.ObjectIDFromHex(id)
			Expect(err).NotTo(HaveOccurred())
			_, err = db.Collection(appmodel.RecordMeta.Collection).UpdateByID(
				ctx, objID, bson.M{"$set": bson.M{"createdAt": createdAt}},
			)
			Expect(err).NotTo(HaveOccurred())
		}

		It("should return the newest record per app and env", func() {
			otherAppID := "test-app-" + stringx.Random(6)
			now := time.Now().UTC().Truncate(time.Millisecond)

			err := recordStore.Create(ctx, &recordA)

			idA := recordA.ID.Hex()
			Expect(err).NotTo(HaveOccurred())
			persistCreatedAt(idA, now.Add(-2*time.Second))

			recordB.ImageTag = "v1.0.1"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			prod := recordA
			prod.ID = bson.NilObjectID
			prod.EnvName = "production"
			prod.ImageTag = "prod-v1"
			err = recordStore.Create(ctx, &prod)
			Expect(err).NotTo(HaveOccurred())

			otherStag := recordA
			otherStag.ID = bson.NilObjectID
			otherStag.AppID = otherAppID
			otherStag.ImageTag = "other-stag-v1"
			err = recordStore.Create(ctx, &otherStag)
			idOther := otherStag.ID.Hex()
			Expect(err).NotTo(HaveOccurred())
			persistCreatedAt(idOther, now.Add(-time.Second))

			otherStagNew := otherStag
			otherStagNew.ID = bson.NilObjectID
			otherStagNew.ImageTag = "other-stag-v2"
			err = recordStore.Create(ctx, &otherStagNew)
			Expect(err).NotTo(HaveOccurred())

			latest, err := appmodel.ListLatestByApps(ctx, db, []string{appID, otherAppID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(2))
			Expect(latest[appID]).To(HaveLen(2))
			Expect(latest[appID][envName].ImageTag).To(Equal("v1.0.1"))
			Expect(latest[appID]["production"].ImageTag).To(Equal("prod-v1"))
			Expect(latest[otherAppID]).To(HaveLen(1))
			Expect(latest[otherAppID][envName].ImageTag).To(Equal("other-stag-v2"))
		})

		It("should only return records of the given traffic lane", func() {
			recordA.TrafficLaneName = trafficLaneName
			recordA.ImageTag = "base-v1"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.TrafficLaneName = "feature-lane"
			recordB.ImageTag = "lane-v1"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			latest, err := appmodel.ListLatestByApps(ctx, db, []string{appID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(1))
			Expect(latest[appID][envName].ImageTag).To(Equal("base-v1"))

			laneLatest, err := appmodel.ListLatestByApps(ctx, db, []string{appID}, "feature-lane")
			Expect(err).NotTo(HaveOccurred())
			Expect(laneLatest).To(HaveLen(1))
			Expect(laneLatest[appID][envName].ImageTag).To(Equal("lane-v1"))
		})

		It("should not return records of apps not in the list", func() {
			otherAppID := "other-app-" + stringx.Random(6)

			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			other := recordA
			other.ID = bson.NilObjectID
			other.AppID = otherAppID
			other.ImageTag = "other-v1"
			err = recordStore.Create(ctx, &other)
			Expect(err).NotTo(HaveOccurred())

			latest, err := appmodel.ListLatestByApps(ctx, db, []string{appID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(1))
			Expect(latest).To(HaveKey(appID))
			Expect(latest).NotTo(HaveKey(otherAppID))
			Expect(latest[appID][envName].ImageTag).To(Equal("v1.0.0"))
		})
	})

	Context("List", func() {
		It("should support pagination and sorting", func() {
			// 创建多条记录
			recordA.ImageTag = "v1.0.0"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			recordB.ImageTag = "v1.0.1"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			recordC := recordB
			recordC.ID = bson.NilObjectID
			recordC.ImageTag = "v1.0.2"
			err = recordStore.Create(ctx, &recordC)
			Expect(err).NotTo(HaveOccurred())

			// 测试第一页（按创建时间降序排列）
			records, total, err := listRecords(envName, trafficLaneName, "", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(3)))
			Expect(records).To(HaveLen(2))
			Expect(records[0].ImageTag).To(Equal("v1.0.2"))
			Expect(records[1].ImageTag).To(Equal("v1.0.1"))

			// 测试第二页
			records, total, err = listRecords(envName, trafficLaneName, "", 2, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(3)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].ImageTag).To(Equal("v1.0.0"))
		})

		It("should filter by keyword (imageTag and operator)", func() {
			// 创建多条记录
			recordA.ImageTag = "v1.0.0-RELEASE"
			recordA.Creator = "admin"
			recordA.Updater = "admin"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.ImageTag = "v2.0.0"
			recordB.Creator = "blueking"
			recordB.Updater = "blueking"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			recordC := recordB
			recordC.ID = bson.NilObjectID
			recordC.ImageTag = "v2.1.0+build.123"
			recordC.Creator = "blueking-admin"
			recordC.Updater = "blueking-admin"
			err = recordStore.Create(ctx, &recordC)
			Expect(err).NotTo(HaveOccurred())

			// 按 imageTag 搜索
			records, total, err := listRecords(envName, trafficLaneName, "v2", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(2))

			// 按 operator 搜索
			_, total, err = listRecords(envName, trafficLaneName, "blueking", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
		})

		It("should filter by envName and trafficLaneName", func() {
			// 为不同环境和泳道创建记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "production"
			recordB.TrafficLaneName = "feature-lane"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 按 envName 过滤
			records, total, err := listRecords("staging", trafficLaneName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records[0].EnvName).To(Equal("staging"))

			// 按 trafficLaneName 过滤
			records, total, err = listRecords("production", "feature-lane", "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records[0].TrafficLaneName).To(Equal("feature-lane"))
		})
	})

	Context("ListByImageTag", func() {
		It("should return records matching the specified imageTag", func() {
			// 创建匹配 tag 的记录（不同状态）
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			recordA.Status = appmodel.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "production"
			recordB.ImageTag = "v1.0.0"
			recordB.Status = appmodel.StatusFailed
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 创建不匹配 tag 的记录
			recordC := recordA
			recordC.ID = bson.NilObjectID
			recordC.ImageTag = "v2.0.0"
			recordC.EnvName = "staging"
			err = recordStore.Create(ctx, &recordC)
			Expect(err).NotTo(HaveOccurred())

			// 查询 v1.0.0 的记录，应返回 2 条（不限状态）
			records, total, err := listByImageTag("v1.0.0", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(2))
			for _, r := range records {
				Expect(r.ImageTag).To(Equal("v1.0.0"))
			}
		})

		It("should support pagination and sort by createdAt descending", func() {
			// 创建 3 条同 tag 的记录
			for i, env := range []string{"dev", "staging", "production"} {
				r := recordA
				r.ID = bson.NilObjectID
				r.EnvName = env
				r.ImageTag = "v1.0.0"
				r.Message = "deploy-" + string(rune('A'+i))
				err := recordStore.Create(ctx, &r)
				Expect(err).NotTo(HaveOccurred())
				// 等待确保时间差
				if i < 2 {
					time.Sleep(5 * time.Millisecond)
				}
			}

			// 第一页，每页 2 条
			records, total, err := listByImageTag("v1.0.0", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(3)))
			Expect(records).To(HaveLen(2))
			// 按 createdAt 降序，最新的在前
			Expect(records[0].EnvName).To(Equal("production"))
			Expect(records[1].EnvName).To(Equal("staging"))

			// 第二页
			records, total, err = listByImageTag("v1.0.0", 2, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(3)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].EnvName).To(Equal("dev"))
		})

		It("should not return records from other apps", func() {
			otherAppID := "other-app-" + stringx.Random(6)

			// 当前应用的记录
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 另一个应用的同 tag 记录
			recordB.AppID = otherAppID
			recordB.EnvName = "production"
			recordB.ImageTag = "v1.0.0"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 只应返回当前应用的记录
			records, total, err := listByImageTag("v1.0.0", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].AppID).To(Equal(appID))
		})
	})

	Context("ListImageTagDeployedEnvs", func() {
		It("should return empty list when no records exist", func() {
			pairs, err := appmodel.ListImageTagDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(BeEmpty())
		})

		It("should deduplicate records with same imageTag and envName", func() {
			// 同一 tag + 同一环境部署两次
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "staging"
			recordB.ImageTag = "v1.0.0"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			pairs, err := appmodel.ListImageTagDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(HaveLen(1))
			Expect(pairs[0].ImageTag).To(Equal("v1.0.0"))
			Expect(pairs[0].EnvName).To(Equal("staging"))
		})

		It("should not return records from other apps", func() {
			otherAppID := "other-app-" + stringx.Random(6)

			// 当前应用的记录
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 另一个应用的记录
			recordB.AppID = otherAppID
			recordB.EnvName = "production"
			recordB.ImageTag = "v2.0.0"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 只应返回当前应用的记录
			pairs, err := appmodel.ListImageTagDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(HaveLen(1))
			Expect(pairs[0].ImageTag).To(Equal("v1.0.0"))
			Expect(pairs[0].EnvName).To(Equal("staging"))
		})

		It("should handle multiple tags across multiple envs", func() {
			// v1.0.0 → staging + production, v1.1.0 → staging, v2.0.0 → production
			for i, pair := range []struct{ tag, env string }{
				{"v1.0.0", "staging"},
				{"v1.0.0", "production"},
				{"v1.1.0", "staging"},
				{"v2.0.0", "production"},
			} {
				r := recordA
				r.ID = bson.NilObjectID
				r.EnvName = pair.env
				r.ImageTag = pair.tag
				r.Message = "deploy-" + string(rune('A'+i))
				err := recordStore.Create(ctx, &r)
				Expect(err).NotTo(HaveOccurred())
			}

			pairs, err := appmodel.ListImageTagDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(HaveLen(4))

			pairSet := make(map[string]bool, len(pairs))
			for _, p := range pairs {
				pairSet[p.ImageTag+"|"+p.EnvName] = true
			}
			Expect(pairSet).To(HaveKey("v1.0.0|staging"))
			Expect(pairSet).To(HaveKey("v1.0.0|production"))
			Expect(pairSet).To(HaveKey("v1.1.0|staging"))
			Expect(pairSet).To(HaveKey("v2.0.0|production"))
		})

		It("should only return deployed records and ignore non-deployed ones", func() {
			// 同一 (imageTag, envName) 组合，一条 deployed，一条 failed
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			recordA.Status = appmodel.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "staging"
			recordB.ImageTag = "v1.0.0"
			recordB.Status = appmodel.StatusFailed
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 另一个环境只有 failed 和 canceled 记录
			failedRecord := recordA
			failedRecord.ID = bson.NilObjectID
			failedRecord.EnvName = "production"
			failedRecord.ImageTag = "v2.0.0"
			failedRecord.Status = appmodel.StatusFailed
			err = recordStore.Create(ctx, &failedRecord)
			Expect(err).NotTo(HaveOccurred())

			canceledRecord := recordA
			canceledRecord.ID = bson.NilObjectID
			canceledRecord.EnvName = "production"
			canceledRecord.ImageTag = "v3.0.0"
			canceledRecord.Status = appmodel.StatusCanceled
			err = recordStore.Create(ctx, &canceledRecord)
			Expect(err).NotTo(HaveOccurred())

			pairs, err := appmodel.ListImageTagDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			// 只应返回 staging 的 deployed 记录，failed/canceled 均不返回
			Expect(pairs).To(HaveLen(1))
			Expect(pairs[0].ImageTag).To(Equal("v1.0.0"))
			Expect(pairs[0].EnvName).To(Equal("staging"))
		})
	})

	Context("HasActiveDeployments", func() {
		It("should return false when no records exist", func() {
			hasActive, err := appmodel.HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return true when latest record is deployed", func() {
			// 创建一条 deployed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = appmodel.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := appmodel.HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeTrue())
		})

		It("should return false when latest record is uninstalled", func() {
			// 先创建一条 deployed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = appmodel.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(5 * time.Millisecond)

			// 再创建一条 uninstalled 记录（最新的）
			recordB.EnvName = "staging"
			recordB.TrafficLaneName = "base"
			recordB.Status = appmodel.StatusUninstalled
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := appmodel.HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return false when latest record is failed", func() {
			// 创建一条 failed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = appmodel.StatusFailed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := appmodel.HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return true when latest record is rolled back", func() {
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.ReleaseMode = appmodel.ReleaseModeBatch
			recordA.Status = appmodel.StatusRolledBack
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := appmodel.HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeTrue())
		})

		It("should return true if at least one env-lane has deployed as latest", func() {
			// env1: deployed
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = appmodel.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// env2: uninstalled (已卸载)
			time.Sleep(5 * time.Millisecond)
			recordB.EnvName = "production"
			recordB.TrafficLaneName = "base"
			recordB.Status = appmodel.StatusUninstalled
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 存在一个活跃部署，应返回 true
			hasActive, err := appmodel.HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeTrue())
		})

		It("should not count records from other apps", func() {
			otherAppID := "other-app-" + stringx.Random(6)

			// 当前应用：无记录
			hasActive, err := appmodel.HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())

			// 另一个应用：有 deployed 记录
			recordOther := recordA
			recordOther.ID = bson.NilObjectID
			recordOther.AppID = otherAppID
			recordOther.EnvName = "staging"
			recordOther.Status = appmodel.StatusDeployed
			err = recordStore.Create(ctx, &recordOther)
			Expect(err).NotTo(HaveOccurred())

			// 当前应用仍应返回 false
			hasActive, err = appmodel.HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})
	})
})
