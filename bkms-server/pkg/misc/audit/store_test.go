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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("OperationRecordStore", func() {
	var (
		ctx     context.Context
		db      *mongo.Database
		records *OperationRecordStore
		base    time.Time
	)

	// mustCreate 以 base 为基准、offset 为偏移设置创建时间，便于断言排序
	mustCreate := func(offset time.Duration, username string, opts ...Option) *OperationRecord {
		r := newOperationRecord(username, OperationTypeCreate, ResourceTypeApp, "res", opts...)
		r.CreatedAt = base.Add(offset)
		r.UpdatedAt = r.CreatedAt
		Expect(records.Create(ctx, r)).To(Succeed())
		return r
	}
	usernames := func(page *store.Page[OperationRecord]) []string {
		return lo.Map(page.Items, func(r OperationRecord, _ int) string { return r.Username })
	}

	BeforeEach(func() {
		ctx = context.Background()
		Expect(testutil.CleanupMetaColl(OperationRecordMeta)).To(Succeed())
		db = testutil.NewDB(GinkgoT())
		records = store.Of(db, OperationRecordMeta)
		base = time.Now().Truncate(time.Millisecond)
	})

	Describe("ListRecords", func() {
		BeforeEach(func() {
			mustCreate(1*time.Minute, "alice", WithWorkspaceID("ws-1"), WithAppID("app-1"), WithEnvName("dev"))
			mustCreate(2*time.Minute, "alice", WithWorkspaceID("ws-1"), WithAppID("app-2"), WithEnvName("prod"),
				WithResult(ResultFailed))
			mustCreate(3*time.Minute, "bob", WithWorkspaceID("ws-1"), WithAppID("app-1"), WithEnvName("dev"))
			mustCreate(4*time.Minute, "carol", WithWorkspaceID("ws-2"))
		})

		list := func(opts ListOptions) *store.Page[OperationRecord] {
			if opts.Page == 0 {
				opts.Page, opts.PageSize = 1, 10
			}
			page, err := ListRecords(ctx, db, opts)
			Expect(err).NotTo(HaveOccurred())
			return page
		}

		It("should list newest first and count all matched records", func() {
			page := list(ListOptions{WorkspaceID: "ws-1"})
			Expect(page.Total).To(Equal(int64(3)))
			Expect(usernames(page)).To(Equal([]string{"bob", "alice", "alice"}))
		})

		It("should paginate in a stable order", func() {
			first := list(ListOptions{Page: 1, PageSize: 3})
			second := list(ListOptions{Page: 2, PageSize: 3})
			Expect(first.Total).To(Equal(int64(4)))
			Expect(usernames(first)).To(Equal([]string{"carol", "bob", "alice"}))
			Expect(usernames(second)).To(Equal([]string{"alice"}))
		})

		It("should match username case-insensitively by substring", func() {
			page := list(ListOptions{Username: strings.ToUpper(" LIC ")})
			Expect(usernames(page)).To(Equal([]string{"alice", "alice"}))
		})

		It("should filter by group fields", func() {
			page := list(ListOptions{WorkspaceID: "ws-1", AppID: "app-1", EnvName: "dev"})
			Expect(usernames(page)).To(Equal([]string{"bob", "alice"}))
		})

		It("should filter by operation result", func() {
			Expect(usernames(list(ListOptions{Result: ResultFailed}))).To(Equal([]string{"alice"}))
			Expect(list(ListOptions{OpType: OperationTypeDelete}).Total).To(BeZero())
			Expect(list(ListOptions{ResType: ResourceTypeApp}).Total).To(Equal(int64(4)))
		})

		It("should filter by an inclusive time range", func() {
			page := list(ListOptions{StartedAt: base.Add(2 * time.Minute), EndedAt: base.Add(3 * time.Minute)})
			Expect(usernames(page)).To(Equal([]string{"bob", "alice"}))
		})

		It("should ignore a time range that is not increasing", func() {
			page := list(ListOptions{StartedAt: base.Add(3 * time.Minute), EndedAt: base.Add(2 * time.Minute)})
			Expect(page.Total).To(Equal(int64(4)))
		})
	})

	Describe("latest operation times", func() {
		BeforeEach(func() {
			mustCreate(1*time.Minute, "alice", WithWorkspaceID("ws-1"), WithAppID("app-1"))
			mustCreate(3*time.Minute, "alice", WithWorkspaceID("ws-1"), WithAppID("app-2"))
			mustCreate(2*time.Minute, "alice", WithWorkspaceID("ws-2"), WithAppID("app-1"))
			mustCreate(5*time.Minute, "bob", WithWorkspaceID("ws-1"), WithAppID("app-1"))
		})

		It("should return the latest time of the user in each workspace", func() {
			m, err := GetLatestOperationTimesByWorkspacesForUser(ctx, db, []string{"ws-1", "ws-2", "ws-3"}, "alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(HaveLen(2))
			Expect(m["ws-1"]).To(BeTemporally("==", base.Add(3*time.Minute)))
			Expect(m["ws-2"]).To(BeTemporally("==", base.Add(2*time.Minute)))
		})

		It("should return the latest time of the user for each app in the workspace", func() {
			m, err := GetLatestOperationTimesByAppsForUser(ctx, db, "ws-1", []string{"app-1", "app-2"}, "alice")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(HaveLen(2))
			Expect(m["app-1"]).To(BeTemporally("==", base.Add(1*time.Minute)))
			Expect(m["app-2"]).To(BeTemporally("==", base.Add(3*time.Minute)))
		})
	})
})
