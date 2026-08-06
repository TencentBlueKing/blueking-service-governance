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

package build

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("RecordStoreMongo", func() {
	var (
		store *RecordStoreMongo
		ctx   context.Context
		appID string
	)

	BeforeEach(func() {
		var err error
		store, err = NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
		appID = "store-test-app-" + stringx.Random(8)
	})

	Describe("Create", func() {
		It("should create a record with auto-incrementing build num and reject duplicate appID+buildID", func() {
			// 第一条记录：自动分配序号 1
			record := &Record{
				WorkspaceID:  "ws-test",
				AppID:        appID,
				BuildID:      "b-" + stringx.Random(8),
				PipelineID:   "p-test",
				ChartVersion: "0.0.1",
				Status:       StatusRunning,
				Operator:     "test-user",
			}
			err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.Num).To(Equal(int64(1)))
			Expect(record.StartedAt).NotTo(BeZero())

			// 同一 appID 下序号自增：2, 3
			for i := int64(2); i <= 3; i++ {
				r := &Record{
					WorkspaceID:  "ws-test",
					AppID:        appID,
					BuildID:      "b-" + stringx.Random(8),
					PipelineID:   "p-test",
					ChartVersion: "0.0." + stringx.Random(1),
					Status:       StatusRunning,
					Operator:     "test-user",
				}
				err = store.Create(ctx, r)
				Expect(err).NotTo(HaveOccurred())
				Expect(r.Num).To(Equal(i))
			}

			// 重复的 appID + buildID 应报错
			dupBuildID := "b-dup-" + stringx.Random(8)
			dup := &Record{
				WorkspaceID:  "ws-test",
				AppID:        appID,
				BuildID:      dupBuildID,
				PipelineID:   "p-test",
				ChartVersion: "0.0.1",
				Status:       StatusRunning,
				Operator:     "test-user",
			}
			err = store.Create(ctx, dup)
			Expect(err).NotTo(HaveOccurred())

			dup.ChartVersion = "0.0.2"
			err = store.Create(ctx, dup)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})
	})

	Describe("Get", func() {
		It("should get a record by appID and buildID", func() {
			buildID := "b-get-" + stringx.Random(8)
			record := &Record{
				WorkspaceID:  "ws-test",
				AppID:        appID,
				BuildID:      buildID,
				PipelineID:   "p-test",
				ChartVersion: "1.2.3",
				Status:       StatusRunning,
				Operator:     "test-user",
			}
			err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.Get(ctx, appID, buildID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.AppID).To(Equal(appID))
			Expect(got.BuildID).To(Equal(buildID))
			Expect(got.ChartVersion).To(Equal("1.2.3"))
			Expect(got.Status).To(Equal(StatusRunning))
			Expect(got.Operator).To(Equal("test-user"))
			Expect(got.Num).To(Equal(int64(1)))
		})

		It("should return ErrRecordNotFound when record does not exist", func() {
			_, err := store.Get(ctx, appID, "non-existent-build-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(ErrRecordNotFound.Error()))
		})
	})

	Describe("Update", func() {
		It("should update status, extras and endedAt of an existing record", func() {
			buildID := "b-upd-" + stringx.Random(8)
			record := &Record{
				WorkspaceID:  "ws-test",
				AppID:        appID,
				BuildID:      buildID,
				PipelineID:   "p-test",
				ChartVersion: "0.1.0",
				Status:       StatusRunning,
				Operator:     "test-user",
			}
			err := store.Create(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			now := time.Now()
			record.Status = StatusSuccess
			record.EndedAt = &now
			record.Extras = map[string]string{"BK_CI_GIT_REPO_HEAD_COMMIT_ID": "abc1234"}
			err = store.Update(ctx, record)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.Get(ctx, appID, buildID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Status).To(Equal(StatusSuccess))
			Expect(got.EndedAt).NotTo(BeNil())
			Expect(got.Extras).To(HaveKeyWithValue("BK_CI_GIT_REPO_HEAD_COMMIT_ID", "abc1234"))
		})
	})

	// 创建若干条具有不同 chartVersion 的构建记录，作为 List 的测试数据
	createRecords := func(items []*Record) {
		for _, r := range items {
			r.WorkspaceID = "ws-test"
			r.AppID = appID
			r.PipelineID = "p-test"
			if r.Status == "" {
				r.Status = StatusRunning
			}
			if r.Operator == "" {
				r.Operator = "test-user"
			}
			time.Sleep(5 * time.Millisecond)
			Expect(store.Create(ctx, r)).To(Succeed())
		}
	}

	Describe("List", func() {
		It("should list records ordered by startedAt desc with pagination and keyword filter", func() {
			createRecords([]*Record{
				{BuildID: "b1-" + stringx.Random(4), ChartVersion: "0.0.1", Operator: "alice"},
				{BuildID: "b2-" + stringx.Random(4), ChartVersion: "0.0.2", Operator: "bob"},
				{BuildID: "b3-" + stringx.Random(4), ChartVersion: "0.0.3", Operator: "alice"},
				{BuildID: "b4-" + stringx.Random(4), ChartVersion: "0.0.4", Operator: "carol"},
			})

			records, total, err := store.List(ctx, appID, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(4)))
			Expect(records).To(HaveLen(4))
			// 倒序：最后插入的版本在最前
			Expect(records[0].ChartVersion).To(Equal("0.0.4"))

			// keyword 命中 operator
			records, total, err = store.List(ctx, appID, "alice", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(2))

			// keyword 命中 chartVersion
			records, total, err = store.List(ctx, appID, "0.0.2", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records[0].ChartVersion).To(Equal("0.0.2"))

			// 分页
			records, total, err = store.List(ctx, appID, "", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(4)))
			Expect(records).To(HaveLen(2))
		})
	})
})
