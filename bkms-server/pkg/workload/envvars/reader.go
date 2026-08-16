/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package envvars

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// UnifiedEnvVarsReader reads env vars from all sources with unified source semantics.
type UnifiedEnvVarsReader struct {
	scopedEnvVarStore ScopedEnvVarStore
	appDepsVarReader  *depenvvars.Reader
	polarisVarReader  *polarisenvvars.Reader
}

// NewUnifiedEnvVarsReader creates a new unified env vars reader.
func NewUnifiedEnvVarsReader(
	store ScopedEnvVarStore,
	appDepsVarReader *depenvvars.Reader,
	polarisVarReader *polarisenvvars.Reader,
) *UnifiedEnvVarsReader {
	return &UnifiedEnvVarsReader{
		scopedEnvVarStore: store,
		appDepsVarReader:  appDepsVarReader,
		polarisVarReader:  polarisVarReader,
	}
}

// ListVars lists all env vars available to an app in the given environment. It's the main
// entry point to get env vars for other modules, e.g. app plugin, config rendering, etc.
//
// The returned list is ordered from low to high priority so later items override earlier
// items when EnvVariableList.ToMap is used for config rendering.
func (r *UnifiedEnvVarsReader) ListVars(
	ctx context.Context,
	environment envmodel.Environment,
	app *bkmsapp.Application,
	am *appmodel.AppModel,
) (envvartypes.EnvVariableList, error) {
	// Merge the "background" vars and the vars defined in AppModel. The "background" vars are those
	// that come from builtin and scoped env var store.
	bgVars, err := r.ListAppBgVars(ctx, environment, app)
	if err != nil {
		return nil, errors.Wrap(err, "list app background vars")
	}
	envVars := bgVars.GetDataList()

	if am == nil {
		return envVars, nil
	}
	// Append env vars defined in the AppModel, which have the highest priority.
	for _, v := range am.Workload.EnvVars {
		envVars = append(envVars, envvartypes.EnvVariableObj{
			Key:         v.Key,
			Value:       v.Value,
			Description: v.Description,
			IsSensitive: v.IsSensitive,
		})
	}
	return envVars, nil
}

// ListEnvBgVars lists environment background vars:
// - Builtin
// - Scoped workspace vars
// - Scoped envType vars
// The returned list includes all vars with the same key, sorted by source priority
// from low to high, so conflict checks can display every source.
func (r *UnifiedEnvVarsReader) ListEnvBgVars(
	ctx context.Context, environment envmodel.Environment,
) (envvartypes.EnvVariableRichList, error) {
	builtinVars, err := ListBuiltinVars(ctx, r.scopedEnvVarStore, environment, nil)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list builtin vars")
	}

	scopedVars, err := r.scopedEnvVarStore.List(
		ctx,
		environment.WorkspaceID,
		WithScopes(
			envvartypes.ScopeWorkspace,
			envvartypes.ScopeEnvType(environment.Type),
		),
	)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list scoped background vars")
	}

	lst := envvartypes.EnvVariableRichList{Vars: wrapEnvVarsWithSource(
		builtinVars,
		envvartypes.ConflictedSource{Source: envvartypes.EnvVarSourceBuiltin},
	)}
	scopedItems, err := wrapScopedVarsWithOwnSource(scopedVars)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, err
	}
	lst.Vars = append(lst.Vars, scopedItems...)
	lst.SortBySourcePriority()
	return lst, nil
}

// ListAppBgVars lists app background vars:
// - Builtin (including app-level builtins when app is provided)
// - Scoped workspace vars
// - Scoped envType vars
// - Scoped env vars
// The returned list includes all vars with the same key, sorted by source priority
// from low to high, so conflict checks can display every source.
func (r *UnifiedEnvVarsReader) ListAppBgVars(
	ctx context.Context, environment envmodel.Environment, app *bkmsapp.Application,
) (envvartypes.EnvVariableRichList, error) {
	builtinVars, err := ListBuiltinVars(ctx, r.scopedEnvVarStore, environment, app)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list builtin vars")
	}

	scopedVars, err := r.scopedEnvVarStore.List(
		ctx,
		environment.WorkspaceID,
		WithScopes(
			envvartypes.ScopeWorkspace,
			envvartypes.ScopeEnvType(environment.Type),
			envvartypes.ScopeEnv(environment.Name),
		),
	)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list scoped background vars")
	}

	lst := envvartypes.EnvVariableRichList{Vars: wrapEnvVarsWithSource(
		builtinVars,
		envvartypes.ConflictedSource{Source: envvartypes.EnvVarSourceBuiltin},
	)}
	scopedItems, err := wrapScopedVarsWithOwnSource(scopedVars)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, err
	}
	lst.Vars = append(lst.Vars, scopedItems...)

	if r.appDepsVarReader != nil {
		// Append vars from depservice instances attached to this app (if any).
		instVars, err := r.appDepsVarReader.ListEnvVarsForApp(ctx, environment, app)
		if err != nil {
			return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list dep service instance env vars")
		}
		lst.Vars = append(lst.Vars, instVars.Vars...)
	}

	if r.polarisVarReader != nil {
		// Append vars from polaris configs attached to this app (if any).
		polarisVars, err := r.polarisVarReader.ListEnvVarsForApp(ctx, environment, app)
		if err != nil {
			return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list polaris config env vars")
		}
		lst.Vars = append(lst.Vars, polarisVars.Vars...)
	}

	lst.SortBySourcePriority()
	return lst, nil
}

