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

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ValidateAppEnvByID validates and returns a standard env by an `envID`
// parameter, then checks the requested permission of the already loaded app
// in it. It's used by APIs that refer to an env by ID in the app's context,
// e.g. creating a feature env from a source env.
func ValidateAppEnvByID(
	ctx context.Context,
	registry *storereg.Registry,
	app *bkmsapp.Application,
	envID string,
	permType Type,
) (*envmodel.Environment, error) {
	env, err := getEnvByID(ctx, registry, envID)
	if err != nil {
		return nil, err
	}
	if env.WorkspaceID != app.WorkspaceID || env.IsFeatureEnv() {
		return nil, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "environment %s not found", envID)
	}
	if err = ValidateAppEnv(ctx, registry, app, env, permType); err != nil {
		return nil, err
	}
	return env, nil
}

// ValidateAppEnv checks the requested permission of an already loaded app in
// an already loaded env. The env must be available to the app, which is
// guaranteed when it's loaded by envmodel.AppEnvByName.
//
// A feature env is authorized by its source standard env, using the current
// ID and type of the source env.
func ValidateAppEnv(
	ctx context.Context,
	registry *storereg.Registry,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	permType Type,
) error {
	authEnvs, err := getAuthEnvs(ctx, registry, []envmodel.Environment{*env})
	if err != nil {
		return err
	}
	return checkAppEnvPerm(ctx, app, authEnvs[0], permType)
}

// ValidateAppEnvNames checks the requested permission of an already
// validated app in each env named in `envNames`. It's used by app-level
// operations affecting some envs, e.g. deleting an image used by them. Names
// that don't refer to an env available to the app are skipped, since no env
// is affected.
func ValidateAppEnvNames(
	ctx context.Context,
	registry *storereg.Registry,
	app *bkmsapp.Application,
	envNames []string,
	permType Type,
) error {
	if len(envNames) == 0 {
		return nil
	}
	envs, err := listAppEnvs(ctx, registry, app, store.Where("name", bson.M{"$in": lo.Uniq(envNames)}))
	if err != nil {
		return err
	}
	return requireAppEnvs(ctx, registry, app, envs, permType)
}

// ValidateAppInAllEnvs validates and returns an app by an `appID` parameter,
// then checks the requested permission of the app in all envs. It's used by
// app-level data that takes effect in all envs, e.g. polaris configs and app
// env vars.
//
//   - TypeView checks view_app, and view_app_in_env in every standard env.
//   - TypeEdit checks edit_app only: it's granted to roles covering all envs
//     only, so it already stands for editing in all envs.
//   - TypeDeploy checks edit_app, and deploy_app_to_env in every standard env.
//
// Feature envs are authorized by their source envs, so they need no check.
func ValidateAppInAllEnvs(
	ctx context.Context,
	registry *storereg.Registry,
	appID string,
	permType Type,
) (*bkmsapp.Application, error) {
	app, err := ValidateAppByID(ctx, registry, appID, lo.Ternary(permType == TypeView, TypeView, TypeEdit))
	if err != nil || permType == TypeEdit {
		return app, err
	}

	envs, err := listStdEnvs(ctx, registry, app.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if err = requireAppEnvs(ctx, registry, app, envs, permType); err != nil {
		return nil, err
	}
	return app, nil
}

// requireAppEnvs checks the requested permission of the app in every env of
// `envs`. View is checked with one batched request, since it's used by reads;
// edit and deploy are only checked by writes affecting a few envs, so each
// env is checked separately.
func requireAppEnvs(
	ctx context.Context, registry *storereg.Registry, app *bkmsapp.Application, envs []envmodel.Environment,
	permType Type,
) error {
	if permType == TypeView {
		viewable, err := viewableAppEnvNames(ctx, registry, app, envs)
		if err != nil {
			return err
		}
		return deniedEnvsError(app.WorkspaceID, "view app "+app.ID, envs, viewable)
	}

	authEnvs, err := getAuthEnvs(ctx, registry, envs)
	if err != nil {
		return err
	}
	// 特性环境与来源环境共用同一个鉴权环境，只需检查一次
	for _, authEnv := range lo.UniqBy(authEnvs, func(env *envmodel.Environment) bson.ObjectID { return env.ID }) {
		if err = checkAppEnvPerm(ctx, app, authEnv, permType); err != nil {
			return err
		}
	}
	return nil
}

// checkAppEnvPerm checks the permission of the app in the env used for
// authorization, with the app-env actions which evaluate the app and the env
// against the same policy.
func checkAppEnvPerm(
	ctx context.Context, app *bkmsapp.Application, authEnv *envmodel.Environment, permType Type,
) error {
	workspaceID, envType, envID := app.WorkspaceID, authEnv.Type, authEnv.ID.Hex()
	permMgr := perm.NewManager()

	var err error
	switch permType {
	case TypeView:
		err = permMgr.HasViewAppInEnvPerm(ctx, workspaceID, app.ID, envType, envID)
	case TypeEdit:
		err = permMgr.HasEditAppInEnvPerm(ctx, workspaceID, app.ID, envType, envID)
	case TypeDeploy:
		err = permMgr.HasDeployAppToEnvPerm(ctx, workspaceID, app.ID, envType, envID)
	default:
		return bkerrs.Errorf(bkerrs.ErrCodeInternalServerError, "invalid permission type: %v", permType)
	}
	if err != nil {
		return bkerrs.WrapIAMNoPermission(err, workspaceID, "check app env perm")
	}
	return nil
}
