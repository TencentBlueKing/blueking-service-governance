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

package handler

import (
	"context"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// listVisibleGroups 返回当前用户能查看的空间操作记录范围，全部可见时返回 nil；特性环境按来源环境判断
func listVisibleGroups(
	ctx context.Context, registry *storereg.Registry, workspaceID string,
) (*audit.VisibleGroups, error) {
	apps, err := store.Of(registry.DB, bkmsapp.AppMeta).List(ctx, bkmsapp.InWorkspace(workspaceID))
	if err != nil {
		return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list apps of workspace %s", workspaceID)
	}
	envs, err := store.Of(registry.DB, envmodel.EnvMeta).List(ctx, envmodel.InWorkspace(workspaceID))
	if err != nil {
		return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list envs of workspace %s", workspaceID)
	}

	viewableEnvs, err := ginperm.FilterViewableEnvs(ctx, registry, workspaceID, envs)
	if err != nil {
		return nil, err
	}
	viewableApps, err := perm.NewManager().FilterViewableApps(
		ctx, workspaceID, lo.Map(apps, func(app bkmsapp.Application, _ int) string { return app.ID }),
	)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "filter viewable apps")
	}
	envsByApp := map[string][]envmodel.Environment{}
	for _, app := range apps {
		if viewableApps.Has(app.ID) {
			envsByApp[app.ID] = lo.Filter(envs, func(env envmodel.Environment, _ int) bool {
				return env.IsAvailableToApp(app.ID)
			})
		}
	}
	viewableByApp, err := ginperm.FilterViewableAppEnvNamesByApp(ctx, registry, workspaceID, envsByApp)
	if err != nil {
		return nil, err
	}

	groups := &audit.VisibleGroups{
		EnvNames:    lo.Map(viewableEnvs, func(env envmodel.Environment, _ int) string { return env.Name }),
		AppEnvNames: make(map[string][]string, len(envsByApp)),
	}
	all := viewableApps.Size() == len(apps) && len(viewableEnvs) == len(envs)
	for appID, appEnvs := range envsByApp {
		viewable := viewableByApp[appID]
		// 全部环境可见时记为 nil，以便已删除环境的记录仍然可见
		if viewable.Size() == len(appEnvs) {
			groups.AppEnvNames[appID] = nil
			continue
		}
		groups.AppEnvNames[appID] = append([]string{}, viewable.ToSlice()...)
		all = false
	}
	if all {
		return nil, nil
	}
	return groups, nil
}
