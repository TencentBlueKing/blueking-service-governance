package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspecserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// ResourcesSpecInput is a complete resources section used by a rule.
type ResourcesSpecInput struct {
	Replicas       *int32  `json:"replicas" binding:"required,gte=0"`
	CPURequests    *string `json:"cpuRequests" binding:"required"`
	CPULimits      *string `json:"cpuLimits" binding:"required"`
	MemoryRequests *string `json:"memoryRequests" binding:"required"`
	MemoryLimits   *string `json:"memoryLimits" binding:"required"`
}

func (input *ResourcesSpecInput) toModel() *appspec.ResourcesSpec {
	if input == nil {
		return nil
	}
	return &appspec.ResourcesSpec{
		Replicas:       input.Replicas,
		CPURequests:    input.CPURequests,
		CPULimits:      input.CPULimits,
		MemoryRequests: input.MemoryRequests,
		MemoryLimits:   input.MemoryLimits,
	}
}

// ResourcesRuleInput is the create and replace body for a resources rule.
type ResourcesRuleInput struct {
	EnvType string              `json:"envType" binding:"required"`
	Spec    *ResourcesSpecInput `json:"spec" binding:"required"`
}

// ToModel converts the input to a resources rule definition.
func (input ResourcesRuleInput) ToModel() appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: input.EnvType,
		Spec:    &appspec.AppSpec{Resources: input.Spec.toModel()},
	}
}

// ResourcesRuleOutputObj is one resources rule.
type ResourcesRuleOutputObj struct {
	RuleOutputFields
	Spec *appspecserializer.AppSpecResourcesOutput `json:"spec"`
}

// FromModel fills a resources rule response.
func (output *ResourcesRuleOutputObj) FromModel(rule appdefaults.Rule) *ResourcesRuleOutputObj {
	output.fromModel(rule)
	output.Spec = new(appspecserializer.AppSpecResourcesOutput).FromModel(rule.Spec.Resources)
	return output
}

// ListResourcesRulesOutput is the list response for resources rules.
type ListResourcesRulesOutput struct {
	Data []*ResourcesRuleOutputObj `json:"data"`
}

// ResourcesRuleOutput is the create and update response for a resources rule.
type ResourcesRuleOutput struct {
	Data *ResourcesRuleOutputObj `json:"data"`
}
