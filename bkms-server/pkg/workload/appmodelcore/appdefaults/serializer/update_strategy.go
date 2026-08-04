package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspecserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// UpdateStrategySpecInput is a complete update-strategy section used by a rule.
type UpdateStrategySpecInput struct {
	MaxUnavailable *string `json:"maxUnavailable" binding:"required"`
	MaxSurge       *string `json:"maxSurge" binding:"required"`
}

func (input *UpdateStrategySpecInput) toModel() *appspec.UpdateStrategySpec {
	if input == nil {
		return nil
	}
	return &appspec.UpdateStrategySpec{
		MaxUnavailable: input.MaxUnavailable,
		MaxSurge:       input.MaxSurge,
	}
}

// UpdateStrategyRuleInput is the create and replace body for an update-strategy rule.
type UpdateStrategyRuleInput struct {
	EnvType string                   `json:"envType" binding:"required"`
	Spec    *UpdateStrategySpecInput `json:"spec" binding:"required"`
}

// ToModel converts the input to an update-strategy rule definition.
func (input UpdateStrategyRuleInput) ToModel() appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: input.EnvType,
		Spec:    &appspec.AppSpec{UpdateStrategy: input.Spec.toModel()},
	}
}

// UpdateStrategyRuleOutputObj is one update-strategy rule.
type UpdateStrategyRuleOutputObj struct {
	RuleOutputFields
	Spec *appspecserializer.AppSpecUpdateStrategyOutput `json:"spec"`
}

// FromModel fills an update-strategy rule response.
func (output *UpdateStrategyRuleOutputObj) FromModel(rule appdefaults.Rule) *UpdateStrategyRuleOutputObj {
	output.fromModel(rule)
	output.Spec = new(appspecserializer.AppSpecUpdateStrategyOutput).FromModel(rule.Spec.UpdateStrategy)
	return output
}

// ListUpdateStrategyRulesOutput is the list response for update-strategy rules.
type ListUpdateStrategyRulesOutput struct {
	Data []*UpdateStrategyRuleOutputObj `json:"data"`
}

// UpdateStrategyRuleOutput is the create and update response for an update-strategy rule.
type UpdateStrategyRuleOutput struct {
	Data *UpdateStrategyRuleOutputObj `json:"data"`
}
