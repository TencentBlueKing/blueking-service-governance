package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspecserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// LifecycleRuleInput is the create and replace body for a lifecycle rule.
type LifecycleRuleInput struct {
	EnvType string                                   `json:"envType" binding:"required"`
	Spec    *appspecserializer.AppSpecLifecycleInput `json:"spec" binding:"required"`
}

// ToModel converts the input to a lifecycle rule definition.
func (input LifecycleRuleInput) ToModel() appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: input.EnvType,
		Spec:    &appspec.AppSpec{Lifecycle: input.Spec.ToModel()},
	}
}

// LifecycleRuleOutputObj is one lifecycle rule.
type LifecycleRuleOutputObj struct {
	RuleOutputFields
	Spec *appspecserializer.AppSpecLifecycleOutput `json:"spec"`
}

// FromModel fills a lifecycle rule response.
func (output *LifecycleRuleOutputObj) FromModel(rule appdefaults.Rule) *LifecycleRuleOutputObj {
	output.fromModel(rule)
	output.Spec = new(appspecserializer.AppSpecLifecycleOutput).FromModel(rule.Spec.Lifecycle)
	return output
}

// ListLifecycleRulesOutput is the list response for lifecycle rules.
type ListLifecycleRulesOutput struct {
	Data []*LifecycleRuleOutputObj `json:"data"`
}

// LifecycleRuleOutput is the create and update response for a lifecycle rule.
type LifecycleRuleOutput struct {
	Data *LifecycleRuleOutputObj `json:"data"`
}
