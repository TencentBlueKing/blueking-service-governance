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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"context"
	"sort"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
)

// ErrNoticeGroupNotFound 告警组不存在的语义错误，调用方可据此跳过。
var ErrNoticeGroupNotFound = errors.New("user group not found")

// syncRoleCodes 以下角色的成员会被同步到蓝鲸监控的 APM 告警组中
var syncRoleCodes = []string{
	perm.RoleCodeAdmin,
	perm.RoleCodeSre,
}

const (
	// userGroupNameTmpl 告警组名称模板：envName 会被填充到 %s
	userGroupNameTmpl = "【APM】%s 告警组"

	// userGroupUserTypeUser 通知接收人员类型：用户
	userGroupUserTypeUser = "user"

	// saveUserGroupDefaultTimezone 保存告警组时的默认时区
	// （bkmonitor 写接口要求 timezone 非空）
	saveUserGroupDefaultTimezone = "Asia/Shanghai"

	// apmSyncRetryInterval 重试间隔
	apmSyncRetryInterval = 2 * time.Second

	// apmSyncRetryMaxElapsed 重试最大总耗时（含首次），超时后停止重试
	apmSyncRetryMaxElapsed = 10 * time.Second
)

// userGroupClientFactory 创建底层 bkmonitor 客户端的工厂函数，便于测试 mock
type userGroupClientFactory func(operator string) (bkmapi.Client, error)

// UserGroupService 封装 bkmonitor 告警组业务逻辑
type UserGroupService struct {
	// newClient 创建底层 bkmonitor 客户端的工厂函数
	newClient userGroupClientFactory

	// permMgr 用于查询 workspace 下 admin/sre 角色成员
	permMgr perm.Manager

	// envStore 用于查询 workspace 下全部环境列表
	envStore envmodel.EnvironmentStore
}

// NewUserGroupService 创建 UserGroupService 实例。
func NewUserGroupService(
	permMgr perm.Manager,
	envStore envmodel.EnvironmentStore,
) *UserGroupService {
	return &UserGroupService{
		newClient: bkmapi.New,
		permMgr:   permMgr,
		envStore:  envStore,
	}
}

// GetByEnv 按 bkMonitorProjectID 与 envName 定位告警组并拉取详情。
// 告警组不存在时返回 ErrNoticeGroupNotFound。
func (s *UserGroupService) GetByEnv(
	ctx context.Context, bkMonitorProjectID int64, envName, operator string,
) (*bkmapi.UserGroupDetail, error) {
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}

	groupName := BuildUserGroupName(envName)
	groups, err := client.SearchUserGroups(ctx, &bkmapi.SearchUserGroupsReq{
		BkBizIDs: []int64{bkMonitorProjectID},
		Name:     groupName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "search user groups by name(%s)", groupName)
	}

	// bkm 是模糊匹配
	var targetID int64
	for _, g := range groups {
		if g == nil {
			continue
		}
		if g.Name == groupName && g.BkBizID == bkMonitorProjectID {
			targetID = g.ID
			break
		}
	}
	if targetID == 0 {
		return nil, errors.Wrapf(ErrNoticeGroupNotFound, "bk_biz_id=%d, name=%s", bkMonitorProjectID, groupName)
	}

	detail, err := client.SearchUserGroupDetail(ctx, &bkmapi.SearchUserGroupDetailReq{ID: targetID})
	if err != nil {
		return nil, errors.Wrapf(err, "search user group detail, bk_biz_id=%d, id=%d", bkMonitorProjectID, targetID)
	}
	return detail, nil
}

