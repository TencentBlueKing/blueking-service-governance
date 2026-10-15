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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("CollectConfigWarnings", func() {
	var (
		ctx                   context.Context
		diApp                 *fxtest.App
		appConfigFileDefStore *appcfg.AppConfigFileDefStore
		configStore           *polaris.ConfigStore
		db                    *mongo.Database
		cfgProvider           appcfg.MountableFileProvider
		testAppID             string
	)

	BeforeEach(func() {
		ctx = context.Background()
		testAppID = "test-validate-" + stringx.Random(6)

		diApp = fxtest.New(
			GinkgoT(),
			appcfg.FxModule,
			polaris.FxModule,
			fx.Populate(),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())
		configStore = store.Of(db, polaris.ConfigMeta)
		appConfigFileDefStore = testutil.NewStore(GinkgoT(), appcfg.AppConfigFileDefMeta)
		cfgProvider = appcfg.NewMountableFileProvider(db)
	})

	AfterEach(func() {
		// 清理测试数据
		_ = appmodel.DeleteAppModel(ctx, db, testAppID)
		_, _ = store.Of(database.DB(), appcfg.AppConfigFileMeta).
			DeleteMany(ctx, appcfg.FilesByApp(testAppID))
		_, _ = appConfigFileDefStore.DeleteMany(ctx, appcfg.DefsByApp(testAppID))
		_, _ = configStore.DeleteMany(ctx, polaris.ByApp(testAppID))

		diApp.RequireStop()
	})

	// 辅助函数：创建 tRPC 类型的 AppModel
	createTrpcAppModel := func() {
		err := appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{
			AppID: testAppID,
			Workload: appmodel.Workload{
				Type: appmodel.WorkloadTypeTrpc,
				Name: "test-workload",
			},
		})
		Expect(err).NotTo(HaveOccurred())
	}

	// 辅助函数：创建 framework def + 默认文件，返回 defID
	createDefAndDefaultFile := func(content string) bson.ObjectID {
		def := &appcfg.AppConfigFileDef{
			AppID:      testAppID,
			Name:       appcfg.DefaultAppConfigFileName,
			ConfigKind: appcfg.ConfigKindFramework,
		}
		err := appConfigFileDefStore.Create(ctx, def)
		defID := def.ID
		Expect(err).NotTo(HaveOccurred())

		acf := appcfg.AppConfigFile{
			DefID:   defID,
			AppID:   testAppID,
			EnvName: appcfg.EnvNameDefault,
			Type:    appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Content:           &content,
			},
		}
		err = store.Of(db, appcfg.AppConfigFileMeta).Create(ctx, &acf)
		Expect(err).NotTo(HaveOccurred())
		return defID
	}

	// 辅助函数：在已有 def 下添加环境级配置文件
	addEnvFile := func(defID bson.ObjectID, envName, content string) {
		acf := appcfg.AppConfigFile{
			DefID:   defID,
			AppID:   testAppID,
			EnvName: envName,
			Type:    appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Content:           &content,
			},
		}
		err := store.Of(db, appcfg.AppConfigFileMeta).Create(ctx, &acf)
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
			warnings := polaris.CollectConfigWarnings(
				ctx,
				db,
				cfgProvider,
				config,
			)
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
				createDefAndDefaultFile(trpcYAML)
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
				warnings := polaris.CollectConfigWarnings(
					ctx,
					db,
					cfgProvider,
					config,
				)
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
				createDefAndDefaultFile(trpcYAML)
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
				warnings := polaris.CollectConfigWarnings(
					ctx,
					db,
					cfgProvider,
					config,
				)
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
				warnings := polaris.CollectConfigWarnings(
					ctx,
					db,
					cfgProvider,
					config,
				)
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
				createDefAndDefaultFile(trpcYAML)
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
				warnings := polaris.CollectConfigWarnings(
					ctx,
					db,
					cfgProvider,
					config,
				)
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when config file has no server.service section", func() {
			BeforeEach(func() {
				trpcYAML := `server:
  app: myapp
`
				createDefAndDefaultFile(trpcYAML)
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
				warnings := polaris.CollectConfigWarnings(
					ctx,
					db,
					cfgProvider,
					config,
				)
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
				// 创建共享 def + 空默认文件，并为各环境添加独立内容
				defaultYAML := `server:
  service: []
`
				defID := createDefAndDefaultFile(defaultYAML)

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
				addEnvFile(defID, "dev", devYAML)
				addEnvFile(defID, "prod", prodYAML)
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
				warnings := polaris.CollectConfigWarnings(
					ctx,
					db,
					cfgProvider,
					config,
				)
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
				defID := createDefAndDefaultFile(sameYAML)
				addEnvFile(defID, "dev", sameYAML)
				addEnvFile(defID, "staging", sameYAML)
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
				warnings := polaris.CollectConfigWarnings(
					ctx,
					db,
					cfgProvider,
					config,
				)
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
				createDefAndDefaultFile(appYAML)
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
				warnings := polaris.CollectConfigWarnings(
					ctx,
					db,
					cfgProvider,
					config,
				)
				Expect(warnings).To(BeEmpty())
			})
		})
	})
})
