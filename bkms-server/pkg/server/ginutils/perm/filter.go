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

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// FilterViewableEnvs returns the envs among `envs` that the current user has
// permission to view. A feature env is checked by its source env.
func FilterViewableEnvs(
	ctx context.Context,
	registry *storereg.Registry,
	workspaceID string,
	envs []envmodel.Environment,
) ([]envmodel.Environment, error) {
	if len(envs) == 0 {
		return envs, nil
	}
	authEnvs, err := getAuthEnvs(ctx, registry, envs)
	if err != nil {
		return nil, err
	}
	refs, _ := groupByAuthEnv(envs, authEnvs)
	viewable, err := perm.NewManager().FilterViewableEnvs(ctx, workspaceID, refs)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "filter viewable envs")
	}
	return lo.Filter(envs, func(_ envmodel.Environment, i int) bool { return viewable.Has(authEnvs[i].ID.Hex()) }), nil
}

// FilterViewableAppEnvNames returns the names among `envNames` in which the
// current user can view the already validated app's data. It's used by
// app-level lists that mix data of multiple envs, so that each item bound to
// an env is shown only when its env is viewable. Names that don't refer to an
// env available to the app are dropped.
func FilterViewableAppEnvNames(
	ctx context.Context,
	registry *storereg.Registry,
	app *bkmsapp.Application,
	envNames []string,
) (*set.StringSet, error) {
	envNames = lo.Uniq(lo.Compact(envNames))
	if len(envNames) == 0 {
		return set.NewStringSet(), nil
	}
	envs, err := listAppEnvs(ctx, registry, app, store.Where("name", bson.M{"$in": envNames}))
	if err != nil {
		return nil, err
	}
	return viewableAppEnvNames(ctx, registry, app, envs)
}

// FilterByViewableAppEnvs returns the items of the already validated app that
// the current user can view, where `envName` returns the env an item is bound
// to. Env-independent items, whose env name is empty, are always kept.
func FilterByViewableAppEnvs[T any](
	ctx context.Context,
	registry *storereg.Registry,
	app *bkmsapp.Application,
	items []T,
	envName func(T) string,
) ([]T, error) {
	viewable, err := FilterViewableAppEnvNames(
		ctx, registry, app, lo.Map(items, func(item T, _ int) string { return envName(item) }),
	)
	if err != nil {
		return nil, err
	}
	return lo.Filter(items, func(item T, _ int) bool {
		name := envName(item)
		return name == "" || viewable.Has(name)
	}), nil
}

// ListViewableAppEnvNames returns the names of the envs available to the
// already validated app in which the current user can view the app's data.
// It's used by paginated lists that must filter by env in the query.
//
// `all` is true when every env is viewable, so that callers need no filter,
// which also keeps data of deleted envs visible as before.
func ListViewableAppEnvNames(
	ctx context.Context, registry *storereg.Registry, app *bkmsapp.Application,
) (names []string, all bool, err error) {
	envs, err := listAppEnvs(ctx, registry, app)
	if err != nil {
		return nil, false, err
	}
	viewable, err := viewableAppEnvNames(ctx, registry, app, envs)
	if err != nil {
		return nil, false, err
	}
	return append([]string{}, viewable.ToSlice()...), viewable.Size() == len(envs), nil
}

// FilterViewableAppEnvNamesByApp returns, for each app in `envsByApp`, the
// names of its envs in which the current user can view the app's data, with
// one IAM request for all apps. The envs of each app must be available to
// it, and a feature env is checked by its source env.
func FilterViewableAppEnvNamesByApp(
	ctx context.Context,
	registry *storereg.Registry,
	workspaceID string,
	envsByApp map[string][]envmodel.Environment,
) (map[string]*set.StringSet, error) {
	appIDs := lo.Keys(envsByApp)
	authEnvs, err := getAuthEnvs(ctx, registry, lo.FlatMap(appIDs, func(appID string, _ int) []envmodel.Environment {
		return envsByApp[appID]
	}))
	if err != nil {
		return nil, err
	}

	var pairs []bkiam.AppEnvRef
	namesByApp := make(map[string]map[string][]string, len(appIDs))
	offset := 0
	for _, appID := range appIDs {
		envs := envsByApp[appID]
		refs, namesByAuthEnvID := groupByAuthEnv(envs, authEnvs[offset:offset+len(envs)])
		offset += len(envs)
		for _, ref := range refs {
			pairs = append(pairs, bkiam.AppEnvRef{AppID: appID, Env: ref})
		}
		namesByApp[appID] = namesByAuthEnvID
	}

	allowedByApp, err := perm.NewManager().FilterViewableAppEnvs(ctx, workspaceID, pairs)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "filter viewable app envs")
	}
	result := make(map[string]*set.StringSet, len(appIDs))
	for appID, namesByAuthEnvID := range namesByApp {
		viewable := set.NewStringSet()
		allowed := allowedByApp[appID]
		for authEnvID, names := range namesByAuthEnvID {
			if allowed != nil && allowed.Has(authEnvID) {
				viewable.Append(names...)
			}
		}
		result[appID] = viewable
	}
	return result, nil
}

// viewableAppEnvNames returns the names of `envs` in which the current user
// can view the app's data, with one IAM request.
func viewableAppEnvNames(
	ctx context.Context, registry *storereg.Registry, app *bkmsapp.Application, envs []envmodel.Environment,
) (*set.StringSet, error) {
	viewableByApp, err := FilterViewableAppEnvNamesByApp(
		ctx, registry, app.WorkspaceID, map[string][]envmodel.Environment{app.ID: envs},
	)
	if err != nil {
		return nil, err
	}
	return viewableByApp[app.ID], nil
}
