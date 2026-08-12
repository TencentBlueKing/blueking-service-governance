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

package export

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	coreapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvarsstore "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("ExportService", func() {
	var (
		diApp                 *fxtest.App
		ctx                   context.Context
		workspaceID           string
		environment           envmodel.Environment
		scopedEnvVarStore     envvarsstore.ScopedEnvVarStore
		appStore              coreapp.ApplicationStore
		appModelStore         appmodel.AppModelStore
		appConfigFileStore    appcfg.AppConfigFileStore
		appConfigVersionStore appcfg.AppConfigFileVersionStore
		buildConfigStore      build.ConfigStore
		service               *Service
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			envvarsstore.FxModule,
			depmodel.FxModule,
			coreapp.FxModule,
			appmodel.FxModule,
			appcfg.FxModule,
			build.FxModule,
			fx.Populate(
				&scopedEnvVarStore,
				&appStore,
				&appModelStore,
				&appConfigFileStore,
				&appConfigVersionStore,
				&buildConfigStore,
			),
		)
		diApp.RequireStart()

		// 为每条用例准备稳定的目标环境，让导出测试专注验证 scope 如何被渲染成
		// dotenv 元数据。
		workspaceID = "export-workspace-" + stringx.Random(6)
		environment = envmodel.Environment{
			WorkspaceID: workspaceID,
			Name:        "prod-env",
			Type:        "production",
			Cluster: envmodel.BizCluster{
				ClusterID: "BCS-K8S-00000",
				Namespace: "prod-ns",
			},
		}
		service = NewService(
			scopedEnvVarStore,
			appModelStore,
			envvarsstore.NewUnifiedEnvVarsReader(scopedEnvVarStore, nil, nil),
		)
	})

	AfterEach(func() {
		if scopedEnvVarStore != nil {
			Expect(scopedEnvVarStore.DeleteAll(ctx)).NotTo(HaveOccurred())
		}
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	It("exports public vars as import-compatible dotenv text", func() {
		_, err := scopedEnvVarStore.Create(ctx, envvarsstore.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			Key:         "WORKSPACE_KEY",
			Value:       "workspace-value",
			Description: "workspace desc",
		})
		Expect(err).NotTo(HaveOccurred())
		_, err = scopedEnvVarStore.Create(ctx, envvarsstore.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeEnvType,
			ScopeValue:  "production",
			Key:         "PROD_KEY",
			Value:       "prod-value",
			Description: "prod desc",
		})
		Expect(err).NotTo(HaveOccurred())

		content, err := service.ExportPublic(ctx, workspaceID)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(ContainSubstring("# desc: workspace desc"))
		Expect(content).To(ContainSubstring("# scopeType: workspace"))
		Expect(content).To(ContainSubstring("WORKSPACE_KEY=workspace-value"))
		Expect(content).To(ContainSubstring("# scopeType: envType"))
		Expect(content).To(ContainSubstring("# scopeValue: production"))
		Expect(content).To(ContainSubstring("PROD_KEY=prod-value"))
	})

	It("exports env vars without scope metadata", func() {
		_, err := scopedEnvVarStore.CreateSimpleEnvScopeVar(
			ctx,
			environment,
			"ENV_ONLY_KEY",
			"env-only-value",
			"env desc",
		)
		Expect(err).NotTo(HaveOccurred())

		content, err := service.ExportEnv(ctx, environment)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(ContainSubstring("# desc: env desc"))
		Expect(content).To(ContainSubstring("ENV_ONLY_KEY=env-only-value"))
		Expect(content).NotTo(ContainSubstring("# scopeType:"))
		Expect(content).NotTo(ContainSubstring("# scopeValue:"))
	})

	It("exports app-defined vars without scope metadata", func() {
		app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			WorkspaceID: workspaceID,
			EnvVars: []appmodel.Variable{
				{Key: "APP_MODE", Value: "prod", Description: "app mode"},
			},
		})

		content, err := service.ExportAppDefined(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(ContainSubstring("# desc: app mode"))
		Expect(content).To(ContainSubstring("APP_MODE=prod"))
		Expect(content).NotTo(ContainSubstring("# scopeType:"))
	})

	It("exports effective app env vars for a target environment", func() {
		_, err := scopedEnvVarStore.Create(ctx, envvarsstore.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			Key:         "PUBLIC_KEY",
			Value:       "workspace-value",
			Description: "public desc",
		})
		Expect(err).NotTo(HaveOccurred())

		app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			WorkspaceID: workspaceID,
			EnvVars: []appmodel.Variable{
				{Key: "APP_MODE", Value: "prod", Description: "app mode"},
			},
		})

		content, err := service.ExportEffectiveAppEnv(ctx, app, &environment)
		Expect(err).NotTo(HaveOccurred())
		// 最终生效变量导出应同时带上继承得到的公共变量和应用直接定义变量，
		// 这两个来源会由 unified reader 一起汇总。
		Expect(content).To(ContainSubstring("PUBLIC_KEY=workspace-value"))
		Expect(content).To(ContainSubstring("APP_MODE=prod"))
		Expect(content).To(ContainSubstring("BKMS_APP_NAME="))
	})
})
