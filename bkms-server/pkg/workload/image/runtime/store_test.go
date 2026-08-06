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
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Runtime image store", func() {
	var (
		ctx   context.Context
		store Store
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		store, err = NewStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(store.DeleteAll(ctx)).To(Succeed())
	})

	It("creates and gets a runtime image", func() {
		image := newTestImage(ImageTypeBuilder)
		Expect(store.Create(ctx, image)).To(Succeed())

		created, err := store.GetByID(ctx, image.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(created.Name).To(Equal(image.Name))
		Expect(created.Description).To(Equal(image.Description))
	})

	It("gets a runtime image by ID", func() {
		image := newTestImage(ImageTypeRunner)
		Expect(store.Create(ctx, image)).To(Succeed())

		created, err := store.GetByID(ctx, image.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(created.ID).To(Equal(image.ID))
		Expect(created.Name).To(Equal(image.Name))
		Expect(created.Description).To(Equal(image.Description))
	})

	It("gets a runtime image by type and name", func() {
		image := newTestImage(ImageTypeBuilder)
		Expect(store.Create(ctx, image)).To(Succeed())

		created, err := store.GetByTypeAndName(ctx, ImageTypeBuilder, image.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(created.ID).To(Equal(image.ID))
		Expect(created.Type).To(Equal(ImageTypeBuilder))
		Expect(created.Name).To(Equal(image.Name))
	})

	It("returns not found when type and name do not match", func() {
		image := newTestImage(ImageTypeBuilder)
		Expect(store.Create(ctx, image)).To(Succeed())

		_, err := store.GetByTypeAndName(ctx, ImageTypeRunner, image.Name)
		Expect(err).To(MatchError(ErrRuntimeImageNotFound))
	})

	It("returns not found when getting missing runtime image by ID", func() {
		_, err := store.GetByID(ctx, bson.NewObjectID().Hex())
		Expect(err).To(MatchError(ErrRuntimeImageNotFound))

		_, err = store.GetByID(ctx, "not-a-hex")
		Expect(err).To(MatchError(ErrRuntimeImageNotFound))
	})

	It("rejects duplicate image names in the same type", func() {
		image := newTestImage(ImageTypeBuilder)
		Expect(store.Create(ctx, image)).To(Succeed())

		duplicate := newTestImage(ImageTypeBuilder)
		duplicate.Name = image.Name
		Expect(store.Create(ctx, duplicate)).To(MatchError(ErrRuntimeImageAlreadyExists))
	})

	It("lists runtime images by keyword", func() {
		builder := newTestImage(ImageTypeBuilder)
		builder.Name = "registry.example.com/team/special-builder"
		Expect(store.Create(ctx, builder)).To(Succeed())

		runner := newTestImage(ImageTypeRunner)
		runner.Description = "special runtime runner"
		Expect(store.Create(ctx, runner)).To(Succeed())

		unmatched := newTestImage(ImageTypeBuilder)
		Expect(store.Create(ctx, unmatched)).To(Succeed())

		images, err := store.List(ctx, ListOptions{Keyword: "special"})
		Expect(err).NotTo(HaveOccurred())
		Expect(images).To(HaveLen(2))
		Expect(images).To(ConsistOf(
			HaveField("ID", builder.ID),
			HaveField("ID", runner.ID),
		))
	})

	Context("UpdateDescription", func() {
		It("updates description and refreshes updatedAt while keeping other fields", func() {
			image := newTestImage(ImageTypeBuilder)
			Expect(store.Create(ctx, image)).To(Succeed())

			// 从 DB 读回一次，保证 originalUpdatedAt 使用与存储一致的毫秒精度，避免 in-memory 纳秒时间与
			// mongo 毫秒精度截断后的比较不稳定
			before, err := store.GetByID(ctx, image.ID)
			Expect(err).NotTo(HaveOccurred())
			originalCreatedAt := before.CreatedAt
			originalUpdatedAt := before.UpdatedAt

			// 稍作等待以便 updatedAt 有明显差异
			time.Sleep(50 * time.Millisecond)

			newDescription := "updated description"
			Expect(store.UpdateDescription(ctx, image.Type, image.Name, newDescription)).To(Succeed())

			updated, err := store.GetByID(ctx, image.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Description).To(Equal(newDescription))
			Expect(updated.UpdatedAt.After(originalUpdatedAt)).To(BeTrue())
			// 其余字段保持不变
			Expect(updated.ID).To(Equal(image.ID))
			Expect(updated.Type).To(Equal(image.Type))
			Expect(updated.Name).To(Equal(image.Name))
			Expect(updated.CreatedAt.Equal(originalCreatedAt)).To(BeTrue())
		})

		It("returns not found when the target record does not exist", func() {
			err := store.UpdateDescription(
				ctx,
				ImageTypeBuilder,
				"registry.example.com/team/missing-image",
				"anything",
			)
			Expect(err).To(MatchError(ErrRuntimeImageNotFound))
		})

		It("rejects description that exceeds the maximum length", func() {
			image := newTestImage(ImageTypeRunner)
			Expect(store.Create(ctx, image)).To(Succeed())

			tooLong := strings.Repeat("a", maxImageDescriptionLen+1)
			err := store.UpdateDescription(ctx, image.Type, image.Name, tooLong)
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
