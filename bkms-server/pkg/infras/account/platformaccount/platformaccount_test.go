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

package platformaccount

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("platform account", func() {
	var originCfg *config.Config
	user := auth.User{ID: "alice", TenantID: "default", Cred: auth.UserCredential{BkTicket: "ticket"}}
	userCtx := auth.WithUser(context.Background(), user)

	BeforeEach(func() {
		originCfg = config.G
		config.G = &config.Config{}
		DeferCleanup(func() { config.G = originCfg })
	})

	It("uses the current user when the account is not enabled", func() {
		// 填了账号但没有开启开关，仍视为未启用
		config.G.PlatformAccount = config.PlatformAccountConfig{Username: "bkms-platform", AccessToken: "token"}

		_, ok := Get()
		Expect(ok).To(BeFalse())
		caller, err := GetCaller(userCtx)
		Expect(err).NotTo(HaveOccurred())
		Expect(caller).To(Equal(user))
		Expect(MustGetCaller(userCtx)).To(Equal(user))
	})

	It("fails without a current user when the account is not enabled", func() {
		_, err := GetCaller(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(func() { MustGetCaller(context.Background()) }).To(Panic())
	})

	It("uses the account regardless of the current user when enabled", func() {
		config.G.PlatformAccount = config.PlatformAccountConfig{
			Enabled: true, Username: "bkms-platform", AccessToken: "token",
		}
		account := auth.User{ID: "bkms-platform", Cred: auth.UserCredential{AccessToken: "token"}}

		got, ok := Get()
		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(account))
		Expect(MustGetCaller(userCtx)).To(Equal(account))

		caller, err := GetCaller(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(caller).To(Equal(account))
	})
})
