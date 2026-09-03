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

var _ = Describe("GetEnvContent", func() {
	var diApp *fxtest.App
	var ctx context.Context
	var appStore bkmsapp.ApplicationStore
	var store appcfg.AppConfigFileStore
	var defStore appcfg.AppConfigFileDefStore
	var app *bkmsapp.Application

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
			fx.Populate(&appStore, &store, &defStore),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
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

		It("should return default config when querying non-existent env", func() {
			_, _, content, err := appcfg.GetEnvContent(ctx, store, defStore, app.ID, "test-env")
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal("server:\n  address: 0.0.0.0:8080\n"))
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
			_, _, content, err := appcfg.GetEnvContent(ctx, store, defStore, app.ID, "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal("server:\n  address: 0.0.0.0:9090\n"))
		})

		It("should return default config when querying non-existent env", func() {
			_, _, content, err := appcfg.GetEnvContent(ctx, store, defStore, app.ID, "test-env")
			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal("server:\n  address: 0.0.0.0:8080\n"))
		})
	})

	Context("when the env-specific config is an overlay", func() {
		It("should return the overlay logical file and compiled content", func() {
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

			_, _, content, err := appcfg.GetEnvContent(ctx, store, defStore, app.ID, "prod")

			Expect(err).NotTo(HaveOccurred())
			Expect(content).To(Equal("database:\n  host: ${{ env.OVERLAY_HOST }}\n  port: 3306\n"))
		})
	})
})
