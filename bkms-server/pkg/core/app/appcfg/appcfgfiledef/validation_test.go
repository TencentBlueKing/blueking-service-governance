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

package appcfgfiledef

import (
	"context"
	"errors"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	appcfgapierrs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/apierrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
)

var _ = Describe("validateBSCPConfig when bscp is disabled", func() {
	var (
		originalCfg *config.Config
		ctx         context.Context
		h           *Handler
	)

	BeforeEach(func() {
		originalCfg = config.G
		config.G = &config.Config{}
		config.G.FeatureIntegrations.DisableBSCP = true
		ctx = auth.WithUser(context.Background(), auth.User{ID: "admin"})
		h = &Handler{}
	})

	AfterEach(func() {
		config.G = originalCfg
	})

	It("should return ErrDisabled for bscp source", func() {
		input := &BSCPConfigInput{BizID: "100", ServiceID: "1", ID: "1"}

		cfg, err := h.validateBSCPConfig(ctx, appcfg.ContentSourceTypeBSCP, input, appcfg.FileFormatYAML)
		Expect(err).To(MatchError(bscp.ErrDisabled))
		Expect(cfg).To(BeNil())
	})

	It("should not affect local source", func() {
		cfg, err := h.validateBSCPConfig(ctx, appcfg.ContentSourceTypeLocal, nil, appcfg.FileFormatYAML)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).To(BeNil())
	})
})

var _ = Describe("validateBSCPConfig", func() {
	var (
		ctx context.Context
		h   *Handler
	)
	input := &BSCPConfigInput{BizID: "123", ServiceID: "456", ID: "666"}

	BeforeEach(func() {
		ctx = auth.WithUser(context.Background(), auth.User{ID: "admin"})
		h = &Handler{}
	})

	mockBSCP := func(versions bscp.Versions, content string) {
		mockey.Mock(bscp.New).Return(&bscp.ApiClient{}, nil).Build()
		mockey.Mock((*bscp.ApiClient).ListServiceVersions).Return(versions, nil).Build()
		mockey.Mock((*bscp.ApiClient).GetServiceConfig).Return(bscp.NewKeyValue("stag", content, ""), nil).Build()
	}

	// apiCode 返回 BSCPConfigRules 映射后的错误码，不命中时为空
	apiCode := func(err error) bkerrs.ErrCode {
		var bkErr *bkerrs.Error
		if errors.As(bkerrs.Match(err, appcfgapierrs.BSCPConfigRules), &bkErr) {
			return bkErr.Code()
		}
		return ""
	}

	It("should require the bscp config reference", func() {
		_, err := h.validateBSCPConfig(ctx, appcfg.ContentSourceTypeBSCP, nil, appcfg.FileFormatYAML)
		Expect(err).To(MatchError(appcfg.ErrBSCPConfigRequired))
		Expect(apiCode(err)).To(Equal(bkerrs.ErrCodeInvalidArgument))
	})

	It("should record the latest fully released version", func() {
		mockey.PatchConvey("released", GinkgoT(), func() {
			mockBSCP(bscp.Versions{{ID: "91011", IsFullyReleased: true}}, "foo: bar\n")

			cfg, err := h.validateBSCPConfig(ctx, appcfg.ContentSourceTypeBSCP, input, appcfg.FileFormatYAML)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).To(Equal(&appcfg.BSCPConfig{
				BizID: "123", ServiceID: "456", VersionID: "91011", ConfigID: "666",
			}))
		})
	})

	It("should reject the service without fully released version", func() {
		mockey.PatchConvey("not released", GinkgoT(), func() {
			mockBSCP(bscp.Versions{{ID: "91011", IsFullyReleased: false}}, "")

			_, err := h.validateBSCPConfig(ctx, appcfg.ContentSourceTypeBSCP, input, appcfg.FileFormatYAML)
			Expect(err).To(MatchError(appcfg.ErrBSCPNoFullyReleasedVersion))
			Expect(apiCode(err)).To(Equal(bkerrs.ErrCodeInvalidRequest))
		})
	})

	It("should validate content only for yaml files", func() {
		mockey.PatchConvey("invalid yaml", GinkgoT(), func() {
			mockBSCP(bscp.Versions{{ID: "91011", IsFullyReleased: true}}, "foo: [bar")

			_, err := h.validateBSCPConfig(ctx, appcfg.ContentSourceTypeBSCP, input, appcfg.FileFormatYAML)
			Expect(err).To(MatchError(appcfg.ErrBSCPContentInvalidYAML))
			Expect(apiCode(err)).To(Equal(bkerrs.ErrCodeInvalidArgument))

			_, err = h.validateBSCPConfig(ctx, appcfg.ContentSourceTypeBSCP, input, appcfg.FileFormat("ini"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should leave bscp call failures unmatched", func() {
		mockey.PatchConvey("call failed", GinkgoT(), func() {
			mockey.Mock(bscp.New).Return(&bscp.ApiClient{}, nil).Build()
			mockey.Mock((*bscp.ApiClient).ListServiceVersions).Return(nil, errors.New("connection refused")).Build()

			_, err := h.validateBSCPConfig(ctx, appcfg.ContentSourceTypeBSCP, input, appcfg.FileFormatYAML)
			Expect(err).To(HaveOccurred())
			Expect(apiCode(err)).To(BeEmpty())
		})
	})
})

var _ = Describe("validateConfigKindForApp", func() {
	DescribeTable("allows or rejects config kinds by app",
		func(app *bkmsapp.Application, kind appcfg.ConfigKind, errSubstring string) {
			err := validateConfigKindForApp(app, kind)
			if errSubstring == "" {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(MatchError(ContainSubstring(errSubstring)))
		},
		Entry("plain for trpc", &bkmsapp.Application{Type: bkmsapp.AppTypeTRPC}, appcfg.ConfigKindPlain, ""),
		Entry("plain for blank",
			&bkmsapp.Application{Type: bkmsapp.AppTypeBKMSApp, Framework: "blank", Language: "python"},
			appcfg.ConfigKindPlain, ""),
		Entry("plain for helm", &bkmsapp.Application{Type: bkmsapp.AppTypeHelm}, appcfg.ConfigKindPlain,
			"only supported for appmodel applications"),
		Entry("framework for helm", &bkmsapp.Application{Type: bkmsapp.AppTypeHelm}, appcfg.ConfigKindFramework, ""),
		Entry("framework for legacy trpc", &bkmsapp.Application{Type: bkmsapp.AppTypeTRPC},
			appcfg.ConfigKindFramework, ""),
		Entry("framework for legacy taf", &bkmsapp.Application{Type: bkmsapp.AppTypeTAF},
			appcfg.ConfigKindFramework, ""),
		Entry("framework for bkmsApp trpc",
			&bkmsapp.Application{Type: bkmsapp.AppTypeBKMSApp, Framework: "trpc", Language: "go"},
			appcfg.ConfigKindFramework, ""),
		Entry("framework for blank",
			&bkmsapp.Application{Type: bkmsapp.AppTypeBKMSApp, Framework: "blank", Language: "go"},
			appcfg.ConfigKindFramework, `framework "blank" does not support framework config file`),
	)
})
