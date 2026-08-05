// Package migrate migrates legacy tkeRouteEni component mounts onto AppSpec sections.
package migrate

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

const tkeRouteEniComponentName = "tkeRouteEni"

// Result is a short summary of a dry-run or applied migration.
type Result struct {
	// DryRun indicates whether this run only reported changes without writing.
	DryRun bool `yaml:"dryRun"`
	// AppSpecWrites is the number of AppSpec tkeRouteEni.enabled=true writes performed (or planned).
	AppSpecWrites int `yaml:"appSpecWrites"`
	// AppComponentsRemoved is the number of app_models component instances/refs removed (or planned).
	AppComponentsRemoved int `yaml:"appComponentsRemoved"`
	// WorkspaceComponentsRemoved is the number of workspace_components documents removed (or planned).
	WorkspaceComponentsRemoved int `yaml:"workspaceComponentsRemoved"`
	// AppIDs lists application IDs that had tkeRouteEni mounts touched by this run.
	AppIDs []string `yaml:"appIDs,omitempty"`
}

// Migrator migrates tkeRouteEni component mounts onto AppSpec sections.
type Migrator struct {
	db            *mongo.Database
	appStore      bkmsapp.ApplicationStore
	appSpecStore  appspec.AppSpecStore
	appModelStore appmodel.AppModelStore
	wsCompStore   workspace.WorkspaceCompsStore
	compDefStore  component.ComponentDefStore
}

// New creates a tkeRouteEni component migrator with reference-count hooks wired.
func New(
	db *mongo.Database,
	appStore bkmsapp.ApplicationStore,
	appSpecStore appspec.AppSpecStore,
	appModelStore appmodel.AppModelStore,
	wsCompStore workspace.WorkspaceCompsStore,
	compDefStore component.ComponentDefStore,
) *Migrator {
	appModelStore.SetComponentHooks(appmodel.NewComponentRefCountHooks(compDefStore))
	wsCompStore.SetComponentHooks(workspace.NewComponentRefCountHooks(compDefStore))
	return &Migrator{
		db:            db,
		appStore:      appStore,
		appSpecStore:  appSpecStore,
		appModelStore: appModelStore,
		wsCompStore:   wsCompStore,
		compDefStore:  compDefStore,
	}
}

type wsCompKey struct {
	workspaceID string
	name        string
}

func makeWsCompKey(workspaceID, name string) wsCompKey {
	return wsCompKey{workspaceID: workspaceID, name: name}
}

// Run migrates tkeRouteEni mounts to AppSpec. dryRun=true only reports counts.
func (m *Migrator) Run(
	ctx context.Context, dryRun bool,
) (Result, error) {
	result := Result{DryRun: dryRun}
	enabled := true
	enabledSpec := &appspec.TkeRouteEniSpec{Enabled: &enabled}
	seenApp := map[string]struct{}{}

	wsComps, err := m.listWorkspaceTkeRouteEni(ctx)
	if err != nil {
		return result, err
	}
	// workspace components are unique by (workspaceID, name), not name alone.
	wsByKey := make(map[wsCompKey]*workspace.Component, len(wsComps))
	wsNames := make([]string, 0, len(wsComps))
	for _, c := range wsComps {
		wsByKey[makeWsCompKey(c.WorkspaceID, c.Name)] = c
		wsNames = append(wsNames, c.Name)
	}

	appModels, err := m.listAffectedAppModels(ctx, wsNames)
	if err != nil {
		return result, err
	}
	appWorkspaceIDs, err := m.loadAppWorkspaceIDs(ctx, appModels)
	if err != nil {
		return result, err
	}

	// 1) Write AppSpec + remove app components / refs.
	for _, am := range appModels {
		workspaceID := appWorkspaceIDs[am.AppID]
		for _, comp := range am.Components {
			if comp == nil {
				continue
			}
			var envNames []string
			switch {
			case comp.Type == tkeRouteEniComponentName && comp.RefWorkspaceCompName == "":
				envNames = []string{appspec.DefaultEnvName}
			case comp.RefWorkspaceCompName != "":
				if workspaceID == "" {
					return result, fmt.Errorf("application %s not found for workspace component ref", am.AppID)
				}
				wsComp, ok := wsByKey[makeWsCompKey(workspaceID, comp.RefWorkspaceCompName)]
				if !ok {
					// Same ref name may exist in another workspace; ignore non-local matches.
					continue
				}
				switch wsComp.ScopeType {
				case component.ScopeTypeGlobal:
					envNames = []string{appspec.DefaultEnvName}
				case component.ScopeTypeEnvironment:
					envNames = wsComp.ScopeEnvNames
				}
			default:
				continue
			}

			for _, envName := range envNames {
				if dryRun {
					result.AppSpecWrites++
					continue
				}
				wrote, enableErr := m.enableTkeRouteEni(ctx, am.AppID, envName, enabledSpec)
				if enableErr != nil {
					return result, enableErr
				}
				if wrote {
					result.AppSpecWrites++
					log.Infof(ctx, "set tkeRouteEni.enabled=true appID=%s envName=%q", am.AppID, envName)
				}
			}

			if dryRun {
				result.AppComponentsRemoved++
			} else if err = m.appModelStore.RemoveComponent(ctx, am.AppID, comp.Name); err != nil {
				return result, errors.Wrapf(err, "remove component appID=%s name=%s", am.AppID, comp.Name)
			} else {
				result.AppComponentsRemoved++
				log.Infof(ctx, "removed app component appID=%s name=%s", am.AppID, comp.Name)
			}
			if _, ok := seenApp[am.AppID]; !ok {
				seenApp[am.AppID] = struct{}{}
				result.AppIDs = append(result.AppIDs, am.AppID)
			}
		}
	}

	// 2) Remove workspace components.
	for _, wsComp := range wsComps {
		if dryRun {
			result.WorkspaceComponentsRemoved++
			continue
		}
		if err = m.wsCompStore.Remove(ctx, wsComp.ID); err != nil {
			return result, errors.Wrapf(err, "remove workspace component %s/%s", wsComp.WorkspaceID, wsComp.Name)
		}
		result.WorkspaceComponentsRemoved++
		log.Infof(ctx, "removed workspace component workspaceID=%s name=%s", wsComp.WorkspaceID, wsComp.Name)
	}

	return result, nil
}

