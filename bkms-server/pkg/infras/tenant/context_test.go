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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

var _ = Describe("Tenant context helpers", func() {
	It("stores and loads tenant id from context", func() {
		ctx := tenant.WithTenantID(context.Background(), "tenant-a")

		tenantID, ok := tenant.GetTenantID(ctx)
		Expect(ok).To(BeTrue())
		Expect(tenantID).To(Equal("tenant-a"))
	})

	It("returns false when tenant id is not present", func() {
		tenantID, ok := tenant.GetTenantID(context.Background())
		Expect(ok).To(BeFalse())
		Expect(tenantID).To(BeEmpty())
	})

	Describe("ScopedKey", func() {
		It("prefixes the key with the tenant in context", func() {
			ctx := tenant.WithTenantID(context.Background(), "tenant-a")

			key, err := tenant.ScopedKey(ctx, "lock:helm-deploy:app:env:")
			Expect(err).NotTo(HaveOccurred())
			Expect(key).To(Equal("tenant:tenant-a:lock:helm-deploy:app:env:"))
		})

		It("gives different keys for the same resource in different tenants", func() {
			keyA, err := tenant.ScopedKey(tenant.WithTenantID(context.Background(), "tenant-a"), "lock:x")
			Expect(err).NotTo(HaveOccurred())
			keyB, err := tenant.ScopedKey(tenant.WithTenantID(context.Background(), "tenant-b"), "lock:x")
			Expect(err).NotTo(HaveOccurred())
			Expect(keyA).NotTo(Equal(keyB))
		})

		It("returns ErrTenantIDRequired when context has no tenant", func() {
			_, err := tenant.ScopedKey(context.Background(), "lock:x")
			Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
		})
	})

	Describe("TenantHeader", func() {
		It("returns the tenant header from context", func() {
			ctx := tenant.WithTenantID(context.Background(), "tenant-a")

			Expect(tenant.TenantHeader(ctx)).To(Equal(map[string]string{"X-Bk-Tenant-Id": "tenant-a"}))
		})

		It("returns nil when context has no tenant", func() {
			Expect(tenant.TenantHeader(context.Background())).To(BeNil())
		})
	})
})
