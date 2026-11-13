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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
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

var _ = Describe("validateConfigKindForApp", func() {
	DescribeTable(
		"allows or rejects config kinds by app",
		func(app *bkmsapp.Application, kind appcfg.ConfigKind, errSubstring string) {
			err := validateConfigKindForApp(app, kind)
			if errSubstring == "" {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(MatchError(ContainSubstring(errSubstring)))
		},
		Entry("plain for trpc", &bkmsapp.Application{Type: appruntime.AppTypeTRPC}, appcfg.ConfigKindPlain, ""),
		Entry("plain for blank",
			&bkmsapp.Application{Type: appruntime.AppTypeBKMSApp, Framework: "blank", Language: "python"},
			appcfg.ConfigKindPlain, ""),
		Entry("plain for helm", &bkmsapp.Application{Type: appruntime.AppTypeHelm}, appcfg.ConfigKindPlain,
			"only supported for appmodel applications"),
		Entry(
			"framework for helm",
			&bkmsapp.Application{Type: appruntime.AppTypeHelm},
			appcfg.ConfigKindFramework,
			"",
		),
		Entry("framework for legacy trpc", &bkmsapp.Application{Type: appruntime.AppTypeTRPC},
			appcfg.ConfigKindFramework, ""),
		Entry("framework for legacy taf", &bkmsapp.Application{Type: appruntime.AppTypeTAF},
			appcfg.ConfigKindFramework, ""),
		Entry("framework for bkmsApp trpc",
			&bkmsapp.Application{Type: appruntime.AppTypeBKMSApp, Framework: "trpc", Language: "go"},
			appcfg.ConfigKindFramework, ""),
		Entry("framework for blank",
			&bkmsapp.Application{Type: appruntime.AppTypeBKMSApp, Framework: "blank", Language: "go"},
			appcfg.ConfigKindFramework, `framework "blank" does not support framework config file`),
	)
})
