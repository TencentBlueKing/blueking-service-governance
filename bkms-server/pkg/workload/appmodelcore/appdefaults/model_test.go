package appdefaults_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	probesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/probe"
)

var _ = Describe("Application default rule model", func() {
	validSections := map[appdefaults.ConfigType]*appspec.AppSpec{
		appspec.AppSpecSectionResources: {
			Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi"),
		},
		appspec.AppSpecSectionUpdateStrategy: {
			UpdateStrategy: updateStrategy("25%", "25%"),
		},
		appspec.AppSpecSectionDevMode: {
			DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(true)},
		},
		appspec.AppSpecSectionLifecycle: {
			Lifecycle: &appspec.LifecycleSpec{TerminationGracePeriodSeconds: lo.ToPtr(int64(30))},
		},
		appspec.AppSpecSectionProbe: {
			Probes: &appspec.ProbeSpec{
				Liveness: &probesection.Probe{
					Handler: &probesection.Handler{Type: "EXEC", Command: []string{"true"}},
				},
			},
		},
		appspec.AppSpecSectionLabels: {
			Labels: &appspec.LabelsSpec{Labels: map[string]string{"team": "platform"}},
		},
		appspec.AppSpecSectionAnnotations: {
			Annotations: &appspec.AnnotationsSpec{Annotations: map[string]string{"owner": "platform"}},
		},
		appspec.AppSpecSectionTkeRouteEni: {
			TkeRouteEni: &appspec.TkeRouteEniSpec{Enabled: lo.ToPtr(false)},
		},
	}

	It("accepts every supported AppSpec section", func() {
		Expect(validSections).To(HaveLen(8))
		for configType, spec := range validSections {
			rule := &appdefaults.Rule{
				WorkspaceID: "workspace-all-sections",
				ConfigType:  configType,
				EnvType:     "production",
				Spec:        spec,
			}
			Expect(appdefaults.ValidateRule(rule)).To(Succeed(), "section %s", configType)
		}
	})

	It("rejects a section that does not match configType", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-section-mismatch",
			ConfigType:  appspec.AppSpecSectionLabels,
			EnvType:     "production",
			Spec:        validSections[appspec.AppSpecSectionAnnotations],
		}
		Expect(errors.Is(appdefaults.ValidateRule(rule), appdefaults.ErrInvalidRule)).To(BeTrue())
	})

	It("rejects more than one section in a rule", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-multiple-sections",
			ConfigType:  appspec.AppSpecSectionLabels,
			EnvType:     "production",
			Spec: &appspec.AppSpec{
				Labels:      validSections[appspec.AppSpecSectionLabels].Labels,
				Annotations: validSections[appspec.AppSpecSectionAnnotations].Annotations,
			},
		}
		Expect(errors.Is(appdefaults.ValidateRule(rule), appdefaults.ErrInvalidRule)).To(BeTrue())
	})

	It("requires every resources and update strategy field", func() {
		resources := resourcesSpec(1, "1", "2", "2Gi", "4Gi")
		resources.CPURequests = nil
		resourceRule := &appdefaults.Rule{
			WorkspaceID: "workspace-complete-sections",
			ConfigType:  appspec.AppSpecSectionResources,
			Spec:        &appspec.AppSpec{Resources: resources},
		}
		Expect(errors.Is(appdefaults.ValidateRule(resourceRule), appdefaults.ErrInvalidRule)).To(BeTrue())

		strategyRule := &appdefaults.Rule{
			WorkspaceID: "workspace-complete-sections",
			ConfigType:  appspec.AppSpecSectionUpdateStrategy,
			Spec: &appspec.AppSpec{
				UpdateStrategy: &appspec.UpdateStrategySpec{MaxUnavailable: lo.ToPtr("25%")},
			},
		}
		Expect(errors.Is(appdefaults.ValidateRule(strategyRule), appdefaults.ErrInvalidRule)).To(BeTrue())
	})

	It("requires explicit boolean values", func() {
		devModeRule := &appdefaults.Rule{
			WorkspaceID: "workspace-boolean-sections",
			ConfigType:  appspec.AppSpecSectionDevMode,
			EnvType:     "production",
			Spec:        &appspec.AppSpec{DevMode: &appspec.DevModeSpec{}},
		}
		Expect(errors.Is(appdefaults.ValidateRule(devModeRule), appdefaults.ErrInvalidRule)).To(BeTrue())

		tkeRouteEniRule := &appdefaults.Rule{
			WorkspaceID: "workspace-boolean-sections",
			ConfigType:  appspec.AppSpecSectionTkeRouteEni,
			EnvType:     "production",
			Spec:        &appspec.AppSpec{TkeRouteEni: &appspec.TkeRouteEniSpec{}},
		}
		Expect(errors.Is(appdefaults.ValidateRule(tkeRouteEniRule), appdefaults.ErrInvalidRule)).To(BeTrue())
	})

	It("requires an environment type for every rule", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-without-env-type",
			ConfigType:  appspec.AppSpecSectionResources,
			Spec:        validSections[appspec.AppSpecSectionResources],
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})
})
