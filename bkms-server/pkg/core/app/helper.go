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

package app

import (
	"context"
	"sort"
	"time"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

// AppWithDetails app 信息 + 用户最近操作时间
type AppWithDetails struct {
	*Application
	LastOperatedAt time.Time
}

// ListSortByOpTime 查询指定 workspace 下的 app 列表，
// 附带指定用户的最近操作时间，按操作时间倒序排序。
// 不包含权限过滤。
func ListSortByOpTime(
	ctx context.Context,
	appStore ApplicationStore,
	opStore audit.OperationRecordStore,
	workspaceID, username string,
) ([]AppWithDetails, error) {
	apps, err := appStore.ListApps(ctx, &ListOpts{WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return nil, nil
	}

	appIDs := make([]string, 0, len(apps))
	for _, a := range apps {
		appIDs = append(appIDs, a.ID)
	}

	// 查询用户操作时间
	latestOpTimes, err := opStore.GetLatestOperationTimesByAppsForUser(ctx, workspaceID, appIDs, username)
	if err != nil {
		log.Warnf(ctx, "get latest operation times by apps for user: %s: %v", username, err)
		latestOpTimes = nil
	}

	result := make([]AppWithDetails, 0, len(apps))
	for _, app := range apps {
		item := AppWithDetails{
			Application: app,
		}
		if t, ok := latestOpTimes[app.ID]; ok {
			item.LastOperatedAt = t
		}
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].LastOperatedAt.After(result[j].LastOperatedAt)
	})

	return result, nil
}
