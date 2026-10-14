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

package autodeploy_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("RecordStore", func() {
	var (
		ctx             context.Context
		db              *mongo.Database
		records         *autodeploy.RecordStore
		appID           string
		trafficLaneName string
	)

	// newRecord 构造同一应用下指定环境的记录，buildID 保证 appID + buildID 唯一索引不冲突。
	newRecord := func(envName, imageTag, laneName string) *autodeploy.Record {
		return &autodeploy.Record{
			WorkspaceID:     "test-workspace-" + stringx.Random(6),
			AppID:           appID,
			AppType:         "trpc",
			EnvName:         envName,
			TrafficLaneName: laneName,
			BuildID:         "build-" + stringx.Random(8),
			ImageTag:        imageTag,
			Stage:           autodeploy.StageBuild,
			Status:          "pending",
			Operator:        "admin",
			StartedAt:       time.Now(),
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		records = store.Of(db, autodeploy.RecordMeta)
		appID = "test-app-" + stringx.Random(6)
		trafficLaneName = ""
		DeferCleanup(func() {
			Expect(testutil.CleanupMetaColl(autodeploy.RecordMeta)).To(Succeed())
		})
	})

	Describe("Create", func() {
		It("should reject a duplicated appID + buildID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(records.Create(ctx, record)).To(Succeed())

			dup := newRecord("stag", "v2", trafficLaneName)
			dup.BuildID = record.BuildID
			Expect(records.Create(ctx, dup)).To(MatchError(store.ErrDuplicate))
		})
	})

	Describe("UpdateRecord", func() {
		It("should only update mutable fields", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(records.Create(ctx, record)).To(Succeed())

			created, err := records.Get(ctx, autodeploy.ByBuild(appID, record.BuildID))
			Expect(err).NotTo(HaveOccurred())

			endedAt := time.Now()
			created.DeployID = "deploy-" + stringx.Random(8)
			created.Stage = autodeploy.StageDeploy
			created.Status = "deployed"
			created.Message = "ok"
			created.EndedAt = endedAt
			// 不在更新范围内的字段，不应写入
			created.ImageTag = "v2"

			Expect(autodeploy.UpdateRecord(ctx, db, created)).To(Succeed())

			got, err := records.Get(ctx, autodeploy.ByBuild(appID, record.BuildID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.DeployID).To(Equal(created.DeployID))
			Expect(got.Stage).To(Equal(autodeploy.StageDeploy))
			Expect(got.Status).To(Equal("deployed"))
			Expect(got.Message).To(Equal("ok"))
			Expect(got.EndedAt.Unix()).To(Equal(endedAt.Unix()))
			// UpdateRecord 只更新指定字段，ImageTag 等保持不变
			Expect(got.ImageTag).To(Equal("v1"))
		})

		It("should return ErrRecordNotFound when updating a missing record", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			record.ID = bson.NewObjectID()

			err := autodeploy.UpdateRecord(ctx, db, record)
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})
	})

	Describe("GetLatest", func() {
		It("should return the newest record for the env and traffic lane", func() {
			older := newRecord("stag", "stag-v1", trafficLaneName)
			Expect(records.Create(ctx, older)).To(Succeed())
			time.Sleep(5 * time.Millisecond)

			newer := newRecord("stag", "stag-v2", trafficLaneName)
			Expect(records.Create(ctx, newer)).To(Succeed())
			Expect(records.Create(ctx, newRecord("prod", "prod-v1", trafficLaneName))).To(Succeed())
			Expect(records.Create(ctx, newRecord("stag", "lane-v1", "lane-a"))).To(Succeed())

			got, err := autodeploy.GetLatest(ctx, db, appID, "stag", trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ImageTag).To(Equal("stag-v2"))
			Expect(got.BuildID).To(Equal(newer.BuildID))
		})

		It("should return ErrRecordNotFound when no record exists", func() {
			_, err := autodeploy.GetLatest(ctx, db, appID, "stag", trafficLaneName)
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})
	})

	Describe("ListLatestByApps", func() {
		// Create 会把 CreatedAt 写成 time.Now()。BSON DateTime 只有毫秒精度，
		// 连续插入可能落在同一毫秒，导致 $sort + $first 取「最新」不稳定。
		// 创建后再显式回写 createdAt，用明确时间差拉开新旧记录，避免 CI 抖动影响结果。
		createAt := func(record *autodeploy.Record, createdAt time.Time) {
			GinkgoHelper()
			Expect(records.Create(ctx, record)).To(Succeed())
			// GenericStore 不允许更新 createdAt，直接写集合
			_, err := db.Collection(autodeploy.RecordMeta.Collection).UpdateByID(
				ctx, record.ID, bson.M{"$set": bson.M{"createdAt": createdAt}},
			)
			Expect(err).NotTo(HaveOccurred())
		}

		It("should return the newest record per app and env", func() {
			otherAppID := "test-app-" + stringx.Random(6)
			now := time.Now().UTC().Truncate(time.Millisecond)

			createAt(newRecord("stag", "stag-v1", trafficLaneName), now.Add(-2*time.Second))
			Expect(records.Create(ctx, newRecord("stag", "stag-v2", trafficLaneName))).To(Succeed())
			Expect(records.Create(ctx, newRecord("prod", "prod-v1", trafficLaneName))).To(Succeed())

			otherStag := newRecord("stag", "other-stag-v1", trafficLaneName)
			otherStag.AppID = otherAppID
			createAt(otherStag, now.Add(-time.Second))

			otherStagNew := newRecord("stag", "other-stag-v2", trafficLaneName)
			otherStagNew.AppID = otherAppID
			Expect(records.Create(ctx, otherStagNew)).To(Succeed())

			latest, err := autodeploy.ListLatestByApps(ctx, db, []string{appID, otherAppID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(2))
			Expect(latest[appID]).To(HaveLen(2))
			Expect(latest[appID]["stag"].ImageTag).To(Equal("stag-v2"))
			Expect(latest[appID]["prod"].ImageTag).To(Equal("prod-v1"))
			Expect(latest[otherAppID]).To(HaveLen(1))
			Expect(latest[otherAppID]["stag"].ImageTag).To(Equal("other-stag-v2"))
		})

		It("should only return records of the given traffic lane", func() {
			Expect(records.Create(ctx, newRecord("stag", "base-v1", trafficLaneName))).To(Succeed())
			Expect(records.Create(ctx, newRecord("stag", "lane-v1", "feature-lane"))).To(Succeed())

			latest, err := autodeploy.ListLatestByApps(ctx, db, []string{appID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(1))
			Expect(latest[appID]["stag"].ImageTag).To(Equal("base-v1"))

			laneLatest, err := autodeploy.ListLatestByApps(ctx, db, []string{appID}, "feature-lane")
			Expect(err).NotTo(HaveOccurred())
			Expect(laneLatest).To(HaveLen(1))
			Expect(laneLatest[appID]["stag"].ImageTag).To(Equal("lane-v1"))
		})

		It("should not return records of apps not in the list", func() {
			otherAppID := "other-app-" + stringx.Random(6)

			Expect(records.Create(ctx, newRecord("stag", "mine-v1", trafficLaneName))).To(Succeed())

			other := newRecord("stag", "other-v1", trafficLaneName)
			other.AppID = otherAppID
			Expect(records.Create(ctx, other)).To(Succeed())

			latest, err := autodeploy.ListLatestByApps(ctx, db, []string{appID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(1))
			Expect(latest).To(HaveKey(appID))
			Expect(latest).NotTo(HaveKey(otherAppID))
			Expect(latest[appID]["stag"].ImageTag).To(Equal("mine-v1"))
		})

		It("should omit apps that have no record", func() {
			latest, err := autodeploy.ListLatestByApps(ctx, db, []string{"non-existent-app"}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(BeEmpty())
		})

		It("should return an empty map for empty appIDs", func() {
			latest, err := autodeploy.ListLatestByApps(ctx, db, nil, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).NotTo(BeNil())
			Expect(latest).To(BeEmpty())
		})
	})

	Describe("ByBuild", func() {
		It("should match the record by appID and buildID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(records.Create(ctx, record)).To(Succeed())

			got, err := records.Get(ctx, autodeploy.ByBuild(appID, record.BuildID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.BuildID).To(Equal(record.BuildID))
			Expect(got.ImageTag).To(Equal("v1"))
		})

		It("should not match another app's record with the same buildID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(records.Create(ctx, record)).To(Succeed())

			_, err := records.Get(ctx, autodeploy.ByBuild("other-app", record.BuildID))
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})
	})

	Describe("ByDeploy", func() {
		It("should match the record by appID and deployID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(records.Create(ctx, record)).To(Succeed())

			deployID := "deploy-" + stringx.Random(8)
			record.DeployID = deployID
			record.Stage = autodeploy.StageDeploy
			record.Status = "deploying"
			Expect(autodeploy.UpdateRecord(ctx, db, record)).To(Succeed())

			got, err := records.Get(ctx, autodeploy.ByDeploy(appID, deployID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.BuildID).To(Equal(record.BuildID))
			Expect(got.DeployID).To(Equal(deployID))
			Expect(got.Stage).To(Equal(autodeploy.StageDeploy))
		})

		It("should not match another app's record with the same deployID", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(records.Create(ctx, record)).To(Succeed())

			deployID := "deploy-" + stringx.Random(8)
			record.DeployID = deployID
			Expect(autodeploy.UpdateRecord(ctx, db, record)).To(Succeed())

			_, err := records.Get(ctx, autodeploy.ByDeploy("other-app", deployID))
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})

		It("should not match empty deployID", func() {
			// Create 后 DeployID 为空；ByDeploy(appID, "") 不应误命中
			Expect(records.Create(ctx, newRecord("stag", "v1", trafficLaneName))).To(Succeed())

			_, err := records.Get(ctx, autodeploy.ByDeploy(appID, ""))
			Expect(err).To(MatchError(autodeploy.ErrRecordNotFound))
		})
	})

	// 保留一个组合场景，覆盖 Create → UpdateRecord → GetLatest 主路径
	Describe("end-to-end flow", func() {
		It("should reflect deploy progress through store APIs", func() {
			record := newRecord("stag", "v1", trafficLaneName)
			Expect(records.Create(ctx, record)).To(Succeed())

			deployID := "deploy-" + stringx.Random(8)
			record.DeployID = deployID
			record.Stage = autodeploy.StageDeploy
			record.Status = "deployed"
			record.Message = "done"
			record.EndedAt = time.Now()
			Expect(autodeploy.UpdateRecord(ctx, db, record)).To(Succeed())

			latest, err := autodeploy.GetLatest(ctx, db, appID, "stag", trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.DeployID).To(Equal(deployID))
			Expect(latest.Status).To(Equal("deployed"))

			byDeploy, err := records.Get(ctx, autodeploy.ByDeploy(appID, deployID))
			Expect(err).NotTo(HaveOccurred())
			Expect(byDeploy.BuildID).To(Equal(record.BuildID))
		})
	})
})
