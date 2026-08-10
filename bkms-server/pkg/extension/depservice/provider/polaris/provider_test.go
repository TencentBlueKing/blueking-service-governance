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

package polaris

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	"github.com/h2non/gock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

var _ = Describe("Test polaris provider", func() {
	var planConfig map[string]any
	var p *Provider
	var ctx context.Context
	var testURL string

	BeforeEach(func() {
		testURL = "http://foo.example.com:8080"
		ctx = context.Background()
		planConfig = map[string]any{
			"baseUrl": testURL,
		}
		p, _ = NewProvider(planConfig)
		// 拦截 provider 的自定义 HTTP client，使 gock 能够 mock 请求
		gock.InterceptClient(p.httpCli)
	})

	AfterEach(func() {
		gock.RestoreClient(p.httpCli)
		gock.Off()
	})

	It("test create instance", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			token := stringx.Random(12)
			gock.New(testURL).
				Post("/naming/v1/services").
				Reply(200).
				JSON(map[string]any{"responses": []map[string]any{{"service": map[string]any{"token": token}}}})

			result, err := p.CreateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				&CreateParams{
					PolarisName:      "test-service",
					PolarisNamespace: "test-namespace",
					Owners:           "test-user1,test-user2",
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.InstConfig["token"]).To(Equal(token))
			Expect(result.Credentials["token"]).To(Equal(token))
		})
	})

	It("test delete instance", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			gock.New(testURL).
				Post("/naming/v1/services/delete").
				Reply(200).
				JSON(map[string]any{})

			_, err := p.DeleteInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("test delete instance with incomplete config is no-op", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			result, err := p.DeleteInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Async).To(BeFalse())
		})
	})

	It("test create error from polaris api", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			gock.New(testURL).
				Post("/naming/v1/services").
				Reply(500).
				JSON(map[string]any{"info": "internal server error"})

			_, err := p.CreateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				&CreateParams{
					PolarisName:      "test-service",
					PolarisNamespace: "test-namespace",
					Owners:           "test-user",
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("internal server error"))
		})
	})

	It("test create missing required params", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			_, err := p.CreateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				&CreateParams{},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PolarisName"))
		})
	})

	It("test new provider missing baseUrl", func() {
		_, err := NewProvider(map[string]any{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("baseUrl is required"))
	})
})
