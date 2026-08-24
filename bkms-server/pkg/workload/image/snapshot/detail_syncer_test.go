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
	"fmt"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
)

var _ = Describe("DetailSyncer", func() {
	var (
		ctx    context.Context
		store  SnapshotStore
		syncer *DetailSyncer
		info   *RepoKeyInfo
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		store, err = NewSnapshotStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		syncer = NewDetailSyncer(store)
		info = &RepoKeyInfo{
			RepoKey:  "repo-key-%s" + stringx.Random(24),
			RepoName: "fixture/sample",
			Username: "user",
			Password: "pass",
		}
		mockey.Mock(registry.New).To(func(username, password string, insecure bool) *registry.Client {
			Expect(username).To(Equal("user"))
			Expect(password).To(Equal("pass"))
			Expect(insecure).To(BeTrue())
			return &registry.Client{}
		}).Build()
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		Expect(store.DeleteAll(ctx)).To(Succeed())
	})

	Describe("SyncDetails", func() {
		It("should sync details and converge status on success", func() {
			err := store.UpsertSnapshots(ctx, info.RepoKey, []Image{
				{Tag: TagLatest},
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.Mock((*registry.Client).GetTagDetail).
				To(func(_ *registry.Client, _ context.Context, repoName, tag string) (registry.ImageDetail, error) {
					Expect(repoName).To(Equal(info.RepoName))
					builtAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
					return registry.ImageDetail{
						Tag:     tag,
						Digest:  fmt.Sprintf("sha256:%s", tag),
						Size:    int64(len(tag)),
						BuiltAt: builtAt,
					}, nil
				}).
				Build()

			err = syncer.SyncDetails(ctx, info)
			Expect(err).NotTo(HaveOccurred())

			snapshots, total, listErr := store.ListByRepoKey(ctx, info.RepoKey, "", 1, 10)
			Expect(listErr).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(3))
			Expect(snapshots).To(HaveLen(3))

			snapshotsByTag := make(map[string]Image, len(snapshots))
			for _, snapshot := range snapshots {
				snapshotsByTag[snapshot.Tag] = snapshot
			}
			Expect(snapshotsByTag).To(HaveKey(TagLatest))
			Expect(snapshotsByTag).To(HaveKey("v1.0.0"))
			Expect(snapshotsByTag).To(HaveKey("v1.1.0"))
			for tag, snapshot := range snapshotsByTag {
				Expect(snapshot.Digest).To(Equal(fmt.Sprintf("sha256:%s", tag)))
				Expect(snapshot.BuiltAt).NotTo(BeNil())
				Expect(snapshot.Size).To(Equal(int64(len(tag))))
			}

			status, getErr := store.GetStatus(ctx, info.RepoKey)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(status.RefreshStatus).To(Equal(RefreshStatusIdle))
			Expect(status.LastDetailSyncedAt).NotTo(BeNil())
			Expect(status.LastError).To(BeEmpty())
		})

		It("should keep syncing remaining tags when one detail request fails", func() {
			err := store.UpsertSnapshots(ctx, info.RepoKey, []Image{
				{Tag: "v1.0.0"},
				{Tag: "v1.1.0"},
				{Tag: "v2.0.0"},
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.Mock((*registry.Client).GetTagDetail).
				To(func(_ *registry.Client, _ context.Context, _, tag string) (registry.ImageDetail, error) {
					if tag == "v1.1.0" {
						return registry.ImageDetail{}, errors.Errorf("detail lookup failed")
					}
					return registry.ImageDetail{
						Tag:     tag,
						Digest:  fmt.Sprintf("sha256:%s", tag),
						Size:    128,
						BuiltAt: time.Now(),
					}, nil
				}).
				Build()

			err = syncer.SyncDetails(ctx, info)
			Expect(err).NotTo(HaveOccurred())

			snapshots, total, listErr := store.ListByRepoKey(ctx, info.RepoKey, "", 1, 10)
			Expect(listErr).NotTo(HaveOccurred())
			Expect(total).To(BeEquivalentTo(3))
			Expect(snapshots).To(HaveLen(3))

			snapshotsByTag := make(map[string]Image, len(snapshots))
			for _, snapshot := range snapshots {
				snapshotsByTag[snapshot.Tag] = snapshot
			}
			Expect(snapshotsByTag["v1.0.0"].Digest).To(Equal("sha256:v1.0.0"))
			Expect(snapshotsByTag["v1.0.0"].BuiltAt).NotTo(BeNil())
			Expect(snapshotsByTag["v2.0.0"].Digest).To(Equal("sha256:v2.0.0"))
			Expect(snapshotsByTag["v2.0.0"].BuiltAt).NotTo(BeNil())
			Expect(snapshotsByTag["v1.1.0"].Digest).To(BeEmpty())
			Expect(snapshotsByTag["v1.1.0"].BuiltAt).To(BeNil())

			status, getErr := store.GetStatus(ctx, info.RepoKey)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(status.RefreshStatus).To(Equal(RefreshStatusIdle))
			Expect(status.LastError).To(ContainSubstring("get detail for v1.1.0"))
		})

		It("should finish cleanly when there is no tag to sync", func() {
			// UpsertSnapshots 不会写入 builtAt，需要额外调用 UpdateDetail 设置详情
			err := store.UpsertSnapshots(ctx, info.RepoKey, []Image{
				{Tag: "v0.9.0"},
			})
			Expect(err).NotTo(HaveOccurred())
			builtAt := time.Now().Add(-time.Hour)
			err = store.UpdateDetail(ctx, info.RepoKey, "v0.9.0", &registry.ImageDetail{
				Tag:     "v0.9.0",
				Digest:  "sha256:v0.9.0",
				Size:    64,
				BuiltAt: builtAt,
			})
			Expect(err).NotTo(HaveOccurred())
			mockey.Mock((*registry.Client).GetTagDetail).
				To(func(*registry.Client, context.Context, string, string) (registry.ImageDetail, error) {
					Fail("GetTagDetail should not be called")
					return registry.ImageDetail{}, nil
				}).
				Build()

			err = syncer.SyncDetails(ctx, info)
			Expect(err).NotTo(HaveOccurred())

			status, getErr := store.GetStatus(ctx, info.RepoKey)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(status.RefreshStatus).To(Equal(RefreshStatusIdle))
			Expect(status.LastDetailSyncedAt).NotTo(BeNil())
			Expect(status.LastError).To(BeEmpty())
		})

		It("should re-fetch pending tags and keep stale details on lookup failure", func() {
			err := store.UpsertSnapshots(ctx, info.RepoKey, []Image{
				{Tag: "core-test-01"},
				{Tag: "core-test-02"},
			})
			Expect(err).NotTo(HaveOccurred())
			staleBuiltAt := time.Date(2026, 4, 13, 14, 32, 31, 0, time.UTC)
			err = store.UpdateDetail(ctx, info.RepoKey, "core-test-01", &registry.ImageDetail{
				Tag:     "core-test-01",
				Digest:  "sha256:a376477a",
				Size:    317,
				BuiltAt: staleBuiltAt,
			})
			Expect(err).NotTo(HaveOccurred())
			err = store.UpdateDetail(ctx, info.RepoKey, "core-test-02", &registry.ImageDetail{
				Tag:     "core-test-02",
				Digest:  "sha256:old02",
				Size:    128,
				BuiltAt: staleBuiltAt,
			})
			Expect(err).NotTo(HaveOccurred())

			// 两个标签详情均已补全，只有被标记为待刷新才会重新拉取
			err = store.MarkDetailSyncPending(ctx, info.RepoKey, []string{"core-test-01", "core-test-02"})
			Expect(err).NotTo(HaveOccurred())

			freshBuiltAt := time.Date(2026, 4, 23, 11, 52, 14, 0, time.UTC)
			mockey.Mock((*registry.Client).GetTagDetail).
				To(func(_ *registry.Client, _ context.Context, _, tag string) (registry.ImageDetail, error) {
					if tag == "core-test-02" {
						return registry.ImageDetail{}, errors.Errorf("registry unreachable")
					}
					return registry.ImageDetail{
						Tag:     tag,
						Digest:  "sha256:9f3c21be",
						Size:    512,
						BuiltAt: freshBuiltAt,
					}, nil
				}).
				Build()

			err = syncer.SyncDetails(ctx, info)
			Expect(err).NotTo(HaveOccurred())

			snapshots, _, listErr := store.ListByRepoKey(ctx, info.RepoKey, "", 1, 10)
			Expect(listErr).NotTo(HaveOccurred())
			Expect(snapshots).To(HaveLen(2))
			snapshotsByTag := make(map[string]Image, len(snapshots))
			for _, snapshot := range snapshots {
				snapshotsByTag[snapshot.Tag] = snapshot
			}
			Expect(snapshotsByTag["core-test-01"].Digest).To(Equal("sha256:9f3c21be"))
			Expect(snapshotsByTag["core-test-01"].Size).To(Equal(int64(512)))
			Expect(snapshotsByTag["core-test-01"].BuiltAt.UTC()).To(Equal(freshBuiltAt))
			Expect(snapshotsByTag["core-test-02"].Digest).To(Equal("sha256:old02"))
			Expect(snapshotsByTag["core-test-02"].Size).To(Equal(int64(128)))
			Expect(snapshotsByTag["core-test-02"].BuiltAt.UTC()).To(Equal(staleBuiltAt))
			// 成功的标签清除待刷新标记；失败的保留，等待后续刷新重试
			Expect(snapshotsByTag["core-test-01"].DetailSyncPending).To(BeFalse())
			Expect(snapshotsByTag["core-test-02"].DetailSyncPending).To(BeTrue())

			status, getErr := store.GetStatus(ctx, info.RepoKey)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(status.RefreshStatus).To(Equal(RefreshStatusIdle))
			Expect(status.LastError).To(ContainSubstring("get detail for core-test-02"))
		})

		It("should skip sync when status is refreshing", func() {
			// 先将状态设为 refreshing
			err := store.UpsertStatus(ctx, &RepoSnapshotStatus{
				RepoKey:       info.RepoKey,
				RefreshStatus: RefreshStatusRefreshing,
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.Mock((*registry.Client).GetTagDetail).
				To(func(*registry.Client, context.Context, string, string) (registry.ImageDetail, error) {
					Fail("GetTagDetail should not be called")
					return registry.ImageDetail{}, nil
				}).
				Build()

			err = syncer.SyncDetails(ctx, info)
			Expect(err).NotTo(HaveOccurred())

			// 状态不应被修改，仍为 refreshing
			status, getErr := store.GetStatus(ctx, info.RepoKey)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(status).NotTo(BeNil())
			Expect(status.RefreshStatus).To(Equal(RefreshStatusRefreshing))
		})

		It("should skip sync when status is detail_syncing", func() {
			// 先将状态设为 detail_syncing
			err := store.UpsertStatus(ctx, &RepoSnapshotStatus{
				RepoKey:       info.RepoKey,
				RefreshStatus: RefreshStatusDetailSyncing,
			})
			Expect(err).NotTo(HaveOccurred())

			mockey.Mock((*registry.Client).GetTagDetail).
				To(func(*registry.Client, context.Context, string, string) (registry.ImageDetail, error) {
					Fail("GetTagDetail should not be called")
					return registry.ImageDetail{}, nil
				}).
				Build()

			err = syncer.SyncDetails(ctx, info)
			Expect(err).NotTo(HaveOccurred())

			// 状态不应被修改，仍为 detail_syncing
			status, getErr := store.GetStatus(ctx, info.RepoKey)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(status).NotTo(BeNil())
			Expect(status.RefreshStatus).To(Equal(RefreshStatusDetailSyncing))
		})
	})
})
