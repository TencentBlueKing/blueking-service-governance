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

package runtime

import (
	"context"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
)

var _ = Describe("Runtime image store", func() {
	var (
		ctx    context.Context
		images *Store
	)

	BeforeEach(func() {
		ctx = context.Background()
		images = testutil.NewStore(GinkgoT(), Meta)
	})

	AfterEach(func() {
		Expect(testutil.CleanupCollection(Meta.Collection)).To(Succeed())
	})

	It("creates a runtime image with a hex ID and trimmed name", func() {
		image := newTestImage(ImageTypeBuilder)
		name := image.Name
		image.Name = "  " + name + "  "
		Expect(images.Create(ctx, image)).To(Succeed())
		Expect(image.Name).To(Equal(name))

		created, err := images.GetByID(ctx, image.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(created.Name).To(Equal(name))
		Expect(created.Description).To(Equal(image.Description))
	})

	It("rejects invalid runtime images on create", func() {
		image := newTestImage(ImageType("unknown"))
		Expect(images.Create(ctx, image)).To(MatchError(ContainSubstring("unsupported runtime image type")))

		image = newTestImage(ImageTypeBuilder)
		image.Name = image.Name + ":latest"
		Expect(images.Create(ctx, image)).To(MatchError(ContainSubstring("must not contain tag or digest")))
	})

	It("gets a runtime image by type and name", func() {
		image := newTestImage(ImageTypeBuilder)
		Expect(images.Create(ctx, image)).To(Succeed())

		created, err := images.Get(ctx, ByTypeAndName(ImageTypeBuilder, " "+image.Name+" "))
		Expect(err).NotTo(HaveOccurred())
		Expect(created.ID).To(Equal(image.ID))
		Expect(created.Type).To(Equal(ImageTypeBuilder))

		_, err = images.Get(ctx, ByTypeAndName(ImageTypeRunner, image.Name))
		Expect(err).To(MatchError(ErrRuntimeImageNotFound))
	})

	It("rejects duplicate image names in the same type", func() {
		image := newTestImage(ImageTypeBuilder)
		Expect(images.Create(ctx, image)).To(Succeed())

		duplicate := newTestImage(ImageTypeBuilder)
		duplicate.Name = image.Name
		Expect(images.Create(ctx, duplicate)).To(MatchError(ErrRuntimeImageAlreadyExists))
	})

	It("lists runtime images by type and keyword", func() {
		builder := newTestImage(ImageTypeBuilder)
		builder.Name = "registry.example.com/team/special-builder"
		Expect(images.Create(ctx, builder)).To(Succeed())

		runner := newTestImage(ImageTypeRunner)
		runner.Description = "SPECIAL runtime runner"
		Expect(images.Create(ctx, runner)).To(Succeed())

		unmatched := newTestImage(ImageTypeBuilder)
		Expect(images.Create(ctx, unmatched)).To(Succeed())

		list, err := images.List(ctx, KeywordMatches(" special "))
		Expect(err).NotTo(HaveOccurred())
		Expect(list).To(ConsistOf(HaveField("ID", builder.ID), HaveField("ID", runner.ID)))

		list, err = images.List(ctx, ByType(ImageTypeBuilder), KeywordMatches(""))
		Expect(err).NotTo(HaveOccurred())
		Expect(list).To(ConsistOf(HaveField("ID", builder.ID), HaveField("ID", unmatched.ID)))
	})

	Context("UpdateDescription", func() {
		It("updates description and refreshes updatedAt while keeping other fields", func() {
			image := newTestImage(ImageTypeBuilder)
			Expect(images.Create(ctx, image)).To(Succeed())

			// 从 DB 读回一次，保证 originalUpdatedAt 使用与存储一致的毫秒精度，避免 in-memory 纳秒时间与
			// mongo 毫秒精度截断后的比较不稳定
			before, err := images.GetByID(ctx, image.ID)
			Expect(err).NotTo(HaveOccurred())

			// 稍作等待以便 updatedAt 有明显差异
			time.Sleep(50 * time.Millisecond)

			newDescription := "updated description"
			Expect(UpdateDescription(ctx, images, image.Type, image.Name, newDescription)).To(Succeed())

			updated, err := images.GetByID(ctx, image.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Description).To(Equal(newDescription))
			Expect(updated.UpdatedAt.After(before.UpdatedAt)).To(BeTrue())
			// 其余字段保持不变
			Expect(updated.Type).To(Equal(image.Type))
			Expect(updated.Name).To(Equal(image.Name))
			Expect(updated.CreatedAt.Equal(before.CreatedAt)).To(BeTrue())
		})

		It("returns not found when the target record does not exist", func() {
			err := UpdateDescription(
				ctx,
				images,
				ImageTypeBuilder,
				"registry.example.com/team/missing-image",
				"anything",
			)
			Expect(err).To(MatchError(ErrRuntimeImageNotFound))
		})

		It("rejects description that exceeds the maximum length", func() {
			image := newTestImage(ImageTypeRunner)
			Expect(images.Create(ctx, image)).To(Succeed())

			tooLong := strings.Repeat("a", maxImageDescriptionLen+1)
			err := UpdateDescription(ctx, images, image.Type, image.Name, tooLong)
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(ErrRuntimeImageNotFound))
		})
	})
})

func newTestImage(imageType ImageType) *Image {
	name := "registry.example.com/team/image-" + stringx.Random(6)
	return &Image{
		Type:        imageType,
		Name:        name,
		Description: "test image",
	}
}
