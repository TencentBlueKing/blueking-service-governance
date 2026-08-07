package appdefaults

import (
	"context"
	"fmt"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// ResolvedAppSpec contains the AppSpecs selected for one new application.
type ResolvedAppSpec struct {
	// Default always contains the platform resources and update-strategy
	// baselines.
	Default appspec.AppSpec
	// Environments contains only standard environments whose type has at least
	// one rule. Platform defaults do not create environment entries.
	Environments []*appspec.AppSpec
}

// Resolve builds the AppSpecs used to initialize a new AppModel application.
func Resolve(
	ctx context.Context,
	ruleStore RuleStore,
	envStore envmodel.EnvironmentStore,
	workspaceID, appID string,
) (*ResolvedAppSpec, error) {
	rules, err := ruleStore.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace application defaults: %w", err)
	}
	environments, err := envStore.ListStdEnvs(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list standard environments: %w", err)
	}

	defaultSpec := newPlatformDefaultSpec(appID)
	// Environment specs contain only sections configured for that environment
	// type. Environments without a matching rule need no initial AppSpec.
	var envSpecs []*appspec.AppSpec
	for _, env := range environments {
		spec := &appspec.AppSpec{AppID: appID, EnvName: env.Name}
		for _, rule := range rules {
			if rule.EnvType == env.Type {
				rule.applyTo(spec)
			}
		}
		if spec.HasConfiguredSections() {
			envSpecs = append(envSpecs, spec)
		}
	}

	return &ResolvedAppSpec{Default: defaultSpec, Environments: envSpecs}, nil
}
