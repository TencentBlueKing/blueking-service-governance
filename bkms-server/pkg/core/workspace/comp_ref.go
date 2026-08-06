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

package workspace

import (
	"context"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// ComponentRefResolver 用于构建空间组件的引用关系
type ComponentRefResolver struct {
	appStore      bkmsapp.ApplicationStore
	appModelStore appmodel.AppModelStore
}

// NewComponentRefResolver 创建 ComponentRefBuilder 实例
func NewComponentRefResolver(
	appStore bkmsapp.ApplicationStore,
	appModelStore appmodel.AppModelStore,
) *ComponentRefResolver {
	return &ComponentRefResolver{
		appStore:      appStore,
		appModelStore: appModelStore,
	}
}

// BuildRefMap 构建指定 workspace 下所有 TRPC 应用对空间组件的引用关系
// key: 空间组件名称, value: 引用该组件的应用 ID 列表
func (b *ComponentRefResolver) BuildRefMap(ctx context.Context, workspaceID string) (map[string][]string, error) {
	apps, err := b.appStore.ListApps(ctx, &bkmsapp.ListOpts{
		WorkspaceID: workspaceID,
		AppType:     bkmsapp.AppTypeTRPC,
	})
	if err != nil {
		return nil, errors.Wrap(err, "list apps")
	}

	// 构建组件引用关系映射, key 为组件名称, value 为引用该组件的 appID 列表
	compRefMap := make(map[string][]string)
	for _, app := range apps {
		appModel, err := b.appModelStore.GetAppModel(ctx, app.ID)
		if err != nil {
			// 正常情况下 trpc App 都应该有应用模型，不过这里避免异常 App 阻塞环境组件获取
			if errors.Is(err, appmodel.ErrAppModelNotFound) {
				log.Warnf(ctx, "app %s model not found, skip", app.ID)
				continue
			}
			return nil, errors.Wrap(err, "get app model")
		}
		for _, comp := range appModel.Components {
			if comp.RefWorkspaceCompName != "" {
				compRefMap[comp.RefWorkspaceCompName] = append(compRefMap[comp.RefWorkspaceCompName], app.ID)
			}
		}
	}

	return compRefMap, nil
}
