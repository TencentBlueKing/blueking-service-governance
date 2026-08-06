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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runtime image model", func() {
	Describe("validateName", func() {
		It("accepts registry ports and multi-level paths", func() {
			image := &Image{Name: "registry.example.com:5000/team/runtime/base"}
			err := image.validateName()
			Expect(err).NotTo(HaveOccurred())
		})

		It("accepts repository paths with repeated separators", func() {
			image := &Image{Name: "ghcr.io/org/repo--name"}
			err := image.validateName()
			Expect(err).NotTo(HaveOccurred())
		})

		It("accepts Docker Hub official images", func() {
			image := &Image{Name: "golang"}
			err := image.validateName()
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects uppercase repository paths", func() {
			image := &Image{Name: "registry.example.com/team/Runtime"}
			err := image.validateName()
			Expect(err).To(HaveOccurred())
		})

		It("rejects image tags", func() {
			image := &Image{Name: "registry.example.com/team/runtime:latest"}
			err := image.validateName()
			Expect(err).To(HaveOccurred())
		})

		It("rejects image digests", func() {
			image := &Image{Name: "registry.example.com/team/runtime@sha256:abc"}
			err := image.validateName()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("validateType", func() {
		It("accepts supported types", func() {
			Expect((&Image{Type: ImageTypeBuilder}).validateType()).To(Succeed())
			Expect((&Image{Type: ImageTypeRunner}).validateType()).To(Succeed())
		})

		It("rejects unsupported types", func() {
			Expect((&Image{Type: ImageType("unknown")}).validateType()).To(HaveOccurred())
		})
	})

	Describe("validateDescription", func() {
		It("accepts description at length limit", func() {
			image := &Image{Description: strings.Repeat("测", maxImageDescriptionLen)}
			err := image.validateDescription()
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects description exceeding length limit", func() {
			image := &Image{Description: strings.Repeat("测", maxImageDescriptionLen) + "试"}
			err := image.validateDescription()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParseTaggedImageReference", func() {
		It("parses tagged references with registry ports and multi-level paths", func() {
			ref, err := ParseTaggedImageReference("registry.example.com:5000/team/runtime/base:v1.2.3")

			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Name).To(Equal("registry.example.com:5000/team/runtime/base"))
			Expect(ref.Tag).To(Equal("v1.2.3"))
		})

		It("parses Docker Hub official images", func() {
			ref, err := ParseTaggedImageReference("golang:1.24")

			Expect(err).NotTo(HaveOccurred())
			Expect(ref.Name).To(Equal("golang"))
			Expect(ref.Tag).To(Equal("1.24"))
		})

		It("rejects references without tag", func() {
			_, err := ParseTaggedImageReference("golang")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must contain tag"))
		})

		It("rejects references containing digest", func() {
			_, err := ParseTaggedImageReference("debian:12@sha256:abc")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid image reference"))
		})
	})
})
