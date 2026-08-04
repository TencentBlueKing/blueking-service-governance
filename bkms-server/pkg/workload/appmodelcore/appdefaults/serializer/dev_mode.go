package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// DevModeSpecInput is the configurable part of a dev-mode section.
type DevModeSpecInput struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

func (input *DevModeSpecInput) toModel() *appspec.DevModeSpec {
	if input == nil {
		return nil
	}
	return &appspec.DevModeSpec{Enabled: input.Enabled}
}

// DevModeRuleInput is the create and replace body for a dev-mode rule.
type DevModeRuleInput struct {
	EnvType string            `json:"envType" binding:"required"`
	Spec    *DevModeSpecInput `json:"spec" binding:"required"`
}

// ToModel converts the input to a dev-mode rule definition.
func (input DevModeRuleInput) ToModel() appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: input.EnvType,
		Spec:    &appspec.AppSpec{DevMode: input.Spec.toModel()},
	}
}

// DevModeSpecOutput is the configurable part of a dev-mode section.
type DevModeSpecOutput struct {
	Enabled bool `json:"enabled"`
}

// DevModeRuleOutputObj is one dev-mode rule.
type DevModeRuleOutputObj struct {
	RuleOutputFields
	Spec *DevModeSpecOutput `json:"spec"`
}

// FromModel fills a dev-mode rule response.
func (output *DevModeRuleOutputObj) FromModel(rule appdefaults.Rule) *DevModeRuleOutputObj {
	output.fromModel(rule)
	output.Spec = &DevModeSpecOutput{}
	if rule.Spec.DevMode.Enabled != nil {
		output.Spec.Enabled = *rule.Spec.DevMode.Enabled
	}
	return output
}

// ListDevModeRulesOutput is the list response for dev-mode rules.
type ListDevModeRulesOutput struct {
	Data []*DevModeRuleOutputObj `json:"data"`
}

// DevModeRuleOutput is the create and update response for a dev-mode rule.
type DevModeRuleOutput struct {
	Data *DevModeRuleOutputObj `json:"data"`
}
