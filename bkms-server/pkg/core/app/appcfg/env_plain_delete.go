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

package appcfg

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// PlainEnvInstanceCleaner 在环境物理删除前同步清理 plain 环境实例。
type PlainEnvInstanceCleaner struct {
	appStore     bkmsapp.ApplicationStore
	fileStore    AppConfigFileStore
	versionStore AppConfigFileVersionStore
}

// NewPlainEnvInstanceCleaner 创建 plain 环境实例清理器。
func NewPlainEnvInstanceCleaner(
	appStore bkmsapp.ApplicationStore,
	fileStore AppConfigFileStore,
	versionStore AppConfigFileVersionStore,
) *PlainEnvInstanceCleaner {
	return &PlainEnvInstanceCleaner{
		appStore:     appStore,
		fileStore:    fileStore,
		versionStore: versionStore,
	}
}

// CleanupBeforeDelete 在环境删除前清理对应应用下的 plain 环境实例。
func (c *PlainEnvInstanceCleaner) CleanupBeforeDelete(
	ctx context.Context,
	environment envmodel.Environment,
) error {
	appIDs := make([]string, 0)
	if environment.IsFeatureEnv() {
		if environment.OwnerAppID != "" {
			appIDs = append(appIDs, environment.OwnerAppID)
		}
	} else {
		apps, err := c.appStore.ListApps(ctx, &bkmsapp.ListOpts{WorkspaceID: environment.WorkspaceID})
		if err != nil {
			return errors.Wrap(err, "list apps for workspace")
		}
		for _, app := range apps {
			appIDs = append(appIDs, app.ID)
		}
	}

	cfgService := NewAppConfigFileService(c.fileStore, c.versionStore)
	for _, appID := range appIDs {
		if err := cfgService.CleanupPlainEnvInstancesByEnv(ctx, appID, environment.Name); err != nil {
			return err
		}
	}
	return nil
}
