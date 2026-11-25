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

// Package customrole 维护自定义角色与空间内应用、环境的关系：生成授权所需的数据，
// 并在应用或环境删除、环境类型变更时收敛自定义角色的授权。
package customrole

import (
	"context"
	"slices"
	"strconv"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// BuildData 校验并补齐权限，校验范围内的应用和标准环境都属于该空间，生成授权所需的数据。
//
// 用户提交的定义不合法时返回的错误满足 errors.Is(err, role.ErrInvalidCustomRole)。
//
// 读取的模型：app、env、APM 实例配置。
func BuildData(
	ctx context.Context,
	db *mongo.Database,
	ws *workspace.Workspace,
	perms []role.Permission,
	appScope role.AppScope,
	envScope role.EnvScope,
) (bkiam.CustomRoleData, error) {
	appScope.AppIDs = lo.Uniq(appScope.AppIDs)
	envScope.EnvTypes = lo.Uniq(envScope.EnvTypes)
	envScope.EnvIDs = lo.Uniq(envScope.EnvIDs)

	normalized, err := role.NormalizePermissions(perms, appScope, envScope)
	if err != nil {
		return bkiam.CustomRoleData{}, err
	}
	data, err := loadData(ctx, db, ws, normalized, appScope, envScope, bson.NilObjectID)
	if err != nil {
		return data, err
	}
	if appScope.Mode == role.AppScopeApps && len(data.Apps) != len(appScope.AppIDs) {
		return data, errors.Wrap(role.ErrInvalidCustomRole, "some apps do not exist in the workspace")
	}
	if envScope.Mode == role.EnvScopeEnvs && len(data.Envs) != len(envScope.EnvIDs) {
		return data, errors.Wrap(role.ErrInvalidCustomRole, "some envs do not exist or are not standard envs")
	}
	return data, nil
}

// loadData 加载范围内应用与环境的展示信息，已不存在或正被删除（excludeEnvID）的资源会被忽略
func loadData(
	ctx context.Context,
	db *mongo.Database,
	ws *workspace.Workspace,
	perms []role.Permission,
	appScope role.AppScope,
	envScope role.EnvScope,
	excludeEnvID bson.ObjectID,
) (bkiam.CustomRoleData, error) {
	data := bkiam.CustomRoleData{
		WorkspaceID:   ws.ID,
		WorkspaceName: ws.DisplayName,
		Permissions:   perms,
		AppScope:      appScope,
		EnvScope:      envScope,
	}
	if appScope.Mode == role.AppScopeApps && len(appScope.AppIDs) > 0 {
		apps, err := store.Of(db, bkmsapp.AppMeta).List(
			ctx, bkmsapp.InWorkspace(ws.ID), store.Where("id", bson.M{"$in": appScope.AppIDs}),
		)
		if err != nil {
			return data, errors.Wrapf(err, "list apps of workspace %s", ws.ID)
		}
		data.Apps = lo.Map(apps, func(app bkmsapp.Application, _ int) bkiam.ResourceRef {
			return bkiam.ResourceRef{ID: app.ID, Name: lo.CoalesceOrEmpty(app.DisplayName, app.Name)}
		})
	}
	if envScope.Mode == role.EnvScopeEnvs && len(envScope.EnvIDs) > 0 {
		oids := lo.FilterMap(envScope.EnvIDs, func(id string, _ int) (any, bool) {
			oid, err := bson.ObjectIDFromHex(id)
			return oid, err == nil && oid != excludeEnvID
		})
		if len(oids) > 0 {
			envs, err := store.Of(db, envmodel.EnvMeta).List(
				ctx, envmodel.StdEnvsIn(ws.ID), store.Where("_id", bson.M{"$in": oids}),
			)
			if err != nil {
				return data, errors.Wrapf(err, "list envs of workspace %s", ws.ID)
			}
			data.Envs = lo.Map(envs, func(env envmodel.Environment, _ int) bkiam.EnvResourceRef {
				return bkiam.EnvResourceRef{
					ID: env.ID.Hex(), Name: lo.CoalesceOrEmpty(env.DisplayName, env.Name), Type: env.Type,
				}
			})
		}
	}
	monitor, err := loadMonitorData(ctx, db, ws, perms, envScope, data.Envs, excludeEnvID)
	if err != nil {
		return data, err
	}
	data.BKMonitor = monitor
	return data, nil
}

// loadMonitorData 加载范围内环境绑定的 APM 应用，监控项目尚未就绪时返回 nil
func loadMonitorData(
	ctx context.Context,
	db *mongo.Database,
	ws *workspace.Workspace,
	perms []role.Permission,
	envScope role.EnvScope,
	envs []bkiam.EnvResourceRef,
	excludeEnvID bson.ObjectID,
) (*bkiam.CustomRoleMonitorData, error) {
	spaceID := ws.BkSystems.BkMonitorProjectID
	if spaceID == "" {
		return nil, nil
	}
	data := &bkiam.CustomRoleMonitorData{SpaceID: spaceID, SpaceName: ws.DisplayName}
	// 无应用查看权限时不授予监控权限，但仍返回非空 BKMonitor 以便修改角色时撤销旧监控策略
	if !lo.Contains(perms, role.Perm.AppView) {
		return data, nil
	}

	var envIDs []bson.ObjectID
	switch envScope.Mode {
	case role.EnvScopeAll:
		data.AllApms = true
		return data, nil
	case role.EnvScopeTypes:
		typedEnvs, err := store.Of(db, envmodel.EnvMeta).List(
			ctx, envmodel.StdEnvsIn(ws.ID), store.Where("type", bson.M{"$in": envScope.EnvTypes}),
		)
		if err != nil {
			return nil, errors.Wrapf(err, "list envs of types %v", envScope.EnvTypes)
		}
		envIDs = lo.FilterMap(typedEnvs, func(env envmodel.Environment, _ int) (bson.ObjectID, bool) {
			return env.ID, env.ID != excludeEnvID
		})
	case role.EnvScopeEnvs:
		envIDs = lo.FilterMap(envs, func(env bkiam.EnvResourceRef, _ int) (bson.ObjectID, bool) {
			oid, err := bson.ObjectIDFromHex(env.ID)
			return oid, err == nil && oid != excludeEnvID
		})
	}
	if len(envIDs) == 0 {
		return data, nil
	}
	// 多个环境可能共用一个 APM 应用，共用时授权会覆盖到范围外的环境
	apms, err := store.Of(db, bkmonitor.ApmInstConfigMeta).List(
		ctx, bkmonitor.ApmInWorkspace(ws.ID), store.Where("associatedEnvs.envID", bson.M{"$in": envIDs}),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list apms of workspace %s", ws.ID)
	}
	data.Apms = lo.Map(apms, func(apm bkmonitor.ApmInstConfig, _ int) bkiam.ResourceRef {
		return bkiam.ResourceRef{ID: strconv.FormatInt(apm.ApmID, 10), Name: apm.Name}
	})
	return data, nil
}

// PruneApp 从自定义角色的指定应用中去掉 appID，并重新授权。去掉后为空的范围不覆盖任何应用。
//
// 读取的模型：workspace、app、env、APM 实例配置。
func PruneApp(ctx context.Context, db *mongo.Database, permMgr perm.Manager, workspaceID, appID string) error {
	return reapply(ctx, db, permMgr, workspaceID, bson.NilObjectID, func(_ *workspace.Workspace, r *role.Role) bool {
		if r.AppScope == nil || r.AppScope.Mode != role.AppScopeApps || !lo.Contains(r.AppScope.AppIDs, appID) {
			return false
		}
		r.AppScope.AppIDs = lo.Without(r.AppScope.AppIDs, appID)
		return true
	})
}

// PruneEnv 在标准环境删除时收敛自定义角色的授权：
//   - 指定了该环境的角色：从 EnvIDs 中移除并重新授权（移除后为空的范围不覆盖任何环境）；
//   - 按环境类型覆盖该环境的角色：重新计算剩余环境绑定的 APM 应用并重新授权。
//
// 环境删除 Hook 在数据库记录物理删除前执行，因此通过 excludeEnvID 将正被删除的环境排除在外。
//
// 读取的模型：workspace、app、env、APM 实例配置。
func PruneEnv(ctx context.Context, db *mongo.Database, permMgr perm.Manager, env *envmodel.Environment) error {
	if env.IsFeatureEnv() {
		return nil
	}
	envID := env.ID.Hex()
	return reapply(ctx, db, permMgr, env.WorkspaceID, env.ID, func(ws *workspace.Workspace, r *role.Role) bool {
		if r.EnvScope == nil || r.EnvScope.IsAll() {
			return false
		}
		if r.EnvScope.Mode == role.EnvScopeEnvs && lo.Contains(r.EnvScope.EnvIDs, envID) {
			r.EnvScope.EnvIDs = lo.Without(r.EnvScope.EnvIDs, envID)
			return true
		}
		return r.EnvScope.Mode == role.EnvScopeTypes &&
			lo.Contains(r.EnvScope.EnvTypes, env.Type) &&
			ws.BkSystems.BkMonitorProjectID != "" &&
			lo.Contains(r.Permissions, role.Perm.AppView)
	})
}

// ReapplyEnv 在标准环境类型变更后重新授权受影响的自定义角色：
//   - 指定了该环境的角色：授权路径包含 env_type，需按新类型重新授权；
//   - 按变更前或变更后环境类型覆盖该环境的角色：监控 APM 应用范围随环境类型增减，需重新授权。
//
// 读取的模型：workspace、app、env、APM 实例配置。
func ReapplyEnv(
	ctx context.Context, db *mongo.Database, permMgr perm.Manager, before, after *envmodel.Environment,
) error {
	if after.IsFeatureEnv() || before.Type == after.Type {
		return nil
	}
	envID := after.ID.Hex()
	return reapply(
		ctx, db, permMgr, after.WorkspaceID, bson.NilObjectID,
		func(ws *workspace.Workspace, r *role.Role) bool {
			if r.EnvScope == nil || r.EnvScope.IsAll() {
				return false
			}
			if r.EnvScope.Mode == role.EnvScopeEnvs && lo.Contains(r.EnvScope.EnvIDs, envID) {
				return true
			}
			return (CoversEnv(*r.EnvScope, before) || CoversEnv(*r.EnvScope, after)) &&
				ws.BkSystems.BkMonitorProjectID != "" &&
				lo.Contains(r.Permissions, role.Perm.AppView)
		},
	)
}

// ReapplyEnvApm 重新授权覆盖了标准环境 env 的自定义角色，用于环境绑定的 APM 应用变化之后：
// 环境创建并绑定、或改绑 APM 应用。环境范围为全部的角色授予的是空间下全部 APM 应用，不需要重新授权。
//
// 读取的模型：workspace、app、env、APM 实例配置。
func ReapplyEnvApm(ctx context.Context, db *mongo.Database, permMgr perm.Manager, env *envmodel.Environment) error {
	if env.IsFeatureEnv() {
		return nil
	}
	return reapply(
		ctx, db, permMgr, env.WorkspaceID, bson.NilObjectID,
		func(ws *workspace.Workspace, r *role.Role) bool {
			return ws.BkSystems.BkMonitorProjectID != "" &&
				lo.Contains(r.Permissions, role.Perm.AppView) &&
				r.EnvScope != nil && !r.EnvScope.IsAll() && CoversEnv(*r.EnvScope, env)
		},
	)
}

// ReapplyAll 重新授权空间下的全部自定义角色，用于监控项目就绪之后补上监控授权。
//
// 读取的模型：workspace、app、env、APM 实例配置。
func ReapplyAll(ctx context.Context, db *mongo.Database, permMgr perm.Manager, workspaceID string) error {
	return reapply(
		ctx, db, permMgr, workspaceID, bson.NilObjectID,
		func(_ *workspace.Workspace, r *role.Role) bool {
			return lo.Contains(r.Permissions, role.Perm.AppView)
		},
	)
}

// reapply 对 mutate 返回 true 的自定义角色按修改后的定义重新授权
func reapply(
	ctx context.Context,
	db *mongo.Database,
	permMgr perm.Manager,
	workspaceID string,
	excludeEnvID bson.ObjectID,
	mutate func(ws *workspace.Workspace, r *role.Role) bool,
) error {
	ws, err := store.Of(db, workspace.Meta).Get(ctx, workspace.ByWorkspaceID(workspaceID))
	if err != nil {
		return errors.Wrapf(err, "get workspace %s", workspaceID)
	}
	roles, err := permMgr.ListRoles(ctx, workspaceID)
	if err != nil {
		return errors.Wrapf(err, "list roles of workspace %s", workspaceID)
	}
	for _, r := range roles {
		if !r.IsCustom() {
			continue
		}
		if err = reapplyRole(ctx, db, permMgr, ws, r, excludeEnvID, mutate); err != nil {
			return err
		}
	}
	return nil
}

func reapplyRole(
	ctx context.Context,
	db *mongo.Database,
	permMgr perm.Manager,
	ws *workspace.Workspace,
	r *role.Role,
	excludeEnvID bson.ObjectID,
	mutate func(ws *workspace.Workspace, r *role.Role) bool,
) error {
	// 在副本上修改，避免改动调用方或缓存中的角色对象
	r = cloneRole(r)
	if !mutate(ws, r) {
		return nil
	}
	data, err := loadData(ctx, db, ws, r.Permissions, lo.FromPtr(r.AppScope), lo.FromPtr(r.EnvScope), excludeEnvID)
	if err != nil {
		return errors.Wrapf(err, "build data of role %s", r.ID)
	}
	if _, err = permMgr.UpdateCustomRole(ctx, r.ID, r.DisplayName, r.Description, data); err != nil {
		return errors.Wrapf(err, "reapply role %s", r.ID)
	}
	return nil
}

// cloneRole 复制角色，范围中的切片也一并复制
func cloneRole(r *role.Role) *role.Role {
	c := *r
	c.Permissions = slices.Clone(r.Permissions)
	if r.AppScope != nil {
		appScope := *r.AppScope
		appScope.AppIDs = slices.Clone(r.AppScope.AppIDs)
		c.AppScope = &appScope
	}
	if r.EnvScope != nil {
		envScope := *r.EnvScope
		envScope.EnvTypes = slices.Clone(r.EnvScope.EnvTypes)
		envScope.EnvIDs = slices.Clone(r.EnvScope.EnvIDs)
		c.EnvScope = &envScope
	}
	return &c
}