func (m *Migrator) loadAppWorkspaceIDs(
	ctx context.Context, appModels []*appmodel.AppModel,
) (map[string]string, error) {
	ids := make([]string, 0, len(appModels))
	for _, am := range appModels {
		ids = append(ids, am.AppID)
	}
	apps, err := m.appStore.GetAppsByIDs(ctx, ids)
	if err != nil {
		return nil, errors.Wrap(err, "load applications for workspace lookup")
	}
	out := make(map[string]string, len(apps))
	for _, app := range apps {
		out[app.ID] = app.WorkspaceID
	}
	return out, nil
}

func (m *Migrator) enableTkeRouteEni(
	ctx context.Context, appID, envName string, spec *appspec.TkeRouteEniSpec,
) (bool, error) {
	existing, err := m.appSpecStore.Get(ctx, appID, envName)
	if err == nil && existing.TkeRouteEni != nil &&
		existing.TkeRouteEni.Enabled != nil && *existing.TkeRouteEni.Enabled {
		return false, nil
	}
	if err != nil && !errors.Is(err, appspec.ErrAppSpecNotFound) {
		return false, errors.Wrapf(err, "get app spec appID=%s envName=%q", appID, envName)
	}
	if envName == appspec.DefaultEnvName {
		err = appspec.SetDefaultSection(
			ctx, m.appSpecStore, m.appModelStore, appID,
			appspec.TkeRouteEniSection, spec, appspec.SectionWriteModeReplace,
		)
	} else {
		err = appspec.SetEnvSection(
			ctx, m.appSpecStore, appID, envName,
			appspec.TkeRouteEniSection, spec, appspec.SectionWriteModeReplace,
		)
	}
	return err == nil, err
}

func (m *Migrator) listWorkspaceTkeRouteEni(ctx context.Context) ([]*workspace.Component, error) {
	cursor, err := m.db.Collection("workspace_components").Find(ctx, bson.M{"type": tkeRouteEniComponentName})
	if err != nil {
		return nil, errors.Wrap(err, "list workspace tkeRouteEni components")
	}
	defer cursor.Close(ctx)
	var items []*workspace.Component
	if err = cursor.All(ctx, &items); err != nil {
		return nil, errors.Wrap(err, "decode workspace tkeRouteEni components")
	}
	return items, nil
}

func (m *Migrator) listAffectedAppModels(ctx context.Context, wsCompNames []string) ([]*appmodel.AppModel, error) {
	orFilters := []bson.M{{"components.type": tkeRouteEniComponentName}}
	if len(wsCompNames) > 0 {
		orFilters = append(orFilters, bson.M{"components.refWorkspaceCompName": bson.M{"$in": wsCompNames}})
	}
	cursor, err := m.db.Collection(appmodel.CollectionName).Find(ctx, bson.M{"$or": orFilters})
	if err != nil {
		return nil, errors.Wrap(err, "list app models with tkeRouteEni mounts")
	}
	defer cursor.Close(ctx)
	var items []*appmodel.AppModel
	if err = cursor.All(ctx, &items); err != nil {
		return nil, errors.Wrap(err, "decode app models with tkeRouteEni mounts")
	}
	return items, nil
}
