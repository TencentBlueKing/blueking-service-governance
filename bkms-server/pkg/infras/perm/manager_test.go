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

package perm

import (
	"sync"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

var _ = Describe("NewManager", func() {
	var originalConfig *config.Config

	BeforeEach(func() {
		originalConfig = config.G
		config.G = &config.Config{}
		managerOnce = sync.Once{}
		manager = nil
		managerErr = nil
	})

	AfterEach(func() {
		config.G = originalConfig
		managerOnce = sync.Once{}
		manager = nil
		managerErr = nil
	})

	It("returns and reuses the stub manager when UseStubPerm is enabled", func() {
		config.G.Development.UseStubPerm = true

		first := NewManager()
		second := NewManager()

		Expect(first).To(BeAssignableToTypeOf(&StubAllowAnyManager{}))
		Expect(second).To(BeIdenticalTo(first))
	})

	It("builds the local manager by default when UseStubPerm is disabled", func() {
		config.G.Development.UseStubPerm = false
		fakeSvc := &bkiam.IAMService{}

		mockey.PatchConvey("default local manager", GinkgoT(), func() {
			mockey.Mock(newIAMService).Return(fakeSvc, nil).Build()

			got, err := buildManager()

			Expect(err).NotTo(HaveOccurred())
			localMgr, ok := got.(*LocalManager)
			Expect(ok).To(BeTrue())
			Expect(localMgr.svc).To(BeIdenticalTo(fakeSvc))
		})
	})

	It("wraps IAMService construction errors from the default local path", func() {
		config.G.Development.UseStubPerm = false
		upstreamErr := errors.New("iam service unavailable")

		mockey.PatchConvey("default local manager error", GinkgoT(), func() {
			mockey.Mock(newIAMService).Return(nil, upstreamErr).Build()

			got, err := buildManager()

			Expect(got).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("build iam service"))
			Expect(err.Error()).To(ContainSubstring("iam service unavailable"))
		})
	})
})
