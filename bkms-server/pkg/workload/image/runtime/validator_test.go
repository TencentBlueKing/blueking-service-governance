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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

var _ = Describe("ImageReferenceValidator", func() {
	var (
		ctx       context.Context
		images    *Store
		validator *ImageReferenceValidator
	)

	BeforeEach(func() {
		ctx = context.Background()
		images = testutil.NewStore(GinkgoT(), Meta)
		validator = NewImageReferenceValidator(testutil.NewDB(GinkgoT()), &snapshot.SnapshotStoreMongo{})
	})

	AfterEach(func() {
		Expect(testutil.CleanupCollection(Meta.Collection)).To(Succeed())
	})

	createRuntimeImage := func(imageType ImageType, name string) {
		Expect(images.Create(ctx, &Image{Type: imageType, Name: name})).To(Succeed())
	}

	mockSnapshotTag := func(exists bool, err error) {
		mockey.Mock((*snapshot.SnapshotStoreMongo).HasTag).Return(exists, err).Build()
	}

	It("accepts an existing runtime image and snapshot tag", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			createRuntimeImage(ImageTypeBuilder, "golang")
			mockSnapshotTag(true, nil)

			ref, err := validator.ValidateTaggedReference(ctx, ImageTypeBuilder, "golang:1.24")

			Expect(err).NotTo(HaveOccurred())
			Expect(ref).To(Equal(&ImageReference{Name: "golang", Tag: "1.24"}))
		})
	})

	It("returns not found when runtime image name does not exist", func() {
		_, err := validator.ValidateTaggedReference(ctx, ImageTypeBuilder, "golang:1.24")

		Expect(err).To(MatchError(ErrRuntimeImageNotFound))
	})

	It("returns not found when runtime image type does not match", func() {
		createRuntimeImage(ImageTypeRunner, "golang")

		_, err := validator.ValidateTaggedReference(ctx, ImageTypeBuilder, "golang:1.24")

		Expect(err).To(MatchError(ErrRuntimeImageNotFound))
	})

	It("returns tag not found when snapshot tag does not exist", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			createRuntimeImage(ImageTypeRunner, "debian")
			mockSnapshotTag(false, nil)

			_, err := validator.ValidateTaggedReference(ctx, ImageTypeRunner, "debian:12")

			Expect(err).To(MatchError(ErrRuntimeImageTagNotFound))
		})
	})
})
