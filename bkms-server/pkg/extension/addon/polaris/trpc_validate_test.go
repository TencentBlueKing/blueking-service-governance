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

package polaris_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("CollectConfigWarnings", func() {
	var (
		ctx                context.Context
		diApp              *fxtest.App
		appModelStore      appmodel.AppModelStore
		appConfigFileStore appcfg.AppConfigFileStore
		polarisConfigStore polaris.PolarisConfigStore
		testAppID          string
	)

	BeforeEach(func() {
		ctx = context.Background()
		testAppID = "test-validate-" + stringx.Random(6)

		diApp = fxtest.New(
			GinkgoT(),
			appmodel.FxModule,
			appcfg.FxModule,
			polaris.FxModule,
			fx.Populate(
				&appModelStore,
				&appConfigFileStore,
				&polarisConfigStore,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		// 清理测试数据
		_ = appModelStore.DeleteAppModel(ctx, testAppID)
		_, _ = appConfigFileStore.DeleteByApp(ctx, testAppID)
		_ = polarisConfigStore.DeleteByApp(ctx, testAppID)

		diApp.RequireStop()
	})

	// 辅助函数：创建 tRPC 类型的 AppModel
	createTrpcAppModel := func() {
		err := appModelStore.CreateAppModel(ctx, &appmodel.AppModel{
			AppID: testAppID,
			Workload: appmodel.Workload{
				Type: appmodel.WorkloadTypeTrpc,
				Name: "test-workload",
			},
		})
		Expect(err).NotTo(HaveOccurred())
	}

	// 辅助函数：创建应用级别 tRPC 配置文件
	createAppLevelConfigFile := func(content string) {
		acf := appcfg.AppConfigFile{
			AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
				AppID:             testAppID,
				EnvName:           appcfg.EnvNameDefault,
				Name:              appcfg.DefaultAppConfigFileName,
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Content:           &content,
			},
		}
		_, err := appConfigFileStore.Add(ctx, acf)
		Expect(err).NotTo(HaveOccurred())
	}

	// 辅助函数：创建环境级别 tRPC 配置文件
	createEnvConfigFile := func(envName, content string) {
		acf := appcfg.AppConfigFile{
			AppConfigFileContentSpec: appcfg.AppConfigFileContentSpec{
				AppID:             testAppID,
				EnvName:           envName,
				Name:              envName,
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Content:           &content,
			},
		}
		_, err := appConfigFileStore.Add(ctx, acf)
		Expect(err).NotTo(HaveOccurred())
	}

	Context("when scope env names is empty", func() {
		BeforeEach(func() {
			createTrpcAppModel()
		})

		It("should skip service name validation", func() {
			config := &polaris.PolarisConfig{
				Name:  "my-polaris",
				AppID: testAppID,
				Properties: polaris.Properties{
					PolarisName: "trpc.app.server.service",
				},
			}
			warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
			Expect(warnings).To(BeEmpty())
		})
	})

	// 原 global scope 场景：改为对 ScopeEnvNames 环境校验，使用 app-level 配置回退。
	Context("when app is tRPC type with app-level config", func() {
		BeforeEach(func() {
			createTrpcAppModel()
		})

		Context("when config file has matching service name", func() {
			BeforeEach(func() {
				trpcYAML := `server:
  service:
    - name: trpc.app.server.service
      ip: 0.0.0.0
      port: 8080
`
				createAppLevelConfigFile(trpcYAML)
			})

			It("should return no warnings", func() {
				config := &polaris.PolarisConfig{
					Name:  "my-polaris",
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName: "trpc.app.server.service",
					},
					ScopeEnvNames: []string{"dev"},
				}
				warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when config file has non-matching service name", func() {
			BeforeEach(func() {
				trpcYAML := `server:
  service:
    - name: trpc.app.server.other
      ip: 0.0.0.0
      port: 8080
`
				createAppLevelConfigFile(trpcYAML)
			})

			It("should return warning about name mismatch", func() {
				config := &polaris.PolarisConfig{
					Name:  "my-polaris",
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName: "trpc.app.server.service",
					},
					ScopeEnvNames: []string{"dev"},
				}
				warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0]).To(ContainSubstring("环境 'dev'"))
				Expect(warnings[0]).To(ContainSubstring("推荐与 tRPC 配置中的服务名"))
				Expect(warnings[0]).To(ContainSubstring("trpc.app.server.service"))
				Expect(warnings[0]).To(ContainSubstring("trpc.app.server.other"))
			})

			It("should skip the warning for immediate-register configs", func() {
				config := &polaris.PolarisConfig{
					Name:  "my-polaris",
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName:  "trpc.app.server.service",
						RegisterMode: polaris.RegisterModeImmediate,
					},
					ScopeEnvNames: []string{"dev"},
				}
				warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when config file has multiple services and one matches", func() {
			BeforeEach(func() {
				trpcYAML := `server:
  service:
    - name: trpc.app.server.service1
      port: 8080
    - name: trpc.app.server.service2
      port: 8081
`
				createAppLevelConfigFile(trpcYAML)
			})

			It("should return no warnings when polaris name matches one of the services", func() {
				config := &polaris.PolarisConfig{
					Name:  "my-polaris",
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName: "trpc.app.server.service2",
					},
					ScopeEnvNames: []string{"dev"},
				}
				warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when config file has no server.service section", func() {
			BeforeEach(func() {
				trpcYAML := `server:
  app: myapp
`
				createAppLevelConfigFile(trpcYAML)
			})

			It("should return warning about name mismatch with empty service list", func() {
				config := &polaris.PolarisConfig{
					Name:  "my-polaris",
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName: "trpc.app.server.service",
					},
					ScopeEnvNames: []string{"dev"},
				}
				warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0]).To(ContainSubstring("[my-polaris]"))
				Expect(warnings[0]).To(ContainSubstring("环境 'dev'"))
				Expect(warnings[0]).To(ContainSubstring("推荐与 tRPC 配置中的服务名"))
			})
		})
	})

	Context("when app is tRPC type with environment scope", func() {
		BeforeEach(func() {
			createTrpcAppModel()
		})

		Context("when env-specific config matches", func() {
			BeforeEach(func() {
				devYAML := `server:
  service:
    - name: trpc.app.server.dev-svc
      port: 8080
`
				prodYAML := `server:
  service:
    - name: trpc.app.server.prod-svc
      port: 9090
`
				createEnvConfigFile("dev", devYAML)
				createEnvConfigFile("prod", prodYAML)
			})

			It("should validate each environment independently", func() {
				config := &polaris.PolarisConfig{
					Name:  "my-polaris",
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName: "trpc.app.server.dev-svc",
					},
					ScopeEnvNames: []string{"dev", "prod"},
				}
				warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
				// dev 匹配，prod 不匹配
				Expect(warnings).To(HaveLen(1))
				Expect(warnings[0]).To(ContainSubstring("环境 'prod'"))
				Expect(warnings[0]).To(ContainSubstring("推荐与 tRPC 配置中的服务名"))
			})
		})

		Context("when all environments match", func() {
			BeforeEach(func() {
				sameYAML := `server:
  service:
    - name: trpc.app.server.svc
      port: 8080
`
				createEnvConfigFile("dev", sameYAML)
				createEnvConfigFile("staging", sameYAML)
			})

			It("should return no warnings", func() {
				config := &polaris.PolarisConfig{
					Name:  "my-polaris",
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName: "trpc.app.server.svc",
					},
					ScopeEnvNames: []string{"dev", "staging"},
				}
				warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when env config not found but app-level config exists", func() {
			BeforeEach(func() {
				appYAML := `server:
  service:
    - name: trpc.app.server.svc
      port: 8080
`
				createAppLevelConfigFile(appYAML)
			})

			It("should fall back to app-level config", func() {
				config := &polaris.PolarisConfig{
					Name:  "my-polaris",
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName: "trpc.app.server.svc",
					},
					ScopeEnvNames: []string{"dev"},
				}
				warnings := polaris.CollectConfigWarnings(ctx, appModelStore, appConfigFileStore, config)
				Expect(warnings).To(BeEmpty())
			})
		})
	})
})
