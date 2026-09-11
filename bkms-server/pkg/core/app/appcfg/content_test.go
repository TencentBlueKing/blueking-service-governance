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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("GetFrameworkMountableFile", func() {
	var diApp *fxtest.App
	var ctx context.Context
	var appStore bkmsapp.ApplicationStore
	var store appcfg.AppConfigFileStore
	var defStore appcfg.AppConfigFileDefStore
	var versionStore appcfg.AppConfigFileVersionStore
	var app *bkmsapp.Application
	var provider appcfg.MountableFileProvider
	var svc *appcfg.AppCfgFileDefService

	BeforeEach(func() {
		var err error

		ctx = context.Background()
		err = testutil.CleanupCollection("app_config_file_defs")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("app_config_files")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("applications")
		Expect(err).NotTo(HaveOccurred())

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			fx.Populate(&appStore, &store, &defStore, &versionStore),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		provider = appcfg.NewMountableFileProvider(store, defStore, versionStore)
		base := appcfg.NewBaseAppCfgFileService(defStore, store, versionStore)
		svc = appcfg.NewAppCfgFileDefService(base)
	})

	createDef := func(name string) bson.ObjectID {
		id, err := defStore.Add(ctx, appcfg.AppConfigFileDef{
			AppID:      app.ID,
			Name:       name,
			ConfigKind: appcfg.ConfigKindFramework,
			Creator:    "tester",
		})
		Expect(err).NotTo(HaveOccurred())
		return id
	}

	createFrameworkFile := func(name, content string) *appcfg.AppConfigFileWithDef {
		result, err := svc.Create(ctx, appcfg.CreateCfgFileParams{
			AppID:             app.ID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              name,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatYAML,
			Content:           &content,
			Creator:           "tester",
			Description:       "init",
			ConfigKind:        appcfg.ConfigKindFramework,
		})
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	AfterEach(func() {
		diApp.RequireStop()
	})

	Context("when only app-level default config exists", func() {
		BeforeEach(func() {
			defID := createDef(appcfg.DefaultAppConfigFileName)
			defaultContent := "server:\n  address: 0.0.0.0:8080\n"
			_, err := store.Add(ctx, appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   app.ID,
				EnvName: appcfg.EnvNameDefault,
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &defaultContent,
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fall back to default config when querying non-existent env", func() {
			cfwc, err := provider.GetFrameworkMountableFile(ctx, app.ID, "test-env")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfwc.Content).To(Equal("server:\n  address: 0.0.0.0:8080\n"))
		})
	})

	Context("when env-specific config exists", func() {
		BeforeEach(func() {
			defID := createDef(appcfg.DefaultAppConfigFileName)
			defaultContent := "server:\n  address: 0.0.0.0:8080\n"
			_, err := store.Add(ctx, appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   app.ID,
				EnvName: appcfg.EnvNameDefault,
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &defaultContent,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			prodContent := "server:\n  address: 0.0.0.0:9090\n"
			_, err = store.Add(ctx, appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   app.ID,
				EnvName: "prod",
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &prodContent,
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return env-specific config when querying that env", func() {
			cfwc, err := provider.GetFrameworkMountableFile(ctx, app.ID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfwc.Content).To(Equal("server:\n  address: 0.0.0.0:9090\n"))
		})

		It("should fall back to default config when querying non-existent env", func() {
			cfwc, err := provider.GetFrameworkMountableFile(ctx, app.ID, "test-env")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfwc.Content).To(Equal("server:\n  address: 0.0.0.0:8080\n"))
		})
	})

	Context("when the env-specific config is an overlay", func() {
		It("should return the compiled content after overlay merge", func() {
			defID := createDef(appcfg.DefaultAppConfigFileName)
			baseContent := "database:\n  host: ${{ env.BASE_HOST }}\n  port: 3306\n"
			baseID, err := store.Add(ctx, appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   app.ID,
				EnvName: appcfg.EnvNameDefault,
				Type:    appcfg.AppConfigFileTypeNormal,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &baseContent,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			overlayContent := "overlayVersion: \"2\"\npatches:\n- database:\n    host: ${{ env.OVERLAY_HOST }}\n"
			_, err = store.Add(ctx, appcfg.AppConfigFile{
				DefID:   defID,
				AppID:   app.ID,
				EnvName: "prod",
				Type:    appcfg.AppConfigFileTypeOverlay,
				VersionedContent: appcfg.VersionedContent{
					ContentSourceType:   appcfg.ContentSourceTypeLocal,
					Format:              appcfg.FileFormatYAML,
					BaseAppConfigFileID: &baseID,
					OverlayContent:      &overlayContent,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			cfwc, err := provider.GetFrameworkMountableFile(ctx, app.ID, "prod")

			Expect(err).NotTo(HaveOccurred())
			Expect(cfwc.Content).To(Equal("database:\n  host: ${{ env.OVERLAY_HOST }}\n  port: 3306\n"))
		})

		It("should return compiled content after switching to independent config via service path", func() {
			result := createFrameworkFile(
				appcfg.DefaultAppConfigFileName,
				"database:\n  host: ${{ env.BASE_HOST }}\n  port: 3306\n",
			)
			def, err := defStore.GetByID(ctx, result.Def.ID)
			Expect(err).NotTo(HaveOccurred())

			isUnified := false
			err = svc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
				IsUnifiedConfig: &isUnified,
				Operator:        "editor",
			})
			Expect(err).NotTo(HaveOccurred())

			overlay := "overlayVersion: \"2\"\npatches:\n- database:\n    host: ${{ env.OVERLAY_HOST }}\n"
			envFile, compiled, isNewFile, err := svc.PrepareEnvContentUpdate(ctx, def, "prod", overlay, "editor")
			Expect(err).NotTo(HaveOccurred())
			Expect(isNewFile).To(BeTrue())
			Expect(compiled).To(Equal("database:\n  host: ${{ env.OVERLAY_HOST }}\n  port: 3306\n"))
			_, err = svc.CreateFileWithVersion(ctx, *envFile, def.Name, "create prod overlay", "editor")
			Expect(err).NotTo(HaveOccurred())

			cfwc, err := provider.GetFrameworkMountableFile(ctx, app.ID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfwc.Content).To(Equal("database:\n  host: ${{ env.OVERLAY_HOST }}\n  port: 3306\n"))
		})
	})

	Context("when multiple framework defs exist", func() {
		It("should return an error", func() {
			createFrameworkFile("values-a.yaml", "a: 1\n")
			createFrameworkFile("values-b.yaml", "b: 2\n")

			_, err := provider.GetFrameworkMountableFile(ctx, app.ID, "prod")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("multiple framework config files found"))
		})
	})
})

var _ = Describe("ListPlainMountableFiles", func() {
	var diApp *fxtest.App
	var ctx context.Context
	var appStore bkmsapp.ApplicationStore
	var store appcfg.AppConfigFileStore
	var defStore appcfg.AppConfigFileDefStore
	var versionStore appcfg.AppConfigFileVersionStore
	var app *bkmsapp.Application
	var provider appcfg.MountableFileProvider

	BeforeEach(func() {
		var err error

		ctx = context.Background()
		err = testutil.CleanupCollection("app_config_file_defs")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("app_config_files")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("applications")
		Expect(err).NotTo(HaveOccurred())

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			fx.Populate(&appStore, &store, &defStore, &versionStore),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		provider = appcfg.NewMountableFileProvider(store, defStore, versionStore)
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	createPlainDef := func(name, mountDir string) bson.ObjectID {
		id, err := defStore.Add(ctx, appcfg.AppConfigFileDef{
			AppID:      app.ID,
			Name:       name,
			ConfigKind: appcfg.ConfigKindPlain,
			MountDir:   mountDir,
			Creator:    "tester",
		})
		Expect(err).NotTo(HaveOccurred())
		return id
	}

	addDefaultFile := func(defID bson.ObjectID, content string) {
		_, err := store.Add(ctx, appcfg.AppConfigFile{
			DefID:   defID,
			AppID:   app.ID,
			EnvName: appcfg.EnvNameDefault,
			Type:    appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
			},
		})
		Expect(err).NotTo(HaveOccurred())
	}

	It("should return empty when no plain defs exist", func() {
		result, err := provider.ListPlainMountableFiles(ctx, app.ID, "prod")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("should return multiple plain files with correct mount dirs", func() {
		defA := createPlainDef("nginx.conf", "/etc/nginx")
		addDefaultFile(defA, "worker_processes 4;")

		defB := createPlainDef("redis.conf", "/etc/redis")
		addDefaultFile(defB, "maxmemory 256mb")

		result, err := provider.ListPlainMountableFiles(ctx, app.ID, "prod")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(2))

		names := []string{result[0].Name, result[1].Name}
		Expect(names).To(ContainElements("nginx.conf", "redis.conf"))
	})

	It("should use env-specific file when it exists", func() {
		defID := createPlainDef("app.conf", "/etc/app")
		addDefaultFile(defID, "default-content")

		envContent := "prod-content"
		_, err := store.Add(ctx, appcfg.AppConfigFile{
			DefID:   defID,
			AppID:   app.ID,
			EnvName: "prod",
			Type:    appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &envContent,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		result, err := provider.ListPlainMountableFiles(ctx, app.ID, "prod")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].Content).To(Equal("prod-content"))
		Expect(result[0].MountDir).To(Equal("/etc/app"))
	})

	It("should fall back to default when env-specific file does not exist", func() {
		defID := createPlainDef("app.conf", "/etc/app")
		addDefaultFile(defID, "default-content")

		result, err := provider.ListPlainMountableFiles(ctx, app.ID, "staging")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].Content).To(Equal("default-content"))
	})

	It("should return empty when mounted env names is explicitly empty", func() {
		defID, err := defStore.Add(ctx, appcfg.AppConfigFileDef{
			AppID:      app.ID,
			Name:       "app.conf",
			ConfigKind: appcfg.ConfigKindPlain,
			MountDir:   "/etc/app",
			Creator:    "tester",
			EnvConfigMode: appcfg.EnvConfigMode{
				IsUnifiedConfig: true,
				MountedEnvNames: []string{},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		addDefaultFile(defID, "default-content")

		result, err := provider.ListPlainMountableFiles(ctx, app.ID, "staging")

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})
})
