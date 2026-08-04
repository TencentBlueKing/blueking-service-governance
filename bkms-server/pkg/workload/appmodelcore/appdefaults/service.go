package appdefaults

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// Service owns application-default rule lifecycle policy.
type Service struct {
	store    RuleStore
	envStore envmodel.EnvironmentStore
}

// Resolution contains the AppSpecs selected for one new application.
type Resolution struct {
	// Default always contains the platform resources and update-strategy
	// baselines.
	Default appspec.AppSpec
	// Environments contains only standard environments whose type has at least
	// one rule. Platform defaults do not create environment entries.
	Environments []*appspec.AppSpec
}

// NewService creates an application-default rule service.
func NewService(store RuleStore, envStore envmodel.EnvironmentStore) *Service {
	return &Service{store: store, envStore: envStore}
}

// List returns one section's workspace rules.
func (s *Service) List(ctx context.Context, workspaceID string, configType ConfigType) ([]Rule, error) {
	return s.store.ListByConfigType(ctx, workspaceID, configType)
}

// Resolve builds the AppSpecs used to initialize a new AppModel application.
func (s *Service) Resolve(ctx context.Context, workspaceID, appID string) (*Resolution, error) {
	rules, err := s.store.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace application defaults: %w", err)
	}
	environments, err := s.envStore.ListStdEnvs(ctx, workspaceID)
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

	return &Resolution{Default: defaultSpec, Environments: envSpecs}, nil
}

// Create validates and creates a rule, returning the persisted value.
func (s *Service) Create(
	ctx context.Context,
	workspaceID string,
	configType ConfigType,
	definition RuleDefinition,
) (*Rule, error) {
	rule := &Rule{
		WorkspaceID: workspaceID,
		ConfigType:  configType,
		EnvType:     definition.EnvType,
		Spec:        definition.Spec,
	}
	if err := ValidateRule(rule); err != nil {
		return nil, err
	}
	if err := s.store.Create(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// Update replaces a rule's environment type and section configuration.
func (s *Service) Update(
	ctx context.Context,
	workspaceID string,
	configType ConfigType,
	id bson.ObjectID,
	definition RuleDefinition,
) (*Rule, *Rule, error) {
	current, err := s.store.Get(ctx, workspaceID, configType, id)
	if err != nil {
		return nil, nil, err
	}
	updated := *current
	updated.EnvType = definition.EnvType
	updated.Spec = definition.Spec
	if err = ValidateRule(&updated); err != nil {
		return nil, nil, err
	}
	if err = s.store.Update(ctx, workspaceID, configType, &updated); err != nil {
		return nil, nil, err
	}
	return current, &updated, nil
}

// Delete deletes a rule and returns its previous value for audit.
func (s *Service) Delete(
	ctx context.Context,
	workspaceID string,
	configType ConfigType,
	id bson.ObjectID,
) (*Rule, error) {
	current, err := s.store.Get(ctx, workspaceID, configType, id)
	if err != nil {
		return nil, err
	}
	if err = s.store.Delete(ctx, workspaceID, configType, id); err != nil {
		return nil, err
	}
	return current, nil
}