// EnsureMembersForEnvs 以 expectedMembers 为目标，对每个环境的告警组做"查缺补漏"式增量同步：
func (s *UserGroupService) EnsureMembersForEnvs(
	ctx context.Context,
	bkMonitorProjectID int64, envNames, expectedMembers []string, operator string,
) {
	if len(envNames) == 0 || len(expectedMembers) == 0 {
		log.Warnf(
			ctx, "skip ensure members: envNames=%d, expectedMembers=%d",
			len(envNames), len(expectedMembers),
		)
		return
	}

	log.Debugf(
		ctx, "start ensuring bkm user group members, bk_biz_id=%d, envCount=%d, expected=%v",
		bkMonitorProjectID, len(envNames), expectedMembers,
	)

	for _, envName := range envNames {
		if err := s.ensureMembersForEnvOnce(
			ctx, bkMonitorProjectID, expectedMembers, envName, operator,
		); err != nil {
			if errors.Is(err, ErrNoticeGroupNotFound) {
				log.Warnf(ctx, "user group not found, skip ensure, bk_biz_id=%d, env=%s", bkMonitorProjectID, envName)
				continue
			}
			log.Errorf(
				ctx, "ensure members to user group failed, bk_biz_id=%d, env=%s, err=%v",
				bkMonitorProjectID, envName, err,
			)
			continue
		}
	}
}

// ensureMembersForEnvOnce 针对单个环境做一次"查缺补漏"式同步。
// 与 EnsureMembersForEnvs 不同：错误（含 ErrNoticeGroupNotFound）原样返回，供调用方判定是否重试。
func (s *UserGroupService) ensureMembersForEnvOnce(
	ctx context.Context,
	bkMonitorProjectID int64,
	expectedMembers []string,
	envName, operator string,
) error {
	detail, err := s.GetByEnv(ctx, bkMonitorProjectID, envName, operator)
	if err != nil {
		return err
	}

	if sErr := s.saveMembers(ctx, detail, expectedMembers, nil, operator); sErr != nil {
		return errors.Wrapf(sErr,
			"save members to user group(id=%d, name=%s)", detail.ID, detail.Name,
		)
	}
	log.Debugf(ctx, "ensure members to user group success, env=%s, user_group_id=%d", envName, detail.ID)
	return nil
}

// RemoveMemberForEnvs 将 userID 从每个环境告警组的 type=user 成员中移除。
func (s *UserGroupService) RemoveMemberForEnvs(
	ctx context.Context,
	bkMonitorProjectID int64,
	envNames []string,
	userID, operator string,
) {
	if userID == "" || len(envNames) == 0 {
		return
	}

	log.Debugf(
		ctx, "start removing member from bkm user groups, bk_biz_id=%d, envCount=%d, userID=%s",
		bkMonitorProjectID, len(envNames), userID,
	)

	toRemove := []string{userID}
	for _, envName := range envNames {
		detail, err := s.GetByEnv(ctx, bkMonitorProjectID, envName, operator)
		if err != nil {
			if errors.Is(err, ErrNoticeGroupNotFound) {
				log.Warnf(
					ctx, "user group not found, skip remove, bk_biz_id=%d, env=%s", bkMonitorProjectID, envName,
				)
				continue
			}
			log.Errorf(
				ctx, "get user group detail failed, bk_biz_id=%d, env=%s, err=%v", bkMonitorProjectID, envName, err,
			)
			continue
		}

		if sErr := s.saveMembers(ctx, detail, nil, toRemove, operator); sErr != nil {
			log.Errorf(
				ctx, "remove member from user group failed, env=%s, user_group_id=%d, userID=%s, err=%v",
				envName, detail.ID, userID, sErr,
			)
			continue
		}
		log.Debugf(
			ctx, "remove member from user group success, env=%s, user_group_id=%d, userID=%s",
			envName, detail.ID, userID,
		)
	}
}

// saveMembers 按增量对比规则更新 DutyArranges[0].Users 并回写。
// 仅增删 Type==user 成员，Type==group 一律保留；无变更时跳过 SaveUserGroup。
func (s *UserGroupService) saveMembers(
	ctx context.Context,
	detail *bkmapi.UserGroupDetail,
	toAdd, toRemove []string,
	operator string,
) error {
	if detail == nil {
		return errors.New("detail is nil")
	}

	if len(detail.DutyArranges) == 0 {
		log.Warnf(ctx, "user group(id=%d, name=%s) has no duty arranges, skip save members", detail.ID, detail.Name)
		return nil
	}

	newUsers, changed := ApplyMemberDiff(detail.DutyArranges[0].Users, toAdd, toRemove)
	if !changed {
		log.Warnf(
			ctx, "user group(id=%d, name=%s) members unchanged after diff, skip save, users_count=%d",
			detail.ID, detail.Name, len(newUsers),
		)
		return nil
	}

	client, err := s.newClient(operator)
	if err != nil {
		return errors.Wrap(err, "new bkmonitor client")
	}

	req := buildSaveUserGroupReq(detail, newUsers, operator)
	if _, err = client.SaveUserGroup(ctx, req); err != nil {
		return errors.Wrapf(err, "save user group, id=%d, name=%s", detail.ID, detail.Name)
	}

	log.Debugf(ctx, "user group(id=%d, name=%s) members synced, users_count=%d", detail.ID, detail.Name, len(newUsers))
	return nil
}

