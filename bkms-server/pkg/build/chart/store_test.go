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
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("RecordStore", func() {
	var (
		db      *mongo.Database
		records *RecordStore
		ctx     context.Context
		appID   string
	)

	BeforeEach(func() {
		db = testutil.NewDB(GinkgoT())
		records = store.Of(db, RecordMeta)
		ctx = context.Background()
		appID = "store-test-app-" + stringx.Random(8)
	})

	Describe("CreateRecord", func() {
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
			err := CreateRecord(ctx, db, record)
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
				err = CreateRecord(ctx, db, r)
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
			err = CreateRecord(ctx, db, dup)
			Expect(err).NotTo(HaveOccurred())

			dup.ChartVersion = "0.0.2"
			err = CreateRecord(ctx, db, dup)
			Expect(err).To(MatchError(store.ErrDuplicate))
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
			err := CreateRecord(ctx, db, record)
			Expect(err).NotTo(HaveOccurred())

			got, err := records.Get(ctx, ByBuild(appID, buildID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.AppID).To(Equal(appID))
			Expect(got.BuildID).To(Equal(buildID))
			Expect(got.ChartVersion).To(Equal("1.2.3"))
			Expect(got.Status).To(Equal(StatusRunning))
			Expect(got.Operator).To(Equal("test-user"))
			Expect(got.Num).To(Equal(int64(1)))
		})

		It("should return ErrRecordNotFound when record does not exist", func() {
			_, err := records.Get(ctx, ByBuild(appID, "non-existent-build-id"))
			Expect(err).To(MatchError(ErrRecordNotFound))

			err = UpdateRecord(ctx, records, &Record{AppID: appID, BuildID: "non-existent-build-id"})
			Expect(err).To(MatchError(ErrRecordNotFound))
		})
	})

	Describe("UpdateRecord", func() {
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
			err := CreateRecord(ctx, db, record)
			Expect(err).NotTo(HaveOccurred())

			now := time.Now()
			record.Status = StatusSuccess
			record.EndedAt = &now
			record.Extras = map[string]string{"BK_CI_GIT_REPO_HEAD_COMMIT_ID": "abc1234"}
			err = UpdateRecord(ctx, records, record)
			Expect(err).NotTo(HaveOccurred())

			got, err := records.Get(ctx, ByBuild(appID, buildID))
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
			Expect(CreateRecord(ctx, db, r)).To(Succeed())
		}
	}

	listPage := func(keyword string, page, size int64) *store.Page[Record] {
		result, err := records.ListPage(ctx, page, size, ByApp(appID), KeywordMatches(keyword), NewestFirst())
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	Describe("ListPage", func() {
		It("should list records ordered by startedAt desc with pagination and keyword filter", func() {
			createRecords([]*Record{
				{BuildID: "b1-" + stringx.Random(4), ChartVersion: "0.0.1", Operator: "alice"},
				{BuildID: "b2-" + stringx.Random(4), ChartVersion: "0.0.2", Operator: "bob"},
				{BuildID: "b3-" + stringx.Random(4), ChartVersion: "0.0.3", Operator: "alice"},
				{BuildID: "b4-" + stringx.Random(4), ChartVersion: "0.0.4", Operator: "carol"},
			})

			page := listPage("", 1, 10)
			Expect(page.Total).To(Equal(int64(4)))
			Expect(page.Items).To(HaveLen(4))
			// 倒序：最后插入的版本在最前
			Expect(page.Items[0].ChartVersion).To(Equal("0.0.4"))

			// keyword 命中 operator
			page = listPage("alice", 1, 10)
			Expect(page.Total).To(Equal(int64(2)))
			Expect(page.Items).To(HaveLen(2))

			// keyword 命中 chartVersion
			page = listPage("0.0.2", 1, 10)
			Expect(page.Total).To(Equal(int64(1)))
			Expect(page.Items[0].ChartVersion).To(Equal("0.0.2"))

			// 分页
			page = listPage("", 2, 3)
			Expect(page.Total).To(Equal(int64(4)))
			Expect(page.HasNext).To(BeFalse())
			Expect(page.Items).To(HaveLen(1))
			Expect(page.Items[0].ChartVersion).To(Equal("0.0.1"))
		})
	})
})
