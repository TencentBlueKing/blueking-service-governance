package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspecserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// ProbeRuleInput is the create and replace body for a probe rule.
type ProbeRuleInput struct {
	EnvType string                               `json:"envType" binding:"required"`
	Spec    *appspecserializer.AppSpecProbeInput `json:"spec" binding:"required"`
}

// ToModel converts the input to a probe rule definition.
func (input ProbeRuleInput) ToModel() appdefaults.RuleDefinition {
	return appdefaults.RuleDefinition{
		EnvType: input.EnvType,
		Spec:    &appspec.AppSpec{Probes: input.Spec.ToModel()},
	}
}

// ProbeRuleOutputObj is one probe rule.
type ProbeRuleOutputObj struct {
	RuleOutputFields
	Spec *appspecserializer.AppSpecProbeOutput `json:"spec"`
}

// FromModel fills a probe rule response.
func (output *ProbeRuleOutputObj) FromModel(rule appdefaults.Rule) *ProbeRuleOutputObj {
	output.fromModel(rule)
	output.Spec = new(appspecserializer.AppSpecProbeOutput).FromModel(rule.Spec.Probes)
	return output
}

// ListProbeRulesOutput is the list response for probe rules.
type ListProbeRulesOutput struct {
	Data []*ProbeRuleOutputObj `json:"data"`
}

// ProbeRuleOutput is the create and update response for a probe rule.
type ProbeRuleOutput struct {
	Data *ProbeRuleOutputObj `json:"data"`
}
