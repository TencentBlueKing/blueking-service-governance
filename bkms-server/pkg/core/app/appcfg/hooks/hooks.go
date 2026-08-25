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

// Package hooks 注册配置文件模块相关的领域事件钩子，如环境删除时清理 plain 环境实例
package hooks

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

const CleanupPlainEnvInstancesHookName = "appcfg.cleanup_plain_env_instances"

// RegisterDeleteHooks 注册配置文件模块的环境删除 Hook。
func RegisterDeleteHooks(
	appStore bkmsapp.ApplicationStore,
	fileStore appcfg.AppConfigFileStore,
	versionStore appcfg.AppConfigFileVersionStore,
) {
	bkmsenv.RegisterDeleteHook(
		CleanupPlainEnvInstancesHookName,
		newCleanupPlainEnvInstancesHook(appStore, fileStore, versionStore),
	)
}

// newCleanupPlainEnvInstancesHook 创建一个在环境删除前清理 plain 环境实例的 Hook。
func newCleanupPlainEnvInstancesHook(
	appStore bkmsapp.ApplicationStore,
	fileStore appcfg.AppConfigFileStore,
	versionStore appcfg.AppConfigFileVersionStore,
) bkmsenv.DeleteHook {
	return func(ctx context.Context, environment envmodel.Environment) error {
		appIDs := make([]string, 0)
		if environment.IsFeatureEnv() {
			if environment.OwnerAppID != "" {
				appIDs = append(appIDs, environment.OwnerAppID)
			}
		} else {
			apps, err := appStore.ListApps(ctx, &bkmsapp.ListOpts{WorkspaceID: environment.WorkspaceID})
			if err != nil {
				return errors.Wrap(err, "list apps for workspace")
			}
			for _, app := range apps {
				appIDs = append(appIDs, app.ID)
			}
		}

		cfgService := appcfg.NewAppConfigFileService(fileStore, versionStore)
		for _, appID := range appIDs {
			if err := cfgService.CleanupPlainEnvInstancesByEnv(ctx, appID, environment.Name); err != nil {
				return err
			}
		}
		return nil
	}
}
