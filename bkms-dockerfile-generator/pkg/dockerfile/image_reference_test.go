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

package dockerfile

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dockerfile image reference helpers", func() {
	DescribeTable("checks fragments only in image tags",
		func(image string, fragment string, expected bool) {
			Expect(imageTagContains(image, fragment)).To(Equal(expected))
		},
		Entry("empty image", "", "alpine", false),
		Entry("empty fragment", "golang:1.25-alpine", "", false),
		Entry("image without tag", "golang", "alpine", false),
		Entry("registry port without tag", "registry.example.com:5000/golang", "alpine", false),
		Entry("registry namespace contains fragment", "registry.example.com/alpine/golang:1.25", "alpine", false),
		Entry("registry port with Debian tag", "registry.example.com:5000/golang:1.25", "alpine", false),
		Entry("Alpine tag", "golang:1.25-alpine", "alpine", true),
		Entry("Alpine tag with digest", "golang:1.25.3-alpine3.22@sha256:abcd", "alpine", true),
		Entry("registry port with Alpine tag and digest", "registry.example.com:5000/golang:1.25.3-alpine3.22@sha256:abcd", "alpine", true),
		Entry("Debian tag with digest", "golang:1.25@sha256:abcd", "alpine", false),
	)
})
