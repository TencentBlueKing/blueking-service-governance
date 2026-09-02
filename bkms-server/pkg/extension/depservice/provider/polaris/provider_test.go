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

	It("test update instance owners", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			gock.New(testURL).
				Put("/naming/v1/services").
				JSON([]map[string]any{{
					"name":      "test-service",
					"namespace": "test-namespace",
					"token":     "test-token",
					"owners":    "lisi,wangwu",
				}}).
				Reply(200).
				JSON(map[string]any{})

			err := p.UpdateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
				&UpdateParams{Owners: "lisi,wangwu"},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("test update instance polaris api error", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			gock.New(testURL).
				Put("/naming/v1/services").
				Reply(500).
				JSON(map[string]any{"info": "invalid owners"})

			err := p.UpdateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
				&UpdateParams{Owners: "bad"},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid owners"))
		})
	})

	It("test update instance missing required params", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			err := p.UpdateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
				&UpdateParams{},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("owners or metadata is required"))
		})
	})

	It("test create instance with metadata", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			token := stringx.Random(12)
			gock.New(testURL).
				Post("/naming/v1/services").
				JSON([]map[string]any{{
					"name":      "test-service",
					"namespace": "test-namespace",
					"owners":    "test-user",
					"metadata": map[string]string{
						"internal-enable-dynamic-weight": "true",
					},
				}}).
				Reply(200).
				JSON(map[string]any{"responses": []map[string]any{{"service": map[string]any{"token": token}}}})

			result, err := p.CreateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				&CreateParams{
					PolarisName:      "test-service",
					PolarisNamespace: "test-namespace",
					Owners:           "test-user",
					Metadata:         map[string]string{"internal-enable-dynamic-weight": "true"},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.InstConfig["token"]).To(Equal(token))
		})
	})

	It("test update instance merges metadata", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			gock.New(testURL).
				Get("/naming/v1/services").
				MatchParam("name", "test-service").
				MatchParam("namespace", "test-namespace").
				Reply(200).
				JSON(map[string]any{
					"services": []map[string]any{{
						"name":      "test-service",
						"namespace": "test-namespace",
						"metadata":  map[string]string{"keep": "yes", "old": "1"},
					}},
				})
			gock.New(testURL).
				Put("/naming/v1/services").
				JSON([]map[string]any{{
					"name":      "test-service",
					"namespace": "test-namespace",
					"token":     "test-token",
					"owners":    "lisi",
					"metadata":  map[string]string{"keep": "yes", "new": "2"},
				}}).
				Reply(200).
				JSON(map[string]any{})

			err := p.UpdateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
				&UpdateParams{
					Owners:               "lisi",
					Metadata:             map[string]string{"new": "2"},
					MetadataKeysToDelete: []string{"old"},
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("test update instance omits owners when not provided", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			gock.New(testURL).
				Get("/naming/v1/services").
				MatchParam("name", "test-service").
				MatchParam("namespace", "test-namespace").
				Reply(200).
				JSON(map[string]any{
					"services": []map[string]any{{
						"name":      "test-service",
						"namespace": "test-namespace",
						"metadata":  map[string]string{},
					}},
				})
			gock.New(testURL).
				Put("/naming/v1/services").
				JSON([]map[string]any{{
					"name":      "test-service",
					"namespace": "test-namespace",
					"token":     "test-token",
					"metadata":  map[string]string{"k": "v"},
				}}).
				Reply(200).
				JSON(map[string]any{})

			err := p.UpdateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
				&UpdateParams{Metadata: map[string]string{"k": "v"}},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("test update instance deletes metadata keys leaving empty map", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			gock.New(testURL).
				Get("/naming/v1/services").
				MatchParam("name", "test-service").
				MatchParam("namespace", "test-namespace").
				Reply(200).
				JSON(map[string]any{
					"services": []map[string]any{{
						"name":      "test-service",
						"namespace": "test-namespace",
						"metadata": map[string]string{
							"internal-enable-dynamic-weight": "true",
						},
					}},
				})
			gock.New(testURL).
				Put("/naming/v1/services").
				JSON([]map[string]any{{
					"name":      "test-service",
					"namespace": "test-namespace",
					"token":     "test-token",
					"owners":    "lisi",
					"metadata":  map[string]string{},
				}}).
				Reply(200).
				JSON(map[string]any{})

			err := p.UpdateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
				&UpdateParams{
					Owners:               "lisi",
					MetadataKeysToDelete: []string{"internal-enable-dynamic-weight"},
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("test update instance get metadata failure", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			defer gock.Off()

			gock.New(testURL).
				Get("/naming/v1/services").
				Reply(500).
				JSON(map[string]any{"info": "query failed"})

			err := p.UpdateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
				&UpdateParams{
					Owners:   "lisi",
					Metadata: map[string]string{"k": "v"},
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("query failed"))
		})
	})

	It("test mergeServiceMetadata overlays then deletes", func() {
		merged := mergeServiceMetadata(
			map[string]string{"keep": "yes", "old": "1"},
			map[string]string{"new": "2"},
			[]string{"old"},
		)
		Expect(merged).To(Equal(map[string]string{"keep": "yes", "new": "2"}))
	})

	It("test update instance invalid params type", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			err := p.UpdateInstance(
				ctx,
				"test-inst-id",
				&types.ServicePlanConfig{Config: planConfig},
				map[string]any{
					"polarisName":      "test-service",
					"polarisNamespace": "test-namespace",
					"token":            "test-token",
				},
				&CreateParams{},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("*polaris.UpdateParams"))
		})
	})
})
