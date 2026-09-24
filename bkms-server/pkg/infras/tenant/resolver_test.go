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

package tenant_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

var _ = Describe("ValidateTenantMode", func() {
	Context("when multi-tenant mode is disabled", func() {
		It("normalizes empty tenant to default", func() {
			tenantID, err := tenant.ValidateTenantMode("", false)
			Expect(err).NotTo(HaveOccurred())
			Expect(tenantID).To(Equal(tenant.DefaultTenantID))
		})

		It("accepts the default tenant", func() {
			tenantID, err := tenant.ValidateTenantMode(tenant.DefaultTenantID, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(tenantID).To(Equal(tenant.DefaultTenantID))
		})

		It("rejects a non-default tenant", func() {
			_, err := tenant.ValidateTenantMode("tenant-a", false)
			Expect(err).To(MatchError(tenant.ErrTenantIDInvalid))
		})
	})

	Context("when multi-tenant mode is enabled", func() {
		It("requires an explicit tenant id", func() {
			_, err := tenant.ValidateTenantMode("", true)
			Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
		})

		It("returns the requested tenant id", func() {
			tenantID, err := tenant.ValidateTenantMode("tenant-a", true)
			Expect(err).NotTo(HaveOccurred())
			Expect(tenantID).To(Equal("tenant-a"))
		})
	})
})
