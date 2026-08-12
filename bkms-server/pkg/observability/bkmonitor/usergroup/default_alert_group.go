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

package usergroup

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	distlock "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/lock"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
)

const (
	// defaultAlertUserGroupLockTimeoutSeconds 控制“确保默认告警组存在”这段临界区的锁 TTL。
	// 这里只覆盖查组/建组短流程，不包含后续默认策略初始化，因此超时保持较小即可。
	defaultAlertUserGroupLockTimeoutSeconds = 5
)

var (
	// defaultAlertUserGroupLockRetryInterval 是抢锁失败后的重试间隔。
	defaultAlertUserGroupLockRetryInterval = 100 * time.Millisecond
	// defaultAlertUserGroupLockWaitTimeout 限制等待 workspace 锁的最长时间，避免在无取消上下文下无限轮询。
	defaultAlertUserGroupLockWaitTimeout = 10 * time.Second
)

type defaultAlertRoleManager interface {
	ListRoles(ctx context.Context, workspaceID string) ([]*role.Role, error)
	ListRoleMembers(ctx context.Context, roleID string) ([]string, error)
}

// buildDefaultAlertUserGroupName 默认告警组命名
func buildDefaultAlertUserGroupName(workspaceID string) string {
	return fmt.Sprintf("【BKMS】%s 默认告警组", workspaceID)
}

// buildDefaultAlertUserGroupLockKey 返回 workspace 维度的默认告警组分布式锁 key。
// 同一工作空间下默认告警组的“查名 -> 创建”需要串行化，不同工作空间之间则允许并行。
func buildDefaultAlertUserGroupLockKey(workspaceID string) string {
	return fmt.Sprintf("lock:bkmonitor-default-alert-group:%s", workspaceID)
}

// ResolveDefaultAlertNoticeGroupIDs 解析工作空间默认告警通知用户组 ID。
func ResolveDefaultAlertNoticeGroupIDs(
	ctx context.Context,
	ws *workspace.Workspace,
	groupSvc *Service,
	permMgr defaultAlertRoleManager,
	operator string,
) ([]int64, error) {
	if ws == nil {
		return nil, errors.New("workspace is nil")
	}
	groupName := buildDefaultAlertUserGroupName(ws.ID)

	// 1. 快路径：无锁复用已存在的默认组。
	//    大多数场景下默认组早已创建完成，这里直接返回，避免为常见读路径引入锁开销。
	group, err := groupSvc.FindByName(ctx, ws, groupName, operator)
	if err != nil {
		return nil, errors.Wrap(err, "find default bkmonitor user group by name")
	}
	if group != nil {
		return []int64{group.ID}, nil
	}

	// 2. 仅在“确认默认组不存在”时才竞争 workspace 维度分布式锁。
	//    锁只保护默认组存在性收敛（查组/建组），不覆盖后续默认策略初始化，避免临界区过大。
	groupLock := distlock.NewRedisLock(
		buildDefaultAlertUserGroupLockKey(ws.ID),
		defaultAlertUserGroupLockTimeoutSeconds,
	)
	waitCtx, cancel := context.WithTimeout(ctx, defaultAlertUserGroupLockWaitTimeout)
	defer cancel()
	for {
		if groupLock.Acquire(waitCtx) {
			defer groupLock.Release(waitCtx)
			// 3. 拿到锁后做二次检查，防止在抢锁期间其他请求已经完成建组。
			return createDefaultAlertGroupIfMissing(waitCtx, ws, groupSvc, permMgr, operator, groupName)
		}

		// 4. 未抢到锁时，不直接报错，也不盲目继续创建。
		//    先等待一个短周期，让持锁方完成建组；若上下文超时/取消，则终止等待。
		select {
		case <-waitCtx.Done():
			return nil, errors.Wrap(waitCtx.Err(), "wait default alert user group lock")
		case <-time.After(defaultAlertUserGroupLockRetryInterval):
		}

		// 5. 等待后先复查默认组是否已出现；若仍不存在，则继续下一轮抢锁。
		//    这样可以在“别的并发请求刚创建完成”时尽快复用，避免重复建组。
		group, err = groupSvc.FindByName(waitCtx, ws, groupName, operator)
		if err != nil {
			return nil, errors.Wrap(err, "find default bkmonitor user group by name after lock contention")
		}
		if group != nil {
			return []int64{group.ID}, nil
		}
	}
}

