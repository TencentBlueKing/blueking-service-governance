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

// Package admincmd_test 单元测试
package admincmd_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/trpc/admincmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var _ = Describe("TrpcAdminService", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var adminService *admincmd.TrpcAdminService

	// Store instances
	var appStore bkmsapp.ApplicationStore
	var appConfigFileStore appcfg.AppConfigFileStore
	var appConfigFileVersionStore appcfg.AppConfigFileVersionStore
	var buildConfigStore build.ConfigStore
	var scopedEnvVarStore envvars.ScopedEnvVarStore
	var appDepsVarReader *depenvvars.Reader
	var polarisVarReader *polarisenvvars.Reader
	var appModelStore appmodel.AppModelStore
	var envService *bkmsenv.EnvService

	// Test data
	var testApp *bkmsapp.Application
	var testAppModel *appmodel.AppModel
	var testEnv *envmodel.Environment

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			appmodel.FxModule,
			bkmsenv.FxModule,
			envvars.FxModule,
			build.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(
				&appStore,
				&appConfigFileStore,
				&appConfigFileVersionStore,
				&buildConfigStore,
				&scopedEnvVarStore,
				&appDepsVarReader,
				&polarisVarReader,
				&appModelStore,
				&envService,
			),
		)
		diApp.RequireStart()

		// 使用 dbfactory 创建测试用 tRPC 应用（包含 App、AppModel 和 AppConfigFile）
		testApp, testAppModel = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigFileVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			EnvVars: []appmodel.Variable{
				{Key: "MODEL_ADMIN_PORT", Value: "7070"},
			},
		})
		// 设置 TrpcSpec（dbfactory 创建的应用默认没有 TrpcSpec）
		testApp.TrpcSpec = &bkmsapp.TrpcSpec{Language: "go"}

		// 使用 dbfactory 创建测试用 Environment
		testEnv = dbfactory.Env(ctx, envService, testApp.WorkspaceID)

		// 创建 TrpcAdminService（跳过验证逻辑）
		adminService = &admincmd.TrpcAdminService{
			AppConfigFileStore: appConfigFileStore,
			AppModelStore:      appModelStore,
			EnvVarsReader:      envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader),
		}
		adminService.App = testApp
		adminService.AppModel = testAppModel
		adminService.Env = testEnv
		adminService.EnvName = testEnv.Name
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Describe("GetAdminConfig", func() {
		Context("when config file exists for specific environment", func() {
			var configFileID bson.ObjectID

			BeforeEach(func() {
				// 创建配置文件内容
				configContent := `server:
  admin:
    port: "11014"
    admin_port: "11015"
  port: "8080"
  admin_port: "8081"`

				var err error
				acf, err := appcfg.NewAppConfigFileService(appConfigFileStore, appConfigFileVersionStore).Create(
					ctx,
					appcfg.CreateCfgFileParams{
						AppID:             testApp.ID,
						EnvName:           testEnv.Name,
						Name:              "trpc_go.yaml",
						Type:              appcfg.AppConfigFileTypeNormal,
						ContentSourceType: appcfg.ContentSourceTypeLocal,
						Format:            appcfg.FileFormatYAML,
						Content:           &configContent,
					},
				)
				Expect(err).NotTo(HaveOccurred())
				configFileID = acf.ID
				Expect(configFileID).NotTo(Equal(bson.ObjectID{}))
			})

			It("should return parsed admin config", func() {
				cfg, err := adminService.GetAdminConfig(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Server.Admin.Port).To(Equal("11014"))
				Expect(cfg.Server.Admin.AdminPort).To(Equal("11015"))
				Expect(cfg.Server.Port).To(Equal("8080"))
				Expect(cfg.Server.AdminPort).To(Equal("8081"))
			})
		})
	})

	Describe("ResolvePort", func() {
		BeforeEach(func() {
			// AppModel 已在全局 BeforeEach 中通过 dbfactory.TrpcApplication 创建
			// 并且已包含 MODEL_ADMIN_PORT 环境变量

			// 添加环境变量（使用 dbfactory 创建的环境）
			_, err := scopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *testEnv, "ADMIN_PORT", "9090", "")
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when port is a direct number", func() {
			It("should return the port number directly", func() {
				port, err := adminService.ResolvePort(ctx, "11014")
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(int32(11014)))
			})
		})

		Context("when port contains invalid value", func() {
			It("should return error for non-numeric port", func() {
				_, err := adminService.ResolvePort(ctx, "abc")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unknown port config value"))
			})

			It("should return error for empty port", func() {
				_, err := adminService.ResolvePort(ctx, "")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when port contains environment variable", func() {
			It("should resolve ${VAR_NAME} format", func() {
				port, err := adminService.ResolvePort(ctx, "${ADMIN_PORT}")
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(int32(9090)))
			})

			It("should resolve ${VAR_NAME} prefix", func() {
				port, err := adminService.ResolvePort(ctx, "1${ADMIN_PORT}")
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(int32(19090)))
			})
		})

		Context("when port is from AppModel envVars", func() {
			It("should resolve from AppModel envVars first", func() {
				port, err := adminService.ResolvePort(ctx, "${MODEL_ADMIN_PORT}")
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(int32(7070)))
			})
		})
	})

	Describe("GetAdminPort", func() {
		// 定义配置设置函数类型
		type configSetupFunc func(cfg *admincmd.AdminConfig)

		DescribeTable(
			"should return correct admin port for different languages",
			func(language string, setupConfig configSetupFunc, expectedPort string, expectError bool, errorMsg string) {
				// 设置语言
				testApp.TrpcSpec.Language = language

				// 创建并配置 AdminConfig
				adminConfig := new(admincmd.AdminConfig)
				if setupConfig != nil {
					setupConfig(adminConfig)
				}

				// 调用 GetAdminPort
				port, err := adminService.GetAdminPort(adminConfig)

				// 验证结果
				if expectError {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(errorMsg))
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(port).To(Equal(expectedPort))
				}
			},
			Entry("Go language - server.admin.port", "go",
				func(cfg *admincmd.AdminConfig) { cfg.Server.Admin.Port = "11014" },
				"11014", false, ""),
			Entry("Python language - server.admin.port", "python",
				func(cfg *admincmd.AdminConfig) { cfg.Server.Admin.Port = "11015" },
				"11015", false, ""),
			Entry("C++ language - server.admin_port", "cpp",
				func(cfg *admincmd.AdminConfig) { cfg.Server.AdminPort = "11016" },
				"11016", false, ""),
			Entry("Node language - server.admin_port", "node",
				func(cfg *admincmd.AdminConfig) { cfg.Server.AdminPort = "11017" },
				"11017", false, ""),
			Entry("Java language - server.admin.admin_port", "java",
				func(cfg *admincmd.AdminConfig) { cfg.Server.Admin.AdminPort = "11018" },
				"11018", false, ""),
			Entry("unsupported language - should return error", "unknown",
				nil,
				"", true, "unsupported trpc language type"),
		)
	})

	Describe("Precheck", func() {
		// 辅助函数：创建指定内容的配置文件
		createConfigFile := func(content string) {
			_, err := appcfg.NewAppConfigFileService(appConfigFileStore, appConfigFileVersionStore).Create(
				ctx,
				appcfg.CreateCfgFileParams{
					AppID:             testApp.ID,
					EnvName:           testEnv.Name,
					Name:              "trpc_go.yaml",
					Type:              appcfg.AppConfigFileTypeNormal,
					ContentSourceType: appcfg.ContentSourceTypeLocal,
					Format:            appcfg.FileFormatYAML,
					Content:           &content,
				},
			)
			Expect(err).NotTo(HaveOccurred())
		}

		DescribeTable("should validate admin IP correctly",
			func(language, configYAML string, expectError bool, errorSubstr string) {
				testApp.TrpcSpec.Language = language
				createConfigFile(configYAML)

				err := adminService.Precheck(ctx)
				if expectError {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(errorSubstr))
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			},
			Entry("go - IP 0.0.0.0, pass", "go",
				"server:\n  admin:\n    ip: \"0.0.0.0\"\n    port: \"11014\"",
				false, ""),
			Entry("go - IP 127.0.0.1, pass", "go",
				"server:\n  admin:\n    ip: \"127.0.0.1\"\n    port: \"11014\"",
				false, ""),
			Entry("go - IP not configured, error", "go",
				"server:\n  admin:\n    port: \"11014\"",
				true, "admin server IP is not configured"),
			Entry("go - invalid IP, error", "go",
				"server:\n  admin:\n    ip: \"10.0.0.1\"\n    port: \"11014\"",
				true, "admin server binding IP can only be 0.0.0.0 or 127.0.0.1"),
			Entry("cpp - server.admin_ip 0.0.0.0, pass", "cpp",
				"server:\n  admin_ip: \"0.0.0.0\"\n  admin_port: \"11014\"",
				false, ""),
			Entry("java - invalid admin_ip, error", "java",
				"server:\n  admin:\n    admin_ip: \"127.0.0.2\"\n    admin_port: \"11014\"",
				true, "admin server binding IP can only be 0.0.0.0 or 127.0.0.1"),
		)
	})
})
