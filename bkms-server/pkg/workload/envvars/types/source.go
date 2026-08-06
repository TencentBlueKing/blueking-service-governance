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

package types

// EnvVarSource identifies where an env var comes from.
type EnvVarSource string

const (
	// EnvVarSourceBuiltin is for system built-in env vars.
	// INFO: A scoped env var with isBuiltin set to true is considered as builtin source not scoped.
	EnvVarSourceBuiltin EnvVarSource = "builtin"

	// EnvVarSourceScopedWorkspace is for scoped env vars defined at workspace scope.
	EnvVarSourceScopedWorkspace EnvVarSource = "scopedWorkspace"
	// EnvVarSourceScopedEnvType is for scoped env vars defined at envType scope.
	EnvVarSourceScopedEnvType EnvVarSource = "scopedEnvType"
	// EnvVarSourceScopedEnv is for scoped env vars defined at env scope.
	EnvVarSourceScopedEnv EnvVarSource = "scopedEnv"

	// EnvVarSourceAppDeps is for env vars produced by depservice service instances
	// (provider builtin + user-defined CustomEnvVars). They are scoped between ScopedEnv
	// (40) and App (50) so that they cannot be overridden by any scoped env var, but can
	// still be overridden by AppModel-level env vars.
	EnvVarSourceAppDeps EnvVarSource = "appDeps"

	// EnvVarSourcePolaris is for env vars produced by polaris configs (polarisToken /
	// servicePort). They sit just above AppDeps so they cannot be overridden by any scoped
	// env var, but can still be overridden by AppModel-level env vars.
	EnvVarSourcePolaris EnvVarSource = "polaris"

	// EnvVarSourceApp is for app-defined env vars (e.g. AppModel env vars).
	EnvVarSourceApp EnvVarSource = "app"
)

// EnvVarSourcePriority gets the priority of an env var source.
func EnvVarSourcePriority(source EnvVarSource) int {
	switch source {
	case EnvVarSourceBuiltin:
		return 10
	case EnvVarSourceScopedWorkspace:
		return 20
	case EnvVarSourceScopedEnvType:
		return 30
	case EnvVarSourceScopedEnv:
		return 40
	case EnvVarSourceAppDeps:
		return 45
	case EnvVarSourcePolaris:
		return 46
	case EnvVarSourceApp:
		return 50
	default:
		return 0
	}
}

// ConflictedSource identifies one source and its source value.
//
// SourceValue examples:
// - builtin: ""
// - scopedWorkspace: workspace ID
// - scopedEnvType: environment type
// - scopedEnv: environment name
// - serviceInstance: service instance name
// - app: app ID
type ConflictedSource struct {
	Source      EnvVarSource
	SourceValue string
}

// EnvVarConflictedInfo describes the conflict status of an env var, including which other sources
// it conflicts with and whether it is the effective one among the conflicts.
type EnvVarConflictedInfo struct {
	// ConflictedSources records all other sources that conflict with the current env var.
	ConflictedSources []ConflictedSource

	// OverrideConflicted indicates whether the current env var is the effective one in its conflict set.
	// INFO: Any env var with this field set to false is considered overridden and should not be used in runtime.
	OverrideConflicted bool

	// ConflictedDetail contains more detailed conflict information.
	ConflictedDetail string
}
