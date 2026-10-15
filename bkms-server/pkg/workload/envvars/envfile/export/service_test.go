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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	coreapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	envvarsstore "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("ExportService", func() {
	var (
		diApp       *fxtest.App
		ctx         context.Context
		workspaceID string
		environment envmodel.Environment
		db          *mongo.Database
		appStore    *coreapp.ApplicationStore
		service     *Service
	)

	BeforeEach(func() {
		ctx = context.Background()
		appStore = testutil.NewStore(GinkgoT(), coreapp.AppMeta)
		diApp = fxtest.New(
			GinkgoT(),
			envvarsstore.FxModule,
			appcfg.FxModule,
			fx.Populate(),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())

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
			db,
			envvarsstore.NewUnifiedEnvVarsReader(db, nil, nil),
		)
	})

	AfterEach(func() {
		if db != nil {
			Expect(testutil.CleanupMetaColl(envvarsstore.ScopedEnvVarMeta)).NotTo(HaveOccurred())
		}
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	It("exports public vars as import-compatible dotenv text", func() {
		err := store.Of(db, envvarsstore.ScopedEnvVarMeta).Create(ctx, &envvarsstore.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			Key:         "WORKSPACE_KEY",
			Value:       "workspace-value",
			Description: "workspace desc",
		})
		Expect(err).NotTo(HaveOccurred())
		err = store.Of(db, envvarsstore.ScopedEnvVarMeta).Create(ctx, &envvarsstore.ScopedEnvVar{
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
		_, err := envvarsstore.CreateSimpleEnvScopeVar(
			ctx, db,
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
			AppStore: appStore,
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
		err := store.Of(db, envvarsstore.ScopedEnvVarMeta).Create(ctx, &envvarsstore.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			Key:         "PUBLIC_KEY",
			Value:       "workspace-value",
			Description: "public desc",
		})
		Expect(err).NotTo(HaveOccurred())

		app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore: appStore,
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

	It("exports the env-overridden resource limit instead of the stored default", func() {
		app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore: appStore,
		}, &dbfactory.TrpcApplicationOpts{WorkspaceID: workspaceID})

		am, err := store.Of(db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
		Expect(err).NotTo(HaveOccurred())
		am.Workload.Resources = map[string]string{"cpu": "1-2", "memory": "2Gi-4Gi"}
		Expect(appmodel.UpdateAppModel(ctx, db, am)).To(Succeed())
		Expect(appspec.SetEnv(ctx, db, &appspec.AppSpec{
			AppID:   app.ID,
			EnvName: environment.Name,
			Resources: &appspec.ResourcesSpec{
				CPURequests:    lo.ToPtr("1"),
				CPULimits:      lo.ToPtr("4"),
				MemoryRequests: lo.ToPtr("2Gi"),
				MemoryLimits:   lo.ToPtr("4Gi"),
			},
		})).To(Succeed())

		content, err := service.ExportEffectiveAppEnv(ctx, app, &environment)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).To(ContainSubstring("BKMS_CPU_LIMIT=4"))
		Expect(content).To(ContainSubstring("BKMS_MEMORY_LIMIT=4Gi"))
	})
})
