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

package params_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/params"
)

var _ = Describe("NormalizeInstIDs", func() {
	It("splits comma-separated values and trims whitespace", func() {
		Expect(params.NormalizeInstIDs(" pod-1, pod-2 ,pod-3 ", ",")).To(Equal([]string{
			"pod-1",
			"pod-2",
			"pod-3",
		}))
	})

	It("filters empty values", func() {
		Expect(params.NormalizeInstIDs("pod-1,, ,pod-2,", ",")).To(Equal([]string{
			"pod-1",
			"pod-2",
		}))
	})

	It("returns nil for empty input", func() {
		Expect(params.NormalizeInstIDs("", ",")).To(BeNil())
	})

	It("returns nil when all items are empty after trim", func() {
		Expect(params.NormalizeInstIDs(", , ,", ",")).To(BeNil())
	})

	It("works with single item", func() {
		Expect(params.NormalizeInstIDs("pod-1", ",")).To(Equal([]string{"pod-1"}))
	})

	It("works with custom separator", func() {
		Expect(params.NormalizeInstIDs("a;b;c", ";")).To(Equal([]string{"a", "b", "c"}))
	})
})
