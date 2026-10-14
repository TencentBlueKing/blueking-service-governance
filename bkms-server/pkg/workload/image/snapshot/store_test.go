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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// cleanupSnapshotCollections 清空镜像快照与仓库快照状态集合
func cleanupSnapshotCollections() {
	Expect(testutil.CleanupMetaColl(ImageMeta)).To(Succeed())
	Expect(testutil.CleanupMetaColl(RepoStatusMeta)).To(Succeed())
}

// listSnapshots 查询仓库第一页（最多 10 条）快照
func listSnapshots(ctx context.Context, db *mongo.Database, repoKey string) ([]Image, int64, error) {
	result, err := ListImagePage(ctx, db, repoKey, "", 1, 10)
	if err != nil {
		return nil, 0, err
	}
	return result.Items, result.Total, nil
}

var _ = Describe("Snapshot store", func() {
	const repoKey = "testrepokey1234567890abcdef123456"

	var ctx context.Context
	var db *mongo.Database

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		DeferCleanup(cleanupSnapshotCollections)
	})

	listPage := func(repo, keyword string, page, size int, conds ...store.Opt) *store.Page[Image] {
		result, err := ListImagePage(ctx, db, repo, keyword, page, size, conds...)
		Expect(err).NotTo(HaveOccurred())
		return result
	}
	tagsOf := func(images []Image) []string {
		return lo.Map(images, func(img Image, _ int) string { return img.Tag })
	}
	getImage := func(tag string) *Image {
		img, err := store.Of(db, ImageMeta).Get(ctx, ByRepoTag(repoKey, tag))
		Expect(err).NotTo(HaveOccurred())
		return img
	}

	Describe("UpsertTags", func() {
		It("inserts new snapshots", func() {
			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v1.1.0"})).To(Succeed())

			result := listPage(repoKey, "", 1, 10)
			Expect(result.Total).To(BeEquivalentTo(2))
			Expect(tagsOf(result.Items)).To(ConsistOf("v1.0.0", "v1.1.0"))
		})

		It("does not duplicate or reset details of existing snapshots", func() {
			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0"})).To(Succeed())
			Expect(UpdateDetail(ctx, db, repoKey, "v1.0.0", &registry.ImageDetail{Digest: "sha256:abc"})).To(Succeed())

			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v1.0.0"})).To(Succeed())

			result := listPage(repoKey, "", 1, 10)
			Expect(result.Total).To(BeEquivalentTo(1))
			Expect(result.Items[0].Digest).To(Equal("sha256:abc"))
		})

		It("ignores empty tags", func() {
			Expect(UpsertTags(ctx, db, repoKey, nil)).To(Succeed())
			Expect(listPage(repoKey, "", 1, 10).Total).To(BeZero())
		})
	})

	Describe("ListImagePage", func() {
		BeforeEach(func() {
			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v1.1.0", "v2.0.0", "v3.0.0", "latest"})).
				To(Succeed())
		})

		It("supports pagination", func() {
			first := listPage(repoKey, "", 1, 2)
			Expect(first.Total).To(BeEquivalentTo(5))
			Expect(first.Items).To(HaveLen(2))
			Expect(first.HasNext).To(BeTrue())

			second := listPage(repoKey, "", 2, 2)
			Expect(second.Items).To(HaveLen(2))
			Expect(tagsOf(second.Items)).NotTo(ContainElements(tagsOf(first.Items)))
		})

		It("sorts by builtAt desc with unbuilt tags last, then by tag desc", func() {
			now := time.Now()
			Expect(UpdateDetail(ctx, db, repoKey, "v1.0.0", &registry.ImageDetail{BuiltAt: now})).To(Succeed())
			Expect(UpdateDetail(ctx, db, repoKey, "v1.1.0", &registry.ImageDetail{BuiltAt: now.Add(-time.Hour)})).
				To(Succeed())

			Expect(tagsOf(listPage(repoKey, "", 1, 10).Items)).
				To(Equal([]string{"v1.0.0", "v1.1.0", "v3.0.0", "v2.0.0", "latest"}))
		})

		It("filters by keyword case-insensitively", func() {
			result := listPage(repoKey, "V1", 1, 10)
			Expect(result.Total).To(BeEquivalentTo(2))
			Expect(tagsOf(result.Items)).To(ConsistOf("v1.0.0", "v1.1.0"))
		})

		It("restricts results to the given tags", func() {
			result := listPage(repoKey, "", 1, 10, TagIn([]string{"v1.0.0", "v2.0.0"}))
			Expect(result.Total).To(BeEquivalentTo(2))
			Expect(tagsOf(result.Items)).To(ConsistOf("v1.0.0", "v2.0.0"))
		})

		It("combines the given tags with keyword filtering", func() {
			result := listPage(repoKey, "v1", 1, 10, TagIn([]string{"v1.0.0", "v1.1.0", "v2.0.0", "latest"}))
			Expect(tagsOf(result.Items)).To(ConsistOf("v1.0.0", "v1.1.0"))
		})

		It("returns empty when the given tags are empty", func() {
			result := listPage(repoKey, "", 1, 10, TagIn(nil))
			Expect(result.Total).To(BeZero())
			Expect(result.Items).To(BeEmpty())
		})
	})

	Describe("DeleteTagsNotIn", func() {
		It("deletes tags not in the active list", func() {
			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v1.1.0", "v2.0.0"})).To(Succeed())

			deleted, err := DeleteTagsNotIn(ctx, db, repoKey, []string{"v1.0.0", "v2.0.0"})
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted).To(BeEquivalentTo(1))
			Expect(tagsOf(listPage(repoKey, "", 1, 10).Items)).To(ConsistOf("v1.0.0", "v2.0.0"))
		})

		It("deletes all tags when the active list is empty", func() {
			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0"})).To(Succeed())

			deleted, err := DeleteTagsNotIn(ctx, db, repoKey, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted).To(BeEquivalentTo(1))
		})
	})

	Describe("ByRepoTag", func() {
		BeforeEach(func() {
			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v1.1.0"})).To(Succeed())
		})

		It("checks whether a tag exists", func() {
			images := store.Of(db, ImageMeta)
			Expect(images.Exists(ctx, ByRepoTag(repoKey, "v1.0.0"))).To(BeTrue())
			Expect(images.Exists(ctx, ByRepoTag(repoKey, "v2.0.0"))).To(BeFalse())
		})

		It("deletes the specified tag only, and is a no-op for a missing tag", func() {
			images := store.Of(db, ImageMeta)
			Expect(images.DeleteMany(ctx, ByRepoTag(repoKey, "v1.1.0"))).To(BeEquivalentTo(1))
			Expect(images.DeleteMany(ctx, ByRepoTag(repoKey, "not-found"))).To(BeZero())
			Expect(tagsOf(listPage(repoKey, "", 1, 10).Items)).To(Equal([]string{"v1.0.0"}))
		})
	})

	Describe("details sync", func() {
		BeforeEach(func() {
			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v1.1.0", "latest"})).To(Succeed())
		})

		It("updates snapshot detail fields", func() {
			builtAt := time.Now().Add(-time.Hour)
			Expect(UpdateDetail(ctx, db, repoKey, "v1.0.0", &registry.ImageDetail{
				Digest:  "sha256:abc123def456",
				Size:    52428800,
				BuiltAt: builtAt,
			})).To(Succeed())

			img := getImage("v1.0.0")
			Expect(img.Digest).To(Equal("sha256:abc123def456"))
			Expect(img.Size).To(Equal(int64(52428800)))
			Expect(img.BuiltAt).NotTo(BeNil())
			Expect(*img.BuiltAt).To(BeTemporally("~", builtAt, time.Millisecond))
		})

		It("ignores updating the detail of a missing tag", func() {
			Expect(UpdateDetail(ctx, db, repoKey, "not-found", &registry.ImageDetail{Digest: "x"})).To(Succeed())
		})

		It("lists tags without details, marked as pending, or latest", func() {
			Expect(ListUnsyncedDetailTags(ctx, db, repoKey)).To(ConsistOf("v1.0.0", "v1.1.0", "latest"))

			for _, tag := range []string{"v1.0.0", "v1.1.0", "latest"} {
				Expect(UpdateDetail(ctx, db, repoKey, tag, &registry.ImageDetail{BuiltAt: time.Now()})).To(Succeed())
			}
			Expect(ListUnsyncedDetailTags(ctx, db, repoKey)).To(Equal([]string{"latest"}))

			Expect(MarkDetailSyncPending(ctx, db, repoKey, []string{"v1.0.0", "not-found"})).To(Succeed())
			Expect(ListUnsyncedDetailTags(ctx, db, repoKey)).To(ConsistOf("v1.0.0", "latest"))

			// 详情同步成功后清除重新拉取标记
			Expect(UpdateDetail(ctx, db, repoKey, "v1.0.0", &registry.ImageDetail{BuiltAt: time.Now()})).To(Succeed())
			Expect(getImage("v1.0.0").DetailSyncPending).To(BeFalse())
			Expect(ListUnsyncedDetailTags(ctx, db, repoKey)).To(Equal([]string{"latest"}))
		})

		It("does not touch updatedAt when marking tags as pending", func() {
			before := getImage("v1.0.0").UpdatedAt
			Expect(MarkDetailSyncPending(ctx, db, repoKey, []string{"v1.0.0"})).To(Succeed())

			img := getImage("v1.0.0")
			Expect(img.DetailSyncPending).To(BeTrue())
			Expect(img.UpdatedAt).To(BeTemporally("==", before))
		})
	})

	Describe("ListAllTags", func() {
		It("returns all tags for a repo", func() {
			Expect(UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v1.1.0", "v2.0.0"})).To(Succeed())
			Expect(ListAllTags(ctx, db, repoKey)).To(ConsistOf("v1.0.0", "v1.1.0", "v2.0.0"))
		})

		It("returns empty for non-existent repoKey", func() {
			Expect(ListAllTags(ctx, db, "nonexistent")).To(BeEmpty())
		})
	})

	Describe("repo status", func() {
		getStatus := func() *RepoSnapshotStatus {
			status, err := GetRepoStatus(ctx, db, repoKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).NotTo(BeNil())
			return status
		}
		resetToIdle := func() {
			Expect(UpsertRepoStatus(ctx, db, &RepoSnapshotStatus{RepoKey: repoKey, RefreshStatus: RefreshStatusIdle})).
				To(Succeed())
		}

		It("returns nil for non-existent status", func() {
			status, err := GetRepoStatus(ctx, db, "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(BeNil())
		})

		It("keeps repoName and timestamps that are not provided", func() {
			refreshedAt := time.Now().Add(-time.Minute)
			Expect(UpsertRepoStatus(ctx, db, &RepoSnapshotStatus{
				RepoKey:         repoKey,
				RepoName:        "mirrors/app",
				RefreshStatus:   RefreshStatusIdle,
				LastRefreshedAt: &refreshedAt,
				LastError:       "boom",
			})).To(Succeed())

			resetToIdle()

			status := getStatus()
			Expect(status.RepoName).To(Equal("mirrors/app"))
			Expect(status.LastRefreshedAt).NotTo(BeNil())
			Expect(*status.LastRefreshedAt).To(BeTemporally("~", refreshedAt, time.Millisecond))
			Expect(status.LastError).To(BeEmpty())
		})

		It("sets refreshing status atomically", func() {
			Expect(TrySetRefreshing(ctx, db, repoKey)).To(BeTrue())
			Expect(TrySetRefreshing(ctx, db, repoKey)).To(BeFalse())
			Expect(getStatus().RefreshStatus).To(Equal(RefreshStatusRefreshing))
		})

		It("allows re-acquiring refreshing after reset to idle", func() {
			Expect(TrySetRefreshing(ctx, db, repoKey)).To(BeTrue())
			resetToIdle()
			Expect(TrySetRefreshing(ctx, db, repoKey)).To(BeTrue())
		})

		It("sets detail_syncing status atomically when idle", func() {
			Expect(TrySetDetailSyncing(ctx, db, repoKey)).To(BeTrue())
			Expect(getStatus().RefreshStatus).To(Equal(RefreshStatusDetailSyncing))
			Expect(TrySetDetailSyncing(ctx, db, repoKey)).To(BeFalse())
		})

		It("rejects detail syncing while refreshing (mutual exclusion)", func() {
			Expect(TrySetRefreshing(ctx, db, repoKey)).To(BeTrue())
			Expect(TrySetDetailSyncing(ctx, db, repoKey)).To(BeFalse())
		})

		It("allows re-acquiring detail syncing after reset to idle", func() {
			Expect(TrySetDetailSyncing(ctx, db, repoKey)).To(BeTrue())
			resetToIdle()
			Expect(TrySetDetailSyncing(ctx, db, repoKey)).To(BeTrue())
		})

		It("clears the last error when acquiring", func() {
			Expect(UpsertRepoStatus(ctx, db, &RepoSnapshotStatus{
				RepoKey: repoKey, RefreshStatus: RefreshStatusIdle, LastError: "boom",
			})).To(Succeed())
			Expect(TrySetRefreshing(ctx, db, repoKey)).To(BeTrue())
			Expect(getStatus().LastError).To(BeEmpty())
		})
	})
})
