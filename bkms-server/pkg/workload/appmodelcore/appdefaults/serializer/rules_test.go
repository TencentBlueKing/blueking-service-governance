package serializer_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspecserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

type conversionFactory func() (appdefaults.RuleDefinition, *appspec.AppSpec)

type outputFactory func(appdefaults.Rule) any

func pointerTo[T any](value T) *T {
	return &value
}

var _ = Describe("Rule serializers", func() {
	DescribeTable(
		"should convert rule inputs into definitions containing exactly one AppSpec section",
		func(factory conversionFactory) {
			definition, expectedSpec := factory()

			Expect(definition.EnvType).To(Equal("production"))
			Expect(definition.Spec).To(Equal(expectedSpec))
		},
		Entry("resources", conversionFactory(func() (
			appdefaults.RuleDefinition,
			*appspec.AppSpec,
		) {
			input := serializer.ResourcesRuleInput{
				EnvType: "production",
				Spec: &serializer.ResourcesSpecInput{
					Replicas:       pointerTo(int32(3)),
					CPURequests:    pointerTo("1500m"),
					CPULimits:      pointerTo("2"),
					MemoryRequests: pointerTo("2Gi"),
					MemoryLimits:   pointerTo("4Gi"),
				},
			}
			expected := &appspec.AppSpec{Resources: &appspec.ResourcesSpec{
				Replicas:       pointerTo(int32(3)),
				CPURequests:    pointerTo("1500m"),
				CPULimits:      pointerTo("2"),
				MemoryRequests: pointerTo("2Gi"),
				MemoryLimits:   pointerTo("4Gi"),
			}}
			return input.ToModel(), expected
		})),
		Entry("update strategy", conversionFactory(func() (
			appdefaults.RuleDefinition,
			*appspec.AppSpec,
		) {
			input := serializer.UpdateStrategyRuleInput{
				EnvType: "production",
				Spec: &serializer.UpdateStrategySpecInput{
					MaxUnavailable: pointerTo("10%"),
					MaxSurge:       pointerTo("20%"),
				},
			}
			expected := &appspec.AppSpec{UpdateStrategy: &appspec.UpdateStrategySpec{
				MaxUnavailable: pointerTo("10%"),
				MaxSurge:       pointerTo("20%"),
			}}
			return input.ToModel(), expected
		})),
		Entry("dev mode with disabled value", conversionFactory(func() (
			appdefaults.RuleDefinition,
			*appspec.AppSpec,
		) {
			input := serializer.DevModeRuleInput{
				EnvType: "production",
				Spec:    &serializer.DevModeSpecInput{Enabled: pointerTo(false)},
			}
			expected := &appspec.AppSpec{DevMode: &appspec.DevModeSpec{Enabled: pointerTo(false)}}
			return input.ToModel(), expected
		})),
		Entry("lifecycle", conversionFactory(func() (
			appdefaults.RuleDefinition,
			*appspec.AppSpec,
		) {
			sectionInput := &appspecserializer.AppSpecLifecycleInput{
				PostStart: &appspecserializer.LifecycleHandlerInput{
					Type: "EXEC",
					Exec: &appspecserializer.LifecycleExecActionInput{
						ShCommand: "touch /tmp/started",
					},
				},
				TerminationGracePeriodSeconds: pointerTo(int64(30)),
			}
			input := serializer.LifecycleRuleInput{
				EnvType: "production",
				Spec:    sectionInput,
			}
			expected := &appspec.AppSpec{Lifecycle: sectionInput.ToModel()}
			return input.ToModel(), expected
		})),
		Entry("probe", conversionFactory(func() (
			appdefaults.RuleDefinition,
			*appspec.AppSpec,
		) {
			sectionInput := &appspecserializer.AppSpecProbeInput{
				Liveness: &appspecserializer.ProbeInput{
					ProbeHandler:  &appspecserializer.ProbeHandlerInput{Type: "TCP", Port: 8080},
					PeriodSeconds: pointerTo(int32(10)),
				},
			}
			input := serializer.ProbeRuleInput{
				EnvType: "production",
				Spec:    sectionInput,
			}
			expected := &appspec.AppSpec{Probes: sectionInput.ToModel()}
			return input.ToModel(), expected
		})),
		Entry("labels", conversionFactory(func() (
			appdefaults.RuleDefinition,
			*appspec.AppSpec,
		) {
			sectionInput := &appspecserializer.AppSpecLabelsInput{
				Labels: map[string]string{"app.kubernetes.io/name": "demo"},
			}
			input := serializer.LabelsRuleInput{
				EnvType: "production",
				Spec:    sectionInput,
			}
			expected := &appspec.AppSpec{Labels: sectionInput.ToModel()}
			return input.ToModel(), expected
		})),
		Entry("annotations", conversionFactory(func() (
			appdefaults.RuleDefinition,
			*appspec.AppSpec,
		) {
			sectionInput := &appspecserializer.AppSpecAnnotationsInput{
				Annotations: map[string]string{"example.com/owner": "platform"},
			}
			input := serializer.AnnotationsRuleInput{
				EnvType: "production",
				Spec:    sectionInput,
			}
			expected := &appspec.AppSpec{Annotations: sectionInput.ToModel()}
			return input.ToModel(), expected
		})),
		Entry("TKE Route ENI with disabled value", conversionFactory(func() (
			appdefaults.RuleDefinition,
			*appspec.AppSpec,
		) {
			input := serializer.TkeRouteEniRuleInput{
				EnvType: "production",
				Spec:    &serializer.TkeRouteEniSpecInput{Enabled: pointerTo(false)},
			}
			expected := &appspec.AppSpec{TkeRouteEni: &appspec.TkeRouteEniSpec{Enabled: pointerTo(false)}}
			return input.ToModel(), expected
		})),
	)

	DescribeTable(
		"should expose only the public rule fields in JSON",
		func(spec *appspec.AppSpec, factory outputFactory, expectedSpec map[string]any) {
			rule := appdefaults.Rule{
				ID:          bson.NewObjectID(),
				WorkspaceID: "workspace-1",
				EnvType:     "production",
				Spec:        spec,
				CreatedAt:   time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
			}

			encoded, err := json.Marshal(factory(rule))
			Expect(err).NotTo(HaveOccurred())

			var payload map[string]any
			Expect(json.Unmarshal(encoded, &payload)).To(Succeed())
			Expect(payload).To(HaveLen(5))
			Expect(payload).To(HaveKeyWithValue("id", rule.ID.Hex()))
			Expect(payload).To(HaveKeyWithValue("envType", "production"))
			Expect(payload).To(HaveKey("createdAt"))
			Expect(payload).To(HaveKey("updatedAt"))
			Expect(payload).To(HaveKeyWithValue("spec", expectedSpec))
		},
		Entry(
			"resources",
			&appspec.AppSpec{Resources: &appspec.ResourcesSpec{
				Replicas:       pointerTo(int32(1)),
				CPURequests:    pointerTo("1"),
				CPULimits:      pointerTo("2"),
				MemoryRequests: pointerTo("2Gi"),
				MemoryLimits:   pointerTo("4Gi"),
			}},
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.ResourcesRuleOutputObj).FromModel(rule)
			}),
			map[string]any{
				"replicas":       float64(1),
				"cpuRequests":    "1",
				"cpuLimits":      "2",
				"memoryRequests": "2Gi",
				"memoryLimits":   "4Gi",
			},
		),
		Entry(
			"update strategy",
			&appspec.AppSpec{UpdateStrategy: &appspec.UpdateStrategySpec{
				MaxUnavailable: pointerTo("25%"),
				MaxSurge:       pointerTo("25%"),
			}},
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.UpdateStrategyRuleOutputObj).FromModel(rule)
			}),
			map[string]any{"maxUnavailable": "25%", "maxSurge": "25%"},
		),
		Entry(
			"dev mode with disabled value",
			&appspec.AppSpec{DevMode: &appspec.DevModeSpec{Enabled: pointerTo(false)}},
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.DevModeRuleOutputObj).FromModel(rule)
			}),
			map[string]any{"enabled": false},
		),
		Entry(
			"lifecycle",
			&appspec.AppSpec{Lifecycle: &appspec.LifecycleSpec{}},
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.LifecycleRuleOutputObj).FromModel(rule)
			}),
			map[string]any{
				"postStart":                     nil,
				"preStop":                       nil,
				"terminationGracePeriodSeconds": nil,
			},
		),
		Entry(
			"probe",
			&appspec.AppSpec{Probes: &appspec.ProbeSpec{}},
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.ProbeRuleOutputObj).FromModel(rule)
			}),
			map[string]any{"liveness": nil, "readiness": nil, "startup": nil},
		),
		Entry(
			"labels",
			&appspec.AppSpec{Labels: &appspec.LabelsSpec{
				Labels: map[string]string{"app": "demo"},
			}},
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.LabelsRuleOutputObj).FromModel(rule)
			}),
			map[string]any{"labels": map[string]any{"app": "demo"}},
		),
		Entry(
			"annotations",
			&appspec.AppSpec{Annotations: &appspec.AnnotationsSpec{
				Annotations: map[string]string{"example.com/owner": "platform"},
			}},
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.AnnotationsRuleOutputObj).FromModel(rule)
			}),
			map[string]any{"annotations": map[string]any{"example.com/owner": "platform"}},
		),
		Entry(
			"TKE Route ENI with disabled value",
			&appspec.AppSpec{TkeRouteEni: &appspec.TkeRouteEniSpec{Enabled: pointerTo(false)}},
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.TkeRouteEniRuleOutputObj).FromModel(rule)
			}),
			map[string]any{"enabled": false},
		),
	)
})
