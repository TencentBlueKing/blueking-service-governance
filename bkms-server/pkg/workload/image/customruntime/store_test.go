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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

var _ = Describe("Custom runtime image store", func() {
	var (
		ctx   context.Context
		store Store
		diApp *fxtest.App
	)

	BeforeEach(func() {
		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
		ctx = context.Background()
	})

	AfterEach(func() {
		Expect(store.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	Describe("Upsert", func() {
		It("creates a record and fills identity fields", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(store.Upsert(ctx, image)).To(Succeed())
			Expect(image.ID).NotTo(BeEmpty())
			Expect(image.CreatedAt).NotTo(BeZero())
			Expect(image.UpdatedAt).NotTo(BeZero())

			got, err := store.GetByWorkspaceTypeAndName(ctx, image.WorkspaceID, image.Type, image.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(image.ID))
			Expect(got.Name).To(Equal(image.Name))
		})

		It("refreshes only updatedAt when the unique key already exists", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(store.Upsert(ctx, image)).To(Succeed())

			before, err := store.GetByWorkspaceTypeAndName(ctx, image.WorkspaceID, image.Type, image.Name)
			Expect(err).NotTo(HaveOccurred())
			originalID := before.ID
			originalCreatedAt := before.CreatedAt
			originalUpdatedAt := before.UpdatedAt

			// 从 DB 读回的 updatedAt 是毫秒精度，稍等再写，避免两次 upsert 时间戳无法区分
			time.Sleep(50 * time.Millisecond)

			duplicate := &Image{
				WorkspaceID: image.WorkspaceID,
				Type:        image.Type,
				Name:        image.Name,
			}
			Expect(store.Upsert(ctx, duplicate)).To(Succeed())

			got, err := store.GetByWorkspaceTypeAndName(ctx, image.WorkspaceID, image.Type, image.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(originalID))
			Expect(got.CreatedAt.Equal(originalCreatedAt)).To(BeTrue())
			Expect(got.UpdatedAt.After(originalUpdatedAt)).To(BeTrue())
		})

		It("stores builder and runner records separately for the same name", func() {
			builder := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(store.Upsert(ctx, builder)).To(Succeed())

			runner := newTestImage("ws-demo", ImageTypeRunner)
			runner.Name = builder.Name
			Expect(store.Upsert(ctx, runner)).To(Succeed())

			gotBuilder, err := store.GetByWorkspaceTypeAndName(ctx, "ws-demo", ImageTypeBuilder, builder.Name)
			Expect(err).NotTo(HaveOccurred())
			gotRunner, err := store.GetByWorkspaceTypeAndName(ctx, "ws-demo", ImageTypeRunner, builder.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotBuilder.ID).NotTo(Equal(gotRunner.ID))
		})

		It("rejects an image whose name contains a tag", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			image.Name += ":latest"
			err := store.Upsert(ctx, image)
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(ErrCustomRuntimeImageNotFound))
		})
	})

	Describe("GetByWorkspaceTypeAndName", func() {
		It("returns not found when the record does not exist", func() {
			_, err := store.GetByWorkspaceTypeAndName(
				ctx, "ws-demo", ImageTypeBuilder, "docker.bkrepo.example.com/demo/repo/missing",
			)
			Expect(err).To(MatchError(ErrCustomRuntimeImageNotFound))
		})

		It("does not match the same name under a different workspace", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(store.Upsert(ctx, image)).To(Succeed())

			_, err := store.GetByWorkspaceTypeAndName(ctx, "ws-other", image.Type, image.Name)
			Expect(err).To(MatchError(ErrCustomRuntimeImageNotFound))
		})
	})

	Describe("List", func() {
		It("lists records in one workspace and ignores other workspaces", func() {
			inWorkspace := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(store.Upsert(ctx, inWorkspace)).To(Succeed())

			otherWorkspace := newTestImage("ws-other", ImageTypeBuilder)
			Expect(store.Upsert(ctx, otherWorkspace)).To(Succeed())

			images, err := store.List(ctx, "ws-demo", ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(images).To(HaveLen(1))
			Expect(images[0].ID).To(Equal(inWorkspace.ID))
		})

		It("filters by type and keyword", func() {
			builder := newTestImage("ws-demo", ImageTypeBuilder)
			builder.Name = "docker.bkrepo.example.com/demo/repo/special-golang"
			Expect(store.Upsert(ctx, builder)).To(Succeed())

			runner := newTestImage("ws-demo", ImageTypeRunner)
			runner.Name = "docker.bkrepo.example.com/demo/repo/special-alpine"
			Expect(store.Upsert(ctx, runner)).To(Succeed())

			unmatched := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(store.Upsert(ctx, unmatched)).To(Succeed())

			images, err := store.List(ctx, "ws-demo", ListOptions{
				Type:    ImageTypeBuilder,
				Keyword: "special",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(images).To(HaveLen(1))
			Expect(images[0].ID).To(Equal(builder.ID))
		})

		It("rejects an empty workspace ID", func() {
			_, err := store.List(ctx, "  ", ListOptions{})
			Expect(err).To(MatchError(ContainSubstring("workspaceID is required")))
		})

		It("returns an empty list when the workspace has no records", func() {
			images, err := store.List(ctx, "ws-empty", ListOptions{Type: ImageTypeBuilder})
			Expect(err).NotTo(HaveOccurred())
			Expect(images).To(BeEmpty())
		})
	})
})

func newTestImage(workspaceID string, imageType ImageType) *Image {
	// 每次随机仓库名，避免用例之间因唯一索引互相干扰
	return &Image{
		WorkspaceID: workspaceID,
		Type:        imageType,
		Name:        "docker.bkrepo.example.com/demo/repo/image-" + stringx.Random(6),
	}
}
