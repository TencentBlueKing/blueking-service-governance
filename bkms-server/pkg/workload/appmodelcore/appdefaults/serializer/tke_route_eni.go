package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// TkeRouteEniSpecInput is the configurable TKE Route ENI section.
type TkeRouteEniSpecInput struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

func (input *TkeRouteEniSpecInput) toModel() *appspec.TkeRouteEniSpec {
	if input == nil {
		return nil
	}
	return &appspec.TkeRouteEniSpec{Enabled: input.Enabled}
}

// TkeRouteEniRuleInput is the create and replace body for a TKE Route ENI rule.
type TkeRouteEniRuleInput struct {
	EnvType string                `json:"envType" binding:"required"`
	Spec    *TkeRouteEniSpecInput `json:"spec" binding:"required"`
}

// ToModel converts the input to a TKE Route ENI rule definition.
func (input TkeRouteEniRuleInput) ToModel() appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: input.EnvType,
		Spec:    &appspec.AppSpec{TkeRouteEni: input.Spec.toModel()},
	}
}

// TkeRouteEniSpecOutput is the configurable TKE Route ENI section.
type TkeRouteEniSpecOutput struct {
	Enabled bool `json:"enabled"`
}

// TkeRouteEniRuleOutputObj is one TKE Route ENI rule.
type TkeRouteEniRuleOutputObj struct {
	RuleOutputFields
	Spec *TkeRouteEniSpecOutput `json:"spec"`
}

// FromModel fills a TKE Route ENI rule response.
func (output *TkeRouteEniRuleOutputObj) FromModel(rule appdefaults.Rule) *TkeRouteEniRuleOutputObj {
	output.fromModel(rule)
	output.Spec = &TkeRouteEniSpecOutput{}
	if rule.Spec.TkeRouteEni.Enabled != nil {
		output.Spec.Enabled = *rule.Spec.TkeRouteEni.Enabled
	}
	return output
}

// ListTkeRouteEniRulesOutput is the list response for TKE Route ENI rules.
type ListTkeRouteEniRulesOutput struct {
	Data []*TkeRouteEniRuleOutputObj `json:"data"`
}

// TkeRouteEniRuleOutput is the create and update response for a TKE Route ENI rule.
type TkeRouteEniRuleOutput struct {
	Data *TkeRouteEniRuleOutputObj `json:"data"`
}