// listPermMgrMembers 聚合 workspace 下 admin、sre 成员，按 ID 去重并过滤空字符串。
func (s *UserGroupService) listPermMgrMembers(
	ctx context.Context,
	workspaceID string,
) ([]string, error) {
	if s.permMgr == nil {
		return nil, errors.New("permMgr is nil")
	}
	memberSet := make(map[string]bool)

	for _, roleCode := range syncRoleCodes {
		role, err := s.permMgr.GetRole(ctx, workspaceID, roleCode)
		if err != nil {
			return nil, errors.Wrapf(err, "get role(%s) of workspace(%s)", roleCode, workspaceID)
		}
		members, err := s.permMgr.ListRoleMembers(ctx, role.ID)
		if err != nil {
			return nil, errors.Wrapf(err,
				"list role(%s) members of workspace(%s)", roleCode, workspaceID,
			)
		}
		for _, m := range members {
			if m == "" {
				continue
			}
			memberSet[m] = true
		}
	}

	result := make([]string, 0, len(memberSet))
	for m := range memberSet {
		result = append(result, m)
	}
	sort.Strings(result)

	return result, nil
}

// listEnvNames 查询 workspace 下所有环境名。
func (s *UserGroupService) listEnvNames(
	ctx context.Context,
	workspaceID string,
) ([]string, error) {
	if s.envStore == nil {
		return nil, errors.New("envStore is nil")
	}

	envs, err := s.envStore.ListStdEnvs(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "list envs of workspace(%s)", workspaceID)
	}
	names := make([]string, 0, len(envs))
	for _, e := range envs {
		names = append(names, e.Name)
	}
	return names, nil
}

// SyncMembersForWorkspace 以 admin、sre 全量成员为目标，对 workspace 下所有环境的告警组，做增量同步
func (s *UserGroupService) SyncMembersForWorkspace(
	ctx context.Context,
	ws *workspace.Workspace,
	operator string,
) {
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		log.Errorf(ctx, "resolve bkMonitorProjectID failed, skip sync, ws=%s, err=%v", ws.ID, err)
		return
	}

	expectedMembers, err := s.listPermMgrMembers(ctx, ws.ID)
	if err != nil {
		log.Errorf(ctx, "list admin/sre members failed, skip sync, ws=%s, err=%v", ws.ID, err)
		return
	}
	if len(expectedMembers) == 0 {
		log.Warnf(ctx, "no admin/sre members, skip sync, ws=%s, bk_biz_id=%d", ws.ID, bkMonitorProjectID)
		return
	}

	envNames, err := s.listEnvNames(ctx, ws.ID)
	if err != nil {
		log.Errorf(ctx, "list env names failed, skip sync, ws=%s, err=%v", ws.ID, err)
		return
	}
	if len(envNames) == 0 {
		log.Warnf(
			ctx, "no envs under workspace, skip sync, ws=%s, bk_biz_id=%d, memberCount=%d",
			ws.ID, bkMonitorProjectID, len(expectedMembers),
		)
		return
	}

	log.Debugf(
		ctx, "sync admin/sre to user groups, ws=%s, bk_biz_id=%d, envCount=%d, memberCount=%d",
		ws.ID, bkMonitorProjectID, len(envNames), len(expectedMembers),
	)
	s.EnsureMembersForEnvs(ctx, bkMonitorProjectID, envNames, expectedMembers, operator)
}

