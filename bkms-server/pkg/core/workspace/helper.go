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
	"sort"
	"time"

	"github.com/samber/lo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

// WorkspaceWithOpTime workspace 信息 + 用户最近操作时间
type WorkspaceWithOpTime struct {
	Workspace
	LastOperatedAt time.Time
}

// ListSortByOpTime 查询所有 workspace 并按指定用户的最近操作时间倒序排序。
// 不包含权限过滤
func ListSortByOpTime(
	ctx context.Context,
	wsStore WorkspaceStore,
	opStore audit.OperationRecordStore,
	username string,
) ([]WorkspaceWithOpTime, error) {
	readyState := StateReady
	workspaces, err := wsStore.List(ctx, &ListOptions{State: &readyState})
	if err != nil {
		return nil, err
	}
	if len(workspaces) == 0 {
		return nil, nil
	}

	workspaceIDs := lo.Map(workspaces, func(ws Workspace, _ int) string { return ws.ID })

	latestOpTimes, err := opStore.GetLatestOperationTimesByWorkspacesForUser(ctx, workspaceIDs, username)
	if err != nil {
		log.Warnf(ctx, "get latest operation times by workspaces for user: %s: %v", username, err)
		latestOpTimes = nil
	}

	result := make([]WorkspaceWithOpTime, 0, len(workspaces))
	for i := range workspaces {
		item := WorkspaceWithOpTime{Workspace: workspaces[i]}
		if t, ok := latestOpTimes[workspaces[i].ID]; ok {
			item.LastOperatedAt = t
		}
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		if !result[i].LastOperatedAt.Equal(result[j].LastOperatedAt) {
			return result[i].LastOperatedAt.After(result[j].LastOperatedAt)
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}
