package appdefaults

import (
	"fmt"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ValidateRule requires a valid environment type and exactly one complete,
// supported AppSpec section.
func ValidateRule(rule *Rule) error {
	if rule.ConfigType != appspec.AppSpecSectionResources && rule.ConfigType != appspec.AppSpecSectionDevMode {
		return fmt.Errorf("%w: unsupported config type %q", ErrInvalidRule, rule.ConfigType)
	}
	if !bkmsenv.IsValidEnvType(rule.EnvType) {
		return fmt.Errorf("%w: envType must be a valid environment type", ErrInvalidRule)
	}
	if rule.Spec == nil {
		return fmt.Errorf("%w: spec is required", ErrInvalidRule)
	}
	if rule.Spec.AppID != "" || rule.Spec.EnvName != "" {
		return fmt.Errorf("%w: spec identity must be empty in a workspace rule", ErrInvalidRule)
	}

	sections := configuredSections(rule.Spec)
	if len(sections) != 1 || sections[0] != rule.ConfigType {
		return fmt.Errorf("%w: spec must contain only the %s section", ErrInvalidRule, rule.ConfigType)
	}

	if err := validateSpecCompleteness(rule.ConfigType, rule.Spec); err != nil {
		return err
	}

	// Delegate to appspec's own validator for cross-field checks
	// (e.g. request ≤ limit).
	validationSpec := appspec.Clone(rule.Spec)
	validationSpec.AppID = "rule-validation"
	if err := appspec.Validate(validationSpec); err != nil {
		return fmt.Errorf("%w: invalid %s configuration: %v", ErrInvalidRule, rule.ConfigType, err)
	}
	return nil
}

func validateSpecCompleteness(configType ConfigType, spec *appspec.AppSpec) error {
	switch configType {
	case appspec.AppSpecSectionResources:
		r := spec.Resources
		if r.Replicas == nil || r.CPURequests == nil ||
			r.CPULimits == nil || r.MemoryRequests == nil ||
			r.MemoryLimits == nil {
			return fmt.Errorf("%w: resources rule must contain all fields", ErrInvalidRule)
		}
	case appspec.AppSpecSectionDevMode:
		if spec.DevMode.Enabled == nil {
			return fmt.Errorf("%w: devMode rule must contain enabled", ErrInvalidRule)
		}
	}
	return nil
}

func configuredSections(spec *appspec.AppSpec) []ConfigType {
	var sections []ConfigType
	if spec.Resources != nil {
		sections = append(sections, appspec.AppSpecSectionResources)
	}
	if spec.UpdateStrategy != nil {
		sections = append(sections, appspec.AppSpecSectionUpdateStrategy)
	}
	if spec.DevMode != nil {
		sections = append(sections, appspec.AppSpecSectionDevMode)
	}
	if spec.Lifecycle != nil {
		sections = append(sections, appspec.AppSpecSectionLifecycle)
	}
	if spec.Probes != nil {
		sections = append(sections, appspec.AppSpecSectionProbe)
	}
	if spec.Labels != nil {
		sections = append(sections, appspec.AppSpecSectionLabels)
	}
	if spec.Annotations != nil {
		sections = append(sections, appspec.AppSpecSectionAnnotations)
	}
	if spec.TkeRouteEni != nil {
		sections = append(sections, appspec.AppSpecSectionTkeRouteEni)
	}
	return sections
}
