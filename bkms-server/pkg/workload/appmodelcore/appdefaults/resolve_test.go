package appdefaults_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Application default resolution", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var ruleStore appdefaults.RuleStore
	var envService *bkmsenv.EnvService
	var envStore envmodel.EnvironmentStore
	var workspaceStore bkmsworkspace.WorkspaceStore
	var workspace *bkmsworkspace.Workspace

	BeforeEach(func() {
		ctx = context.Background()
		bkmsworkspace.ResetLifecycleHooksForTest()
		bkmsenv.ResetDeleteHooksForTest()

		diApp = fxtest.New(
			GinkgoT(),
			appdefaults.FxModule,
			bkmsenv.FxModule,
			bkmsworkspace.FxModule,
			fx.Populate(&ruleStore, &envService, &envStore, &workspaceStore),
		)
		diApp.RequireStart()
		workspace = dbfactory.Workspace(ctx, workspaceStore)
	})

	AfterEach(func() {
		Expect(ruleStore.Drop(ctx)).To(Succeed())
		Expect(envStore.DeleteAll(ctx)).To(Succeed())
		Expect(workspaceStore.Delete(ctx, workspace.ID)).To(Succeed())
		bkmsworkspace.ResetLifecycleHooksForTest()
		bkmsenv.ResetDeleteHooksForTest()
		diApp.RequireStop()
	})

	It("uses platform defaults without persisting rules", func() {
		resolved, err := appdefaults.Resolve(ctx, ruleStore, envStore, workspace.ID, "app-without-rules")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved.Default.AppID).To(Equal("app-without-rules"))
		Expect(resolved.Default.EnvName).To(Equal(appspec.DefaultEnvName))
		Expect(*resolved.Default.Resources.Replicas).To(Equal(int32(1)))
		Expect(*resolved.Default.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(resolved.Environments).To(BeEmpty())

		persisted, err := ruleStore.List(ctx, workspace.ID)
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

		resolved, err := appdefaults.Resolve(ctx, ruleStore, envStore, workspace.ID, "app-combined-defaults")
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

		resolved, err := appdefaults.Resolve(ctx, ruleStore, envStore, workspace.ID, "app-matching-envs")
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
})

func resourcesRule(workspaceID, envType string) *appdefaults.Rule {
	return &appdefaults.Rule{
		WorkspaceID: workspaceID,
		ConfigType:  appspec.AppSpecSectionResources,
		EnvType:     envType,
		Spec: &appspec.AppSpec{
			Resources: resourcesSpec(2, "500m", "1", "1Gi", "2Gi"),
		},
	}
}

func devModeRule(workspaceID, envType string, enabled bool) *appdefaults.Rule {
	return &appdefaults.Rule{
		WorkspaceID: workspaceID,
		ConfigType:  appspec.AppSpecSectionDevMode,
		EnvType:     envType,
		Spec: &appspec.AppSpec{
			DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(enabled)},
		},
	}
}
