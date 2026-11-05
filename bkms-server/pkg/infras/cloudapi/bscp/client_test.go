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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("DisableBSCP", func() {
	var originalCfg *config.Config

	BeforeEach(func() {
		originalCfg = config.G
		config.G = &config.Config{}
	})

	AfterEach(func() {
		config.G = originalCfg
	})

	It("should be enabled by default", func() {
		Expect(Disabled()).To(BeFalse())
	})

	It("should be enabled when config is not loaded", func() {
		config.G = nil
		Expect(Disabled()).To(BeFalse())
	})

	It("should refuse to create clients when disabled", func() {
		config.G.FeatureIntegrations.DisableBSCP = true
		// stub 也不能绕过开关
		config.G.Development.UseStubBSCP = true
		user := auth.User{ID: "admin"}

		client, err := New(user)
		Expect(err).To(MatchError(ErrDisabled))
		Expect(client).To(BeNil())

		configClient, err := NewConfigClient(user)
		Expect(err).To(MatchError(ErrDisabled))
		Expect(configClient).To(BeNil())
	})

	It("should create clients when not disabled", func() {
		config.G.Development.UseStubBSCP = true
		user := auth.User{ID: "admin"}

		client, err := New(user)
		Expect(err).NotTo(HaveOccurred())
		Expect(client).NotTo(BeNil())

		configClient, err := NewConfigClient(user)
		Expect(err).NotTo(HaveOccurred())
		Expect(configClient).NotTo(BeNil())
	})
})
