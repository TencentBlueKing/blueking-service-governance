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

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// WorkspaceWithOpTime workspace 信息 + 用户最近操作时间
type WorkspaceWithOpTime struct {
	Workspace
	LastOperatedAt time.Time
}

// ListSortByOpTime 查询所有 workspace 并按指定用户的最近操作时间倒序排序。
// 不包含权限过滤
func ListSortByOpTime(ctx context.Context, db *mongo.Database, username string) ([]WorkspaceWithOpTime, error) {
	workspaces, err := store.Of(db, Meta).List(ctx, InState(StateReady))
	if err != nil {
		return nil, errors.Wrap(err, "list ready workspaces")
	}
	if len(workspaces) == 0 {
		return nil, nil
	}

	workspaceIDs := lo.Map(workspaces, func(ws Workspace, _ int) string { return ws.ID })

	latestOpTimes, err := audit.GetLatestOperationTimesByWorkspacesForUser(ctx, db, workspaceIDs, username)
	if err != nil {
		log.Warnf(ctx, "list workspaces sorted by operation time: %v", err)
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

// ListRoleMembers 聚合 workspace 下指定角色（roleCodes）的成员，按 ID 去重并过滤空字符串。
func ListRoleMembers(
	ctx context.Context,
	workspaceID string,
	roleCodes ...string,
) ([]string, error) {
	permMgr := perm.NewManager()

	var allMembers []string
	for _, roleCode := range roleCodes {
		role, err := permMgr.GetRole(ctx, workspaceID, roleCode)
		if err != nil {
			return nil, errors.Wrapf(err, "get role(%s) of workspace(%s)", roleCode, workspaceID)
		}
		members, err := permMgr.ListRoleMembers(ctx, role.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "list role(%s) members of workspace(%s)", roleCode, workspaceID)
		}
		allMembers = append(allMembers, members...)
	}

	// 过滤空字符串、去重并排序
	result := lo.Uniq(lo.Filter(allMembers, func(m string, _ int) bool { return m != "" }))
	sort.Strings(result)

	return result, nil
}
