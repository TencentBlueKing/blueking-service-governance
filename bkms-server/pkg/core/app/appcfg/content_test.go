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

	Context("when framework and plain env-specific config coexist", func() {
		BeforeEach(func() {
			defaultContent := "server:\n  address: 0.0.0.0:8080\n"
			dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:      app.ID,
				EnvName:    appcfg.EnvNameDefault,
				Name:       appcfg.DefaultAppConfigFileName,
				ConfigKind: appcfg.ConfigKindFramework,
				Content:    &defaultContent,
			})

			frameworkProdContent := "server:\n  address: 0.0.0.0:9090\n"
			dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:      app.ID,
				EnvName:    "prod",
				Name:       "prod-framework",
				ConfigKind: appcfg.ConfigKindFramework,
				Content:    &frameworkProdContent,
			})

			plainDefaultContent := "benchmark.enabled=true\n"
			plainRoot := dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:           app.ID,
				EnvName:         appcfg.EnvNameDefault,
				Name:            "custom-env",
				ConfigKind:      appcfg.ConfigKindPlain,
				IsUnifiedConfig: false,
				MountedEnvNames: []string{"prod"},
				MountPath:       "/data/app/conf/custom.env",
				Format:          appcfg.FileFormat("properties"),
				Content:         &plainDefaultContent,
			})
			plainProdContent := "benchmark.enabled=false\n"
			dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
				AppID:                  app.ID,
				EnvName:                "prod",
				Name:                   "custom-env--prod",
				ConfigKind:             appcfg.ConfigKindPlain,
				MountPath:              "/data/app/conf/custom.env",
				DefaultAppConfigFileID: &plainRoot.ID,
				Format:                 appcfg.FileFormat("properties"),
				Content:                &plainProdContent,
			})
		})

		It("should still return the framework env-specific config", func() {
			acf, content, err := appcfg.GetEnvContent(ctx, store, app.ID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(acf.Name).To(Equal("prod-framework"))
			Expect(content).To(Equal("server:\n  address: 0.0.0.0:9090\n"))
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

var _ = Describe("ListEnvPlainContents", func() {
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

	It("should return default plain content when env mode is disabled", func() {
		defaultContent := "feature.enabled=true\n"
		dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
			AppID:           app.ID,
			EnvName:         appcfg.EnvNameDefault,
			Name:            "feature-flags",
			ConfigKind:      appcfg.ConfigKindPlain,
			MountPath:       "/data/app/conf/feature-flags.properties",
			Format:          appcfg.FileFormat("properties"),
			IsUnifiedConfig: true,
			Content:         &defaultContent,
		})

		files, err := appcfg.ListEnvPlainContents(ctx, store, app.ID, "stag")
		Expect(err).NotTo(HaveOccurred())
		Expect(files).To(HaveLen(1))
		Expect(files[0].File.EnvName).To(Equal(appcfg.EnvNameDefault))
		Expect(files[0].File.Name).To(Equal("feature-flags"))
		Expect(files[0].File.MountPath).To(Equal("/data/app/conf/feature-flags.properties"))
		Expect(files[0].Content).To(Equal("feature.enabled=true\n"))
	})

	It("should return env-specific plain content when env mode is enabled", func() {
		defaultContent := "feature.enabled=true\n"
		root := dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
			AppID:           app.ID,
			EnvName:         appcfg.EnvNameDefault,
			Name:            "feature-flags",
			ConfigKind:      appcfg.ConfigKindPlain,
			IsUnifiedConfig: false,
			MountedEnvNames: []string{"prod"},
			MountPath:       "/data/app/conf/feature-flags.properties",
			Format:          appcfg.FileFormat("properties"),
			Content:         &defaultContent,
		})
		prodContent := "feature.enabled=false\n"
		dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
			AppID:                  app.ID,
			EnvName:                "prod",
			Name:                   "feature-flags--prod",
			ConfigKind:             appcfg.ConfigKindPlain,
			DefaultAppConfigFileID: &root.ID,
			MountPath:              "/data/app/conf/feature-flags.properties",
			Format:                 appcfg.FileFormat("properties"),
			Content:                &prodContent,
		})

		files, err := appcfg.ListEnvPlainContents(ctx, store, app.ID, "prod")
		Expect(err).NotTo(HaveOccurred())
		Expect(files).To(HaveLen(1))
		Expect(files[0].File.EnvName).To(Equal("prod"))
		Expect(files[0].File.Name).To(Equal("feature-flags--prod"))
		Expect(files[0].Content).To(Equal("feature.enabled=false\n"))
	})

	It("should skip unified plain files that are not mounted to the target env", func() {
		defaultContent := "feature.enabled=true\n"
		dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
			AppID:           app.ID,
			EnvName:         appcfg.EnvNameDefault,
			Name:            "feature-flags-unified",
			ConfigKind:      appcfg.ConfigKindPlain,
			IsUnifiedConfig: true,
			MountedEnvNames: []string{"prod"},
			MountPath:       "/data/app/conf/feature-flags-unified.properties",
			Format:          appcfg.FileFormat("properties"),
			Content:         &defaultContent,
		})

		stagFiles, err := appcfg.ListEnvPlainContents(ctx, store, app.ID, "stag")
		Expect(err).NotTo(HaveOccurred())
		Expect(stagFiles).To(BeEmpty())

		prodFiles, err := appcfg.ListEnvPlainContents(ctx, store, app.ID, "prod")
		Expect(err).NotTo(HaveOccurred())
		Expect(prodFiles).To(HaveLen(1))
		Expect(prodFiles[0].File.Name).To(Equal("feature-flags-unified"))
	})

	It("should skip plain files that are not mounted to the target env", func() {
		defaultContent := "feature.enabled=true\n"
		dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
			AppID:           app.ID,
			EnvName:         appcfg.EnvNameDefault,
			Name:            "feature-flags",
			ConfigKind:      appcfg.ConfigKindPlain,
			IsUnifiedConfig: false,
			MountedEnvNames: []string{"prod"},
			MountPath:       "/data/app/conf/feature-flags.properties",
			Format:          appcfg.FileFormat("properties"),
			Content:         &defaultContent,
		})

		files, err := appcfg.ListEnvPlainContents(ctx, store, app.ID, "stag")
		Expect(err).NotTo(HaveOccurred())
		Expect(files).To(BeEmpty())
	})

	It("should fallback to default content for referenced env (no independent instance)", func() {
		defaultContent := "feature.enabled=true\n"
		dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
			AppID:           app.ID,
			EnvName:         appcfg.EnvNameDefault,
			Name:            "feature-flags",
			ConfigKind:      appcfg.ConfigKindPlain,
			IsUnifiedConfig: false,
			MountedEnvNames: []string{"prod", "stag"},
			MountPath:       "/data/app/conf/feature-flags.properties",
			Format:          appcfg.FileFormat("properties"),
			Content:         &defaultContent,
		})

		files, err := appcfg.ListEnvPlainContents(ctx, store, app.ID, "prod")
		Expect(err).NotTo(HaveOccurred())
		Expect(files).To(HaveLen(1))
		Expect(files[0].File.EnvName).To(Equal(appcfg.EnvNameDefault))
		Expect(files[0].Content).To(Equal("feature.enabled=true\n"))
	})

	It("should fallback to default content for all-env mode without instance", func() {
		defaultContent := "feature.enabled=true\n"
		dbfactory.AppConfigFile(ctx, store, &dbfactory.AppConfigFileOpts{
			AppID:           app.ID,
			EnvName:         appcfg.EnvNameDefault,
			Name:            "feature-flags",
			ConfigKind:      appcfg.ConfigKindPlain,
			IsUnifiedConfig: false,
			MountPath:       "/data/app/conf/feature-flags.properties",
			Format:          appcfg.FileFormat("properties"),
			Content:         &defaultContent,
		})

		files, err := appcfg.ListEnvPlainContents(ctx, store, app.ID, "any-env")
		Expect(err).NotTo(HaveOccurred())
		Expect(files).To(HaveLen(1))
		Expect(files[0].File.EnvName).To(Equal(appcfg.EnvNameDefault))
		Expect(files[0].Content).To(Equal("feature.enabled=true\n"))
	})
})
