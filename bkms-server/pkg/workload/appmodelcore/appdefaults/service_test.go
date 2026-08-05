package appdefaults_test

import (
	"context"
	"errors"

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

var _ = Describe("Application default rule service", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var store appdefaults.RuleStore
	var service *appdefaults.Service
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
			fx.Populate(&store, &envService, &envStore, &workspaceStore),
		)
		diApp.RequireStart()
		workspace = dbfactory.Workspace(ctx, workspaceStore)
		service = appdefaults.NewService(store, envStore)
	})

	AfterEach(func() {
		Expect(store.Drop(ctx)).To(Succeed())
		Expect(envStore.DeleteAll(ctx)).To(Succeed())
		Expect(workspaceStore.Delete(ctx, workspace.ID)).To(Succeed())
		bkmsworkspace.ResetLifecycleHooksForTest()
		bkmsenv.ResetDeleteHooksForTest()
		diApp.RequireStop()
	})

	It("uses platform defaults without persisting rules", func() {
		listed, err := service.List(ctx, workspace.ID, appspec.AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())
		Expect(listed).To(BeEmpty())

		resolution, err := service.Resolve(ctx, workspace.ID, "app-without-rules")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolution.Default.AppID).To(Equal("app-without-rules"))
		Expect(resolution.Default.EnvName).To(Equal(appspec.DefaultEnvName))
		Expect(*resolution.Default.Resources.Replicas).To(Equal(int32(1)))
		Expect(*resolution.Default.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(resolution.Environments).To(BeEmpty())

		persisted, err := store.List(ctx, workspace.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(persisted).To(BeEmpty())
	})

	It("stores resources and dev mode as independent section rules", func() {
		production := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})

		resourcesRule, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesDefinition("production", 2, "500m", "1", "1Gi", "2Gi"),
		)
		Expect(err).NotTo(HaveOccurred())
		devModeRule, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionDevMode,
			devModeDefinition("production", true),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(devModeRule.ID).NotTo(Equal(resourcesRule.ID))

		persisted, err := store.List(ctx, workspace.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(persisted).To(HaveLen(2))
		configTypes := []appdefaults.ConfigType{persisted[0].ConfigType, persisted[1].ConfigType}
		Expect(configTypes).To(ConsistOf(
			appspec.AppSpecSectionResources,
			appspec.AppSpecSectionDevMode,
		))

		resolution, err := service.Resolve(ctx, workspace.ID, "app-combined-defaults")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolution.Environments).To(HaveLen(1))
		spec := resolution.Environments[0]
		Expect(spec.AppID).To(Equal("app-combined-defaults"))
		Expect(spec.EnvName).To(Equal(production.Name))
		Expect(*spec.Resources.Replicas).To(Equal(int32(2)))
		Expect(*spec.DevMode.Enabled).To(BeTrue())
	})

	It("applies environment type rules to every matching environment", func() {
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

		_, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionDevMode,
			devModeDefinition("production", true),
		)
		Expect(err).NotTo(HaveOccurred())

		resolution, err := service.Resolve(ctx, workspace.ID, "app-matching-envs")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolution.Environments).To(HaveLen(2))

		specs := make(map[string]*appspec.AppSpec, len(resolution.Environments))
		for _, spec := range resolution.Environments {
			specs[spec.EnvName] = spec
		}
		Expect(specs).To(HaveKey(productionA.Name))
		Expect(specs).To(HaveKey(productionB.Name))
		Expect(*specs[productionA.Name].DevMode.Enabled).To(BeTrue())
		Expect(*specs[productionB.Name].DevMode.Enabled).To(BeTrue())
	})

	It("updates and deletes one section without overwriting the other", func() {
		_ = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
		resourcesRule, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesDefinition("production", 1, "1", "2", "2Gi", "4Gi"),
		)
		Expect(err).NotTo(HaveOccurred())
		devModeRule, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionDevMode,
			devModeDefinition("production", true),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(devModeRule.ID).NotTo(Equal(resourcesRule.ID))

		before, updated, err := service.Update(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesRule.ID,
			resourcesDefinition("production", 3, "2", "4", "4Gi", "8Gi"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(*before.Spec.Resources.Replicas).To(Equal(int32(1)))
		Expect(before.Spec.DevMode).To(BeNil())
		Expect(*updated.Spec.Resources.Replicas).To(Equal(int32(3)))
		Expect(updated.Spec.DevMode).To(BeNil())
		persisted, err := store.List(ctx, workspace.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(persisted).To(HaveLen(2))

		_, moved, err := service.Update(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesRule.ID,
			resourcesDefinition("test", 3, "2", "4", "4Gi", "8Gi"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(moved.EnvType).To(Equal("test"))

		deleted, err := service.Delete(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesRule.ID,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(deleted.Spec.Resources).NotTo(BeNil())
		Expect(deleted.Spec.DevMode).To(BeNil())

		resourcesRules, err := service.List(ctx, workspace.ID, appspec.AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())
		Expect(resourcesRules).To(BeEmpty())
		devModeRules, err := service.List(ctx, workspace.ID, appspec.AppSpecSectionDevMode)
		Expect(err).NotTo(HaveOccurred())
		Expect(devModeRules).To(HaveLen(1))
		Expect(devModeRules[0].Spec.Resources).To(BeNil())

		_, err = service.Delete(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionDevMode,
			devModeRule.ID,
		)
		Expect(err).NotTo(HaveOccurred())
		resolution, err := service.Resolve(ctx, workspace.ID, "app-after-deletes")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolution.Environments).To(BeEmpty())
	})

	It("enforces one section per workspace and environment type", func() {
		resourcesRule, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesDefinition("production", 1, "1", "2", "2Gi", "4Gi"),
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesDefinition("production", 2, "2", "4", "4Gi", "8Gi"),
		)
		Expect(errors.Is(err, appdefaults.ErrRuleConflict)).To(BeTrue())

		_, _, err = service.Update(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionDevMode,
			resourcesRule.ID,
			devModeDefinition("production", true),
		)
		Expect(errors.Is(err, appdefaults.ErrRuleNotFound)).To(BeTrue())

		devModeRule, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionDevMode,
			devModeDefinition("production", false),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(devModeRule.ID).NotTo(Equal(resourcesRule.ID))
	})

	It("lists only templates containing the requested section", func() {
		_, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesDefinition("production", 1, "1", "2", "2Gi", "4Gi"),
		)
		Expect(err).NotTo(HaveOccurred())
		_, err = service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionDevMode,
			devModeDefinition("development", false),
		)
		Expect(err).NotTo(HaveOccurred())

		resourcesRules, err := service.List(ctx, workspace.ID, appspec.AppSpecSectionResources)
		Expect(err).NotTo(HaveOccurred())
		Expect(resourcesRules).To(HaveLen(1))
		Expect(resourcesRules[0].EnvType).To(Equal("production"))

		devModeRules, err := service.List(ctx, workspace.ID, appspec.AppSpecSectionDevMode)
		Expect(err).NotTo(HaveOccurred())
		Expect(devModeRules).To(HaveLen(1))
		Expect(devModeRules[0].EnvType).To(Equal("development"))
	})

	It("rejects invalid environment types and incomplete sections", func() {
		_, err := service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionResources,
			resourcesDefinition("invalid", 1, "1", "2", "2Gi", "4Gi"),
		)
		Expect(errors.Is(err, appdefaults.ErrInvalidRule)).To(BeTrue())

		_, err = service.Create(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionDevMode,
			appdefaults.RuleDefinition{
				EnvType: "production",
				Spec:    &appspec.AppSpec{DevMode: &appspec.DevModeSpec{}},
			},
		)
		Expect(errors.Is(err, appdefaults.ErrInvalidRule)).To(BeTrue())
	})
})

func resourcesDefinition(
	envType string,
	replicas int32,
	cpuRequests, cpuLimits, memoryRequests, memoryLimits string,
) appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: envType,
		Spec: &appspec.AppSpec{
			Resources: resourcesSpec(
				replicas,
				cpuRequests,
				cpuLimits,
				memoryRequests,
				memoryLimits,
			),
		},
	}
}

func devModeDefinition(envType string, enabled bool) appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: envType,
		Spec: &appspec.AppSpec{
			DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(enabled)},
		},
	}
}

func resourcesSpec(
	replicas int32,
	cpuRequests, cpuLimits, memoryRequests, memoryLimits string,
) *appspec.ResourcesSpec {
	return &appspec.ResourcesSpec{
		Replicas:       lo.ToPtr(replicas),
		CPURequests:    lo.ToPtr(cpuRequests),
		CPULimits:      lo.ToPtr(cpuLimits),
		MemoryRequests: lo.ToPtr(memoryRequests),
		MemoryLimits:   lo.ToPtr(memoryLimits),
	}
}
