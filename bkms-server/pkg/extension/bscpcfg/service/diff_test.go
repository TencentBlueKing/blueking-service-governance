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

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
)

var _ = Describe("DiffScopes", func() {
	Describe("when current and target are identical", func() {
		It("should return nil", func() {
			current := []bscpapi.CredentialScope{
				{ID: 1, App: "svc-a", Scope: "/**"},
				{ID: 2, App: "svc-b", Scope: "/**"},
			}
			target := []bscpapi.CredentialScopeItem{
				{App: "svc-a", Scope: "/**"},
				{App: "svc-b", Scope: "/**"},
			}

			result := service.DiffScopes(current, target)
			Expect(result).To(BeNil())
		})
	})

	Describe("when target has new apps", func() {
		It("should return AddScope with new apps", func() {
			current := []bscpapi.CredentialScope{
				{ID: 1, App: "svc-a", Scope: "/**"},
			}
			target := []bscpapi.CredentialScopeItem{
				{App: "svc-a", Scope: "/**"},
				{App: "svc-b", Scope: "/**"},
				{App: "svc-c", Scope: "/**"},
			}

			result := service.DiffScopes(current, target)
			Expect(result).NotTo(BeNil())
			Expect(result.AddScope).To(HaveLen(2))
			Expect(result.AlterScope).To(BeEmpty())
			Expect(result.DelID).To(BeEmpty())

			addedApps := make([]string, 0, len(result.AddScope))
			for _, s := range result.AddScope {
				addedApps = append(addedApps, s.App)
			}
			Expect(addedApps).To(ContainElements("svc-b", "svc-c"))
		})
	})

	Describe("when current has apps not in target", func() {
		It("should return DelID with removed app IDs", func() {
			current := []bscpapi.CredentialScope{
				{ID: 1, App: "svc-a", Scope: "/**"},
				{ID: 2, App: "svc-b", Scope: "/**"},
				{ID: 3, App: "svc-c", Scope: "/**"},
			}
			target := []bscpapi.CredentialScopeItem{
				{App: "svc-a", Scope: "/**"},
			}

			result := service.DiffScopes(current, target)
			Expect(result).NotTo(BeNil())
			Expect(result.AddScope).To(BeEmpty())
			Expect(result.AlterScope).To(BeEmpty())
			Expect(result.DelID).To(HaveLen(2))
			Expect(result.DelID).To(ContainElements(int64(2), int64(3)))
		})
	})

	Describe("when scope value changed for existing app", func() {
		It("should return AlterScope with updated items", func() {
			current := []bscpapi.CredentialScope{
				{ID: 1, App: "svc-a", Scope: "/old-path/**"},
				{ID: 2, App: "svc-b", Scope: "/**"},
			}
			target := []bscpapi.CredentialScopeItem{
				{App: "svc-a", Scope: "/**"},
				{App: "svc-b", Scope: "/**"},
			}

			result := service.DiffScopes(current, target)
			Expect(result).NotTo(BeNil())
			Expect(result.AddScope).To(BeEmpty())
			Expect(result.AlterScope).To(HaveLen(1))
			Expect(result.DelID).To(BeEmpty())

			Expect(result.AlterScope[0].ID).To(Equal(int64(1)))
			Expect(result.AlterScope[0].App).To(Equal("svc-a"))
			Expect(result.AlterScope[0].Scope).To(Equal("/**"))
		})
	})

	Describe("when mixed operations are needed", func() {
		It("should return add, alter and delete together", func() {
			current := []bscpapi.CredentialScope{
				{ID: 1, App: "svc-keep", Scope: "/**"},
				{ID: 2, App: "svc-alter", Scope: "/old/**"},
				{ID: 3, App: "svc-delete", Scope: "/**"},
			}
			target := []bscpapi.CredentialScopeItem{
				{App: "svc-keep", Scope: "/**"},
				{App: "svc-alter", Scope: "/**"},
				{App: "svc-new", Scope: "/**"},
			}

			result := service.DiffScopes(current, target)
			Expect(result).NotTo(BeNil())

			// 新增 1 个
			Expect(result.AddScope).To(HaveLen(1))
			Expect(result.AddScope[0].App).To(Equal("svc-new"))

			// 更新 1 个
			Expect(result.AlterScope).To(HaveLen(1))
			Expect(result.AlterScope[0].ID).To(Equal(int64(2)))
			Expect(result.AlterScope[0].App).To(Equal("svc-alter"))
			Expect(result.AlterScope[0].Scope).To(Equal("/**"))

			// 删除 1 个
			Expect(result.DelID).To(HaveLen(1))
			Expect(result.DelID[0]).To(Equal(int64(3)))
		})
	})

	Describe("when current is empty", func() {
		It("should add all target items", func() {
			current := []bscpapi.CredentialScope{}
			target := []bscpapi.CredentialScopeItem{
				{App: "svc-a", Scope: "/**"},
				{App: "svc-b", Scope: "/**"},
			}

			result := service.DiffScopes(current, target)
			Expect(result).NotTo(BeNil())
			Expect(result.AddScope).To(HaveLen(2))
			Expect(result.AlterScope).To(BeEmpty())
			Expect(result.DelID).To(BeEmpty())
		})
	})

	Describe("when target is empty", func() {
		It("should delete all current items", func() {
			current := []bscpapi.CredentialScope{
				{ID: 1, App: "svc-a", Scope: "/**"},
				{ID: 2, App: "svc-b", Scope: "/**"},
			}
			target := []bscpapi.CredentialScopeItem{}

			result := service.DiffScopes(current, target)
			Expect(result).NotTo(BeNil())
			Expect(result.AddScope).To(BeEmpty())
			Expect(result.AlterScope).To(BeEmpty())
			Expect(result.DelID).To(HaveLen(2))
		})
	})

	Describe("when both are empty", func() {
		It("should return nil", func() {
			current := []bscpapi.CredentialScope{}
			target := []bscpapi.CredentialScopeItem{}

			result := service.DiffScopes(current, target)
			Expect(result).To(BeNil())
		})
	})
})
