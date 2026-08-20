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

// Package perm provides reusable Gin permission and path-resource validation helpers.
package perm

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Type defines the permission type required by a route.
type Type string

const (
	// TypeView checks view permission.
	TypeView Type = "view"
	// TypeEdit checks edit permission.
	TypeEdit Type = "edit"
	// TypeDelete checks delete permission.
	TypeDelete Type = "delete"
	// TypeDeploy checks deploy permission.
	TypeDeploy Type = "deploy"
)

// ValidateWorkspaceByID validates and returns a workspace by a `workspaceID`
// path parameter, then checks the requested permission. It's usually used in
// APIs with {workspaceID} in path, such as "/workspaces/{workspaceID}/...".
//
// Parameters:
// - registry: stores and services used by Gin handlers.
// - workspaceID: the workspace ID from path parameters.
// - permType: the type of permission to check.
//
// The error returned from this function is an instance of bkerrs and can be
// returned directly to the client without any wrapping.
func ValidateWorkspaceByID(
	ctx context.Context,
	registry *storereg.Registry,
	workspaceID string,
	permType Type,
) (*workspace.Workspace, error) {
	ws, err := getWorkspaceByID(ctx, registry, workspaceID)
	if err != nil {
		return nil, err
	}

	permMgr := perm.NewManager()
	switch permType {
	case TypeView:
		err = permMgr.HasViewWorkspacePerm(ctx, ws.ID)
	case TypeEdit:
		err = permMgr.HasEditWorkspacePerm(ctx, ws.ID)
	case TypeDelete:
		err = permMgr.HasDeleteWorkspacePerm(ctx, ws.ID)
	default:
		return nil, bkerrs.New(bkerrs.ErrCodeInternalServerError, "invalid permission type")
	}
	if err != nil {
		return nil, bkerrs.WrapIAMNoPermission(err, ws.ID, "check workspace perm")
	}
	return ws, nil
}

// ValidateWorkspaceForAppCreate validates the workspace referenced by a
// `workspaceID` path parameter, then checks create-app permission within that
// workspace. It's used by APIs that create apps under
// "/workspaces/{workspaceID}/...".
//
// The error returned from this function is an instance of bkerrs and can be
// returned directly to the client without any wrapping.
func ValidateWorkspaceForAppCreate(
	ctx context.Context,
	registry *storereg.Registry,
	workspaceID string,
) (*workspace.Workspace, error) {
	ws, err := getWorkspaceByID(ctx, registry, workspaceID)
	if err != nil {
		return nil, err
	}

	err = perm.NewManager().HasCreateAppPerm(ctx, ws.ID)
	if err != nil {
		return nil, bkerrs.WrapIAMNoPermission(err, ws.ID, "check app perm")
	}
	return ws, nil
}

func getWorkspaceByID(
	ctx context.Context,
	registry *storereg.Registry,
	workspaceID string,
) (*workspace.Workspace, error) {
	if workspaceID == "" {
		return nil, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "workspaceID is required")
	}

	ws, err := registry.WorkspaceStore.Get(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrWorkspaceNotFound) {
			return nil, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "workspace %s not found", workspaceID)
		}
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace")
	}
	return ws, nil
}

