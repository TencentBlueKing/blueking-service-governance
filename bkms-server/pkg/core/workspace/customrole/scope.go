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

package customrole

import (
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// Definition 返回角色的权限与范围；内置角色的权限从授权模板反推，范围固定为全部应用 × 全部环境。
func Definition(r *role.Role) ([]role.Permission, role.AppScope, role.EnvScope) {
	if r.IsCustom() {
		return r.Permissions, lo.FromPtr(r.AppScope), lo.FromPtr(r.EnvScope)
	}
	return bkiam.BuiltinRolePermissions(r.RoleCode),
		role.AppScope{Mode: role.AppScopeAll}, role.EnvScope{Mode: role.EnvScopeAll}
}

// CoversEnv 环境范围是否覆盖标准环境 env
func CoversEnv(envScope role.EnvScope, env *envmodel.Environment) bool {
	switch envScope.Mode {
	case role.EnvScopeAll:
		return true
	case role.EnvScopeTypes:
		return lo.Contains(envScope.EnvTypes, env.Type)
	case role.EnvScopeEnvs:
		return lo.Contains(envScope.EnvIDs, env.ID.Hex())
	default:
		return false
	}
}

// CanDeploy 角色是否可以把（部分）应用部署到标准环境 env
func CanDeploy(r *role.Role, env *envmodel.Environment) bool {
	perms, _, envScope := Definition(r)
	return lo.Contains(perms, role.Perm.EnvDeploy) && CoversEnv(envScope, env)
}

// MergedScope 多个角色合并后的应用与环境范围，仅用于展示；鉴权仍按单个角色判断，不能用它拼接范围。
type MergedScope struct {
	AllApps  bool
	AppIDs   []string
	AllEnvs  bool
	EnvTypes []string
	EnvIDs   []string
}

// MergeScopes 合并多个角色的应用与环境范围
func MergeScopes(roles []*role.Role) MergedScope {
	var merged MergedScope
	for _, r := range roles {
		_, appScope, envScope := Definition(r)
		if appScope.IsAll() {
			merged.AllApps = true
		} else {
			merged.AppIDs = append(merged.AppIDs, appScope.AppIDs...)
		}
		switch envScope.Mode {
		case role.EnvScopeAll:
			merged.AllEnvs = true
		case role.EnvScopeTypes:
			merged.EnvTypes = append(merged.EnvTypes, envScope.EnvTypes...)
		case role.EnvScopeEnvs:
			merged.EnvIDs = append(merged.EnvIDs, envScope.EnvIDs...)
		}
	}
	if merged.AllApps {
		merged.AppIDs = nil
	}
	if merged.AllEnvs {
		merged.EnvTypes, merged.EnvIDs = nil, nil
	}
	merged.AppIDs = lo.Uniq(merged.AppIDs)
	merged.EnvTypes = lo.Filter(role.EnvTypes, func(t string, _ int) bool { return lo.Contains(merged.EnvTypes, t) })
	merged.EnvIDs = lo.Uniq(merged.EnvIDs)
	return merged
}
