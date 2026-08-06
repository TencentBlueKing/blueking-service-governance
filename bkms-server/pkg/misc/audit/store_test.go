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

package audit

import (
	"context"
	"strings"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("OperationRecordStoreMongo", func() {
	var store OperationRecordStore
	var ctx context.Context

	BeforeEach(func() {
		var err error
		// 创建 store 实例
		store, err = NewOperationRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
	})

	AfterEach(func() {
		if store != nil {
			// 清理测试数据
			_ = store.DeleteAll(ctx)
		}
	})

	Describe("Create", func() {
		It("should create a new operation record successfully", func() {
			record := &OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      "blueking",
				AccessType:    AccessTypeWeb,
				OperationType: OperationTypeCreate,
				ResourceType:  ResourceTypeApp,
				ResourceID:    bson.NewObjectID().Hex(),
				Attribute:     "test-app",
				Result:        ResultSuccess,
			}

			id, err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(BeEmpty())

			// 验证创建的记录可以被获取
			retrieved, err := store.Get(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Username).To(Equal("blueking"))
			Expect(retrieved.OperationType).To(Equal(OperationTypeCreate))
			Expect(retrieved.ResourceType).To(Equal(ResourceTypeApp))
			Expect(retrieved.Attribute).To(Equal(Attribute("test-app")))
		})

		It("should create operation record with data before and after", func() {
			dataBefore := []byte(`{"name": "old-name"}`)
			dataAfter := []byte(`{"name": "new-name"}`)

			record := &OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      "admin",
				AccessType:    AccessTypeAPI,
				OperationType: OperationTypeUpdate,
				ResourceType:  ResourceTypeWorkspace,
				ResourceID:    bson.NewObjectID().Hex(),
				Attribute:     "test-workspace",
				Result:        ResultSuccess,
				Data: OperationData{
					Before: dataBefore,
					After:  dataAfter,
				},
			}

			id, err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := store.Get(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Data.Before).To(Equal(dataBefore))
			Expect(retrieved.Data.After).To(Equal(dataAfter))
		})
	})

	Describe("Get", func() {
		var recordID string

		BeforeEach(func() {
			record := &OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      "getUser",
				AccessType:    AccessTypeWeb,
				OperationType: OperationTypeDelete,
				ResourceType:  ResourceTypeEnv,
				ResourceID:    bson.NewObjectID().Hex(),
				Attribute:     "test-env",
				Result:        ResultSuccess,
			}

			var err error
			recordID, err = store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get operation record by valid ID", func() {
			record, err := store.Get(ctx, recordID)
			Expect(err).NotTo(HaveOccurred())
			Expect(record).NotTo(BeNil())
			Expect(record.Username).To(Equal("getUser"))
			Expect(record.OperationType).To(Equal(OperationTypeDelete))
		})

		It("should return error for invalid ID format", func() {
			_, err := store.Get(ctx, "invalid-id")
			Expect(err).To(HaveOccurred())
		})

		It("should return error for non-existent ID", func() {
			nonExistentID := bson.NewObjectID().Hex()
			_, err := store.Get(ctx, nonExistentID)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Update", func() {
		var recordID string
		var originalRecord *OperationRecord

		BeforeEach(func() {
			originalRecord = &OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      "newUser",
				AccessType:    AccessTypeWeb,
				OperationType: OperationTypeDeploy,
				ResourceType:  ResourceTypeApp,
				ResourceID:    bson.NewObjectID().Hex(),
				Attribute:     "deploy-app",
				Result:        ResultSuccess,
			}

			var err error
			recordID, err = store.Create(ctx, originalRecord)
			Expect(err).NotTo(HaveOccurred())

			// 获取创建的记录以获得完整的 ID
			originalRecord, err = store.Get(ctx, recordID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update operation result", func() {
			originalRecord.Result = ResultSuccess
			err := store.Update(ctx, originalRecord)
			Expect(err).NotTo(HaveOccurred())

			updated, err := store.Get(ctx, recordID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Result).To(Equal(ResultSuccess))
		})

		It("should update data before and after", func() {
			dataBefore := []byte(`{"status": "pending"}`)
			dataAfter := []byte(`{"status": "completed"}`)

			originalRecord.Data.Before = dataBefore
			originalRecord.Data.After = dataAfter
			err := store.Update(ctx, originalRecord)
			Expect(err).NotTo(HaveOccurred())

			updated, err := store.Get(ctx, recordID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Data.Before).To(Equal(dataBefore))
			Expect(updated.Data.After).To(Equal(dataAfter))
		})
	})

	Describe("List", func() {
		var username1, username2 string
		var resourceID1, resourceID2 string

		BeforeEach(func() {
			username1 = "user1-" + stringx.Random(6)
			username2 = "user2-" + stringx.Random(6)
			resourceID1 = bson.NewObjectID().Hex()
			resourceID2 = bson.NewObjectID().Hex()

			// 创建多条测试记录
			records := []*OperationRecord{
				{
					ID:            bson.NewObjectID(),
					Username:      username1,
					AccessType:    AccessTypeWeb,
					OperationType: OperationTypeCreate,
					ResourceType:  ResourceTypeApp,
					ResourceID:    resourceID1,
					Attribute:     "app1",
					Result:        ResultSuccess,
				},
				{
					ID:            bson.NewObjectID(),
					Username:      username1,
					AccessType:    AccessTypeAPI,
					OperationType: OperationTypeUpdate,
					ResourceType:  ResourceTypeApp,
					ResourceID:    resourceID1,
					Attribute:     "app1",
					Result:        ResultSuccess,
				},
				{
					ID:            bson.NewObjectID(),
					Username:      username2,
					AccessType:    AccessTypeWeb,
					OperationType: OperationTypeDelete,
					ResourceType:  ResourceTypeWorkspace,
					ResourceID:    resourceID2,
					Attribute:     "workspace1",
					Result:        ResultFailed,
				},
				{
					ID:            bson.NewObjectID(),
					Username:      username2,
					AccessType:    AccessTypeWeb,
					OperationType: OperationTypeBuild,
					ResourceType:  ResourceTypeApp,
					ResourceID:    resourceID2,
					Attribute:     "app2",
					Result:        ResultSuccess,
				},
			}

			for _, record := range records {
				_, err := store.Create(ctx, record)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should list all records with pagination", func() {
			opts := ListOptions{
				Page:     1,
				PageSize: 5,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(4)))
			Expect(len(records)).To(Equal(4))
		})

		It("should filter by username", func() {
			opts := ListOptions{
				Username: username1,
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(len(records)).To(Equal(2))
			for _, record := range records {
				Expect(record.Username).To(Equal(username1))
			}
		})

		It("should filter by username case-insensitively", func() {
			opts := ListOptions{
				Username: strings.ToUpper(username1),
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(len(records)).To(Equal(2))
			for _, record := range records {
				Expect(record.Username).To(Equal(username1))
			}
		})

		It("should filter by operation type", func() {
			opts := ListOptions{
				OpType:   OperationTypeCreate,
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 1))
			for _, record := range records {
				Expect(record.OperationType).To(Equal(OperationTypeCreate))
			}
		})

		It("should filter by resource type", func() {
			opts := ListOptions{
				ResType:  ResourceTypeApp,
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 3))
			for _, record := range records {
				Expect(record.ResourceType).To(Equal(ResourceTypeApp))
			}
		})

		It("should filter by result", func() {
			opts := ListOptions{
				Result:   ResultSuccess,
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 2))
			for _, record := range records {
				Expect(record.Result).To(Equal(ResultSuccess))
			}
		})

		It("should support pagination", func() {
			// 第一页
			opts := ListOptions{
				Page:     1,
				PageSize: 2,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 4))
			Expect(len(records)).To(Equal(2))

			// 第二页
			opts.Page = 2
			records, total, err = store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 4))
			Expect(len(records)).To(BeNumerically(">=", 2))
		})

		It("should combine multiple filters", func() {
			opts := ListOptions{
				Username: username2,
				ResType:  ResourceTypeApp,
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(len(records)).To(Equal(1))
			Expect(records[0].Username).To(Equal(username2))
			Expect(records[0].ResourceType).To(Equal(ResourceTypeApp))
		})

		It("should filter by workspace ID", func() {
			// 创建带有分组信息的记录
			workspaceID := "workspace-test-" + stringx.Random(6)
			record := &OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      "groupUser",
				AccessType:    AccessTypeWeb,
				OperationType: OperationTypeCreate,
				ResourceType:  ResourceTypeApp,
				ResourceID:    bson.NewObjectID().Hex(),
				Attribute:     "grouped-app",
				Result:        ResultSuccess,
				Group: OperationGroup{
					WorkspaceID: workspaceID,
				},
			}
			_, err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			// 按 workspaceID 过滤
			opts := ListOptions{
				WorkspaceID: workspaceID,
				Page:        1,
				PageSize:    10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 1))
			for _, r := range records {
				Expect(r.Group.WorkspaceID).To(Equal(workspaceID))
			}
		})

		It("should filter by app ID", func() {
			// 创建带有分组信息的记录
			appID := "app-test-" + stringx.Random(6)
			record := &OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      "appUser",
				AccessType:    AccessTypeWeb,
				OperationType: OperationTypeUpdate,
				ResourceType:  ResourceTypeApp,
				ResourceID:    bson.NewObjectID().Hex(),
				Attribute:     "app-with-group",
				Result:        ResultSuccess,
				Group: OperationGroup{
					AppID: appID,
				},
			}
			_, err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			// 按 appID 过滤
			opts := ListOptions{
				AppID:    appID,
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 1))
			for _, r := range records {
				Expect(r.Group.AppID).To(Equal(appID))
			}
		})

		It("should filter by env name", func() {
			// 创建带有分组信息的记录
			workspaceID := "workspace-test-" + stringx.Random(6)
			envName := "staging"
			record := &OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      "envUser",
				AccessType:    AccessTypeWeb,
				OperationType: OperationTypeDeploy,
				ResourceType:  ResourceTypeEnv,
				ResourceID:    bson.NewObjectID().Hex(),
				Attribute:     "env-with-group",
				Result:        ResultSuccess,
				Group: OperationGroup{
					WorkspaceID: workspaceID,
					EnvName:     envName,
				},
			}
			_, err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			// 按 envName 过滤
			opts := ListOptions{
				EnvName:  envName,
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 1))
			for _, r := range records {
				Expect(r.Group.EnvName).To(Equal(envName))
			}
		})

		It("should filter by multiple group fields", func() {
			// 创建带有完整分组信息的记录
			workspaceID := "workspace-test-" + stringx.Random(6)
			appID := "app-test-" + stringx.Random(6)
			envName := "staging"
			record := &OperationRecord{
				ID:            bson.NewObjectID(),
				Username:      "multigroup",
				AccessType:    AccessTypeWeb,
				OperationType: OperationTypeDeploy,
				ResourceType:  ResourceTypeApp,
				ResourceID:    bson.NewObjectID().Hex(),
				Attribute:     "multi-grouped-app",
				Result:        ResultSuccess,
				Group: OperationGroup{
					WorkspaceID: workspaceID,
					AppID:       appID,
					EnvName:     envName,
				},
			}
			_, err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			// 按多个分组字段过滤
			opts := ListOptions{
				WorkspaceID: workspaceID,
				AppID:       appID,
				EnvName:     envName,
				Page:        1,
				PageSize:    10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeNumerically(">=", 1))
			for _, r := range records {
				Expect(r.Group.WorkspaceID).To(Equal(workspaceID))
				Expect(r.Group.AppID).To(Equal(appID))
				Expect(r.Group.EnvName).To(Equal(envName))
			}
		})

		It("should return empty list when no records match", func() {
			opts := ListOptions{
				Username: "nonexistent-user",
				Page:     1,
				PageSize: 10,
			}

			records, total, err := store.List(ctx, opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(len(records)).To(Equal(0))
		})
	})

	Describe("ListOptions.AsFilter", func() {
		It("should create empty filter when no options set", func() {
			opts := ListOptions{}
			filter := opts.AsFilter()
			Expect(filter).To(BeEmpty())
		})

		It("should create filter with username", func() {
			opts := ListOptions{Username: "blueking"}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("username"))
			Expect(filter["username"]).To(Equal(bson.M{
				"$regex": "blueking", "$options": "i",
			}))
		})

		It("should create filter with operation type", func() {
			opts := ListOptions{OpType: OperationTypeCreate}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("operationType"))
			Expect(filter["operationType"]).To(Equal(OperationTypeCreate))
		})

		It("should create filter with resource type", func() {
			opts := ListOptions{ResType: ResourceTypeApp}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("resourceType"))
			Expect(filter["resourceType"]).To(Equal(ResourceTypeApp))
		})

		It("should create filter with result", func() {
			opts := ListOptions{Result: ResultSuccess}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("result"))
			Expect(filter["result"]).To(Equal(ResultSuccess))
		})

		It("should create filter with workspace ID", func() {
			workspaceID := "workspace-test-" + stringx.Random(6)
			opts := ListOptions{WorkspaceID: workspaceID}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("group.workspaceID"))
			Expect(filter["group.workspaceID"]).To(Equal(workspaceID))
		})

		It("should create filter with app ID", func() {
			appID := "app-test-" + stringx.Random(6)
			opts := ListOptions{AppID: appID}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("group.appID"))
			Expect(filter["group.appID"]).To(Equal(appID))
		})

		It("should not create filter only with envName", func() {
			envName := "production"
			opts := ListOptions{EnvName: envName}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("group.envName"))
			Expect(filter["group.envName"]).To(Equal(envName))
		})

		It("should create filter with all group fields", func() {
			workspaceID := "workspace-test-" + stringx.Random(6)
			appID := "app-test-" + stringx.Random(6)
			envName := "staging"
			opts := ListOptions{
				WorkspaceID: workspaceID,
				AppID:       appID,
				EnvName:     envName,
			}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("group.workspaceID"))
			Expect(filter).To(HaveKey("group.appID"))
			Expect(filter).To(HaveKey("group.envName"))
			Expect(filter["group.workspaceID"]).To(Equal(workspaceID))
			Expect(filter["group.appID"]).To(Equal(appID))
			Expect(filter["group.envName"]).To(Equal(envName))
		})

		It("should accept any workspace ID string", func() {
			opts := ListOptions{
				WorkspaceID: "custom-workspace-id",
			}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("group.workspaceID"))
			Expect(filter["group.workspaceID"]).To(Equal("custom-workspace-id"))
		})

		It("should accept any app ID string", func() {
			opts := ListOptions{
				AppID: "custom-app-id",
			}
			filter := opts.AsFilter()
			Expect(filter).To(HaveKey("group.appID"))
			Expect(filter["group.appID"]).To(Equal("custom-app-id"))
		})
	})
})
