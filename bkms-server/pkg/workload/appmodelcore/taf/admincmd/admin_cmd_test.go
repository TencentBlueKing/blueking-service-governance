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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/taf/admincmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var _ = Describe("TafAdminService", func() {
	var ctx context.Context
	var diApp *fxtest.App

	// Store instances
	var appStore bkmsapp.ApplicationStore
	var appConfigFileStore appcfg.AppConfigFileStore
	var appConfigFileVersionStore appcfg.AppConfigFileVersionStore
	var appModelStore appmodel.AppModelStore
	var envStore envmodel.EnvironmentStore
	var envService *bkmsenv.EnvService
	var scopedEnvVarStore envvars.ScopedEnvVarStore
	var buildConfigStore build.ConfigStore
	var appDepsVarReader *depenvvars.Reader
	var polarisVarReader *polarisenvvars.Reader

	// Test data
	var testApp *bkmsapp.Application
	var testAppModel *appmodel.AppModel
	var testEnv *envmodel.Environment

	// 创建 TafAdminService（跳过需要 K8s 和数据库的验证步骤）
	newAdminService := func() *admincmd.TafAdminService {
		svc := &admincmd.TafAdminService{}
		svc.App = testApp
		svc.AppModel = testAppModel
		svc.Env = testEnv
		svc.EnvName = testEnv.Name
		svc.Stores = &admincmd.AdminServiceStores{
			AppConfigFileStore: appConfigFileStore,
			AppModelStore:      appModelStore,
			EnvStore:           envStore,
			AppStore:           appStore,
			EnvVarsReader:      envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader),
		}
		return svc
	}

	// 辅助函数：创建指定内容的 TAF 配置文件
	createConfigFile := func(content string) {
		_, err := appcfg.NewAppConfigFileService(appConfigFileStore, appConfigFileVersionStore).Create(
			ctx,
			appcfg.CreateCfgFileParams{
				AppID:             testApp.ID,
				EnvName:           testEnv.Name,
				Name:              "taf.tafconfig.xml",
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatTAF,
				Content:           &content,
			},
		)
		Expect(err).NotTo(HaveOccurred())
	}

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
				&appModelStore,
				&envStore,
				&envService,
				&scopedEnvVarStore,
				&appDepsVarReader,
				&polarisVarReader,
			),
		)
		diApp.RequireStart()

		// 使用 dbfactory 创建测试用 TAF 应用
		testApp, testAppModel = dbfactory.TafApplication(ctx, &dbfactory.TafApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigFileVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, nil)

		// 使用 dbfactory 创建测试用 Environment
		testEnv = dbfactory.Env(ctx, envService, testApp.WorkspaceID)
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Describe("GetAdminPortFromConfig", func() {
		Context("when TAF config file exists with /taf root", func() {
			BeforeEach(func() {
				createConfigFile(`<taf>
  <application>
    <server>
      local=tcp -h 0.0.0.0 -p 17064 -t 30000
      app=HeroGame
      server=GameCoinServer
    </server>
  </application>
</taf>`)
			})

			It("should extract port from local endpoint", func() {
				ip, port, err := newAdminService().GetAdminConfig(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(int32(17064)))
				Expect(ip).To(Equal("0.0.0.0"))
			})
		})

		Context("when TAF config file exists with /tars root", func() {
			BeforeEach(func() {
				createConfigFile(`<tars>
  <application>
    <server>
      local=tcp -h 0.0.0.0 -p 20000 -t 60000
    </server>
  </application>
</tars>`)
			})

			It("should extract port from /tars root", func() {
				ip, port, err := newAdminService().GetAdminConfig(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(int32(20000)))
				Expect(ip).To(Equal("0.0.0.0"))
			})
		})

		Context("when local endpoint contains template variables ${{VAR}}", func() {
			BeforeEach(func() {
				// 使用包含模板变量的 local 字段，模拟真实场景
				// local=tcp -h ${{BKMS_POD_IP}} -p ${{TAF_SERVER_PORT}} -t 30000
				createConfigFile(`<taf>
  <application>
    <server>
      local=tcp -h ${{env.TAF_SERVER_IP}} -p ${{env.TAF_SERVER_PORT}} -t 30000
      app=HeroGame
      server=GameCoinServer
    </server>
  </application>
</taf>`)

				// 通过 AppModel EnvVars 设置 TAF_SERVER_PORT（AppModel 级变量优先）
				// 更新 AppModel 中的环境变量
				testAppModel.Workload.EnvVars = append(testAppModel.Workload.EnvVars,
					appmodel.Variable{Key: "TAF_SERVER_PORT", Value: "17064"},
					appmodel.Variable{Key: "TAF_SERVER_IP", Value: "0.0.0.0"},
				)
				err := appModelStore.UpdateAppModel(ctx, testAppModel)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should render ${{TAF_SERVER_PORT}} from AppModel EnvVars and extract port", func() {
				ip, port, err := newAdminService().GetAdminConfig(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(int32(17064)))
				Expect(ip).To(Equal("0.0.0.0"))
			})
		})

		Context("when local endpoint uses env-level variable ${{VAR}}", func() {
			BeforeEach(func() {
				createConfigFile(`<taf>
  <application>
    <server>
      local=tcp -h ${{env.TAF_SERVER_IP}} -p ${{env.TAF_SERVER_PORT}} -t 30000
    </server>
  </application>
</taf>`)

				// 通过环境级变量设置 TAF_SERVER_PORT
				_, err := scopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *testEnv, "TAF_SERVER_PORT", "18080", "")
				Expect(err).NotTo(HaveOccurred())
				_, err = scopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *testEnv, "TAF_SERVER_IP", "0.0.0.0", "")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should render ${{TAF_SERVER_PORT}} from env-level vars and extract port", func() {
				ip, port, err := newAdminService().GetAdminConfig(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(port).To(Equal(int32(18080)))
				Expect(ip).To(Equal("0.0.0.0"))
			})
		})

		Context("when use invalid host IP", func() {
			BeforeEach(func() {
				createConfigFile(`<taf>
  <application>
    <server>
      local=tcp -h 10.0.0.1 -p 12345 -t 30000
    </server>
  </application>
</taf>`)
			})

			It("should return error because host IP is not valid", func() {
				_, _, err := newAdminService().GetAdminConfig(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("admin server binding IP can only be 0.0.0.0, 127.0.0.1"))
			})
		})

		Context("when template variable is undefined", func() {
			BeforeEach(func() {
				// 包含未定义变量的配置，端口无法解析
				createConfigFile(`<taf>
  <application>
    <server>
      local=tcp -h 0.0.0.0 -p ${{env.UNDEFINED_PORT}} -t 30000
    </server>
  </application>
</taf>`)
			})

			It("should return error because port is not a valid number", func() {
				_, _, err := newAdminService().GetAdminConfig(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid port number"))
			})
		})

		Context("when config file is missing", func() {
			It("should return error", func() {
				_, _, err := newAdminService().GetAdminConfig(ctx)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when local field is missing from config", func() {
			BeforeEach(func() {
				createConfigFile(`<taf>
  <application>
    <server>
      app=HeroGame
      server=GameCoinServer
    </server>
  </application>
</taf>`)
			})

			It("should return error about missing local field", func() {
				_, _, err := newAdminService().GetAdminConfig(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("'local' field not found"))
			})
		})
	})
})
