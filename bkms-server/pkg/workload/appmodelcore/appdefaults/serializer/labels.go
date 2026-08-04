package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspecserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// LabelsRuleInput is the create and replace body for a labels rule.
type LabelsRuleInput struct {
	EnvType string                                `json:"envType" binding:"required"`
	Spec    *appspecserializer.AppSpecLabelsInput `json:"spec" binding:"required"`
}

// ToModel converts the input to a labels rule definition.
func (input LabelsRuleInput) ToModel() appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: input.EnvType,
		Spec:    &appspec.AppSpec{Labels: input.Spec.ToModel()},
	}
}

// LabelsRuleOutputObj is one labels rule.
type LabelsRuleOutputObj struct {
	RuleOutputFields
	Spec *appspecserializer.AppSpecLabelsOutput `json:"spec"`
}

// FromModel fills a labels rule response.
func (output *LabelsRuleOutputObj) FromModel(rule appdefaults.Rule) *LabelsRuleOutputObj {
	output.fromModel(rule)
	output.Spec = new(appspecserializer.AppSpecLabelsOutput).FromModel(rule.Spec.Labels)
	return output
}

// ListLabelsRulesOutput is the list response for labels rules.
type ListLabelsRulesOutput struct {
	Data []*LabelsRuleOutputObj `json:"data"`
}

// LabelsRuleOutput is the create and update response for a labels rule.
type LabelsRuleOutput struct {
	Data *LabelsRuleOutputObj `json:"data"`
}
