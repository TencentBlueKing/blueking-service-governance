package appdefaults_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	componentdevmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Application default resolution", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var ruleStore *appdefaults.RuleStore
	var db *mongo.Database
	var envService *bkmsenv.EnvService
	var workspaceStore *bkmsworkspace.WorkspaceStore
	var workspace *bkmsworkspace.Workspace

	BeforeEach(func() {
		ctx = context.Background()
		bkmsworkspace.ResetLifecycleHooksForTest()
		bkmsenv.ResetHooksForTest()

		diApp = fxtest.New(
			GinkgoT(),
			bkmsenv.FxModule,
			fx.Populate(&envService),
		)
		diApp.RequireStart()
		workspaceStore = testutil.NewStore(GinkgoT(), bkmsworkspace.Meta)
		db = testutil.NewDB(GinkgoT())
		ruleStore = store.Of(db, appdefaults.RuleMeta)
		workspace = dbfactory.Workspace(ctx, workspaceStore)
	})

	AfterEach(func() {
		// Clear documents without Drop so migration-managed unique indexes survive.
		Expect(testutil.CleanupMetaColl(appdefaults.RuleMeta)).To(Succeed())
		Expect(testutil.CleanupCollection("environments")).To(Succeed())
		Expect(bkmsworkspace.Delete(ctx, workspaceStore, workspace.ID)).To(Succeed())
		bkmsworkspace.ResetLifecycleHooksForTest()
		bkmsenv.ResetHooksForTest()
		diApp.RequireStop()
	})

	It("uses platform defaults without persisting rules", func() {
		resolved, err := appdefaults.Resolve(ctx, db, workspace.ID, "app-without-rules", bkmsapp.AppTypeTRPC)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Default.AppID).To(Equal("app-without-rules"))
		Expect(resolved.Default.EnvName).To(Equal(appspec.DefaultEnvName))
		Expect(*resolved.Default.Resources.Replicas).To(Equal(int32(1)))
		Expect(*resolved.Default.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(resolved.Environments).To(BeEmpty())

		persisted, err := ruleStore.List(ctx, appdefaults.ByWorkspace(workspace.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(persisted).To(BeEmpty())
	})

	It("combines independent section rules for an environment", func() {
		production := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
		Expect(ruleStore.Create(ctx, resourcesRule(workspace.ID, "production"))).To(Succeed())
		Expect(ruleStore.Create(ctx, devModeRule(workspace.ID, "production", true))).To(Succeed())

		resolved, err := appdefaults.Resolve(ctx, db, workspace.ID, "app-combined-defaults", bkmsapp.AppTypeTRPC)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Environments).To(HaveLen(1))
		spec := resolved.Environments[0]
		Expect(spec.AppID).To(Equal("app-combined-defaults"))
		Expect(spec.EnvName).To(Equal(production.Name))
		Expect(*spec.Resources.Replicas).To(Equal(int32(2)))
		Expect(*spec.DevMode.Enabled).To(BeTrue())
	})

	It("applies rules to every environment with the matching type", func() {
		productionA := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
		productionB := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
		_ = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "development",
		})
		Expect(ruleStore.Create(ctx, devModeRule(workspace.ID, "production", true))).To(Succeed())

		resolved, err := appdefaults.Resolve(ctx, db, workspace.ID, "app-matching-envs", bkmsapp.AppTypeTRPC)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Environments).To(HaveLen(2))

		specs := make(map[string]*appspec.AppSpec, len(resolved.Environments))
		for _, spec := range resolved.Environments {
			specs[spec.EnvName] = spec
		}
		Expect(specs).To(HaveKey(productionA.Name))
		Expect(specs).To(HaveKey(productionB.Name))
		Expect(*specs[productionA.Name].DevMode.Enabled).To(BeTrue())
		Expect(*specs[productionB.Name].DevMode.Enabled).To(BeTrue())
	})

	It("derives devMode paths from the app type", func() {
		development := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "development",
		})
		Expect(ruleStore.Create(ctx, devModeRule(workspace.ID, "development", true))).To(Succeed())

		resolved, err := appdefaults.Resolve(ctx, db, workspace.ID, "app-taf-dev-mode", bkmsapp.AppTypeTAF)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Environments).To(HaveLen(1))
		spec := resolved.Environments[0]
		Expect(spec.EnvName).To(Equal(development.Name))
		Expect(*spec.DevMode.Enabled).To(BeTrue())
		Expect(*spec.DevMode.WorkPath).To(Equal(componentdevmode.TafWorkPath))
		Expect(*spec.DevMode.MountPath).To(Equal(componentdevmode.TafMountPath))

		rules, err := ruleStore.List(ctx, appdefaults.ByWorkspace(workspace.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(HaveLen(1))
		Expect(rules[0].Spec.DevMode.WorkPath).To(BeNil())
	})

	It("applies one rule to every environment type it lists", func() {
		production := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
		staging := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "staging",
		})
		_ = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "development",
		})
		Expect(ruleStore.Create(ctx, resourcesRule(workspace.ID, "production", "staging"))).To(Succeed())

		resolved, err := appdefaults.Resolve(ctx, db, workspace.ID, "app-multi-env-types", bkmsapp.AppTypeTRPC)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Environments).To(HaveLen(2))

		specs := make(map[string]*appspec.AppSpec, len(resolved.Environments))
		for _, spec := range resolved.Environments {
			specs[spec.EnvName] = spec
		}
		Expect(specs).To(HaveKey(production.Name))
		Expect(specs).To(HaveKey(staging.Name))
		Expect(*specs[production.Name].Resources.Replicas).To(Equal(int32(2)))
		Expect(*specs[staging.Name].Resources.Replicas).To(Equal(int32(2)))
	})
})

func resourcesRule(workspaceID string, envTypes ...string) *appdefaults.Rule {
	return &appdefaults.Rule{
		WorkspaceID: workspaceID,
		ConfigType:  appspec.AppSpecSectionResources,
		EnvTypes:    envTypes,
		Spec: &appspec.AppSpec{
			Resources: resourcesSpec(2, "500m", "1", "1Gi", "2Gi"),
		},
	}
}

func devModeRule(workspaceID, envType string, enabled bool) *appdefaults.Rule {
	return &appdefaults.Rule{
		WorkspaceID: workspaceID,
		ConfigType:  appspec.AppSpecSectionDevMode,
		EnvTypes:    []string{envType},
		Spec: &appspec.AppSpec{
			DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(enabled)},
		},
	}
}
