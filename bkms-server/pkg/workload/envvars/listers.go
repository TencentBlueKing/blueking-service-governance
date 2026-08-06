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
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// ListBuiltinVars returns built-in env variables available in the given environment.
// If app is not nil, app-related built-in variables are appended at the end.
func ListBuiltinVars(
	ctx context.Context,
	store ScopedEnvVarStore,
	environment envmodel.Environment,
	app *bkmsapp.Application,
) (envvartypes.EnvVariableList, error) {
	envVars := make(envvartypes.EnvVariableList, 0)
	envVars = append(envVars, getEnvBuiltinVars(environment)...)

	scopedVars, err := ListBuiltinScoped(ctx, store, environment)
	if err != nil {
		return nil, errors.Wrap(err, "list built-in scoped env vars")
	}
	for _, item := range scopedVars {
		envVars = append(envVars, envvartypes.EnvVariableObj{
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
			IsBuiltin:   item.IsBuiltin,
			IsSensitive: item.IsSensitive,
		})
	}

	envVars = append(envVars, getRuntimeBuiltinVars()...)

	if app != nil {
		envVars = append(envVars, getAppBuiltinVars(*app)...)
	}
	return envVars, nil
}

// Get the environment-level built-in env vars.
func getEnvBuiltinVars(environment envmodel.Environment) envvartypes.EnvVariableList {
	envVars := make(envvartypes.EnvVariableList, 0)
	envVars = append(envVars, []envvartypes.EnvVariableObj{{
		Key:         EnvVarNameEnvType,
		Value:       environment.Type,
		Description: "The type of the environment",
		IsBuiltin:   true,
	}, {
		Key:         EnvVarNameEnvName,
		Value:       environment.Name,
		Description: "The name of the environment",
		IsBuiltin:   true,
	}}...)

	// Extend the cluster-related vars
	if environment.Cluster.ClusterID != "" {
		envVars = append(envVars, envvartypes.EnvVariableObj{
			Key:         EnvVarNameEnvNS,
			Value:       environment.Cluster.Namespace,
			Description: "The namespace configured in the environment",
			IsBuiltin:   true,
		})
		envVars = append(envVars, envvartypes.EnvVariableObj{
			Key:         EnvVarNameEnvCluster,
			Value:       environment.Cluster.ClusterID,
			Description: "The cluster ID configured in the environment",
			IsBuiltin:   true,
		})
	}
	return envVars
}

// Get the runtime built-in env vars, which does not has any configured value but has Placeholder and
// ValueFrom for config rendering and pod runtime injection.
func getRuntimeBuiltinVars() envvartypes.EnvVariableList {
	vars := make(envvartypes.EnvVariableList, 0, len(RuntimeVars))
	for _, rv := range RuntimeVars {
		vars = append(vars, newRuntimeVariable(rv.Name, rv.Description, rv.FieldPath))
	}
	return vars
}

// newRuntimeVariable creates a RuntimeVariable object from the current three
// RuntimeVar properties: name, description, and Kubernetes Downward API field path.
//
// RuntimeVariable has no actual value; Placeholder and ValueFrom are the effective
// fields for config rendering and pod runtime injection, and they are not exposed
// in user-facing API responses.
func newRuntimeVariable(name, description, fieldPath string) envvartypes.EnvVariableObj {
	return envvartypes.EnvVariableObj{
		Key:         name,
		Description: description,
		IsBuiltin:   true,
		Placeholder: RuntimeVarPlaceholder(name),
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

// Get the app-level built-in env vars.
func getAppBuiltinVars(app bkmsapp.Application) envvartypes.EnvVariableList {
	envVars := make(envvartypes.EnvVariableList, 0)
	envVars = append(envVars, envvartypes.EnvVariableObj{
		Key:         EnvVarNameAppName,
		Value:       app.Name,
		Description: "The name of the application",
		IsBuiltin:   true,
	})

	// NOTE: 目前只有 trpc 和 taf 应用需要注入容器名称变量
	if app.Type == bkmsapp.AppTypeTRPC || app.Type == bkmsapp.AppTypeTAF {
		envVars = append(envVars, envvartypes.EnvVariableObj{
			Key:         EnvVarNameContainerName,
			Value:       defaults.WorkloadMainContainerName,
			Description: "The name of the container",
			IsBuiltin:   true,
		})
	}
	return envVars
}

// ListBuiltinScoped returns built-in scoped env vars matched by the given environment.
func ListBuiltinScoped(
	ctx context.Context, store ScopedEnvVarStore, environment envmodel.Environment,
) ([]ScopedEnvVar, error) {
	matchedVars, err := store.List(
		ctx,
		environment.WorkspaceID,
		WithScopes(
			envvartypes.ScopeWorkspace,
			envvartypes.ScopeEnvType(environment.Type),
			envvartypes.ScopeEnv(environment.Name),
		),
		WithOnlyBuiltin(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "list built-in scoped env vars")
	}
	return matchedVars, nil
}
