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
	"errors"
	"fmt"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("Service", func() {
	var (
		ctx              context.Context
		store            SnapshotStore
		appStore         *bkmsapp.ApplicationStoreMongo
		cfgStore         *build.ConfigStoreMongo
		service          *Service
		oldConfig        *config.Config
		testAppID1       string
		testAppID2       string
		testWorkspaceID1 string
		testWorkspaceID2 string
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		store, err = NewSnapshotStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		appStore, err = bkmsapp.NewApplicationStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		cfgStore, err = build.NewConfigStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		service = NewService(store, cfgStore, appStore)

		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())

		oldConfig = config.G
		config.G = &config.Config{
			Encrypt: config.EncryptConfig{Secret: secret},
		}

		// 生成测试应用 & 工作空间 ID
		testAppID1 = "app-1-" + stringx.Random(12)
		testAppID2 = "app-2-" + stringx.Random(12)
		testWorkspaceID1 = "ws-1-" + stringx.Random(12)
		testWorkspaceID2 = "ws-2-" + stringx.Random(12)

		// 插入测试应用数据
		Expect(appStore.CreateApp(ctx, &bkmsapp.Application{
			ID:          testAppID1,
			Name:        testAppID2,
			WorkspaceID: testWorkspaceID1,
		})).To(Succeed())
		Expect(appStore.CreateApp(ctx, &bkmsapp.Application{
			ID:          testAppID2,
			Name:        testAppID2,
			WorkspaceID: testWorkspaceID2,
		})).To(Succeed())

		// 插入 app-1 的构建配置（外部镜像仓库类型）
		Expect(cfgStore.Create(ctx, &build.Config{
			AppID: testAppID1,
			Image: &build.ImageConfig{
				Name:     "library/busybox",
				Username: "alice",
				Password: "secret",
			},
		})).To(Succeed())

		// 插入 app-2 的构建配置（代码仓库类型）
		Expect(cfgStore.Create(ctx, &build.Config{
			AppID:      testAppID2,
			SourceType: build.SourceTypeCodeRepository,
		})).To(Succeed())
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		Expect(store.DeleteAll(ctx)).To(Succeed())
		config.G = oldConfig
	})

	Describe("ResolveRepoKeyForApp", func() {
		It("should resolve external image registry config directly", func() {
			info, err := service.ResolveRepoKeyForApp(ctx, testAppID1)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.RepoName).To(Equal("library/busybox"))
			Expect(info.Username).To(Equal("alice"))
			Expect(info.Password).To(Equal("secret"))
			Expect(info.RepoKey).To(Equal(GenerateRepoKey("library/busybox", "alice", "secret")))
		})

		It("should resolve platform registry config for non-image source", func() {
			mockey.PatchConvey("mock workspace image registry", GinkgoT(), func() {
				mockey.Mock(workspace.GetWorkspaceImageRegistry).To(
					func(_ context.Context, wsID string) (*bkmsreg.ImageRegistry, error) {
						Expect(wsID).To(Equal(testWorkspaceID2))
						return &bkmsreg.ImageRegistry{
							Registry: "mirrors.tencent.com/bkpaas",
							Username: "bk-user",
							Password: "bk-pass",
						}, nil
					},
				).Build()

				info, err := service.ResolveRepoKeyForApp(ctx, testAppID2)
				Expect(err).NotTo(HaveOccurred())
				Expect(info.RepoName).To(Equal(
					fmt.Sprintf("mirrors.tencent.com/bkpaas/%s", testAppID2),
				))
				Expect(info.Username).To(Equal("bk-user"))
				Expect(info.Password).To(Equal("bk-pass"))
			})
		})
	})

	Describe("ResolveRepoKeyForRepository", func() {
		It("should resolve repository name without credentials", func() {
			info, err := service.ResolveRepoKeyForRepository(" registry.example.com/team/runtime ")
			Expect(err).NotTo(HaveOccurred())
			Expect(info.RepoName).To(Equal("registry.example.com/team/runtime"))
			Expect(info.Username).To(BeEmpty())
			Expect(info.Password).To(BeEmpty())
			Expect(info.RepoKey).To(Equal(GenerateRepoKey("registry.example.com/team/runtime", "", "")))
		})
	})

	Describe("RefreshSnapshots", func() {
		It("should refresh snapshots and enqueue detail sync", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				repoKey := GenerateRepoKey("library/busybox", "alice", "secret")
				err := store.UpsertSnapshots(ctx, repoKey, []Image{{Tag: "stale"}})
				Expect(err).NotTo(HaveOccurred())

				var gotUsername string
				var gotPassword string
				var preparedArgs *ImageDetailSyncArgs
				mockey.Mock(registry.New).To(func(username, password string, insecure bool) *registry.Client {
					gotUsername = username
					gotPassword = password
					Expect(insecure).To(BeTrue())
					return &registry.Client{}
				}).Build()
				mockey.Mock((*registry.Client).ListAllTags).Return([]string{TagLatest, "v1.0.0"}, nil).Build()
				mockey.Mock((*SnapshotStoreMongo).ListUnsyncedDetailTags).
					Return([]string{TagLatest, "v1.0.0"}, nil).
					Build()
				mockey.Mock(NewImageDetailSyncArgs).
					To(func(repoKey, repoName, username, password string) (*ImageDetailSyncArgs, error) {
						preparedArgs = &ImageDetailSyncArgs{
							RepoKey:           repoKey,
							RepoName:          repoName,
							EncryptedUsername: "encrypted-user",
							EncryptedPassword: "encrypted-pass",
						}
						Expect(username).To(Equal("alice"))
						Expect(password).To(Equal("secret"))
						return preparedArgs, nil
					}).
					Build()
				mockey.Mock(taskq.Enqueue).Return(nil).Build()

				result, err := service.RefreshAppSnapshots(ctx, testAppID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("success"))
				Expect(result.AddedTagCnt).To(BeEquivalentTo(2))
				Expect(result.RemovedTagCnt).To(BeEquivalentTo(1))
				Expect(gotUsername).To(Equal("alice"))
				Expect(gotPassword).To(Equal("secret"))

				snapshots, total, err := store.ListByRepoKey(ctx, repoKey, "", 1, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(BeEquivalentTo(2))
				Expect(snapshots).To(HaveLen(2))
				Expect([]string{snapshots[0].Tag, snapshots[1].Tag}).To(ContainElements(TagLatest, "v1.0.0"))

				status, err := store.GetStatus(ctx, repoKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(status.RepoName).To(Equal("library/busybox"))
				Expect(status.RefreshStatus).To(Equal(RefreshStatusIdle))
				Expect(status.LastRefreshedAt).NotTo(BeNil())
				Expect(status.LastError).To(BeEmpty())

				Expect(preparedArgs).NotTo(BeNil())
				Expect(preparedArgs.RepoKey).To(Equal(repoKey))
				Expect(preparedArgs.RepoName).To(Equal("library/busybox"))
			})
		})

		It("should short-circuit when another refresh is in progress", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				repoKey := GenerateRepoKey("library/busybox", "alice", "secret")
				err := store.UpsertStatus(ctx, &RepoSnapshotStatus{
					RepoKey:       repoKey,
					RefreshStatus: RefreshStatusRefreshing,
				})
				Expect(err).NotTo(HaveOccurred())
				mockey.Mock(registry.New).To(func(string, string, bool) *registry.Client {
					Fail("registry.New should not be called")
					return &registry.Client{}
				}).Build()

				result, err := service.RefreshAppSnapshots(ctx, testAppID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("refreshing"))
			})
		})

		It("should mark status back to idle when refresh fails", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(registry.New).Return(&registry.Client{}).Build()
				mockey.Mock((*registry.Client).ListAllTags).Return(nil, errors.New("registry unavailable")).Build()

				_, err := service.RefreshAppSnapshots(ctx, testAppID1)
				Expect(err).To(HaveOccurred())

				repoKey := GenerateRepoKey("library/busybox", "alice", "secret")
				status, getErr := store.GetStatus(ctx, repoKey)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(status.RefreshStatus).To(Equal(RefreshStatusIdle))
				Expect(status.LastError).To(ContainSubstring("list all tags"))
			})
		})

		It("should not submit detail sync task when no new tags found", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				repoKey := GenerateRepoKey("library/busybox", "alice", "secret")
				err := store.UpsertSnapshots(ctx, repoKey, []Image{{Tag: TagLatest}, {Tag: "v1.0.0"}})
				Expect(err).NotTo(HaveOccurred())

				mockey.Mock(registry.New).Return(&registry.Client{}).Build()
				mockey.Mock((*registry.Client).ListAllTags).Return([]string{TagLatest, "v1.0.0"}, nil).Build()
				mockey.Mock((*SnapshotStoreMongo).ListUnsyncedDetailTags).Return([]string{}, nil).Build()

				result, err := service.RefreshAppSnapshots(ctx, testAppID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("success"))
				Expect(result.AddedTagCnt).To(BeEquivalentTo(0))
				Expect(result.RemovedTagCnt).To(BeEquivalentTo(0))
				Expect(result.Message).To(Equal("Snapshot refresh completed, no tags need detail sync"))
			})
		})

		It("should submit detail sync task for a forced tag that already has details", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				repoKey := GenerateRepoKey("library/busybox", "alice", "secret")
				err := store.UpsertSnapshots(ctx, repoKey, []Image{{Tag: "core-test-01"}})
				Expect(err).NotTo(HaveOccurred())
				// 详情已补全，常规查询不会返回它
				err = store.UpdateDetail(ctx, repoKey, "core-test-01", &registry.ImageDetail{
					Tag:     "core-test-01",
					Digest:  "sha256:a376477a",
					BuiltAt: time.Now().Add(-time.Hour),
				})
				Expect(err).NotTo(HaveOccurred())

				mockey.Mock(registry.New).Return(&registry.Client{}).Build()
				mockey.Mock((*registry.Client).ListAllTags).Return([]string{"core-test-01"}, nil).Build()
				mockey.Mock(taskq.Enqueue).Return(nil).Build()

				result, err := service.RefreshAppSnapshots(ctx, testAppID1, "core-test-01")
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("success"))
				Expect(result.AddedTagCnt).To(BeEquivalentTo(0))
				Expect(result.Message).To(Equal(
					"Snapshot refresh completed, and the detail sync task has started asynchronously",
				))

				// 标记已落库，即使本次任务被跳过，后续刷新仍会重新拉取该标签
				tags, listErr := store.ListUnsyncedDetailTags(ctx, repoKey)
				Expect(listErr).NotTo(HaveOccurred())
				Expect(tags).To(ContainElement("core-test-01"))
			})
		})
	})

	Describe("RefreshRepositorySnapshots", func() {
		It("should refresh repository snapshots without credentials", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				repoName := "registry.example.com/team/runtime"
				repoKey := GenerateRepoKey(repoName, "", "")
				err := store.UpsertSnapshots(ctx, repoKey, []Image{{Tag: "stale"}})
				Expect(err).NotTo(HaveOccurred())

				var gotUsername string
				var gotPassword string
				var preparedArgs *ImageDetailSyncArgs
				mockey.Mock(registry.New).To(func(username, password string, insecure bool) *registry.Client {
					gotUsername = username
					gotPassword = password
					Expect(insecure).To(BeTrue())
					return &registry.Client{}
				}).Build()
				mockey.Mock((*registry.Client).ListAllTags).Return([]string{TagLatest, "v1.0.0"}, nil).Build()
				mockey.Mock((*SnapshotStoreMongo).ListUnsyncedDetailTags).Return([]string{TagLatest}, nil).Build()
				mockey.Mock(NewImageDetailSyncArgs).
					To(func(
						repoKey, repoName, username, password string,
					) (*ImageDetailSyncArgs, error) {
						preparedArgs = &ImageDetailSyncArgs{RepoKey: repoKey, RepoName: repoName}
						Expect(username).To(BeEmpty())
						Expect(password).To(BeEmpty())
						return preparedArgs, nil
					}).
					Build()
				mockey.Mock(taskq.Enqueue).Return(nil).Build()

				result, err := service.RefreshRepositorySnapshots(auth.WithMaintenanceUser(ctx), repoName)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("success"))
				Expect(result.AddedTagCnt).To(BeEquivalentTo(2))
				Expect(result.RemovedTagCnt).To(BeEquivalentTo(1))
				Expect(gotUsername).To(BeEmpty())
				Expect(gotPassword).To(BeEmpty())

				snapshots, total, err := store.ListByRepoKey(ctx, repoKey, "", 1, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(total).To(BeEquivalentTo(2))
				Expect(snapshots).To(HaveLen(2))
				Expect([]string{snapshots[0].Tag, snapshots[1].Tag}).To(ContainElements(TagLatest, "v1.0.0"))

				status, err := store.GetStatus(ctx, repoKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(status.RepoName).To(Equal(repoName))
				Expect(status.RefreshStatus).To(Equal(RefreshStatusIdle))
				Expect(status.LastRefreshedAt).NotTo(BeNil())
				Expect(status.LastError).To(BeEmpty())

				Expect(preparedArgs).NotTo(BeNil())
				Expect(preparedArgs.RepoKey).To(Equal(repoKey))
				Expect(preparedArgs.RepoName).To(Equal(repoName))
			})
		})

		It("should keep task enqueue error context", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				repoName := "registry.example.com/team/runtime-failed"
				mockey.Mock(registry.New).Return(&registry.Client{}).Build()
				mockey.Mock((*registry.Client).ListAllTags).Return([]string{TagLatest}, nil).Build()
				mockey.Mock((*SnapshotStoreMongo).ListUnsyncedDetailTags).Return([]string{TagLatest}, nil).Build()
				mockey.Mock(NewImageDetailSyncArgs).
					Return(&ImageDetailSyncArgs{RepoKey: GenerateRepoKey(repoName, "", ""), RepoName: repoName}, nil).
					Build()
				mockey.Mock(taskq.Enqueue).Return(errors.New("asynq unavailable")).Build()

				_, err := service.RefreshRepositorySnapshots(auth.WithMaintenanceUser(ctx), repoName)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("enqueue image detail sync task"))
				Expect(err.Error()).To(ContainSubstring("asynq unavailable"))
			})
		})
	})
})
