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

package appcfg_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("GetEnvContent", func() {
	var diApp *fxtest.App
	var ctx context.Context
	var appStore bkmsapp.ApplicationStore
	var store appcfg.AppConfigFileStore
	var app *bkmsapp.Application

	BeforeEach(func() {
		var err error

		ctx = context.Background()
		err = testutil.CleanupCollection("app_config_files")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("applications")
		Expect(err).NotTo(HaveOccurred())

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			fx.Populate(&appStore, &store),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Context("when only app-level default config exists", func() {
		BeforeEach(func() {
			defaultContent := "server:\n  address: 0.0.0.0:8080\n"
			dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:   app.ID,
				EnvName: appcfg.EnvNameDefault,
				Name:    appcfg.DefaultAppConfigFileName,
				Content: &defaultContent,
			})
		})

		It("should return default config when querying non-existent env", func() {
			acf, content, err := appcfg.GetEnvContent(ctx, store, app.ID, "test-env")
			Expect(err).NotTo(HaveOccurred())
			Expect(acf.Name).To(Equal(appcfg.DefaultAppConfigFileName))
			Expect(content).To(Equal("server:\n  address: 0.0.0.0:8080\n"))
		})
	})

	Context("when env-specific config exists", func() {
		BeforeEach(func() {
			defaultContent := "server:\n  address: 0.0.0.0:8080\n"
			dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:   app.ID,
				EnvName: appcfg.EnvNameDefault,
				Name:    appcfg.DefaultAppConfigFileName,
				Content: &defaultContent,
			})

			prodContent := "server:\n  address: 0.0.0.0:9090\n"
			dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:   app.ID,
				EnvName: "prod",
				Name:    "prod",
				Content: &prodContent,
			})
		})

		It("should return env-specific config when querying that env", func() {
			acf, content, err := appcfg.GetEnvContent(ctx, store, app.ID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(acf.Name).To(Equal("prod"))
			Expect(content).To(Equal("server:\n  address: 0.0.0.0:9090\n"))
		})

		It("should return default config when querying non-existent env", func() {
			acf, content, err := appcfg.GetEnvContent(ctx, store, app.ID, "test-env")
			Expect(err).NotTo(HaveOccurred())
			Expect(acf.Name).To(Equal(appcfg.DefaultAppConfigFileName))
			Expect(content).To(Equal("server:\n  address: 0.0.0.0:8080\n"))
		})
	})

	Context("when the env-specific config is an overlay", func() {
		It("should return the overlay logical file and compiled content", func() {
			baseContent := "database:\n  host: ${{ env.BASE_HOST }}\n  port: 3306\n"
			base := dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:   app.ID,
				EnvName: appcfg.EnvNameDefault,
				Name:    appcfg.DefaultAppConfigFileName,
				Content: &baseContent,
			})

			overlayContent := "overlayVersion: \"2\"\npatches:\n- database:\n    host: ${{ env.OVERLAY_HOST }}\n"
			dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:               app.ID,
				EnvName:             "prod",
				Name:                "prod-overlay",
				Type:                appcfg.AppConfigFileTypeOverlay,
				BaseAppConfigFileID: &base.ID,
				OverlayContent:      &overlayContent,
			})

			acf, content, err := appcfg.GetEnvContent(ctx, store, app.ID, "prod")

			Expect(err).NotTo(HaveOccurred())
			Expect(acf.Name).To(Equal("prod-overlay"))
			Expect(content).To(Equal("database:\n  host: ${{ env.OVERLAY_HOST }}\n  port: 3306\n"))
		})
	})
})
