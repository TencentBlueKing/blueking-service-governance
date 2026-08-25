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

package resources

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = Describe("Resources Spec", func() {
	Describe("Merge", func() {
		It("returns nil when both are nil", func() {
			Expect(Merge(nil, nil)).To(BeNil())
		})

		It("returns a clone of override when base is nil", func() {
			override := &Spec{MemoryRequests: lo.ToPtr("256Mi")}
			Expect(Merge(nil, override)).To(Equal(override))
		})

		It("returns a clone of base when override is nil", func() {
			base := &Spec{MemoryLimits: lo.ToPtr("1Gi")}
			Expect(Merge(base, nil)).To(Equal(base))
		})

		It("overlays only non-nil override fields", func() {
			base := &Spec{
				Replicas:       lo.ToPtr(int32(1)),
				CPURequests:    lo.ToPtr("1"),
				CPULimits:      lo.ToPtr("2"),
				MemoryRequests: lo.ToPtr("2Gi"),
				MemoryLimits:   lo.ToPtr("4Gi"),
			}
			override := &Spec{
				Replicas:    lo.ToPtr(int32(3)),
				CPURequests: lo.ToPtr("500m"),
			}

			Expect(Merge(base, override)).To(Equal(&Spec{
				Replicas:       lo.ToPtr(int32(3)),
				CPURequests:    lo.ToPtr("500m"),
				CPULimits:      lo.ToPtr("2"),
				MemoryRequests: lo.ToPtr("2Gi"),
				MemoryLimits:   lo.ToPtr("4Gi"),
			}))
		})

		It("keeps base memory limits when override only sets memory requests", func() {
			base := &Spec{
				MemoryRequests: lo.ToPtr("256Mi"),
				MemoryLimits:   lo.ToPtr("1Gi"),
			}
			override := &Spec{MemoryRequests: lo.ToPtr("256Mi")}

			Expect(Merge(base, override)).To(Equal(&Spec{
				MemoryRequests: lo.ToPtr("256Mi"),
				MemoryLimits:   lo.ToPtr("1Gi"),
			}))
		})

		It("keeps base cpu limits when override only sets cpu requests", func() {
			base := &Spec{
				CPURequests: lo.ToPtr("0.1"),
				CPULimits:   lo.ToPtr("1"),
			}
			override := &Spec{CPURequests: lo.ToPtr("0.5")}

			Expect(Merge(base, override)).To(Equal(&Spec{
				CPURequests: lo.ToPtr("0.5"),
				CPULimits:   lo.ToPtr("1"),
			}))
		})

		It("keeps base requests when override only sets limits", func() {
			base := &Spec{
				MemoryRequests: lo.ToPtr("2Gi"),
				MemoryLimits:   lo.ToPtr("4Gi"),
			}
			override := &Spec{MemoryLimits: lo.ToPtr("8Gi")}

			Expect(Merge(base, override)).To(Equal(&Spec{
				MemoryRequests: lo.ToPtr("2Gi"),
				MemoryLimits:   lo.ToPtr("8Gi"),
			}))
		})
	})
})
