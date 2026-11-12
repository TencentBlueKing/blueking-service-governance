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

package workspace

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
)

var _ = Describe("BSCP binding", func() {
	var (
		originalCfg *config.Config
		ctx         context.Context
	)

	BeforeEach(func() {
		originalCfg = config.G
		config.G = &config.Config{}
		config.G.Development.UseStubBSCP = true
		ctx = auth.WithUser(context.Background(), auth.User{ID: "admin"})
	})

	AfterEach(func() {
		config.G = originalCfg
	})

	Describe("BindBscpProject", func() {
		It("should return ErrDisabled instead of an empty binding when bscp is disabled", func() {
			config.G.FeatureIntegrations.DisableBSCP = true

			binding, err := BindBscpProject(ctx, "100", "")
			Expect(err).To(MatchError(bscpapi.ErrDisabled))
			Expect(binding).To(BeNil())
		})

		It("should bind the default project when bscp is enabled", func() {
			binding, err := BindBscpProject(ctx, "100", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(binding.ProjectID).To(Equal("1"))
			Expect(binding.ProjectKey).To(Equal("BK-BSCP-00001"))
		})
	})

	Describe("bindDefaultBscpProject", func() {
		It("should skip binding and return an empty binding when bscp is disabled", func() {
			config.G.FeatureIntegrations.DisableBSCP = true

			// ctx 中没有用户，若未跳过会在 auth.MustGetUser 处 panic
			binding, err := bindDefaultBscpProject(context.Background(), "100")
			Expect(err).NotTo(HaveOccurred())
			Expect(*binding).To(Equal(BscpBinding{}))
		})

		It("should bind the default project when bscp is enabled", func() {
			binding, err := bindDefaultBscpProject(ctx, "100")
			Expect(err).NotTo(HaveOccurred())
			Expect(binding.ProjectKey).To(Equal("BK-BSCP-00001"))
		})
	})
})