// ListEnvVars lists all env vars available in the given environment.
func (r *UnifiedEnvVarsReader) ListEnvVars(
	ctx context.Context, environment envmodel.Environment,
) (envvartypes.EnvVariableList, error) {
	bgVars, err := r.ListAppBgVars(ctx, environment, nil)
	if err != nil {
		return nil, err
	}
	return bgVars.ToDeduplicatedList().GetDataList(), nil
}

// ListAppBgVarsForConflicts lists app background vars used by app-level conflict checks.
// It's different from `ListAppBgVars` in that it is not env specific and it includes all
// scoped env vars in the workspace, to reflect the real conflict situation.
func (r *UnifiedEnvVarsReader) ListAppBgVarsForConflicts(
	ctx context.Context,
	workspaceID string,
	app *bkmsapp.Application,
) (envvartypes.EnvVariableRichList, error) {
	// Get the builtin vars.
	//
	// ListBuiltinVars requires an Environment to derive env-level builtin variables
	// (e.g. BKMS_ENV_TYPE, BKMS_ENV_NAME) and to query scope-matched builtin scoped
	// vars from the database. Since app-level conflict checks are not tied to a
	// specific environment, we use a representative env with "production" type here
	// to get a close-to-real builtin variable list. Alternative approaches such as
	// hard-coding the builtin key list have their own maintenance problems, so this
	// is the pragmatic trade-off for now.
	stubEnv := envmodel.Environment{
		WorkspaceID: workspaceID,
		Type:        string(env.TypeProduction),
		Name:        "production",
	}
	builtinVars, err := ListBuiltinVars(ctx, r.scopedEnvVarStore, stubEnv, app)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list builtin vars")
	}

	// Get all scoped vars in the workspace.
	scopedVars, err := r.scopedEnvVarStore.List(ctx, workspaceID)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list workspace scoped env vars")
	}

	bgVars := envvartypes.EnvVariableRichList{Vars: wrapEnvVarsWithSource(
		builtinVars,
		envvartypes.ConflictedSource{Source: envvartypes.EnvVarSourceBuiltin},
	)}
	scopedItems, err := wrapScopedVarsWithOwnSource(scopedVars)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, err
	}
	bgVars.Vars = append(bgVars.Vars, scopedItems...)

	// Include depservice binding keys in the conflict pool. App-level vars have
	// no env dimension, so the reader returns each binding key once, with values
	// aggregated across mapped environments.
	if app != nil && r.appDepsVarReader != nil {
		instVars, err := r.appDepsVarReader.ListAppVarsForConflicts(ctx, workspaceID, app)
		if err != nil {
			return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list dep service instance env vars")
		}
		bgVars.Vars = append(bgVars.Vars, instVars.Vars...)
	}

	// Include polaris config env vars in the conflict pool.
	if app != nil && r.polarisVarReader != nil {
		polarisVars, err := r.polarisVarReader.ListAppVarsForConflicts(ctx, workspaceID, app)
		if err != nil {
			return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list polaris config env vars")
		}
		bgVars.Vars = append(bgVars.Vars, polarisVars.Vars...)
	}

	bgVars.SortBySourcePriority()
	return bgVars, nil
}

// BuildEnvConflictedInfoByKeys builds env-level conflict info for given keys.
func (r *UnifiedEnvVarsReader) BuildEnvConflictedInfoByKeys(
	ctx context.Context,
	keys []string,
	environment envmodel.Environment,
) (map[string]envvartypes.EnvVarConflictedInfo, error) {
	bgVars, err := r.ListEnvBgVars(ctx, environment)
	if err != nil {
		return nil, err
	}

	return buildConflictedInfoByKeys(
		keys,
		envvartypes.ConflictedSource{Source: envvartypes.EnvVarSourceScopedEnv},
		bgVars.Vars,
	), nil
}

