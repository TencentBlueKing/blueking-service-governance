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
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("RecordStoreMongo", func() {
	var store appmodel.RecordStore
	var ctx context.Context

	var workspaceID, appID, envName, trafficLaneName string
	var recordA, recordB appmodel.Record

	BeforeEach(func() {
		var err error

		store, err = appmodel.NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

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

	Context("Create", func() {
		It("should create deploy record successfully", func() {
			recordID, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())
			Expect(recordID).NotTo(BeEmpty())

			// 验证 ID 是有效的 ObjectID
			_, err = bson.ObjectIDFromHex(recordID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should set createdAt and updatedAt timestamps", func() {
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			Expect(recordA.CreatedAt).NotTo(BeZero())
			Expect(recordA.UpdatedAt).NotTo(BeZero())
			Expect(recordA.CreatedAt).To(Equal(recordA.UpdatedAt))
		})

		It("should create multiple records for the same app", func() {
			recordID1, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())
			Expect(recordID1).NotTo(BeEmpty())

			recordID2, err := store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())
			Expect(recordID2).NotTo(BeEmpty())

			Expect(recordID1).NotTo(Equal(recordID2))
		})
	})

	Context("Update", func() {
		It("should update deploy record successfully", func() {
			recordID, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 更新记录
			recordA.ID, err = bson.ObjectIDFromHex(recordID)
			Expect(err).NotTo(HaveOccurred())

			recordA.Message = "updated message"
			recordA.Status = "failed"
			recordA.Extras["error"] = "some error"

			err = store.Update(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			record, err := store.Get(ctx, appID, recordA.ID.Hex())
			Expect(err).NotTo(HaveOccurred())
			Expect(record.Message).To(Equal("updated message"))
			Expect(record.Status).To(Equal(appmodel.Status("failed")))
			Expect(record.Extras["error"]).To(Equal("some error"))
		})

		It("should update updatedAt timestamp", func() {
			recordID, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordA.ID, err = bson.ObjectIDFromHex(recordID)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			// 更新消息
			recordA.Message = "updated"
			err = store.Update(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			record, err := store.Get(ctx, appID, recordA.ID.Hex())
			Expect(err).NotTo(HaveOccurred())
			Expect(recordA.UpdatedAt.UnixMilli()).To(BeNumerically("<", record.UpdatedAt.UnixMilli()))
		})

		It("should return error when updating non-existent record", func() {
			recordA.ID = bson.NewObjectID()
			err := store.Update(ctx, &recordA)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("GetLatest", func() {
		It("should get the latest deploy record", func() {
			// 创建第一条记录
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			// 创建第二条记录（应该是最新的）
			recordID2, err := store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 获取最新记录
			latest, err := store.GetLatest(ctx, appID, envName, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).NotTo(BeNil())
			Expect(latest.ID.Hex()).To(Equal(recordID2))
			Expect(latest.ImageTag).To(Equal("v1.0.1"))
			Expect(latest.Replicas).To(Equal(int32(5)))
			Expect(latest.Creator).To(Equal("blueking"))
		})

		It("should return error when no record exists", func() {
			_, err := store.GetLatest(ctx, "non-existent-app", envName, trafficLaneName)
			Expect(err).To(HaveOccurred())
		})

		It("should filter by envName correctly", func() {
			// 为 staging 环境创建记录
			recordA.EnvName = "staging"
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 为 production 环境创建记录
			recordB.EnvName = "production"
			recordID2, err := store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 获取 production 环境的最新记录
			latest, err := store.GetLatest(ctx, appID, "production", trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.ID.Hex()).To(Equal(recordID2))
			Expect(latest.EnvName).To(Equal("production"))
		})

		It("should filter by trafficLaneName correctly", func() {
			// 为 base 泳道创建记录
			recordA.TrafficLaneName = "base"
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 为 feature 泳道创建记录
			recordB.TrafficLaneName = "feature-lane"
			recordID2, err := store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 获取 feature 泳道的最新记录
			latest, err := store.GetLatest(ctx, appID, envName, "feature-lane")
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.ID.Hex()).To(Equal(recordID2))
			Expect(latest.TrafficLaneName).To(Equal("feature-lane"))
		})

		It("should return the most recent record when multiple exist", func() {
			// 创建多条记录
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.ImageTag = "v1.0.2"
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			recordC := recordB
			recordC.ImageTag = "v1.0.3"
			recordID3, err := store.Create(ctx, &recordC)
			Expect(err).NotTo(HaveOccurred())

			// 获取最新记录应该返回最后创建的那条
			latest, err := store.GetLatest(ctx, appID, envName, trafficLaneName)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.ID.Hex()).To(Equal(recordID3))
			Expect(latest.ImageTag).To(Equal("v1.0.3"))
		})

		It("should get the latest record matching statuses", func() {
			recordA.Status = appmodel.StatusDeployed
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// MongoDB DateTime 精度为毫秒，依赖 createdAt 排序的场景需要等到下一个精度窗口
			time.Sleep(5 * time.Millisecond)

			recordB.Status = appmodel.StatusFailed
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// MongoDB DateTime 精度为毫秒，依赖 createdAt 排序的场景需要等到下一个精度窗口
			time.Sleep(5 * time.Millisecond)

			recordC := recordB
			recordC.ImageTag = "v1.0.2"
			recordC.Status = appmodel.StatusDeployed
			recordID3, err := store.Create(ctx, &recordC)
			Expect(err).NotTo(HaveOccurred())

			latest, err := store.GetLatestByStatuses(
				ctx,
				appID,
				envName,
				trafficLaneName,
				[]appmodel.Status{appmodel.StatusDeployed},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).NotTo(BeNil())
			Expect(latest.ID.Hex()).To(Equal(recordID3))
			Expect(latest.Status).To(Equal(appmodel.StatusDeployed))
			Expect(latest.ImageTag).To(Equal("v1.0.2"))
		})

		It("should return error when no record matches statuses", func() {
			recordA.Status = appmodel.StatusFailed
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.GetLatestByStatuses(
				ctx,
				appID,
				envName,
				trafficLaneName,
				[]appmodel.Status{appmodel.StatusDeployed},
			)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("List", func() {
		It("should support pagination and sorting", func() {
			// 创建多条记录
			recordA.ImageTag = "v1.0.0"
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			recordB.ImageTag = "v1.0.1"
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 等待一段时间（5ms 确保时间差）
			time.Sleep(5 * time.Millisecond)

			recordC := recordB
			recordC.ImageTag = "v1.0.2"
			_, err = store.Create(ctx, &recordC)
			Expect(err).NotTo(HaveOccurred())

			// 测试第一页（按创建时间降序排列）
			records, total, err := store.List(ctx, appID, envName, trafficLaneName, "", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(3)))
			Expect(records).To(HaveLen(2))
			Expect(records[0].ImageTag).To(Equal("v1.0.2"))
			Expect(records[1].ImageTag).To(Equal("v1.0.1"))

			// 测试第二页
			records, total, err = store.List(ctx, appID, envName, trafficLaneName, "", 2, 2)
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
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.ImageTag = "v2.0.0"
			recordB.Creator = "blueking"
			recordB.Updater = "blueking"
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			recordC := recordB
			recordC.ImageTag = "v2.1.0+build.123"
			recordC.Creator = "blueking-admin"
			recordC.Updater = "blueking-admin"
			_, err = store.Create(ctx, &recordC)
			Expect(err).NotTo(HaveOccurred())

			// 按 imageTag 搜索
			records, total, err := store.List(ctx, appID, envName, trafficLaneName, "v2", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(2))

			// 按 operator 搜索
			_, total, err = store.List(ctx, appID, envName, trafficLaneName, "blueking", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))

			// 大小写不敏感搜索
			records, total, err = store.List(ctx, appID, envName, trafficLaneName, "release", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records[0].ImageTag).To(Equal("v1.0.0-RELEASE"))

			// 特殊正则字符处理
			records, total, err = store.List(ctx, appID, envName, trafficLaneName, "v2.1.0+", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records[0].ImageTag).To(Equal("v2.1.0+build.123"))
		})

		It("should filter by envName and trafficLaneName", func() {
			// 为不同环境和泳道创建记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "production"
			recordB.TrafficLaneName = "feature-lane"
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 按 envName 过滤
			records, total, err := store.List(ctx, appID, "staging", trafficLaneName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records[0].EnvName).To(Equal("staging"))

			// 按 trafficLaneName 过滤
			records, total, err = store.List(ctx, appID, "production", "feature-lane", "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records[0].TrafficLaneName).To(Equal("feature-lane"))
		})

		It("should return empty list when no records match", func() {
			// 创建记录
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 不存在的关键字
			records, total, err := store.List(ctx, appID, envName, trafficLaneName, "nonexistent", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(records).To(BeEmpty())

			// 不存在的应用
			records, total, err = store.List(ctx, "non-existent-app", envName, trafficLaneName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(records).To(BeEmpty())
		})
	})

	Context("ListByImageTag", func() {
		It("should return empty list when no records match the tag", func() {
			records, total, err := store.ListByImageTag(ctx, appID, "non-existent-tag", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(records).To(BeEmpty())
		})

		It("should return records matching the specified imageTag", func() {
			// 创建匹配 tag 的记录（不同状态）
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			recordA.Status = appmodel.StatusDeployed
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "production"
			recordB.ImageTag = "v1.0.0"
			recordB.Status = appmodel.StatusFailed
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 创建不匹配 tag 的记录
			recordC := recordA
			recordC.ImageTag = "v2.0.0"
			recordC.EnvName = "staging"
			_, err = store.Create(ctx, &recordC)
			Expect(err).NotTo(HaveOccurred())

			// 查询 v1.0.0 的记录，应返回 2 条（不限状态）
			records, total, err := store.ListByImageTag(ctx, appID, "v1.0.0", 1, 10)
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
				r.EnvName = env
				r.ImageTag = "v1.0.0"
				r.Message = "deploy-" + string(rune('A'+i))
				_, err := store.Create(ctx, &r)
				Expect(err).NotTo(HaveOccurred())
				// 等待确保时间差
				if i < 2 {
					time.Sleep(5 * time.Millisecond)
				}
			}

			// 第一页，每页 2 条
			records, total, err := store.ListByImageTag(ctx, appID, "v1.0.0", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(3)))
			Expect(records).To(HaveLen(2))
			// 按 createdAt 降序，最新的在前
			Expect(records[0].EnvName).To(Equal("production"))
			Expect(records[1].EnvName).To(Equal("staging"))

			// 第二页
			records, total, err = store.ListByImageTag(ctx, appID, "v1.0.0", 2, 2)
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
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 另一个应用的同 tag 记录
			recordB.AppID = otherAppID
			recordB.EnvName = "production"
			recordB.ImageTag = "v1.0.0"
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 只应返回当前应用的记录
			records, total, err := store.ListByImageTag(ctx, appID, "v1.0.0", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].AppID).To(Equal(appID))
		})
	})

	Context("ListImageTagDeployedEnvs", func() {
		It("should return empty list when no records exist", func() {
			pairs, err := store.ListImageTagDeployedEnvs(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(pairs).To(BeEmpty())
		})

		It("should deduplicate records with same imageTag and envName", func() {
			// 同一 tag + 同一环境部署两次
			recordA.EnvName = "staging"
			recordA.ImageTag = "v1.0.0"
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "staging"
			recordB.ImageTag = "v1.0.0"
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			pairs, err := store.ListImageTagDeployedEnvs(ctx, appID)
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
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// 另一个应用的记录
			recordB.AppID = otherAppID
			recordB.EnvName = "production"
			recordB.ImageTag = "v2.0.0"
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 只应返回当前应用的记录
			pairs, err := store.ListImageTagDeployedEnvs(ctx, appID)
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
				r.EnvName = pair.env
				r.ImageTag = pair.tag
				r.Message = "deploy-" + string(rune('A'+i))
				_, err := store.Create(ctx, &r)
				Expect(err).NotTo(HaveOccurred())
			}

			pairs, err := store.ListImageTagDeployedEnvs(ctx, appID)
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
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			recordB.EnvName = "staging"
			recordB.ImageTag = "v1.0.0"
			recordB.Status = appmodel.StatusFailed
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 另一个环境只有 failed 和 canceled 记录
			failedRecord := recordA
			failedRecord.EnvName = "production"
			failedRecord.ImageTag = "v2.0.0"
			failedRecord.Status = appmodel.StatusFailed
			_, err = store.Create(ctx, &failedRecord)
			Expect(err).NotTo(HaveOccurred())

			canceledRecord := recordA
			canceledRecord.EnvName = "production"
			canceledRecord.ImageTag = "v3.0.0"
			canceledRecord.Status = appmodel.StatusCanceled
			_, err = store.Create(ctx, &canceledRecord)
			Expect(err).NotTo(HaveOccurred())

			pairs, err := store.ListImageTagDeployedEnvs(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			// 只应返回 staging 的 deployed 记录，failed/canceled 均不返回
			Expect(pairs).To(HaveLen(1))
			Expect(pairs[0].ImageTag).To(Equal("v1.0.0"))
			Expect(pairs[0].EnvName).To(Equal("staging"))
		})
	})

	Context("HasActiveDeployments", func() {
		It("should return false when no records exist", func() {
			hasActive, err := store.HasActiveDeployments(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return true when latest record is deployed", func() {
			// 创建一条 deployed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = appmodel.StatusDeployed
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := store.HasActiveDeployments(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeTrue())
		})

		It("should return false when latest record is uninstalled", func() {
			// 先创建一条 deployed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = appmodel.StatusDeployed
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(5 * time.Millisecond)

			// 再创建一条 uninstalled 记录（最新的）
			recordB.EnvName = "staging"
			recordB.TrafficLaneName = "base"
			recordB.Status = appmodel.StatusUninstalled
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := store.HasActiveDeployments(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return false when latest record is failed", func() {
			// 创建一条 failed 记录
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = appmodel.StatusFailed
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			hasActive, err := store.HasActiveDeployments(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})

		It("should return true if at least one env-lane has deployed as latest", func() {
			// env1: deployed
			recordA.EnvName = "staging"
			recordA.TrafficLaneName = "base"
			recordA.Status = appmodel.StatusDeployed
			_, err := store.Create(ctx, &recordA)
			Expect(err).NotTo(HaveOccurred())

			// env2: uninstalled (已卸载)
			time.Sleep(5 * time.Millisecond)
			recordB.EnvName = "production"
			recordB.TrafficLaneName = "base"
			recordB.Status = appmodel.StatusUninstalled
			_, err = store.Create(ctx, &recordB)
			Expect(err).NotTo(HaveOccurred())

			// 存在一个活跃部署，应返回 true
			hasActive, err := store.HasActiveDeployments(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeTrue())
		})

		It("should not count records from other apps", func() {
			otherAppID := "other-app-" + stringx.Random(6)

			// 当前应用：无记录
			hasActive, err := store.HasActiveDeployments(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())

			// 另一个应用：有 deployed 记录
			recordOther := recordA
			recordOther.AppID = otherAppID
			recordOther.EnvName = "staging"
			recordOther.Status = appmodel.StatusDeployed
			_, err = store.Create(ctx, &recordOther)
			Expect(err).NotTo(HaveOccurred())

			// 当前应用仍应返回 false
			hasActive, err = store.HasActiveDeployments(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasActive).To(BeFalse())
		})
	})
})
