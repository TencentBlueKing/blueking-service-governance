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

package perm

import (
	"context"
	"errors"

	"github.com/TencentBlueKing/gopkg/collection/set"
	pkgerrors "github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ValidateWorkspaceViewInAllEnvs validates and returns a workspace by a
// `workspaceID` parameter, then checks the view permission of the workspace
// and every standard env in it. It's used by workspace-level data that takes
// effect in multiple envs, e.g. public env vars.
func ValidateWorkspaceViewInAllEnvs(
	ctx context.Context,
	registry *storereg.Registry,
	workspaceID string,
) (*workspace.Workspace, error) {
	ws, err := ValidateWorkspaceByID(ctx, registry, workspaceID, TypeView)
	if err != nil {
		return nil, err
	}
	envs, err := listStdEnvs(ctx, registry, ws.ID)
	if err != nil {
		return nil, err
	}
	viewable, err := FilterViewableEnvs(ctx, registry, ws.ID, envs)
	if err != nil {
		return nil, err
	}
	if err = deniedEnvsError(ws.ID, "view env", envs, namesOf(viewable)); err != nil {
		return nil, err
	}
	return ws, nil
}

// deniedEnvsError returns a no-permission error listing the envs of `envs`
// whose names aren't in `allowed`, or nil if there is none.
func deniedEnvsError(workspaceID, operation string, envs []envmodel.Environment, allowed *set.StringSet) error {
	denied := lo.FilterMap(envs, func(env envmodel.Environment, _ int) (string, bool) {
		return env.Name, !allowed.Has(env.Name)
	})
	if len(denied) == 0 {
		return nil
	}
	return bkerrs.WrapIAMNoPermission(
		pkgerrors.Errorf("no permission to %s in envs %v", operation, denied), workspaceID, "check perm in envs",
	)
}

// getAuthEnvs returns the env used for authorization of each env in `envs`,
// in the same order: the env itself for a standard env, or its source
// standard env for a feature env. The source envs are loaded in one query,
// and each must be a standard env in the same workspace.
func getAuthEnvs(
	ctx context.Context, registry *storereg.Registry, envs []envmodel.Environment,
) ([]*envmodel.Environment, error) {
	sourceIDs := lo.Uniq(lo.FilterMap(envs, func(env envmodel.Environment, _ int) (bson.ObjectID, bool) {
		return env.SourceEnvID, env.IsFeatureEnv()
	}))
	sourceEnvs := map[bson.ObjectID]*envmodel.Environment{}
	if len(sourceIDs) > 0 {
		loaded, err := store.Of(registry.DB, envmodel.EnvMeta).
			List(ctx, store.Where("_id", bson.M{"$in": sourceIDs}))
		if err != nil {
			return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list source envs of feature envs")
		}
		for i := range loaded {
			sourceEnvs[loaded[i].ID] = &loaded[i]
		}
	}

	authEnvs := make([]*envmodel.Environment, len(envs))
	for i := range envs {
		env := &envs[i]
		if !env.IsFeatureEnv() {
			authEnvs[i] = env
			continue
		}
		sourceEnv, ok := sourceEnvs[env.SourceEnvID]
		if !ok || sourceEnv.WorkspaceID != env.WorkspaceID || sourceEnv.IsFeatureEnv() {
			return nil, bkerrs.Errorf(
				bkerrs.ErrCodeInternalServerError,
				"invalid source env %s of feature env %s", env.SourceEnvID.Hex(), env.Name,
			)
		}
		authEnvs[i] = sourceEnv
	}
	return authEnvs, nil
}

// groupByAuthEnv groups `envs` by the env used for their authorization, where
// authEnvs[i] is the auth env of envs[i]. It returns the distinct auth envs
// to check and the env names authorized by each of them.
func groupByAuthEnv(
	envs []envmodel.Environment, authEnvs []*envmodel.Environment,
) ([]bkiam.EnvRef, map[string][]string) {
	var refs []bkiam.EnvRef
	namesByAuthEnvID := make(map[string][]string, len(envs))
	for i, authEnv := range authEnvs {
		authEnvID := authEnv.ID.Hex()
		if _, ok := namesByAuthEnvID[authEnvID]; !ok {
			refs = append(refs, bkiam.EnvRef{ID: authEnvID, Type: authEnv.Type})
		}
		namesByAuthEnvID[authEnvID] = append(namesByAuthEnvID[authEnvID], envs[i].Name)
	}
	return refs, namesByAuthEnvID
}

// namesOf returns the names of `envs`.
func namesOf(envs []envmodel.Environment) *set.StringSet {
	return set.NewStringSetWithValues(lo.Map(envs, func(env envmodel.Environment, _ int) string { return env.Name }))
}

// getEnv returns the env matching `opt`, where `ref` refers to the env in
// errors.
func getEnv(
	ctx context.Context, registry *storereg.Registry, opt store.Opt, ref string,
) (*envmodel.Environment, error) {
	env, err := store.Of(registry.DB, envmodel.EnvMeta).Get(ctx, opt)
	if err != nil {
		if errors.Is(err, envmodel.ErrEnvNotFound) {
			return nil, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "environment %s not found", ref)
		}
		return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get environment %s", ref)
	}
	return env, nil
}

func getEnvByID(ctx context.Context, registry *storereg.Registry, envID string) (*envmodel.Environment, error) {
	if envID == "" {
		return nil, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "envID is required")
	}
	envBsonID, err := bson.ObjectIDFromHex(envID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid env id")
	}
	return getEnv(ctx, registry, store.ByID(envBsonID), envID)
}

// getAppEnvByName returns the env named `envName` that is available to the
// app, including feature envs of the app.
func getAppEnvByName(
	ctx context.Context, registry *storereg.Registry, app *bkmsapp.Application, envName string,
) (*envmodel.Environment, error) {
	return getEnv(ctx, registry, envmodel.AppEnvByName(app.WorkspaceID, app.ID, envName), envName)
}

// listStdEnvs lists the standard envs of the workspace.
func listStdEnvs(
	ctx context.Context, registry *storereg.Registry, workspaceID string,
) ([]envmodel.Environment, error) {
	envs, err := store.Of(registry.DB, envmodel.EnvMeta).List(ctx, envmodel.StdEnvsIn(workspaceID))
	if err != nil {
		return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list envs of workspace %s", workspaceID)
	}
	return envs, nil
}

// listAppEnvs lists the envs available to the app, including feature envs of
// the app, narrowed by extra filters.
func listAppEnvs(
	ctx context.Context, registry *storereg.Registry, app *bkmsapp.Application, opts ...store.Opt,
) ([]envmodel.Environment, error) {
	opts = append([]store.Opt{envmodel.InWorkspace(app.WorkspaceID), envmodel.AvailableToApp(app.ID)}, opts...)
	envs, err := store.Of(registry.DB, envmodel.EnvMeta).List(ctx, opts...)
	if err != nil {
		return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list envs of app %s", app.ID)
	}
	return envs, nil
}
