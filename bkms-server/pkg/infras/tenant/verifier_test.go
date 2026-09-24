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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/bkuser"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

type fakeUserClient struct {
	getUserFunc func(ctx context.Context, bkUsername string) (*bkuser.User, error)
}

func (f fakeUserClient) GetUser(ctx context.Context, bkUsername string) (*bkuser.User, error) {
	return f.getUserFunc(ctx, bkUsername)
}

var _ = Describe("Tenant verifier", func() {
	It("passes when the user belongs to the requested tenant and is enabled", func() {
		verifier := tenant.NewBKUserVerifier(fakeUserClient{
			getUserFunc: func(ctx context.Context, bkUsername string) (*bkuser.User, error) {
				Expect(bkUsername).To(Equal("alice"))
				return &bkuser.User{
					TenantID:   "tenant-a",
					BkUsername: "alice",
					Status:     "enabled",
				}, nil
			},
		})

		err := verifier.Verify(context.Background(), "alice", "tenant-a")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns tenant access denied when the tenant id does not match", func() {
		verifier := tenant.NewBKUserVerifier(fakeUserClient{
			getUserFunc: func(ctx context.Context, bkUsername string) (*bkuser.User, error) {
				return &bkuser.User{
					TenantID:   "tenant-b",
					BkUsername: "alice",
					Status:     "enabled",
				}, nil
			},
		})

		err := verifier.Verify(context.Background(), "alice", "tenant-a")
		Expect(err).To(MatchError(tenant.ErrTenantAccessDenied))
	})

	It("returns tenant user disabled when the bk-user status is not enabled", func() {
		verifier := tenant.NewBKUserVerifier(fakeUserClient{
			getUserFunc: func(ctx context.Context, bkUsername string) (*bkuser.User, error) {
				return &bkuser.User{
					TenantID:   "tenant-a",
					BkUsername: "alice",
					Status:     "disabled",
				}, nil
			},
		})

		err := verifier.Verify(context.Background(), "alice", "tenant-a")
		Expect(err).To(MatchError(tenant.ErrTenantUserDisabled))
	})

	It("returns wrapped client errors", func() {
		verifier := tenant.NewBKUserVerifier(fakeUserClient{
			getUserFunc: func(ctx context.Context, bkUsername string) (*bkuser.User, error) {
				return nil, errors.New("network timeout")
			},
		})

		err := verifier.Verify(context.Background(), "alice", "tenant-a")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("get bk-user user"))
	})
})
