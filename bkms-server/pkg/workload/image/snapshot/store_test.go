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

package snapshot

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
)

var _ = Describe("SnapshotStoreMongo", func() {
	var ctx context.Context
	var mongoClient *mongo.Client
	var store SnapshotStore
	var dbName string

	BeforeEach(func() {
		var err error
		mongoClient, dbName = database.Client(), database.Name()
		ctx = context.Background()
		store, err = NewSnapshotStoreMongo(mongoClient, dbName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := store.DeleteAll(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("UpsertSnapshots", func() {
		It("should insert new snapshots", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			snapshots := []Image{
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
			}

			err := store.UpsertSnapshots(ctx, repoKey, snapshots)
			Expect(err).NotTo(HaveOccurred())

			results, total, err := store.ListByRepoKey(ctx, repoKey, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(2))
			Expect(results).To(HaveLen(2))
		})

		It("should upsert existing snapshots without duplicating", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			snapshots := []Image{
				{Tag: "v1.0.0"},
			}

			err := store.UpsertSnapshots(ctx, repoKey, snapshots)
			Expect(err).NotTo(HaveOccurred())

			// Upsert again with same tag
			snapshots2 := []Image{
				{Tag: "v1.0.0"},
			}
			err = store.UpsertSnapshots(ctx, repoKey, snapshots2)
			Expect(err).NotTo(HaveOccurred())

			results, total, err := store.ListByRepoKey(ctx, repoKey, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(1))
			Expect(results[0].Tag).To(Equal("v1.0.0"))
		})
	})

	Describe("ListByRepoKey", func() {
		BeforeEach(func() {
			repoKey := "testrepokey1234567890abcdef123456"
			snapshots := []Image{
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
				{Tag: "v2.0.0"},
				{Tag: "latest"},
			}
			err := store.UpsertSnapshots(ctx, repoKey, snapshots)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should support pagination", func() {
			repoKey := "testrepokey1234567890abcdef123456"

			results, total, err := store.ListByRepoKey(ctx, repoKey, "", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(4))
			Expect(results).To(HaveLen(2))

			results2, _, err := store.ListByRepoKey(ctx, repoKey, "", 2, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(results2).To(HaveLen(2))
		})

		It("should support keyword filtering", func() {
			repoKey := "testrepokey1234567890abcdef123456"

			results, total, err := store.ListByRepoKey(ctx, repoKey, "v1", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(2))
			Expect(results).To(HaveLen(2))
		})

		It("should return empty for non-existent repoKey", func() {
			results, total, err := store.ListByRepoKey(ctx, "nonexistent", "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(0))
			Expect(results).To(BeNil())
		})
	})

	Describe("DeleteByRepoKeyExcludeTags", func() {
		It("should delete tags not in the active list", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			snapshots := []Image{
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
				{Tag: "v2.0.0"},
			}
			err := store.UpsertSnapshots(ctx, repoKey, snapshots)
			Expect(err).NotTo(HaveOccurred())

			// Only keep v1.0.0 and v2.0.0
			deleted, err := store.DeleteByRepoKeyExcludeTags(ctx, repoKey, []string{"v1.0.0", "v2.0.0"})
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted).To(BeEquivalentTo(1))

			results, total, err := store.ListByRepoKey(ctx, repoKey, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(2))
			tags := make([]string, 0, len(results))
			for _, r := range results {
				tags = append(tags, r.Tag)
			}
			Expect(tags).To(ContainElements("v1.0.0", "v2.0.0"))
		})
	})

	Describe("DeleteByRepoKeyAndTag", func() {
		It("should delete the specified tag only", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			snapshots := []Image{
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
				{Tag: "v2.0.0"},
			}
			err := store.UpsertSnapshots(ctx, repoKey, snapshots)
			Expect(err).NotTo(HaveOccurred())

			deleted, err := store.DeleteByRepoKeyAndTag(ctx, repoKey, "v1.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted).To(BeEquivalentTo(1))

			results, total, err := store.ListByRepoKey(ctx, repoKey, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(2))
			tags := make([]string, 0, len(results))
			for _, r := range results {
				tags = append(tags, r.Tag)
			}
			Expect(tags).To(ContainElements("v1.0.0", "v2.0.0"))
			Expect(tags).NotTo(ContainElement("v1.1.0"))
		})

		It("should be a no-op when the tag does not exist", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			err := store.UpsertSnapshots(ctx, repoKey, []Image{{Tag: "v1.0.0"}})
			Expect(err).NotTo(HaveOccurred())

			deleted, err := store.DeleteByRepoKeyAndTag(ctx, repoKey, "not-found")
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted).To(BeZero())

			results, total, err := store.ListByRepoKey(ctx, repoKey, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(1))
			Expect(results).To(HaveLen(1))
			Expect(results[0].Tag).To(Equal("v1.0.0"))
		})
	})

	Describe("HasTag", func() {
		It("should return true when the tag exists", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			err := store.UpsertSnapshots(ctx, repoKey, []Image{{Tag: "v1.0.0"}})
			Expect(err).NotTo(HaveOccurred())

			exists, err := store.HasTag(ctx, repoKey, "v1.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should return false when the tag does not exist", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			err := store.UpsertSnapshots(ctx, repoKey, []Image{{Tag: "v1.0.0"}})
			Expect(err).NotTo(HaveOccurred())

			exists, err := store.HasTag(ctx, repoKey, "v2.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("GetStatus and TrySetRefreshing", func() {
		It("should return nil for non-existent status", func() {
			status, err := store.GetStatus(ctx, "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(BeNil())
		})

		It("should set refreshing status atomically", func() {
			repoKey := "testrepokey1234567890abcdef123456"

			// First attempt should succeed
			acquired, err := store.TrySetRefreshing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// Second attempt should fail (already refreshing)
			acquired2, err := store.TrySetRefreshing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired2).To(BeFalse())

			// Verify status
			status, err := store.GetStatus(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).NotTo(BeNil())
			Expect(status.RefreshStatus).To(Equal(RefreshStatusRefreshing))
		})

		It("should allow re-acquiring after reset to idle", func() {
			repoKey := "testrepokey1234567890abcdef123456"

			acquired, err := store.TrySetRefreshing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// Reset to idle
			err = store.UpsertStatus(ctx, &RepoSnapshotStatus{
				RepoKey:       repoKey,
				RefreshStatus: RefreshStatusIdle,
			})
			Expect(err).NotTo(HaveOccurred())

			// Should be able to acquire again
			acquired2, err := store.TrySetRefreshing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired2).To(BeTrue())
		})
	})

	Describe("TrySetDetailSyncing", func() {
		It("should set detail_syncing status atomically when idle", func() {
			repoKey := "testrepokey1234567890abcdef123456"

			// 首次调用应成功
			acquired, err := store.TrySetDetailSyncing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// 验证状态为 detail_syncing
			status, err := store.GetStatus(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).NotTo(BeNil())
			Expect(status.RefreshStatus).To(Equal(RefreshStatusDetailSyncing))
		})

		It("should reject when already detail_syncing", func() {
			repoKey := "testrepokey1234567890abcdef123456"

			acquired, err := store.TrySetDetailSyncing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// 第二次调用应返回 false
			acquired2, err := store.TrySetDetailSyncing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired2).To(BeFalse())
		})

		It("should reject when refreshing (mutual exclusion)", func() {
			repoKey := "testrepokey1234567890abcdef123456"

			// 先设为 refreshing
			acquired, err := store.TrySetRefreshing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// 尝试 detail sync 应被拒绝（与 refresh 互斥）
			acquired2, err := store.TrySetDetailSyncing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired2).To(BeFalse())
		})

		It("should allow re-acquiring after reset to idle", func() {
			repoKey := "testrepokey1234567890abcdef123456"

			acquired, err := store.TrySetDetailSyncing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired).To(BeTrue())

			// 重置为 idle
			err = store.UpsertStatus(ctx, &RepoSnapshotStatus{
				RepoKey:       repoKey,
				RefreshStatus: RefreshStatusIdle,
			})
			Expect(err).NotTo(HaveOccurred())

			// 再次调用应成功
			acquired2, err := store.TrySetDetailSyncing(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(acquired2).To(BeTrue())
		})
	})

	Describe("ListAllTags", func() {
		It("should return all tags for a repo", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			snapshots := []Image{
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
				{Tag: "v2.0.0"},
			}
			err := store.UpsertSnapshots(ctx, repoKey, snapshots)
			Expect(err).NotTo(HaveOccurred())

			tags, err := store.ListAllTags(ctx, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(HaveLen(3))
			Expect(tags).To(ContainElements("v1.0.0", "v1.1.0", "v2.0.0"))
		})

		It("should return empty for non-existent repoKey", func() {
			tags, err := store.ListAllTags(ctx, "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(BeEmpty())
		})
	})

	Describe("UpdateDetail", func() {
		It("should update snapshot detail fields", func() {
			repoKey := "testrepokey1234567890abcdef123456"
			snapshots := []Image{
				{Tag: "v1.0.0"},
			}
			err := store.UpsertSnapshots(ctx, repoKey, snapshots)
			Expect(err).NotTo(HaveOccurred())

			builtAt := time.Now().Add(-time.Hour)
			err = store.UpdateDetail(ctx, repoKey, "v1.0.0", &registry.ImageDetail{
				Digest:  "sha256:abc123def456",
				Size:    52428800,
				BuiltAt: builtAt,
			})
			Expect(err).NotTo(HaveOccurred())

			results, _, err := store.ListByRepoKey(ctx, repoKey, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Digest).To(Equal("sha256:abc123def456"))
			Expect(results[0].Size).To(Equal(int64(52428800)))
			Expect(results[0].BuiltAt).NotTo(BeNil())
		})
	})

	Describe("ListByRepoKeyAndTags", func() {
		const repoKey = "testrepokey1234567890abcdef123456"

		BeforeEach(func() {
			snapshots := []Image{
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
				{Tag: "v2.0.0"},
				{Tag: "v3.0.0"},
				{Tag: "latest"},
			}
			err := store.UpsertSnapshots(ctx, repoKey, snapshots)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return only snapshots whose tags are in the given list", func() {
			tags := []string{"v1.0.0", "v2.0.0"}
			results, total, err := store.ListByRepoKeyAndTags(ctx, repoKey, tags, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(2))
			Expect(results).To(HaveLen(2))
			resultTags := make([]string, 0, len(results))
			for _, r := range results {
				resultTags = append(resultTags, r.Tag)
			}
			Expect(resultTags).To(ContainElements("v1.0.0", "v2.0.0"))
		})

		It("should support pagination within the given tags", func() {
			tags := []string{"v1.0.0", "v1.1.0", "v2.0.0", "v3.0.0"}
			results, total, err := store.ListByRepoKeyAndTags(ctx, repoKey, tags, "", 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(4))
			Expect(results).To(HaveLen(2))

			results2, _, err := store.ListByRepoKeyAndTags(ctx, repoKey, tags, "", 2, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(results2).To(HaveLen(2))
		})

		It("should support keyword filtering within the given tags", func() {
			tags := []string{"v1.0.0", "v1.1.0", "v2.0.0", "latest"}
			results, total, err := store.ListByRepoKeyAndTags(ctx, repoKey, tags, "v1", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(2))
			Expect(results).To(HaveLen(2))
			resultTags := make([]string, 0, len(results))
			for _, r := range results {
				resultTags = append(resultTags, r.Tag)
			}
			Expect(resultTags).To(ContainElements("v1.0.0", "v1.1.0"))
		})

		It("should return empty when tags list is empty", func() {
			results, total, err := store.ListByRepoKeyAndTags(ctx, repoKey, []string{}, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(0))
			Expect(results).To(BeNil())
		})

		It("should return empty for non-existent repoKey", func() {
			tags := []string{"v1.0.0", "v2.0.0"}
			results, total, err := store.ListByRepoKeyAndTags(ctx, "nonexistent", tags, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(0))
			Expect(results).To(BeNil())
		})
	})
})