func createDefaultAlertGroupIfMissing(
	ctx context.Context,
	ws *workspace.Workspace,
	groupSvc *Service,
	permMgr defaultAlertRoleManager,
	operator, groupName string,
) ([]int64, error) {
	group, err := groupSvc.FindByName(ctx, ws, groupName, operator)
	if err != nil {
		return nil, errors.Wrap(err, "find default bkmonitor user group by name under lock")
	}
	if group != nil {
		return []int64{group.ID}, nil
	}

	// 仅在默认告警组不存在时，才需要收集空间下 developer / sre 角色成员作为建组来源。
	members, err := listDefaultAlertGroupMembers(ctx, permMgr, ws.ID)
	if err != nil {
		return nil, err
	}
	// 工作空间内没有任何目标角色成员时，无需创建告警组。
	if len(members) == 0 {
		return nil, nil
	}

	// 默认告警组尚不存在，创建新的用户组：仅通过 user 渠道通知，
	// 成员为上述角色成员，并预置告警与动作的分级通知配置。
	detail, err := groupSvc.Save(ctx, ws, &SaveParams{
		Name:         groupName,
		Channels:     []string{"user"},
		Desc:         "BKMS workspace default alert user group",
		AlertNotice:  buildDefaultAlertUserGroupAlertNotices(),
		ActionNotice: buildDefaultAlertUserGroupActionNotices(),
		Users: lo.Map(members, func(member string, _ int) bkmapi.UserGroupUser {
			return bkmapi.UserGroupUser{
				ID:   member,
				Type: "user",
			}
		}),
		Operator: operator,
	})
	if err != nil {
		return nil, errors.Wrap(err, "create default alert user group")
	}
	return []int64{detail.ID}, nil
}

// listDefaultAlertGroupMembers 收集工作空间默认告警组的成员列表（developer 与 sre 两类角色）
func listDefaultAlertGroupMembers(
	ctx context.Context,
	permMgr defaultAlertRoleManager,
	workspaceID string,
) ([]string, error) {
	// 列出工作空间下的全部角色，供后续按 roleCode 检索目标角色。
	roles, err := permMgr.ListRoles(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "list roles of workspace(%s)", workspaceID)
	}

	// 按 roleCode 建立索引，便于定位 developer / sre 角色
	roleByCode := make(map[string]*role.Role, len(roles))
	for _, roleInfo := range roles {
		if roleInfo == nil {
			continue
		}
		roleByCode[roleInfo.RoleCode] = roleInfo
	}

	// 取默认告警组所需的目标角色（developer / sre），缺失则直接报错。
	targetRoles := make([]*role.Role, 0, 2)
	for _, roleCode := range []string{perm.RoleCodeDeveloper, perm.RoleCodeSre} {
		roleInfo := roleByCode[roleCode]
		if roleInfo == nil {
			return nil, errors.Errorf("role(%s) of workspace(%s) not found", roleCode, workspaceID)
		}
		targetRoles = append(targetRoles, roleInfo)
	}

	// 并发查询各目标角色成员，使用集合去重（同一用户可能同时属于两类角色）。
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

	// 将去重后的成员按字典序排序，保证结果稳定可预期。
	result := make([]string, 0, len(memberSet))
	for member := range memberSet {
		result = append(result, member)
	}
	sort.Strings(result)
	return result, nil
}

// 告警阶段默认告警组通知方式
func buildDefaultAlertUserGroupAlertNotices() []bkmapi.AlertNotice {
	return []bkmapi.AlertNotice{{
		TimeRange: "00:00--23:59",
		NotifyConfig: []bkmapi.AlertNoticeConfig{
			{
				Level:      1,
				Type:       []string{},
				NoticeWays: buildDefaultAlertUserGroupUrgentNoticeWays(),
			},
			{
				Level: 2,
				Type:  []string{},
				NoticeWays: []bkmapi.NoticeWay{
					{Name: "weixin", Receivers: []string{}},
				},
			},
			{
				Level: 3,
				Type:  []string{},
				NoticeWays: []bkmapi.NoticeWay{
					{Name: "weixin", Receivers: []string{}},
				},
			},
		},
	}}
}

// 执行阶段默认告警组通知方式
func buildDefaultAlertUserGroupActionNotices() []bkmapi.ActionNotice {
	return []bkmapi.ActionNotice{{
		TimeRange: "00:00--23:59",
		NotifyConfig: []bkmapi.ActionNoticeConfig{
			{
				Phase:      1,
				Type:       []string{},
				NoticeWays: buildDefaultAlertUserGroupUrgentNoticeWays(),
			},
			{
				Phase: 2,
				Type:  []string{},
				NoticeWays: []bkmapi.NoticeWay{
					{Name: "weixin", Receivers: []string{}},
				},
			},
			{
				Phase: 3,
				Type:  []string{},
				NoticeWays: []bkmapi.NoticeWay{
					{Name: "weixin", Receivers: []string{}},
				},
			},
		},
	}}
}

func buildDefaultAlertUserGroupUrgentNoticeWays() []bkmapi.NoticeWay {
	return []bkmapi.NoticeWay{
		{Name: "weixin", Receivers: []string{}},
		{Name: "mail", Receivers: []string{}},
		{Name: "sms", Receivers: []string{}},
		{Name: "voice", Receivers: []string{}},
	}
}
