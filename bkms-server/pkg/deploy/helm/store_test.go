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

package helm

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("DeployRecordStore", func() {
	var recordStore *RecordStore
	var db *mongo.Database
	var ctx context.Context

	var workspaceID, appName, appID, envName, trafficLaneName string
	var recordA, recordB Record

	// listRecords 按 handler 的查询组合分页列出当前应用在指定环境、泳道下的部署记录
	listRecords := func(envName, laneName, keyword string, page, size int64) ([]Record, int64, error) {
		p, err := recordStore.ListPage(ctx, page, size,
			deploytypes.ByAppEnvLane(appID, envName, laneName), KeywordMatches(keyword), deploytypes.NewestFirst(),
		)
		if err != nil {
			return nil, 0, err
		}
		return p.Items, p.Total, nil
	}
	// listByImageTag 按 handler 的查询组合分页列出当前应用使用指定镜像标签的部署记录
	listByImageTag := func(imageTag string, page, size int64) ([]Record, int64, error) {
		p, err := recordStore.ListPage(ctx, page, size, ByImageTag(appID, imageTag), deploytypes.NewestFirst())
		if err != nil {
			return nil, 0, err
		}
		return p.Items, p.Total, nil
	}

	BeforeEach(func() {
		db = testutil.NewDB(GinkgoT())
		recordStore = store.Of(db, RecordMeta)

		ctx = context.Background()
		workspaceID = "test-workspace-" + stringx.Random(6)
		projectCode := "bkms-" + workspaceID
		appName = "test-app-" + stringx.Random(6)
		appID = appName + "-" + stringx.Random(6)
		envName = "staging"
		trafficLaneName = "base"

		recordA = Record{
			WorkspaceID:  workspaceID,
			AppID:        appID,
			Revision:     "674",
			ProjectCode:  projectCode,
			ClusterID:    "BCS-K8S-12345",
			Namespace:    "default",
			ReleaseName:  appName,
			ChartName:    appName,
			ChartVersion: "1.0.0",
			Message:      "deployed by admin",
			Status:       helm.StatusDeployed,
			Operator:     "admin",
			// The test case should update EnvName field
			EnvName:         "to-be-updated",
			TrafficLaneName: trafficLaneName,
		}
		recordB = Record{
			WorkspaceID:  workspaceID,
			AppID:        appID,
			Revision:     "675",
			ProjectCode:  projectCode,
			ClusterID:    "BCS-K8S-54321",
			Namespace:    "blueking",
			ReleaseName:  appName,
			ChartName:    appName,
			ChartVersion: "1.0.1",
			Message:      "failed to deploy",
			Status:       helm.StatusFailed,
			Operator:     "blueking",
			// The test case should update EnvName field
			EnvName:         "to-be-updated",
			TrafficLaneName: trafficLaneName,
		}
	})

	Context("Create List Update Get", func() {
		It("should create, list, update and get deploy record successfully", func() {
			// Set the EnvName to the default one before any actions.
			recordA.EnvName = envName
			recordB.EnvName = envName

			// Create
			err := recordStore.Create(ctx, &recordA)
			recordID := recordA.ID.Hex()
			Expect(err).NotTo(HaveOccurred())

			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// List
			records, total, err := listRecords(envName, trafficLaneName, "", 1, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(1))

			// Update
			err = UpdateRecord(ctx, db, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// Get
			r, err := GetRecord(ctx, db, appID, recordID)
			Expect(err).NotTo(HaveOccurred())
			Expect(r).NotTo(BeNil())
			Expect(r.ID.Hex()).To(Equal(recordID))
			Expect(r.AppID).To(Equal(appID))
			Expect(r.Revision).To(Equal("674"))
			Expect(r.Status).To(Equal(helm.StatusDeployed))
			Expect(r.Operator).To(Equal("admin"))
		})
	})

	Context("GetLatest with statuses", func() {
		It("should get the latest record matching statuses", func() {
			recordA.EnvName = envName
			recordA.Status = helm.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(5 * time.Millisecond)

			recordB.EnvName = envName
			recordB.Status = helm.StatusFailed
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(5 * time.Millisecond)

			recordC := recordB
			recordC.ID = bson.NilObjectID
			recordC.ImageTag = "v1.0.2"
			recordC.Status = helm.StatusDeployed
			err = recordStore.Create(ctx, &recordC)
			recordID3 := recordC.ID.Hex()
			Expect(err).NotTo(HaveOccurred())

			latest, err := GetLatest(
				ctx,
				db,
				appID,
				envName,
				trafficLaneName,
				helm.StatusDeployed,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).NotTo(BeNil())
			Expect(latest.ID.Hex()).To(Equal(recordID3))
			Expect(latest.Status).To(Equal(helm.StatusDeployed))
			Expect(latest.ImageTag).To(Equal("v1.0.2"))
		})

		It("should return ErrLatestDeployRecordNotFound when no record matches statuses", func() {
			recordA.EnvName = envName
			recordA.Status = helm.StatusFailed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			_, err = GetLatest(
				ctx,
				db,
				appID,
				envName,
				trafficLaneName,
				helm.StatusDeployed,
			)
			Expect(err).To(MatchError(ErrLatestDeployRecordNotFound))
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
			_, err = db.Collection(RecordMeta.Collection).UpdateByID(
				ctx, objID, bson.M{"$set": bson.M{"createdAt": createdAt}},
			)
			Expect(err).NotTo(HaveOccurred())
		}

		It("should return the newest record per app and env", func() {
			otherAppID := "test-app-" + stringx.Random(6)
			now := time.Now().UTC().Truncate(time.Millisecond)

			recordA.EnvName = envName
			recordA.ImageTag = "v1.0.0"
			err := recordStore.Create(ctx, &recordA)
			idA := recordA.ID.Hex()
			Expect(err).NotTo(HaveOccurred())
			persistCreatedAt(idA, now.Add(-2*time.Second))

			recordB.EnvName = envName
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
			otherStag.EnvName = envName
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

			latest, err := ListLatestByApps(ctx, db, []string{appID, otherAppID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(2))
			Expect(latest[appID]).To(HaveLen(2))
			Expect(latest[appID][envName].ImageTag).To(Equal("v1.0.1"))
			Expect(latest[appID]["production"].ImageTag).To(Equal("prod-v1"))
			Expect(latest[otherAppID]).To(HaveLen(1))
			Expect(latest[otherAppID][envName].ImageTag).To(Equal("other-stag-v2"))
		})

		It("should only return records of the given traffic lane", func() {
			recordA.EnvName = envName
			recordA.TrafficLaneName = trafficLaneName
			recordA.ImageTag = "base-v1"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = envName
			recordB.TrafficLaneName = "feature-lane"
			recordB.ImageTag = "lane-v1"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			latest, err := ListLatestByApps(ctx, db, []string{appID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(1))
			Expect(latest[appID][envName].ImageTag).To(Equal("base-v1"))

			laneLatest, err := ListLatestByApps(ctx, db, []string{appID}, "feature-lane")
			Expect(err).NotTo(HaveOccurred())
			Expect(laneLatest).To(HaveLen(1))
			Expect(laneLatest[appID][envName].ImageTag).To(Equal("lane-v1"))
		})

		It("should not return records of apps not in the list", func() {
			otherAppID := "other-app-" + stringx.Random(6)

			recordA.EnvName = envName
			recordA.ImageTag = "v1.0.0"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			other := recordA
			other.ID = bson.NilObjectID
			other.AppID = otherAppID
			other.ImageTag = "other-v1"
			err = recordStore.Create(ctx, &other)
			Expect(err).NotTo(HaveOccurred())

			latest, err := ListLatestByApps(ctx, db, []string{appID}, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(1))
			Expect(latest).To(HaveKey(appID))
			Expect(latest).NotTo(HaveKey(otherAppID))
			Expect(latest[appID][envName].ImageTag).To(Equal("v1.0.0"))
		})
	})

	Context("Environment Data Separation", func() {
		It("should only return records with matching envName", func() {
			recordA.EnvName = envName

			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// List with correct envName - should return 1 records
			records, _, err := listRecords(envName, trafficLaneName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(records).To(HaveLen(1))

			// List with incorrect envName - should return 0 records
			records, _, err = listRecords("non-existent-env", trafficLaneName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(records).To(HaveLen(0))
		})
	})

	Context("Keyword Search", func() {
		It("should return records with matching keyword", func() {
			recordA.EnvName = envName
			recordB.EnvName = envName

			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// List with correct keyword - should return 1 records
			records, _, err := listRecords(envName, trafficLaneName, "1.0.0", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(records).To(HaveLen(1))

			// List with incorrect keyword - should return 0 records
			records, _, err = listRecords(envName,
				trafficLaneName,
				"non-existent-keyword",
				1,
				10,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(records).To(HaveLen(0))
		})

		It("should perform case-insensitive search", func() {
			recordA.EnvName = envName
			recordA.Operator = "Admin"
			recordB.EnvName = envName
			recordB.Operator = "blueking"

			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// Search with lowercase - should match recordA (case-insensitive)
			records, _, err := listRecords(envName, trafficLaneName, "admin", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(records).To(HaveLen(1))
			Expect(records[0].Operator).To(Equal("Admin"))

			// Search with uppercase - should match recordB (case-insensitive)
			records, _, err = listRecords(envName, trafficLaneName, "BLUE", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(records).To(HaveLen(1))
			Expect(records[0].Operator).To(Equal("blueking"))
		})

		It("should perform substring matching", func() {
			recordA.EnvName = envName
			recordA.ChartVersion = "v1.2.3-alpha"
			recordB.EnvName = envName
			recordB.ImageTag = "20231015-abc123"

			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// Search for substring in the middle - should match recordA
			records, _, err := listRecords(envName, trafficLaneName, "2.3", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(records).To(HaveLen(1))
			Expect(records[0].ChartVersion).To(Equal("v1.2.3-alpha"))

			// Search for substring at the end - should match recordB
			records, _, err = listRecords(envName, trafficLaneName, "abc123", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(records).To(HaveLen(1))
			Expect(records[0].ImageTag).To(Equal("20231015-abc123"))
		})
	})

	Context("ListByImageTag", func() {
		It("should return records matching the specified imageTag", func() {
			// 创建匹配 tag 的记录（不同状态）
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			recordA.Status = helm.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "production"
			recordB.ImageTag = "v1.0.0"
			// recordB 默认 StatusFailed
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
			pairs, err := ListImageTagDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(BeEmpty())
		})

		It("should deduplicate records with same imageTag and envName", func() {
			// 同一 tag + 同一环境部署两次
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.Status = helm.StatusDeployed
			recordB.EnvName = "staging"
			recordB.ImageTag = "v1.0.0"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			pairs, err := ListImageTagDeployedEnvs(ctx, db, appID)
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
			recordB.Status = helm.StatusDeployed
			recordB.AppID = otherAppID
			recordB.EnvName = "production"
			recordB.ImageTag = "v2.0.0"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 只应返回当前应用的记录
			pairs, err := ListImageTagDeployedEnvs(ctx, db, appID)
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

			pairs, err := ListImageTagDeployedEnvs(ctx, db, appID)
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
			recordA.Status = helm.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "staging"
			recordB.ImageTag = "v1.0.0"
			// recordB 默认 StatusFailed，无需修改
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 另一个环境只有 failed 记录
			failedRecord := recordA
			failedRecord.ID = bson.NilObjectID
			failedRecord.EnvName = "production"
			failedRecord.ImageTag = "v2.0.0"
			failedRecord.Status = helm.StatusFailed
			err = recordStore.Create(ctx, &failedRecord)
			Expect(err).NotTo(HaveOccurred())

			pairs, err := ListImageTagDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			// 只应返回 staging 的 deployed 记录，failed 均不返回
			Expect(pairs).To(HaveLen(1))
			Expect(pairs[0].ImageTag).To(Equal("v1.0.0"))
			Expect(pairs[0].EnvName).To(Equal("staging"))
		})
	})

	Context("ListChartVersionDeployedEnvs", func() {
		It("should return empty list when no records exist", func() {
			pairs, err := ListChartVersionDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(BeEmpty())
		})

		It("should deduplicate records with same chartVersion and envName", func() {
			recordA.EnvName = "staging"
			recordA.ChartVersion = "0.1.0"
			recordA.Status = helm.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.Status = helm.StatusDeployed
			recordB.EnvName = "staging"
			recordB.ChartVersion = "0.1.0"
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			pairs, err := ListChartVersionDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(HaveLen(1))
			Expect(pairs[0].ChartVersion).To(Equal("0.1.0"))
			Expect(pairs[0].EnvName).To(Equal("staging"))
		})

		It("should handle multiple chart versions across multiple envs and ignore non-deployed", func() {
			// 0.1.0 → staging + production, 0.2.0 → staging
			recordA.Status = helm.StatusDeployed
			for _, pair := range []struct{ ver, env string }{
				{"0.1.0", "staging"},
				{"0.1.0", "production"},
				{"0.2.0", "staging"},
			} {
				r := recordA
				r.ID = bson.NilObjectID
				r.EnvName = pair.env
				r.ChartVersion = pair.ver
				err := recordStore.Create(ctx, &r)
				Expect(err).NotTo(HaveOccurred())
			}

			// 一条 failed 记录不应被统计
			failed := recordA
			failed.ID = bson.NilObjectID
			failed.EnvName = "production"
			failed.ChartVersion = "0.3.0"
			failed.Status = helm.StatusFailed
			err := recordStore.Create(ctx, &failed)
			Expect(err).NotTo(HaveOccurred())

			pairs, err := ListChartVersionDeployedEnvs(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(HaveLen(3))

			pairSet := make(map[string]bool, len(pairs))
			for _, p := range pairs {
				pairSet[p.ChartVersion+"|"+p.EnvName] = true
			}
			Expect(pairSet).To(HaveKey("0.1.0|staging"))
			Expect(pairSet).To(HaveKey("0.1.0|production"))
			Expect(pairSet).To(HaveKey("0.2.0|staging"))
			Expect(pairSet).NotTo(HaveKey("0.3.0|production"))
		})
	})

	Context("HasActiveDeployments", func() {
		It("should return false when no records exist", func() {
			hasActive, err := HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return true when latest record is deployed", func() {
			// 创建一条 deployed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = helm.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeTrue())
		})

		It("should return false when latest record is uninstalled", func() {
			// 先创建一条 deployed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = helm.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(5 * time.Millisecond)

			// 再创建一条 uninstalled 记录（最新的）
			recordB.EnvName = "staging"
			recordB.TrafficLaneName = "base"
			recordB.Status = helm.StatusUninstalled
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return false when latest record is failed", func() {
			// 创建一条 failed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = helm.StatusFailed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return true if at least one env-lane has deployed as latest", func() {
			// env1: deployed
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = helm.StatusDeployed
			err := recordStore.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// env2: uninstalled (已卸载)
			time.Sleep(5 * time.Millisecond)
			recordB.EnvName = "production"
			recordB.TrafficLaneName = "base"
			recordB.Status = helm.StatusUninstalled
			err = recordStore.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 存在一个活跃部署，应返回 true
			hasActive, err := HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeTrue())
		})

		It("should not count records from other apps", func() {
			otherAppID := "other-app-" + stringx.Random(6)

			// 当前应用：无记录
			hasActive, err := HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())

			// 另一个应用：有 deployed 记录
			recordOther := recordA
			recordOther.ID = bson.NilObjectID
			recordOther.AppID = otherAppID
			recordOther.EnvName = "staging"
			recordOther.Status = helm.StatusDeployed
			err = recordStore.Create(ctx, &recordOther)
			Expect(err).NotTo(HaveOccurred())

			// 当前应用仍应返回 false
			hasActive, err = HasActiveDeployments(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})
	})
})
