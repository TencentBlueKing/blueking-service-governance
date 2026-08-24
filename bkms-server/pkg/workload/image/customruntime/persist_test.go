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
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

func platformPersistConfig(builder, runner string) *imagebuild.Config {
	return &imagebuild.Config{
		SourceType: imagebuild.SourceTypeCodeRepository,
		CodeRepo: &imagebuild.RepositoryConfig{
			ImageBuildMode: imagebuild.ImageBuildModePlatform,
			PlatformBuildConfig: &imagebuild.PlatformBuildConfig{
				BuilderImage: builder,
				RunnerImage:  runner,
			},
		},
	}
}

var _ = Describe("PersistManager", func() {
	const (
		registryAddr = "docker.bkrepo.example.com/demo/repo"
		imageName    = registryAddr + "/my-golang"
		imageRef     = imageName + ":1.0"
	)

	var (
		ctx         context.Context
		store       Store
		diApp       *fxtest.App
		workspaceID string
		snapshotSvc *snapshot.Service
	)

	BeforeEach(func() {
		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
		ctx = context.Background()
		workspaceID = "ws-persist-" + stringx.Random(6)
		snapshotSvc = &snapshot.Service{}
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		Expect(store.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	newManager := func() *PersistManager {
		mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(
			&bkmsreg.ImageRegistry{Registry: registryAddr}, nil,
		).Build()
		return NewPersistManager(store, snapshotSvc)
	}

	// mockSnapshotStatus 模拟快照状态：refreshedAt 为 nil 表示该镜像从未成功刷新过
	mockSnapshotStatus := func(refreshedAt *time.Time) {
		var status *snapshot.RepoSnapshotStatus
		if refreshedAt != nil {
			status = &snapshot.RepoSnapshotStatus{LastRefreshedAt: refreshedAt}
		}
		mockey.Mock((*snapshot.Service).GetWorkspaceSnapshotStatus).Return(status, nil).Build()
	}

	It("writes a new custom image and refreshes snapshots", func() {
		mockSnapshotStatus(nil)
		refreshCalls := 0
		mockey.Mock((*snapshot.Service).RefreshWorkspaceSnapshots).To(
			func(_ *snapshot.Service, _ context.Context, gotWorkspaceID, gotImageName string) (*snapshot.RefreshResult, error) {
				refreshCalls++
				Expect(gotWorkspaceID).To(Equal(workspaceID))
				Expect(gotImageName).To(Equal(imageName))
				return &snapshot.RefreshResult{}, nil
			},
		).
			Build()

		err := newManager().PersistAfterSave(ctx, workspaceID, platformPersistConfig(imageRef, "debian:12"))

		Expect(err).NotTo(HaveOccurred())
		Expect(refreshCalls).To(Equal(1))

		got, getErr := store.GetByWorkspaceTypeAndName(ctx, workspaceID, ImageTypeBuilder, imageName)
		Expect(getErr).NotTo(HaveOccurred())
		Expect(got.Name).To(Equal(imageName))
		_, runnerErr := store.GetByWorkspaceTypeAndName(ctx, workspaceID, ImageTypeRunner, imageName)
		Expect(runnerErr).To(MatchError(ErrCustomRuntimeImageNotFound))
	})

	It("does not refresh snapshots when the image already has a successful snapshot", func() {
		existing := &Image{WorkspaceID: workspaceID, Type: ImageTypeBuilder, Name: imageName}
		Expect(store.Upsert(ctx, existing)).To(Succeed())
		mockSnapshotStatus(lo.ToPtr(time.Now()))

		refreshCalls := 0
		mockey.Mock((*snapshot.Service).RefreshWorkspaceSnapshots).To(
			func(_ *snapshot.Service, _ context.Context, _, _ string) (*snapshot.RefreshResult, error) {
				refreshCalls++
				return &snapshot.RefreshResult{}, nil
			},
		).Build()

		err := newManager().PersistAfterSave(ctx, workspaceID, platformPersistConfig(imageRef, "debian:12"))

		Expect(err).NotTo(HaveOccurred())
		Expect(refreshCalls).To(Equal(0))
		got, getErr := store.GetByWorkspaceTypeAndName(ctx, workspaceID, ImageTypeBuilder, imageName)
		Expect(getErr).NotTo(HaveOccurred())
		Expect(got.ID).To(Equal(existing.ID))
	})

	It("retries the refresh when the record exists but was never refreshed successfully", func() {
		// 首次刷新失败的现场：记录已落库，但快照状态里没有成功刷新时间
		Expect(store.Upsert(ctx, &Image{
			WorkspaceID: workspaceID, Type: ImageTypeBuilder, Name: imageName,
		})).To(Succeed())
		mockSnapshotStatus(nil)

		refreshCalls := 0
		mockey.Mock((*snapshot.Service).RefreshWorkspaceSnapshots).To(
			func(_ *snapshot.Service, _ context.Context, _, _ string) (*snapshot.RefreshResult, error) {
				refreshCalls++
				return &snapshot.RefreshResult{}, nil
			},
		).Build()

		err := newManager().PersistAfterSave(ctx, workspaceID, platformPersistConfig(imageRef, "debian:12"))

		Expect(err).NotTo(HaveOccurred())
		Expect(refreshCalls).To(Equal(1))
	})

	It("keeps the persisted record when snapshot refresh fails", func() {
		mockSnapshotStatus(nil)
		mockey.Mock((*snapshot.Service).RefreshWorkspaceSnapshots).Return(
			nil, errors.New("refresh failed"),
		).Build()

		err := newManager().PersistAfterSave(ctx, workspaceID, platformPersistConfig(imageRef, "debian:12"))

		Expect(err).To(MatchError(ContainSubstring("refresh failed")))
		_, getErr := store.GetByWorkspaceTypeAndName(ctx, workspaceID, ImageTypeBuilder, imageName)
		Expect(getErr).NotTo(HaveOccurred())
	})
})