// ValidateEnvByID validates and returns an environment by an `envID` path
// parameter, then checks the requested permission. It's usually used in APIs
// with {envID} in path, such as "/envs/{envID}/...".
//
// Parameters:
// - registry: stores and services used by Gin handlers.
// - envID: the environment ID from path parameters.
// - permType: the type of permission to check.
//
// The error returned from this function is an instance of bkerrs and can be
// returned directly to the client without any wrapping.
func ValidateEnvByID(
	ctx context.Context,
	registry *storereg.Registry,
	envID string,
	permType Type,
) (*envmodel.Environment, error) {
	if envID == "" {
		return nil, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "envID is required")
	}

	envBsonID, err := bson.ObjectIDFromHex(envID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "invalid env id")
	}

	env, err := registry.EnvStore.Get(ctx, envBsonID)
	if err != nil {
		if errors.Is(err, envmodel.ErrEnvNotFound) {
			return nil, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "environment %s not found", envID)
		}
		return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get environment %s", envID)
	}

	permMgr := perm.NewManager()
	switch permType {
	case TypeView:
		err = permMgr.HasViewEnvPerm(ctx, env.WorkspaceID, env.Name)
	case TypeEdit:
		err = permMgr.HasEditEnvPerm(ctx, env.WorkspaceID, env.Name)
	case TypeDelete:
		err = permMgr.HasDeleteEnvPerm(ctx, env.WorkspaceID, env.Name)
	case TypeDeploy:
		err = permMgr.HasDeployEnvPerm(ctx, env.WorkspaceID, env.Name)
	default:
		return nil, bkerrs.Errorf(bkerrs.ErrCodeInternalServerError, "invalid permission type: %v", permType)
	}
	if err != nil {
		return nil, bkerrs.WrapIAMNoPermission(err, env.WorkspaceID, "check env perm")
	}
	return env, nil
}

// ValidateAppByID validates and returns an app by an `appID` path parameter,
// then checks the requested permission. It's usually used in APIs with {appID}
// in path, such as "/apps/{appID}/...".
//
// Parameters:
// - registry: stores and services used by Gin handlers.
// - appID: the app ID from path parameters.
// - permType: the type of permission to check.
//
// The error returned from this function is an instance of bkerrs and can be
// returned directly to the client without any wrapping.
func ValidateAppByID(
	ctx context.Context,
	registry *storereg.Registry,
	appID string,
	permType Type,
) (*bkmsapp.Application, error) {
	if appID == "" {
		return nil, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "appID is required")
	}

	app, err := registry.AppStore.GetApp(ctx, appID)
	if err != nil {
		if errors.Is(err, bkmsapp.ErrAppNotFound) {
			return nil, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app %s not found", appID)
		}
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get app")
	}

	permMgr := perm.NewManager()
	switch permType {
	case TypeView:
		err = permMgr.HasViewAppPerm(ctx, app.WorkspaceID, app.ID)
	case TypeEdit:
		err = permMgr.HasEditAppPerm(ctx, app.WorkspaceID, app.ID)
	case TypeDelete:
		err = permMgr.HasDeleteAppPerm(ctx, app.WorkspaceID, app.ID)
	default:
		return nil, bkerrs.New(bkerrs.ErrCodeInternalServerError, "invalid permission type")
	}
	if err != nil {
		return nil, bkerrs.WrapIAMNoPermission(err, app.WorkspaceID, "check app perm")
	}
	return app, nil
}

// ValidateAppEnvByName validates and returns an app and env by `appID` and
// `envName` path or request parameters, then checks the requested app permission.
// It's usually used in APIs with {appID} and {envName} in path or request.
//
// Parameters:
// - registry: stores and services used by Gin handlers.
// - appID: the app ID from path parameters.
// - envName: the environment name from path or request parameters.
// - permType: the type of app permission to check.
//
// The error returned from this function is an instance of bkerrs and can be
// returned directly to the client without any wrapping.
func ValidateAppEnvByName(
	ctx context.Context,
	registry *storereg.Registry,
	appID string,
	envName string,
	permType Type,
) (*bkmsapp.Application, *envmodel.Environment, error) {
	app, err := ValidateAppByID(ctx, registry, appID, permType)
	if err != nil {
		return nil, nil, err
	}
	if envName == "" {
		return nil, nil, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "envName is required")
	}

	env, err := registry.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		if errors.Is(err, envmodel.ErrEnvNotFound) {
			return nil, nil, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "environment %s not found", envName)
		}
		return nil, nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get env %s by name", envName)
	}
	return app, env, nil
}
