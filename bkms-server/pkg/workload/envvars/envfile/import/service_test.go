package importer_test

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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/import"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("ImportService", func() {
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
		service               *importer.Service
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

		// 每条用例都使用独立的 workspace / env 种子数据，这样覆盖与新增的断言
		// 不会受到其他导入场景污染。
		workspaceID = "import-workspace-" + stringx.Random(6)
		environment = envmodel.Environment{
			WorkspaceID: workspaceID,
			Name:        "prod-env",
			Type:        "production",
			Cluster: envmodel.BizCluster{
				ClusterID: "BCS-K8S-00000",
				Namespace: "prod-ns",
			},
		}
		seedScopedEnvVars(ctx, scopedEnvVarStore, workspaceID)
		service = importer.NewService(scopedEnvVarStore, appModelStore)
	})

	AfterEach(func() {
		if scopedEnvVarStore != nil {
			Expect(scopedEnvVarStore.DeleteAll(ctx)).NotTo(HaveOccurred())
		}
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	It("imports public vars with preview-aligned scope resolution", func() {
		err := service.ImportPublic(ctx, workspaceID, `
# desc: workspace key
# scopeType: workspace
WORKSPACE_NEW=new-workspace-value

# desc: override env-type
# scopeType: envType
# scopeValue: development
SHARED_KEY=dev-override
`)
		Expect(err).NotTo(HaveOccurred())

		publicVars, err := scopedEnvVarStore.ListPublic(ctx, workspaceID)
		Expect(err).NotTo(HaveOccurred())

		publicVarsByScope := map[string]envvarsstore.ScopedEnvVar{}
		for _, item := range publicVars {
			publicVarsByScope[string(item.ScopeType)+":"+item.ScopeValue+":"+item.Key] = item
		}
		Expect(publicVarsByScope["workspace::WORKSPACE_NEW"].Value).To(Equal("new-workspace-value"))
		Expect(publicVarsByScope["workspace::WORKSPACE_NEW"].Description).To(Equal("workspace key"))
		Expect(publicVarsByScope["envType:development:SHARED_KEY"].Value).To(Equal("dev-override"))
		Expect(publicVarsByScope["envType:development:SHARED_KEY"].Description).To(Equal("override env-type"))
	})

	It("imports env-scoped vars into the target environment only", func() {
		err := service.ImportEnv(ctx, environment, `
# desc: overwrite existing env key
ENV_ONLY_KEY=updated-env-value

# desc: add another env key
ANOTHER_ENV_KEY=another-value
`)
		Expect(err).NotTo(HaveOccurred())

		envVars, err := scopedEnvVarStore.List(
			ctx,
			workspaceID,
			envvarsstore.WithScopes(envvartypes.ScopeEnv("prod-env")),
		)
		Expect(err).NotTo(HaveOccurred())
		envVarsByKey := map[string]envvarsstore.ScopedEnvVar{}
		for _, item := range envVars {
			envVarsByKey[item.Key] = item
		}
		Expect(envVarsByKey["ENV_ONLY_KEY"].Value).To(Equal("updated-env-value"))
		Expect(envVarsByKey["ENV_ONLY_KEY"].Description).To(Equal("overwrite existing env key"))
		Expect(envVarsByKey["ANOTHER_ENV_KEY"].Value).To(Equal("another-value"))
		Expect(envVarsByKey["ANOTHER_ENV_KEY"].Description).To(Equal("add another env key"))

		// 导入只能影响目标 env scope；其余无关环境作用域应保持种子数据原样。
		otherEnvVars, err := scopedEnvVarStore.List(
			ctx,
			workspaceID,
			envvarsstore.WithScopes(envvartypes.ScopeEnv("other-env")),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(otherEnvVars).To(HaveLen(1))
		Expect(otherEnvVars[0].Key).To(Equal("OTHER_ENV_KEY"))
	})

	It("imports app-defined vars by overwriting and appending in one batch", func() {
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

		err := service.ImportApp(ctx, app, `
# desc: overwrite app var
APP_MODE=test

# desc: add app var
APP_REGION=ap-guangzhou
`)
		Expect(err).NotTo(HaveOccurred())

		envVars, err := appModelStore.ListAppDefinedEnvVars(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		envVarsByKey := map[string]appmodel.Variable{}
		for _, item := range envVars {
			envVarsByKey[item.Key] = item
		}
		Expect(envVarsByKey["APP_MODE"].Value).To(Equal("test"))
		Expect(envVarsByKey["APP_MODE"].Description).To(Equal("overwrite app var"))
		Expect(envVarsByKey["APP_REGION"].Value).To(Equal("ap-guangzhou"))
		Expect(envVarsByKey["APP_REGION"].Description).To(Equal("add app var"))
	})

	It("rejects scope metadata in env import", func() {
		err := service.ImportEnv(ctx, environment, `
# scopeType: workspace
ENV_ONLY_KEY=updated-env-value
`)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("env import does not allow scope metadata"))
	})
})

func seedScopedEnvVars(ctx context.Context, store envvarsstore.ScopedEnvVarStore, workspaceID string) {
	// 预先种入公共作用域的共享 key 和若干 env 作用域数据，用来验证不同导入目标
	// 下的覆盖规则。
	_, err := store.Create(ctx, envvarsstore.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeWorkspace,
		ScopeValue:  "",
		Key:         "SHARED_KEY",
		Value:       "workspace-value",
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = store.Create(ctx, envvarsstore.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeEnvType,
		ScopeValue:  "development",
		Key:         "SHARED_KEY",
		Value:       "env-type-development-value",
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = store.CreateSimpleEnvScopeVar(
		ctx,
		envmodel.Environment{WorkspaceID: workspaceID, Name: "prod-env"},
		"ENV_ONLY_KEY",
		"env-only-value",
		"",
	)
	Expect(err).NotTo(HaveOccurred())

	_, err = store.CreateSimpleEnvScopeVar(
		ctx,
		envmodel.Environment{WorkspaceID: workspaceID, Name: "other-env"},
		"OTHER_ENV_KEY",
		"ignored",
		"",
	)
	Expect(err).NotTo(HaveOccurred())
}
