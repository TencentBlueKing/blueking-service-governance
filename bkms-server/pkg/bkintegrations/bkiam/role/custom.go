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

package role

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

// CustomRoleCode 自定义角色的角色码
const CustomRoleCode = "custom"

// Permission 页面上可勾选的权限项，与内置权限矩阵一致
type Permission = string

// Perm 页面权限项
var Perm = struct {
	SpaceView, SpaceCreate, SpaceEdit, SpaceDelete             Permission
	AppCreate, AppView, AppObservationView, AppEdit, AppDelete Permission
	EnvView, EnvDeploy, EnvCreate, EnvEdit, EnvDelete          Permission
}{
	SpaceView:          "space.view",
	SpaceCreate:        "space.create",
	SpaceEdit:          "space.edit",
	SpaceDelete:        "space.delete",
	AppCreate:          "app.create",
	AppView:            "app.view",
	AppObservationView: "app.observation.view",
	AppEdit:            "app.edit",
	AppDelete:          "app.delete",
	EnvView:            "env.view",
	EnvDeploy:          "env.deploy",
	EnvCreate:          "env.create",
	EnvEdit:            "env.edit",
	EnvDelete:          "env.delete",
}

// AllPermissions 全部权限项，按页面展示顺序排列
var AllPermissions = []Permission{
	Perm.SpaceView, Perm.SpaceCreate, Perm.SpaceEdit, Perm.SpaceDelete,
	Perm.AppCreate, Perm.AppView, Perm.AppObservationView, Perm.AppEdit, Perm.AppDelete,
	Perm.EnvView, Perm.EnvDeploy, Perm.EnvCreate, Perm.EnvEdit, Perm.EnvDelete,
}

// AppScopeMode 应用范围类型
type AppScopeMode = string

const (
	// AppScopeAll 全部应用，包括之后新建的应用
	AppScopeAll AppScopeMode = "all"
	// AppScopeApps 指定应用
	AppScopeApps AppScopeMode = "apps"
)

// EnvScopeMode 环境范围类型
type EnvScopeMode = string

const (
	// EnvScopeAll 全部环境，包括之后新建的环境
	EnvScopeAll EnvScopeMode = "all"
	// EnvScopeTypes 按环境类型，包括该类型下之后新建的环境
	EnvScopeTypes EnvScopeMode = "types"
	// EnvScopeEnvs 指定标准环境
	EnvScopeEnvs EnvScopeMode = "envs"
)

// EnvTypes 可选的环境类型
var EnvTypes = []string{"development", "test", "staging", "production"}

// AppScope 角色的应用范围
type AppScope struct {
	Mode   AppScopeMode `bson:"mode"`
	AppIDs []string     `bson:"appIDs"`
}

// IsAll 是否覆盖全部应用
func (s AppScope) IsAll() bool { return s.Mode == AppScopeAll }

// EnvScope 角色的环境范围
type EnvScope struct {
	Mode     EnvScopeMode `bson:"mode"`
	EnvTypes []string     `bson:"envTypes"`
	// EnvIDs 指定的标准环境 ObjectID
	EnvIDs []string `bson:"envIDs"`
}

// IsAll 是否覆盖全部环境
func (s EnvScope) IsAll() bool { return s.Mode == EnvScopeAll }

// ErrInvalidCustomRole 自定义角色的权限或范围不合法
var ErrInvalidCustomRole = errors.New("invalid custom role")

// NormalizePermissions 校验并补齐权限项：所有角色都带上查看空间，修改或部署类权限带上对应的查看权限。
//
// 返回按 AllPermissions 顺序排列、去重后的权限项。
func NormalizePermissions(perms []Permission, appScope AppScope, envScope EnvScope) ([]Permission, error) {
	if err := validateScopes(appScope, envScope); err != nil {
		return nil, err
	}
	for _, p := range perms {
		if !lo.Contains(AllPermissions, p) {
			return nil, errors.Wrapf(ErrInvalidCustomRole, "unknown permission %s", p)
		}
		if p == Perm.SpaceCreate {
			return nil, errors.Wrapf(ErrInvalidCustomRole, "permission %s is not workspace scoped", p)
		}
	}

	set := lo.SliceToMap(perms, func(p Permission) (Permission, struct{}) { return p, struct{}{} })
	set[Perm.SpaceView] = struct{}{}
	for _, p := range []Permission{Perm.AppObservationView, Perm.AppEdit, Perm.AppDelete, Perm.EnvDeploy} {
		if _, ok := set[p]; ok {
			set[Perm.AppView] = struct{}{}
		}
	}
	for _, p := range []Permission{Perm.EnvDeploy, Perm.EnvEdit, Perm.EnvDelete} {
		if _, ok := set[p]; ok {
			set[Perm.EnvView] = struct{}{}
		}
	}

	// 编辑、删除环境作用于环境下的全部应用，编辑、删除空间作用于整个空间，范围不是全部时授不出任何权限。
	// 创建应用、创建环境挂在空间上，删除应用不涉及环境，都不看范围
	requireAllApps := []Permission{Perm.EnvEdit, Perm.EnvDelete}
	requireAll := []Permission{Perm.SpaceEdit, Perm.SpaceDelete}
	for p := range set {
		if lo.Contains(requireAllApps, p) && !appScope.IsAll() {
			return nil, errors.Wrapf(ErrInvalidCustomRole, "permission %s requires all apps", p)
		}
		if lo.Contains(requireAll, p) && (!appScope.IsAll() || !envScope.IsAll()) {
			return nil, errors.Wrapf(ErrInvalidCustomRole, "permission %s requires all apps and all envs", p)
		}
	}

	return lo.Filter(AllPermissions, func(p Permission, _ int) bool {
		_, ok := set[p]
		return ok
	}), nil
}

func validateScopes(appScope AppScope, envScope EnvScope) error {
	switch appScope.Mode {
	case AppScopeAll:
	case AppScopeApps:
		if len(appScope.AppIDs) == 0 {
			return errors.Wrap(ErrInvalidCustomRole, "app scope requires at least one app")
		}
	default:
		return errors.Wrapf(ErrInvalidCustomRole, "unknown app scope mode %s", appScope.Mode)
	}

	switch envScope.Mode {
	case EnvScopeAll:
	case EnvScopeTypes:
		if len(envScope.EnvTypes) == 0 {
			return errors.Wrap(ErrInvalidCustomRole, "env scope requires at least one env type")
		}
		for _, t := range envScope.EnvTypes {
			if !slices.Contains(EnvTypes, t) {
				return errors.Wrapf(ErrInvalidCustomRole, "unknown env type %s", t)
			}
		}
	case EnvScopeEnvs:
		if len(envScope.EnvIDs) == 0 {
			return errors.Wrap(ErrInvalidCustomRole, "env scope requires at least one env")
		}
	default:
		return errors.Wrapf(ErrInvalidCustomRole, "unknown env scope mode %s", envScope.Mode)
	}
	return nil
}