// BuildAppConflictedInfoByKeys builds app-level conflict info for given keys against the whole workspace.
func (r *UnifiedEnvVarsReader) BuildAppConflictedInfoByKeys(
	ctx context.Context,
	keys []string,
	workspaceID string,
	app *bkmsapp.Application,
) (map[string]envvartypes.EnvVarConflictedInfo, error) {
	bgVars, err := r.ListAppBgVarsForConflicts(ctx, workspaceID, app)
	if err != nil {
		return nil, err
	}

	return buildConflictedInfoByKeys(
		keys,
		envvartypes.ConflictedSource{Source: envvartypes.EnvVarSourceApp, SourceValue: app.ID},
		bgVars.Vars,
	), nil
}

func buildConflictedInfoByKeys(
	keys []string,
	keysSource envvartypes.ConflictedSource,
	bgVars []envvartypes.EnvVariableRichItem,
) map[string]envvartypes.EnvVarConflictedInfo {
	// Build an index to find conflict info faster.
	varsByKey := make(map[string][]envvartypes.EnvVariableRichItem)
	for _, item := range bgVars {
		if item.Obj.Key == "" {
			continue
		}
		// Ignore items with same source.
		if item.Source == keysSource {
			continue
		}
		varsByKey[item.Obj.Key] = append(varsByKey[item.Obj.Key], item)
	}

	result := make(map[string]envvartypes.EnvVarConflictedInfo)
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, ok := result[key]; ok {
			continue
		}
		conflictedItems := varsByKey[key]
		if len(conflictedItems) == 0 {
			continue
		}

		// Found conflicts, build the conflicted info.
		conflictedSources := make([]envvartypes.ConflictedSource, 0, len(conflictedItems))
		conflictedValues := make([]string, 0, len(conflictedItems))
		for _, item := range conflictedItems {
			conflictedSources = append(conflictedSources, item.Source)
			// Some vars may have empty value, we display it as "--" to make it clear.
			conflictedValues = append(conflictedValues, cmp.Or(item.Obj.ValueForDisplay(), "--"))
		}

		result[key] = envvartypes.EnvVarConflictedInfo{
			ConflictedSources:  conflictedSources,
			OverrideConflicted: isHigherPriority(keysSource.Source, conflictedSources),
			ConflictedDetail:   fmt.Sprintf("冲突值为：%s", strings.Join(conflictedValues, ", ")),
		}
	}
	return result
}

func isHigherPriority(source envvartypes.EnvVarSource, conflictedSources []envvartypes.ConflictedSource) bool {
	sourcePriority := envvartypes.EnvVarSourcePriority(source)
	for _, item := range conflictedSources {
		if sourcePriority <= envvartypes.EnvVarSourcePriority(item.Source) {
			return false
		}
	}
	return true
}

// Wraps the env variable object with the source info.
func wrapEnvVarsWithSource(
	vars []envvartypes.EnvVariableObj, source envvartypes.ConflictedSource,
) []envvartypes.EnvVariableRichItem {
	return lo.Map(vars, func(item envvartypes.EnvVariableObj, _ int) envvartypes.EnvVariableRichItem {
		return envvartypes.EnvVariableRichItem{
			Obj:    item,
			Source: source,
		}
	})
}

func wrapScopedVarsWithOwnSource(vars []ScopedEnvVar) ([]envvartypes.EnvVariableRichItem, error) {
	result := make([]envvartypes.EnvVariableRichItem, 0, len(vars))
	for _, item := range vars {
		source, err := conflictedSourceFromScopedEnvVar(item)
		if err != nil {
			return nil, err
		}
		result = append(result, envvartypes.EnvVariableRichItem{
			Obj: envvartypes.EnvVariableObj{
				Key:         item.Key,
				Value:       item.Value,
				Description: item.Description,
				IsBuiltin:   item.IsBuiltin,
				IsSensitive: item.IsSensitive,
			},
			Source: source,
		})
	}
	return result, nil
}

func conflictedSourceFromScopedEnvVar(item ScopedEnvVar) (envvartypes.ConflictedSource, error) {
	switch item.ScopeType {
	case envvartypes.ScopeTypeWorkspace:
		return envvartypes.ConflictedSource{
			Source:      envvartypes.EnvVarSourceScopedWorkspace,
			SourceValue: item.WorkspaceID,
		}, nil
	case envvartypes.ScopeTypeEnvType:
		return envvartypes.ConflictedSource{
			Source:      envvartypes.EnvVarSourceScopedEnvType,
			SourceValue: item.ScopeValue,
		}, nil
	case envvartypes.ScopeTypeEnv:
		return envvartypes.ConflictedSource{
			Source:      envvartypes.EnvVarSourceScopedEnv,
			SourceValue: item.ScopeValue,
		}, nil
	default:
		return envvartypes.ConflictedSource{}, errors.Errorf(
			"unsupported scoped env var scope type %s",
			item.ScopeType,
		)
	}
}
