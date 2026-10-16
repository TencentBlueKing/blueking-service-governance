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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Custom runtime image store", func() {
	var (
		ctx    context.Context
		images *Store
	)

	BeforeEach(func() {
		ctx = context.Background()
		images = testutil.NewStore(GinkgoT(), Meta)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(Meta)).To(Succeed())
	})

	Describe("Register", func() {
		It("creates a record that can be found by key", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(Register(ctx, images, image)).To(Succeed())

			got, err := images.Get(ctx, ByKey(image.WorkspaceID, image.Type, image.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(image.ID))
			Expect(got.Name).To(Equal(image.Name))
		})

		It("stores trimmed workspace ID and name", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			name := image.Name
			image.WorkspaceID, image.Name = " ws-demo ", "  "+name+"  "
			Expect(Register(ctx, images, image)).To(Succeed())

			got, err := images.Get(ctx, ByKey("ws-demo", ImageTypeBuilder, name))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.WorkspaceID).To(Equal("ws-demo"))
			Expect(got.Name).To(Equal(name))
		})

		It("refreshes only updatedAt when the unique key already exists", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(Register(ctx, images, image)).To(Succeed())

			before, err := images.Get(ctx, ByKey(image.WorkspaceID, image.Type, image.Name))
			Expect(err).NotTo(HaveOccurred())

			// 从 DB 读回的 updatedAt 是毫秒精度，稍等再写，避免两次写入的时间戳无法区分
			time.Sleep(50 * time.Millisecond)

			duplicate := &Image{WorkspaceID: image.WorkspaceID, Type: image.Type, Name: image.Name}
			Expect(Register(ctx, images, duplicate)).To(Succeed())

			got, err := images.Get(ctx, ByKey(image.WorkspaceID, image.Type, image.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(before.ID))
			Expect(got.CreatedAt.Equal(before.CreatedAt)).To(BeTrue())
			Expect(got.UpdatedAt.After(before.UpdatedAt)).To(BeTrue())
			Expect(images.Count(ctx, ByWorkspace(image.WorkspaceID))).To(BeEquivalentTo(1))
		})

		It("stores builder and runner records separately for the same name", func() {
			builder := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(Register(ctx, images, builder)).To(Succeed())

			runner := newTestImage("ws-demo", ImageTypeRunner)
			runner.Name = builder.Name
			Expect(Register(ctx, images, runner)).To(Succeed())

			gotBuilder, err := images.Get(ctx, ByKey("ws-demo", ImageTypeBuilder, builder.Name))
			Expect(err).NotTo(HaveOccurred())
			gotRunner, err := images.Get(ctx, ByKey("ws-demo", ImageTypeRunner, builder.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(gotBuilder.ID).NotTo(Equal(gotRunner.ID))
		})

		It("rejects an image whose name contains a tag", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			image.Name += ":latest"
			Expect(Register(ctx, images, image)).To(MatchError(ContainSubstring("must not contain tag or digest")))
		})

		It("rejects an image without workspace ID", func() {
			image := newTestImage("  ", ImageTypeBuilder)
			Expect(Register(ctx, images, image)).To(MatchError(ContainSubstring("workspaceID is required")))
		})
	})

	Describe("Get by key", func() {
		It("returns not found when the record does not exist", func() {
			_, err := images.Get(
				ctx, ByKey("ws-demo", ImageTypeBuilder, "docker.bkrepo.example.com/demo/repo/missing"),
			)
			Expect(err).To(MatchError(ErrCustomRuntimeImageNotFound))
			Expect(err).To(MatchError(store.ErrNotFound))
		})

		It("does not match the same name under a different workspace", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(Register(ctx, images, image)).To(Succeed())

			_, err := images.Get(ctx, ByKey("ws-other", image.Type, image.Name))
			Expect(err).To(MatchError(ErrCustomRuntimeImageNotFound))
		})
	})

	Describe("Exists by workspace and name", func() {
		It("matches by workspace and name regardless of type", func() {
			builder := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(Register(ctx, images, builder)).To(Succeed())
			Expect(images.Exists(ctx, ByWorkspaceAndName("ws-demo", builder.Name))).To(BeTrue())

			// 只落 runner 时同样按名称命中，调用方无需知道类型
			runner := newTestImage("ws-demo", ImageTypeRunner)
			Expect(Register(ctx, images, runner)).To(Succeed())
			Expect(images.Exists(ctx, ByWorkspaceAndName("ws-demo", runner.Name))).To(BeTrue())

			Expect(images.Exists(ctx, ByWorkspaceAndName("ws-other", builder.Name))).To(BeFalse())
			Expect(images.Exists(ctx, ByWorkspaceAndName("ws-demo", builder.Name+"-missing"))).To(BeFalse())
		})

		It("does not match any record when the workspace ID is empty", func() {
			image := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(Register(ctx, images, image)).To(Succeed())
			Expect(images.Exists(ctx, ByWorkspaceAndName("", image.Name))).To(BeFalse())
		})
	})

	Describe("List", func() {
		It("lists records in one workspace and ignores other workspaces", func() {
			inWorkspace := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(Register(ctx, images, inWorkspace)).To(Succeed())
			otherWorkspace := newTestImage("ws-other", ImageTypeBuilder)
			Expect(Register(ctx, images, otherWorkspace)).To(Succeed())

			got, err := images.List(ctx, ByWorkspace("ws-demo"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(1))
			Expect(got[0].ID).To(Equal(inWorkspace.ID))
		})

		It("filters by type and keyword", func() {
			builder := newTestImage("ws-demo", ImageTypeBuilder)
			builder.Name = "docker.bkrepo.example.com/demo/repo/special-golang"
			Expect(Register(ctx, images, builder)).To(Succeed())

			runner := newTestImage("ws-demo", ImageTypeRunner)
			runner.Name = "docker.bkrepo.example.com/demo/repo/special-alpine"
			Expect(Register(ctx, images, runner)).To(Succeed())

			unmatched := newTestImage("ws-demo", ImageTypeBuilder)
			Expect(Register(ctx, images, unmatched)).To(Succeed())

			got, err := images.List(ctx, ByWorkspace("ws-demo"), ByType(ImageTypeBuilder), NameContains("SPECIAL"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(1))
			Expect(got[0].ID).To(Equal(builder.ID))
		})

		It("lists all types ordered by type then name when type is empty", func() {
			runner := newTestImage("ws-demo", ImageTypeRunner)
			runner.Name = "docker.bkrepo.example.com/demo/repo/a"
			builderB := newTestImage("ws-demo", ImageTypeBuilder)
			builderB.Name = "docker.bkrepo.example.com/demo/repo/b"
			builderA := newTestImage("ws-demo", ImageTypeBuilder)
			builderA.Name = "docker.bkrepo.example.com/demo/repo/a"
			for _, img := range []*Image{runner, builderB, builderA} {
				Expect(Register(ctx, images, img)).To(Succeed())
			}

			got, err := images.List(ctx, ByWorkspace("ws-demo"), ByType(""), InListOrder())
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(3))
			Expect([]string{got[0].ID, got[1].ID, got[2].ID}).To(Equal([]string{builderA.ID, builderB.ID, runner.ID}))
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
