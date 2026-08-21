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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Custom runtime image model", func() {
	Describe("validateName", func() {
		It("accepts a full address with registry host and multi-level path", func() {
			image := &Image{Name: "docker.bkrepo.example.com/demo/repo/my-golang"}
			Expect(image.validateName()).To(Succeed())
		})

		It("accepts registry ports", func() {
			image := &Image{Name: "registry.example.com:5000/team/runtime/base"}
			Expect(image.validateName()).To(Succeed())
		})

		It("accepts an explicit docker.io host", func() {
			image := &Image{Name: "docker.io/library/nginx"}
			Expect(image.validateName()).To(Succeed())
		})

		It("rejects short names that normalize to docker.io", func() {
			image := &Image{Name: "nginx"}
			Expect(image.validateName()).To(MatchError(ContainSubstring("must include a registry host")))
		})

		It("rejects names without a registry host segment", func() {
			image := &Image{Name: "library/nginx"}
			Expect(image.validateName()).To(MatchError(ContainSubstring("must include a registry host")))
		})

		It("rejects blank names", func() {
			image := &Image{Name: "   "}
			Expect(image.validateName()).To(MatchError(ContainSubstring("image name is required")))
		})

		It("rejects uppercase repository paths", func() {
			image := &Image{Name: "registry.example.com/team/Runtime"}
			Expect(image.validateName()).To(HaveOccurred())
		})

		It("rejects image tags", func() {
			image := &Image{Name: "registry.example.com/team/runtime:latest"}
			Expect(image.validateName()).To(MatchError(ContainSubstring("must not contain tag or digest")))
		})

		It("rejects image digests", func() {
			image := &Image{Name: "registry.example.com/team/runtime@sha256:abc"}
			Expect(image.validateName()).To(HaveOccurred())
		})
	})

	Describe("validateType", func() {
		It("accepts supported types", func() {
			Expect((&Image{Type: ImageTypeBuilder}).validateType()).To(Succeed())
			Expect((&Image{Type: ImageTypeRunner}).validateType()).To(Succeed())
		})

		It("rejects unsupported and empty types", func() {
			Expect((&Image{Type: ImageType("unknown")}).validateType()).To(HaveOccurred())
			Expect((&Image{}).validateType()).To(HaveOccurred())
		})
	})

	Describe("Validate", func() {
		It("accepts a complete record", func() {
			image := &Image{
				WorkspaceID: "ws-demo",
				Type:        ImageTypeBuilder,
				Name:        "docker.bkrepo.example.com/demo/repo/my-golang",
			}
			Expect(image.Validate()).To(Succeed())
		})

		It("rejects records without a workspace", func() {
			image := &Image{
				Type: ImageTypeRunner,
				Name: "docker.bkrepo.example.com/demo/repo/my-golang",
			}
			Expect(image.Validate()).To(MatchError(ContainSubstring("workspaceID is required")))
		})

		It("rejects a nil receiver", func() {
			var image *Image
			Expect(image.Validate()).To(HaveOccurred())
		})
	})
})
