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

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
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
})

var _ = Describe("IsEnabled", func() {
	It("returns false when global config is nil", func() {
		prev := svccfg.G
		svccfg.G = nil
		DeferCleanup(func() { svccfg.G = prev })

		Expect(tenant.IsEnabled()).To(BeFalse())
	})

	It("follows tenant.enableMultiTenantMode", func() {
		prev := svccfg.G
		svccfg.G = &svccfg.Config{Tenant: svccfg.TenantConfig{EnableMultiTenantMode: true}}
		DeferCleanup(func() { svccfg.G = prev })

		Expect(tenant.IsEnabled()).To(BeTrue())
	})
})
