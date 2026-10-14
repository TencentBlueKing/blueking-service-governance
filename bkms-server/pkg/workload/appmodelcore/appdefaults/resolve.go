package appdefaults

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
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
// appType is used to derive devMode paths, which are not configurable in rules.
func Resolve(
	ctx context.Context,
	db *mongo.Database,
	workspaceID, appID, appType string,
) (*ResolvedAppSpec, error) {
	rules, err := store.Of(db, RuleMeta).List(ctx, ByWorkspace(workspaceID))
	if err != nil {
		return nil, errors.Wrap(err, "list workspace application defaults")
	}
	environments, err := store.Of(db, envmodel.EnvMeta).
		List(ctx, envmodel.StdEnvsIn(workspaceID), envmodel.NewestFirst())
	if err != nil {
		return nil, errors.Wrap(err, "list standard environments")
	}

	defaultSpec := newPlatformDefaultSpec(appID)
	// Environment specs contain only sections configured for that environment
	// type. Environments without a matching rule need no initial AppSpec.
	var envSpecs []*appspec.AppSpec
	for _, env := range environments {
		spec := &appspec.AppSpec{AppID: appID, EnvName: env.Name}
		for _, rule := range rules {
			if lo.Contains(rule.EnvTypes, env.Type) {
				rule.applyTo(spec)
			}
		}
		if spec.DevMode != nil {
			// devMode rules only carry enabled; workPath/mountPath are derived from the app type.
			// Copy before setting paths because applyTo shares the section pointer with the rule.
			devMode := *spec.DevMode
			devMode.SetPathsByAppType(appType)
			spec.DevMode = &devMode
		}
		if spec.HasConfiguredSections() {
			envSpecs = append(envSpecs, spec)
		}
	}

	return &ResolvedAppSpec{Default: defaultSpec, Environments: envSpecs}, nil
}
