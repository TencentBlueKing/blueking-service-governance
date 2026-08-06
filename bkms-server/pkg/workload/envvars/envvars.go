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

// Package envvars 提供应用环境变量的构建和管理功能。
// 该包是构建应用运行时环境变量的唯一权威来源。
package envvars

import (
	"context"
	"fmt"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// Builtin env variable names
const (
	// EnvVarNameAppName is the env variable name for application name
	EnvVarNameAppName = "BKMS_APP_NAME"
	// EnvVarNameContainerName is the env variable name for the name of the container
	EnvVarNameContainerName = "BKMS_CONTAINER_NAME"
	// EnvVarNameEnvType is the env variable name for the type of the environment
	EnvVarNameEnvType = "BKMS_ENV_TYPE"
	// EnvVarNameEnvName is the env variable name for the name of the environment
	EnvVarNameEnvName = "BKMS_ENV_NAME"
	// EnvVarNameEnvNS is the env variable name for the namespace of the environment
	EnvVarNameEnvNS = "BKMS_ENV_NAMESPACE"
	// EnvVarNameEnvCluster is the env variable name for the cluster ID of the environment
	EnvVarNameEnvCluster = "BKMS_ENV_CLUSTER"
	// EnvVarNamePodIP is the env variable name for the pod IP
	EnvVarNamePodIP = "BKMS_POD_IP"
	// EnvVarNamePodName is the env variable name for the pod name
	EnvVarNamePodName = "BKMS_POD_NAME"
	// EnvVarNameNodeIP is the env variable name for the node IP
	EnvVarNameNodeIP = "BKMS_NODE_IP"
)

// RuntimeVarPlaceholder returns the placeholder string for a runtime variable.
// For example, RuntimeVarPlaceholder("BKMS_POD_IP") returns "__#VAR_PLACEHOLDER#__BKMS_POD_IP__".
func RuntimeVarPlaceholder(varName string) string {
	return fmt.Sprintf("__#VAR_PLACEHOLDER#__%s__", varName)
}

// RuntimeVar defines a runtime environment variable that is injected by Kubernetes
// Downward API at pod startup time. At compile time, its value is set to a placeholder
// (e.g., "__#VAR_PLACEHOLDER#__BKMS_POD_IP__") so that config rendering preserves it.
// An init container then replaces the placeholder with the actual runtime value using sed.
type RuntimeVar struct {
	// Name is the environment variable name (e.g., "BKMS_POD_IP").
	Name string
	// Description is a human-readable description.
	Description string
	// FieldPath is the Kubernetes Downward API field path (e.g., "status.podIP").
	FieldPath string
}

// RuntimeVars is the list of all runtime variables that are injected via Kubernetes Downward API.
// These variables have their Value set to a placeholder at compile time and are resolved
// at pod startup by an init container.
var RuntimeVars = []RuntimeVar{
	{Name: EnvVarNamePodIP, Description: "Pod IP address", FieldPath: "status.podIP"},
	{Name: EnvVarNamePodName, Description: "Pod name", FieldPath: "metadata.name"},
	{Name: EnvVarNameNodeIP, Description: "Node IP address", FieldPath: "status.hostIP"},
}

// BuildAppEnvVars returns the env variables for the given application in the environment.
func BuildAppEnvVars(
	ctx context.Context,
	app *bkmsapp.Application,
	am *appmodel.AppModel,
	environment *envmodel.Environment,
	reader *UnifiedEnvVarsReader,
) (envvartypes.EnvVariableList, error) {
	return reader.ListVars(ctx, *environment, app, am)
}
