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

package customruntime

import (
	"context"
	"net/http"
	"time"

	"github.com/bytedance/mockey"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	infrasreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

var _ = Describe("TagQueryManager", func() {
	const (
		workspaceID  = "ws-tag-demo"
		registryAddr = "docker.bkrepo.example.com/demo/repo"
		imageName    = registryAddr + "/my-golang"
	)

	var (
		ctx           context.Context
		store         Store
		snapshotStore snapshot.SnapshotStore
		manager       *TagQueryManager
		diApp         *fxtest.App
		repoKey       string
		listAllCalls  int
		// 以下三者由各用例改写，用于控制镜像源绑定与远程拉取结果
		boundRegistry *bkmsreg.ImageRegistry
		remoteTags    []string
		remoteErr     error
	)

	BeforeEach(func() {
		diApp = fxtest.New(GinkgoT(), FxModule, fx.Populate(&store))
		diApp.RequireStart()
		ctx = context.Background()

		var err error
		snapshotStore, err = snapshot.NewSnapshotStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		manager = NewTagQueryManager(store, snapshot.NewService(snapshotStore, nil, nil))
		repoKey = snapshot.GenerateRepoKey(imageName, "ws-user", "ws-pass")
		listAllCalls = 0
		remoteTags, remoteErr = nil, nil
		boundRegistry = &bkmsreg.ImageRegistry{
			Registry: registryAddr, Username: "ws-user", Password: "ws-pass",
		}

		mockey.Mock(workspace.GetWorkspaceImageRegistry).To(
			func(context.Context, string) (*bkmsreg.ImageRegistry, error) {
				if boundRegistry == nil {
					return nil, bkmsreg.ErrImageRegistryNotFound
				}
				return boundRegistry, nil
			},
		).Build()
		mockey.Mock(infrasreg.New).Return(&infrasreg.Client{}).Build()
		mockey.Mock((*infrasreg.Client).ListAllTags).To(
			func(*infrasreg.Client, context.Context, string) ([]string, error) {
				listAllCalls++
				return remoteTags, remoteErr
			},
		).Build()
		mockey.Mock(taskq.Enqueue).Return(nil).Build()
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		Expect(store.DeleteAll(ctx)).To(Succeed())
		Expect(snapshotStore.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	// seedPersisted 造落库记录与新鲜快照，使该镜像走快照来源且不触发 TTL 懒刷新
	seedPersisted := func(tags ...string) {
		Expect(store.Upsert(ctx, &Image{
			WorkspaceID: workspaceID, Type: ImageTypeBuilder, Name: imageName,
		})).To(Succeed())
		Expect(snapshotStore.UpsertSnapshots(ctx, repoKey, lo.Map(tags,
			func(tag string, _ int) snapshot.Image { return snapshot.Image{Tag: tag} },
		))).To(Succeed())
		refreshedAt := time.Now()
		Expect(snapshotStore.UpsertStatus(ctx, &snapshot.RepoSnapshotStatus{
			RepoKey: repoKey, RepoName: imageName,
			RefreshStatus: snapshot.RefreshStatusIdle, LastRefreshedAt: &refreshedAt,
		})).To(Succeed())
	}

	It("switches from the realtime source to the snapshot source once the image is persisted", func() {
		remoteTags = []string{"v1.0", "v2.0"}

		// 无落库记录时走实时拉取，且不得留下镜像记录或快照
		queried, err := manager.ListTags(ctx, workspaceID, imageName, "", 1, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(queried.Total).To(BeEquivalentTo(2))
		Expect(queried.Status).To(BeNil())
		Expect(listAllCalls).To(Equal(1))

		records, err := store.List(ctx, workspaceID, ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(records).To(BeEmpty())
		_, total, err := snapshotStore.ListByRepoKey(ctx, repoKey, "", 1, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(total).To(BeZero())

		// 落库后同一请求自动切到快照来源，不再打远端
		seedPersisted("v1.0")
		queried, err = manager.ListTags(ctx, workspaceID, imageName, "", 1, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(queried.Total).To(BeEquivalentTo(1))
		Expect(queried.Status).NotTo(BeNil())
		Expect(listAllCalls).To(Equal(1))
	})

	It("filters case-insensitively and slices the requested page", func() {
		remoteTags = []string{"v1.0", "V1.1", "v1.2", "v2.0"}

		// 大小写不敏感，与快照来源的 Mongo 正则过滤口径一致
		queried, err := manager.ListTags(ctx, workspaceID, imageName, "V1", 1, 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(queried.Total).To(BeEquivalentTo(3))
		Expect(queried.Tags).To(HaveLen(2))

		queried, err = manager.ListTags(ctx, workspaceID, imageName, "V1", 2, 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(queried.Tags).To(HaveLen(1))

		// 越界不报错，仍返回过滤后的真实总数
		queried, err = manager.ListTags(ctx, workspaceID, imageName, "V1", 100, 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(queried.Total).To(BeEquivalentTo(3))
		Expect(queried.Tags).To(BeEmpty())
	})

	It("refuses to use the credential outside the workspace registry", func() {
		remoteTags = []string{"v1.0"}

		_, err := manager.ListTags(ctx, workspaceID, "evil.example.com/foo/bar", "", 1, 10)
		Expect(err).To(MatchError(ErrImageNotInWorkspaceRegistry))
		Expect(err).To(MatchError(ContainSubstring(registryAddr)))
		_, err = manager.RefreshTags(ctx, workspaceID, "evil.example.com/foo/bar")
		Expect(err).To(MatchError(ErrImageNotInWorkspaceRegistry))

		// 未落库镜像只走实时查询，显式刷新必须已有记录，避免写出无人读的孤儿快照
		_, err = manager.RefreshTags(ctx, workspaceID, imageName)
		Expect(err).To(MatchError(ErrCustomRuntimeImageNotFound))

		boundRegistry = nil
		_, err = manager.ListTags(ctx, workspaceID, imageName, "", 1, 10)
		Expect(err).To(MatchError(ErrWorkspaceRegistryUnbound))

		// 关键断言：凭证不能被发往非本工作空间镜像源的主机
		Expect(listAllCalls).To(BeZero())
	})

	It("classifies registry errors raised by the realtime source", func() {
		classify := func(raised error) error {
			remoteErr = raised
			_, err := manager.ListTags(ctx, workspaceID, imageName, "", 1, 10)
			return err
		}

		Expect(classify(&transport.Error{StatusCode: http.StatusNotFound})).
			To(MatchError(ErrImageNameNotFound))
		Expect(classify(&transport.Error{StatusCode: http.StatusForbidden})).
			To(MatchError(ErrRegistryAccessDenied))
		// 超时与其它访问失败分开，前者对外要给可识别提示
		Expect(classify(context.DeadlineExceeded)).To(MatchError(ErrRegistryAccessTimeout))
		Expect(classify(errors.New("connection reset"))).To(MatchError(ErrRegistryAccessFailed))
	})

	It("reports success, refreshing and registry failures without erroring out", func() {
		remoteTags = []string{"v1.0", "v2.0"}
		seedPersisted("v1.0", "stale")

		result, err := manager.RefreshTags(ctx, workspaceID, imageName)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(snapshot.RefreshResultSuccess))
		Expect(result.AddedTagCnt).To(BeEquivalentTo(1))
		Expect(result.RemovedTagCnt).To(BeEquivalentTo(1))

		// 已有刷新在进行中：幂等保护生效，不重复发起远程调用
		callsBefore := listAllCalls
		Expect(snapshotStore.UpsertStatus(ctx, &snapshot.RepoSnapshotStatus{
			RepoKey: repoKey, RepoName: imageName, RefreshStatus: snapshot.RefreshStatusRefreshing,
		})).To(Succeed())
		result, err = manager.RefreshTags(ctx, workspaceID, imageName)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(snapshot.RefreshResultRefreshing))
		Expect(listAllCalls).To(Equal(callsBefore))

		// 仓库侧失败折成 failed；seedPersisted 顺带把状态复位以便重新抢到刷新权
		remoteTags, remoteErr = nil, &transport.Error{StatusCode: http.StatusUnauthorized}
		seedPersisted("v1.0")
		result, err = manager.RefreshTags(ctx, workspaceID, imageName)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(snapshot.RefreshResultFailed))
		Expect(result.Message).To(Equal(registryAuthFailedMessage))

		// 快照内容不因刷新失败而清空，状态回置为 idle 并记录原因
		_, total, err := snapshotStore.ListByRepoKey(ctx, repoKey, "", 1, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(total).To(BeNumerically(">", 0))
		status, err := snapshotStore.GetStatus(ctx, repoKey)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.RefreshStatus).To(Equal(snapshot.RefreshStatusIdle))
		Expect(status.LastError).NotTo(BeEmpty())
	})
})