// SyncMembersForEnvWithRetry 在 APM 创建后，对单个环境告警组同步成员
func (s *UserGroupService) SyncMembersForEnvWithRetry(
	ctx context.Context,
	ws *workspace.Workspace,
	envName, operator string,
) {
	if envName == "" {
		log.Warnf(ctx, "envName is empty, skip sync admin/sre for env(with retry)")
		return
	}

	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		log.Errorf(ctx, "resolve bkMonitorProjectID failed, skip sync, ws=%s, err=%v", ws.ID, err)
		return
	}

	expectedMembers, err := s.listPermMgrMembers(ctx, ws.ID)
	if err != nil {
		log.Errorf(
			ctx, "list admin/sre members failed, skip sync(with retry), ws=%s, env=%s, err=%v", ws.ID, envName, err,
		)
		return
	}
	if len(expectedMembers) == 0 {
		log.Warnf(
			ctx, "no admin/sre members, skip sync(with retry), ws=%s, env=%s, bk_biz_id=%d",
			ws.ID, envName, bkMonitorProjectID,
		)
		return
	}

	log.Debugf(
		ctx, "sync admin/sre to user group with retry, ws=%s, env=%s, bk_biz_id=%d, memberCount=%d",
		ws.ID, envName, bkMonitorProjectID, len(expectedMembers),
	)

	// 使用指数退避重试，仅 NotFound 时重试，其他错误立即终止
	// 重试间隔 2s，最大重试时间 10s
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = apmSyncRetryInterval
	bo.MaxElapsedTime = apmSyncRetryMaxElapsed

	retryErr := backoff.Retry(func() error {
		syncErr := s.ensureMembersForEnvOnce(
			ctx, bkMonitorProjectID, expectedMembers, envName, operator,
		)
		if syncErr == nil {
			return nil
		}
		// 非 NotFound 错误 → Permanent，立即终止重试
		if !errors.Is(syncErr, ErrNoticeGroupNotFound) {
			return backoff.Permanent(syncErr)
		}
		log.Debugf(
			ctx, "user group not found, will retry, ws=%s, env=%s, bk_biz_id=%d", ws.ID, envName, bkMonitorProjectID,
		)
		return syncErr
	}, bo)

	if retryErr == nil {
		log.Debugf(
			ctx, "sync admin/sre to user group success, ws=%s, env=%s, bk_biz_id=%d, memberCount=%d",
			ws.ID, envName, bkMonitorProjectID, len(expectedMembers),
		)
		return
	}

	if errors.Is(retryErr, ErrNoticeGroupNotFound) {
		log.Warnf(
			ctx,
			"APM Alarm Group Member Synchronization Failed After Creation, ws=%s, env=%s, bk_biz_id=%d, err=%v",
			ws.ID, envName, bkMonitorProjectID, retryErr,
		)
	} else {
		log.Errorf(
			ctx,
			"sync admin/sre to user group failed(non-NotFound), stop retry, "+
				"ws=%s, env=%s, bk_biz_id=%d, err=%v",
			ws.ID, envName, bkMonitorProjectID, retryErr,
		)
	}
}

// RemoveMemberForWorkspace 将 userID 从该 workspace 下，所有环境的告警组 成员 中移除。
func (s *UserGroupService) RemoveMemberForWorkspace(
	ctx context.Context,
	ws *workspace.Workspace,
	userID, operator string,
) {
	if userID == "" {
		log.Warnf(ctx, "userID is empty, skip remove member for workspace")
		return
	}

	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		log.Errorf(ctx, "resolve bkMonitorProjectID failed, skip sync, ws=%s, err=%v", ws.ID, err)
		return
	}

	envNames, err := s.listEnvNames(ctx, ws.ID)
	if err != nil {
		log.Errorf(ctx, "list env names failed, skip remove, ws=%s, userID=%s, err=%v", ws.ID, userID, err)
		return
	}
	if len(envNames) == 0 {
		log.Warnf(
			ctx, "no envs under workspace, skip remove, ws=%s, bk_biz_id=%d, userID=%s",
			ws.ID, bkMonitorProjectID, userID,
		)
		return
	}

	log.Debugf(
		ctx, "remove member from user groups, ws=%s, bk_biz_id=%d, envCount=%d, userID=%s",
		ws.ID, bkMonitorProjectID, len(envNames), userID,
	)
	s.RemoveMemberForEnvs(ctx, bkMonitorProjectID, envNames, userID, operator)
}
