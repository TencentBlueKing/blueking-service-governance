package appdefaults_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Application default rule model", func() {
	DescribeTable(
		"accepts supported AppSpec sections",
		func(configType appdefaults.ConfigType, spec *appspec.AppSpec) {
			rule := &appdefaults.Rule{
				WorkspaceID: "workspace-supported-sections",
				ConfigType:  configType,
				EnvType:     "production",
				Spec:        spec,
			}
			Expect(appdefaults.ValidateRule(rule)).To(Succeed())
		},
		Entry(
			"resources",
			appspec.AppSpecSectionResources,
			&appspec.AppSpec{Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi")},
		),
		Entry(
			"dev mode with an explicit false value",
			appspec.AppSpecSectionDevMode,
			&appspec.AppSpec{DevMode: &appspec.DevModeSpec{Enabled: lo.ToPtr(false)}},
		),
	)

	It("rejects unsupported sections", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-unsupported-section",
			ConfigType:  appspec.AppSpecSectionLabels,
			EnvType:     "production",
			Spec: &appspec.AppSpec{
				Labels: &appspec.LabelsSpec{Labels: map[string]string{"team": "platform"}},
			},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("rejects a mismatched or additional section", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-section-mismatch",
			ConfigType:  appspec.AppSpecSectionResources,
			EnvType:     "production",
			Spec: &appspec.AppSpec{
				Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi"),
				DevMode:   &appspec.DevModeSpec{Enabled: lo.ToPtr(true)},
			},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("requires every resources field", func() {
		resources := resourcesSpec(1, "1", "2", "2Gi", "4Gi")
		resources.CPURequests = nil
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-incomplete-resources",
			ConfigType:  appspec.AppSpecSectionResources,
			EnvType:     "production",
			Spec:        &appspec.AppSpec{Resources: resources},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("requires an explicit dev-mode value", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-incomplete-dev-mode",
			ConfigType:  appspec.AppSpecSectionDevMode,
			EnvType:     "production",
			Spec:        &appspec.AppSpec{DevMode: &appspec.DevModeSpec{}},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})

	It("requires an environment type and empty AppSpec identity", func() {
		rule := &appdefaults.Rule{
			WorkspaceID: "workspace-invalid-identity",
			ConfigType:  appspec.AppSpecSectionResources,
			Spec: &appspec.AppSpec{
				Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi"),
			},
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())

		rule.EnvType = "production"
		rule.Spec = &appspec.AppSpec{
			AppID:     "application",
			Resources: resourcesSpec(1, "1", "2", "2Gi", "4Gi"),
		}
		Expect(errors.Is(
			appdefaults.ValidateRule(rule),
			appdefaults.ErrInvalidRule,
		)).To(BeTrue())
	})
})

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
