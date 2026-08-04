package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspecserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// AnnotationsRuleInput is the create and replace body for an annotations rule.
type AnnotationsRuleInput struct {
	EnvType string                                     `json:"envType" binding:"required"`
	Spec    *appspecserializer.AppSpecAnnotationsInput `json:"spec" binding:"required"`
}

// ToModel converts the input to an annotations rule definition.
func (input AnnotationsRuleInput) ToModel() appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: input.EnvType,
		Spec:    &appspec.AppSpec{Annotations: input.Spec.ToModel()},
	}
}

// AnnotationsRuleOutputObj is one annotations rule.
type AnnotationsRuleOutputObj struct {
	RuleOutputFields
	Spec *appspecserializer.AppSpecAnnotationsOutput `json:"spec"`
}

// FromModel fills an annotations rule response.
func (output *AnnotationsRuleOutputObj) FromModel(rule appdefaults.Rule) *AnnotationsRuleOutputObj {
	output.fromModel(rule)
	output.Spec = new(appspecserializer.AppSpecAnnotationsOutput).FromModel(rule.Spec.Annotations)
	return output
}

// ListAnnotationsRulesOutput is the list response for annotations rules.
type ListAnnotationsRulesOutput struct {
	Data []*AnnotationsRuleOutputObj `json:"data"`
}

// AnnotationsRuleOutput is the create and update response for an annotations rule.
type AnnotationsRuleOutput struct {
	Data *AnnotationsRuleOutputObj `json:"data"`
}
