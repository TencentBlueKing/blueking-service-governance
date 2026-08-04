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
	probesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/probe"
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

	It("supports all eight sections in an environment type AppSpec", func() {
		_ = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "development",
		})
		production := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})

		configTypes := []appdefaults.ConfigType{
			appspec.AppSpecSectionResources,
			appspec.AppSpecSectionUpdateStrategy,
			appspec.AppSpecSectionDevMode,
			appspec.AppSpecSectionLifecycle,
			appspec.AppSpecSectionProbe,
			appspec.AppSpecSectionLabels,
			appspec.AppSpecSectionAnnotations,
			appspec.AppSpecSectionTkeRouteEni,
		}
		for _, configType := range configTypes {
			_, err := service.Create(ctx, workspace.ID, configType, appdefaults.RuleDefinition{
				EnvType: "production",
				Spec:    validRuleSpec(configType),
			})
			Expect(err).NotTo(HaveOccurred())
		}

		resolution, err := service.Resolve(ctx, workspace.ID, "app-all-sections")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolution.Default.AppID).To(Equal("app-all-sections"))
		Expect(resolution.Default.EnvName).To(Equal(appspec.DefaultEnvName))
		Expect(resolution.Default.Resources).NotTo(BeNil())
		Expect(resolution.Default.UpdateStrategy).NotTo(BeNil())
		Expect(resolution.Environments).To(HaveLen(1))
		productionSpec := resolution.Environments[0]
		Expect(productionSpec.EnvName).To(Equal(production.Name))
		Expect(productionSpec.Resources).NotTo(BeNil())
		Expect(productionSpec.UpdateStrategy).NotTo(BeNil())
		Expect(productionSpec.DevMode).NotTo(BeNil())
		Expect(productionSpec.Lifecycle).NotTo(BeNil())
		Expect(productionSpec.Probes).NotTo(BeNil())
		Expect(productionSpec.Labels).NotTo(BeNil())
		Expect(productionSpec.Annotations).NotTo(BeNil())
		Expect(productionSpec.TkeRouteEni).NotTo(BeNil())
	})

	It("creates only matching environment AppSpecs from environment type rules", func() {
		_ = dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "development",
		})
		production := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
		rule := labelsRule(workspace.ID, "production", "production")
		_, err := service.Create(ctx, workspace.ID, rule.ConfigType, definitionFromRule(rule))
		Expect(err).NotTo(HaveOccurred())

		resolution, err := service.Resolve(ctx, workspace.ID, "app-env-type-only")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolution.Default.AppID).To(Equal("app-env-type-only"))
		Expect(resolution.Default.EnvName).To(Equal(appspec.DefaultEnvName))
		Expect(resolution.Default.Resources).NotTo(BeNil())
		Expect(resolution.Default.UpdateStrategy).NotTo(BeNil())
		Expect(resolution.Environments).To(HaveLen(1))
		Expect(resolution.Environments[0].EnvName).To(Equal(production.Name))
		Expect(resolution.Environments[0].Labels.Labels).To(Equal(map[string]string{
			"source": "production",
		}))
	})

	It("applies platform defaults before environment type rules", func() {
		productionA := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})
		development := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "development",
		})
		productionB := dbfactory.EnvWithOpts(ctx, envService, &dbfactory.EnvOpts{
			WorkspaceID: workspace.ID,
			Type:        "production",
		})

		rules := []appdefaults.Rule{
			resourcesRule(workspace.ID, "production", 2, "500m", "1", "1Gi", "2Gi"),
			updateStrategyRule(workspace.ID, "production", "10%", "30%"),
			labelsRule(workspace.ID, "development", "development"),
		}
		for _, rule := range rules {
			_, err := service.Create(ctx, workspace.ID, rule.ConfigType, definitionFromRule(rule))
			Expect(err).NotTo(HaveOccurred())
		}

		resolution, err := service.Resolve(ctx, workspace.ID, "app-platform-defaults-and-env-types")
		Expect(err).NotTo(HaveOccurred())
		Expect(*resolution.Default.Resources.Replicas).To(Equal(int32(1)))
		Expect(*resolution.Default.UpdateStrategy.MaxUnavailable).To(Equal("25%"))
		Expect(resolution.Default.Labels).To(BeNil())
		Expect(resolution.Environments).To(HaveLen(3))

		specsByEnv := make(map[string]*appspec.AppSpec, len(resolution.Environments))
		for _, spec := range resolution.Environments {
			specsByEnv[spec.EnvName] = spec
		}

		for _, environment := range []*envmodel.Environment{productionA, productionB} {
			productionSpec := specsByEnv[environment.Name]
			Expect(productionSpec).NotTo(BeNil())
			Expect(*productionSpec.Resources.Replicas).To(Equal(int32(2)))
			Expect(*productionSpec.Resources.CPURequests).To(Equal("500m"))
			Expect(*productionSpec.Resources.MemoryLimits).To(Equal("2Gi"))
			Expect(*productionSpec.UpdateStrategy.MaxUnavailable).To(Equal("10%"))
			Expect(*productionSpec.UpdateStrategy.MaxSurge).To(Equal("30%"))
			Expect(productionSpec.Labels).To(BeNil())
		}

		developmentSpec := specsByEnv[development.Name]
		Expect(developmentSpec).NotTo(BeNil())
		Expect(developmentSpec.Resources).To(BeNil())
		Expect(developmentSpec.UpdateStrategy).To(BeNil())
		Expect(developmentSpec.Labels.Labels).To(Equal(map[string]string{
			"source": "development",
		}))
	})

	It("enforces one rule per workspace section and environment type", func() {
		rule := resourcesRule(
			workspace.ID,
			"production",
			2,
			"500m",
			"1",
			"1Gi",
			"2Gi",
		)
		_, err := service.Create(ctx, workspace.ID, rule.ConfigType, definitionFromRule(rule))
		Expect(err).NotTo(HaveOccurred())

		_, err = service.Create(ctx, workspace.ID, rule.ConfigType, definitionFromRule(rule))
		Expect(errors.Is(err, appdefaults.ErrRuleConflict)).To(BeTrue())

		otherSection := updateStrategyRule(
			workspace.ID,
			"production",
			"0",
			"50%",
		)
		_, err = service.Create(
			ctx,
			workspace.ID,
			otherSection.ConfigType,
			definitionFromRule(otherSection),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	It("lists rules for every environment type", func() {
		rules := []appdefaults.Rule{
			labelsRule(workspace.ID, "production", "production"),
			labelsRule(workspace.ID, "development", "development"),
			labelsRule(workspace.ID, "test", "test"),
			labelsRule(workspace.ID, "staging", "staging"),
		}
		for index := range rules {
			_, err := service.Create(
				ctx,
				workspace.ID,
				rules[index].ConfigType,
				definitionFromRule(rules[index]),
			)
			Expect(err).NotTo(HaveOccurred())
		}

		listed, err := service.List(ctx, workspace.ID, appspec.AppSpecSectionLabels)
		Expect(err).NotTo(HaveOccurred())
		Expect(listed).To(HaveLen(4))

		envTypes := make([]string, 0, len(listed))
		for _, rule := range listed {
			envTypes = append(envTypes, rule.EnvType)
		}
		Expect(envTypes).To(ConsistOf(
			"development",
			"test",
			"staging",
			"production",
		))
	})

	DescribeTable(
		"allows changing the target environment type of ordinary rules",
		func(configType appdefaults.ConfigType, spec *appspec.AppSpec) {
			created, err := service.Create(ctx, workspace.ID, configType, appdefaults.RuleDefinition{
				EnvType: "production",
				Spec:    spec,
			})
			Expect(err).NotTo(HaveOccurred())

			_, moved, err := service.Update(
				ctx,
				workspace.ID,
				configType,
				created.ID,
				appdefaults.RuleDefinition{
					EnvType: "test",
					Spec:    spec,
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(moved.EnvType).To(Equal("test"))

			_, _, err = service.Update(
				ctx,
				workspace.ID,
				configType,
				created.ID,
				appdefaults.RuleDefinition{
					Spec: spec,
				},
			)
			Expect(errors.Is(err, appdefaults.ErrInvalidRule)).To(BeTrue())
		},
		Entry(
			"for resources",
			appspec.AppSpecSectionResources,
			&appspec.AppSpec{Resources: resourcesSpec(2, "1", "2", "2Gi", "4Gi")},
		),
		Entry(
			"for update strategy",
			appspec.AppSpecSectionUpdateStrategy,
			&appspec.AppSpec{UpdateStrategy: updateStrategy("0", "50%")},
		),
	)

	It("allows an ordinary rule to change environment type and be deleted", func() {
		rule := labelsRule(workspace.ID, "development", "development")
		created, err := service.Create(ctx, workspace.ID, rule.ConfigType, definitionFromRule(rule))
		Expect(err).NotTo(HaveOccurred())

		_, updated, err := service.Update(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionLabels,
			created.ID,
			appdefaults.RuleDefinition{
				EnvType: "production",
				Spec:    created.Spec,
			},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.EnvType).To(Equal("production"))

		deleted, err := service.Delete(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionLabels,
			created.ID,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(deleted.ID).To(Equal(created.ID))
	})

	It("treats an ID from another section as not found", func() {
		rule := labelsRule(workspace.ID, "production", "production")
		created, err := service.Create(ctx, workspace.ID, rule.ConfigType, definitionFromRule(rule))
		Expect(err).NotTo(HaveOccurred())

		_, _, err = service.Update(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionAnnotations,
			created.ID,
			appdefaults.RuleDefinition{
				EnvType: "production",
				Spec: &appspec.AppSpec{
					Annotations: &appspec.AnnotationsSpec{
						Annotations: map[string]string{"owner": "platform"},
					},
				},
			},
		)
		Expect(errors.Is(err, appdefaults.ErrRuleNotFound)).To(BeTrue())

		_, err = service.Delete(
			ctx,
			workspace.ID,
			appspec.AppSpecSectionAnnotations,
			created.ID,
		)
		Expect(errors.Is(err, appdefaults.ErrRuleNotFound)).To(BeTrue())
	})

	It("rejects an unsupported environment type", func() {
		rule := resourcesRule(
			workspace.ID,
			"invalid-type",
			2,
			"1",
			"2",
			"2Gi",
			"4Gi",
		)
		_, err := service.Create(ctx, workspace.ID, rule.ConfigType, definitionFromRule(rule))
		Expect(errors.Is(err, appdefaults.ErrInvalidRule)).To(BeTrue())
	})

	It("requires an environment type when creating an ordinary rule", func() {
		rule := labelsRule(
			workspace.ID,
			"",
			"missing-env-type",
		)
		_, err := service.Create(ctx, workspace.ID, rule.ConfigType, definitionFromRule(rule))
		Expect(errors.Is(err, appdefaults.ErrInvalidRule)).To(BeTrue())
	})
})

func definitionFromRule(rule appdefaults.Rule) appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: rule.EnvType,
		Spec:    rule.Spec,
	}
}

func labelsRule(
	workspaceID string,
	envType string,
	value string,
) appdefaults.Rule {
	return appdefaults.Rule{
		WorkspaceID: workspaceID,
		ConfigType:  appspec.AppSpecSectionLabels,
		EnvType:     envType,
		Spec: &appspec.AppSpec{
			Labels: &appspec.LabelsSpec{
				Labels: map[string]string{"source": value},
			},
		},
	}
}

func validRuleSpec(configType appdefaults.ConfigType) *appspec.AppSpec {
	switch configType {
	case appspec.AppSpecSectionResources:
		return &appspec.AppSpec{
			Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi"),
		}
	case appspec.AppSpecSectionUpdateStrategy:
		return &appspec.AppSpec{
			UpdateStrategy: updateStrategy("25%", "25%"),
		}
	case appspec.AppSpecSectionDevMode:
		return &appspec.AppSpec{
			DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(true)},
		}
	case appspec.AppSpecSectionLifecycle:
		return &appspec.AppSpec{
			Lifecycle: &appspec.LifecycleSpec{
				TerminationGracePeriodSeconds: lo.ToPtr(int64(30)),
			},
		}
	case appspec.AppSpecSectionProbe:
		return &appspec.AppSpec{
			Probes: &appspec.ProbeSpec{
				Liveness: &probesection.Probe{
					Handler: &probesection.Handler{
						Type:    "EXEC",
						Command: []string{"true"},
					},
				},
			},
		}
	case appspec.AppSpecSectionLabels:
		return &appspec.AppSpec{
			Labels: &appspec.LabelsSpec{
				Labels: map[string]string{"team": "platform"},
			},
		}
	case appspec.AppSpecSectionAnnotations:
		return &appspec.AppSpec{
			Annotations: &appspec.AnnotationsSpec{
				Annotations: map[string]string{"owner": "platform"},
			},
		}
	case appspec.AppSpecSectionTkeRouteEni:
		return &appspec.AppSpec{
			TkeRouteEni: &appspec.TkeRouteEniSpec{Enabled: lo.ToPtr(false)},
		}
	default:
		return nil
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

func resourcesRule(
	workspaceID string,
	envType string,
	replicas int32,
	cpuRequests, cpuLimits, memoryRequests, memoryLimits string,
) appdefaults.Rule {
	return appdefaults.Rule{
		WorkspaceID: workspaceID,
		ConfigType:  appspec.AppSpecSectionResources,
		EnvType:     envType,
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

func updateStrategy(maxUnavailable, maxSurge string) *appspec.UpdateStrategySpec {
	return &appspec.UpdateStrategySpec{
		MaxUnavailable: lo.ToPtr(maxUnavailable),
		MaxSurge:       lo.ToPtr(maxSurge),
	}
}

func updateStrategyRule(
	workspaceID string,
	envType, maxUnavailable, maxSurge string,
) appdefaults.Rule {
	return appdefaults.Rule{
		WorkspaceID: workspaceID,
		ConfigType:  appspec.AppSpecSectionUpdateStrategy,
		EnvType:     envType,
		Spec: &appspec.AppSpec{
			UpdateStrategy: updateStrategy(maxUnavailable, maxSurge),
		},
	}
}
