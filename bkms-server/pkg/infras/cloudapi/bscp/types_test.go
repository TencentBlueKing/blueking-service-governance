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

package bscp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Versions", func() {
	Describe("LatestFullyReleased", func() {
		It("should return the first fully released version", func() {
			versions := Versions{
				{ID: "v3", IsFullyReleased: false},
				{ID: "v2", IsFullyReleased: true},
				{ID: "v1", IsFullyReleased: true},
			}

			got := versions.LatestFullyReleased()
			Expect(got).NotTo(BeNil())
			Expect(got.ID).To(Equal("v2"))
		})

		It("should return nil when no fully released version exists", func() {
			versions := Versions{
				{ID: "v1", IsFullyReleased: false},
			}

			got := versions.LatestFullyReleased()
			Expect(got).To(BeNil())
		})
	})
})
