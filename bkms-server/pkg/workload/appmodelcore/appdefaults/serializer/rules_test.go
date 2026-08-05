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
)

type conversionFactory func() (appdefaults.RuleDefinition, *appspec.AppSpec)

type outputFactory func(appdefaults.Rule) any

func pointerTo[T any](value T) *T {
	return &value
}

var _ = Describe("Rule serializers", func() {
	DescribeTable(
		"converts supported rule inputs into one AppSpec section",
		func(factory conversionFactory) {
			definition, expectedSpec := factory()

			Expect(definition.EnvType).To(Equal("production"))
			Expect(definition.Spec).To(Equal(expectedSpec))
		},
		Entry("resources", conversionFactory(func() (appdefaults.RuleDefinition, *appspec.AppSpec) {
			input := serializer.ResourcesRuleInput{
				EnvType: "production",
				Spec: &serializer.ResourcesSpecInput{
					Replicas:       pointerTo(int32(0)),
					CPURequests:    pointerTo("1500m"),
					CPULimits:      pointerTo("2"),
					MemoryRequests: pointerTo("2Gi"),
					MemoryLimits:   pointerTo("4Gi"),
				},
			}
			expected := &appspec.AppSpec{Resources: &appspec.ResourcesSpec{
				Replicas:       pointerTo(int32(0)),
				CPURequests:    pointerTo("1500m"),
				CPULimits:      pointerTo("2"),
				MemoryRequests: pointerTo("2Gi"),
				MemoryLimits:   pointerTo("4Gi"),
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
	)

	DescribeTable(
		"exposes only the requested section and public rule fields",
		func(factory outputFactory, expectedSpec map[string]any) {
			rule := appdefaults.Rule{
				ID:          bson.NewObjectID(),
				WorkspaceID: "workspace-1",
				EnvType:     "production",
				Spec: &appspec.AppSpec{
					Resources: &appspec.ResourcesSpec{
						Replicas:       pointerTo(int32(1)),
						CPURequests:    pointerTo("1"),
						CPULimits:      pointerTo("2"),
						MemoryRequests: pointerTo("2Gi"),
						MemoryLimits:   pointerTo("4Gi"),
					},
					DevMode: &appspec.DevModeSpec{Enabled: pointerTo(false)},
				},
				CreatedAt: time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
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
			"dev mode with disabled value",
			outputFactory(func(rule appdefaults.Rule) any {
				return new(serializer.DevModeRuleOutputObj).FromModel(rule)
			}),
			map[string]any{"enabled": false},
		),
	)
})
