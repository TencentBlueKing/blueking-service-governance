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
	"fmt"
	"sort"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup"
)

const defaultAlertUserGroupDesc = "BKMS workspace default alert user group"

type defaultAlertRoleManager interface {
	ListRoles(ctx context.Context, workspaceID string) ([]*role.Role, error)
	ListRoleMembers(ctx context.Context, roleID string) ([]string, error)
}

type defaultAlertUserGroupService interface {
	FindByName(ctx context.Context, ws *workspace.Workspace, name, operator string) (*bkmapi.UserGroup, error)
	Save(ctx context.Context, ws *workspace.Workspace, params *usergroup.SaveParams) (*bkmapi.UserGroupDetail, error)
}

func buildDefaultAlertUserGroupName(workspaceID string) string {
	return fmt.Sprintf("【BKMS】%s 默认告警组", workspaceID)
}

func resolveDefaultAlertNoticeGroupIDs(
	ctx context.Context,
	ws *workspace.Workspace,
	groupSvc defaultAlertUserGroupService,
	permMgr defaultAlertRoleManager,
	operator string,
) ([]int64, error) {
	if ws == nil {
		return nil, errors.New("workspace is nil")
	}
	members, err := listDefaultAlertGroupMembers(ctx, permMgr, ws.ID)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, nil
	}

	groupName := buildDefaultAlertUserGroupName(ws.ID)
	group, err := groupSvc.FindByName(ctx, ws, groupName, operator)
	if err != nil {
		return nil, errors.Wrap(err, "find default bkmonitor user group by name")
	}
	if group != nil {
		return []int64{group.ID}, nil
	}

	detail, err := groupSvc.Save(ctx, ws, &usergroup.SaveParams{
		Name:         groupName,
		Channels:     []string{"user"},
		Desc:         defaultAlertUserGroupDesc,
		AlertNotice:  []bkmapi.AlertNotice{{TimeRange: "00:00--23:59"}},
		ActionNotice: []bkmapi.ActionNotice{{TimeRange: "00:00--23:59"}},
		Users:        buildDefaultAlertUserGroupUsers(members),
		Operator:     operator,
	})
	if err != nil {
		return nil, errors.Wrap(err, "create default alert user group")
	}
	return []int64{detail.ID}, nil
}

func listDefaultAlertGroupMembers(
	ctx context.Context,
	permMgr defaultAlertRoleManager,
	workspaceID string,
) ([]string, error) {
	roles, err := permMgr.ListRoles(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "list roles of workspace(%s)", workspaceID)
	}

	roleByCode := make(map[string]*role.Role, len(roles))
	for _, roleInfo := range roles {
		if roleInfo == nil {
			continue
		}
		roleByCode[roleInfo.RoleCode] = roleInfo
	}

	targetRoles := make([]*role.Role, 0, 2)
	for _, roleCode := range []string{perm.RoleCodeDeveloper, perm.RoleCodeSre} {
		roleInfo := roleByCode[roleCode]
		if roleInfo == nil {
			return nil, errors.Errorf("role(%s) of workspace(%s) not found", roleCode, workspaceID)
		}
		targetRoles = append(targetRoles, roleInfo)
	}

	memberSet := make(map[string]struct{})
	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	for _, roleInfo := range targetRoles {
		roleInfo := roleInfo
		g.Go(func() error {
			members, err := permMgr.ListRoleMembers(gCtx, roleInfo.ID)
			if err != nil {
				return errors.Wrapf(err, "list role(%s) members of workspace(%s)", roleInfo.RoleCode, workspaceID)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, member := range members {
				if member == "" {
					continue
				}
				memberSet[member] = struct{}{}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(memberSet))
	for member := range memberSet {
		result = append(result, member)
	}
	sort.Strings(result)
	return result, nil
}

func buildDefaultAlertUserGroupUsers(members []string) []bkmapi.UserGroupUser {
	users := make([]bkmapi.UserGroupUser, 0, len(members))
	for _, member := range members {
		users = append(users, bkmapi.UserGroupUser{
			ID:   member,
			Type: "user",
		})
	}
	return users
}
