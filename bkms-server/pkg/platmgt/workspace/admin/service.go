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

package admin

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
)

const (
	// defaultGrantDuration 是临时管理员授权的默认有效时长。
	defaultGrantDuration = 2 * time.Hour
)

var (
	// ErrWorkspaceAdminAlreadyExists 表示当前用户已经拥有空间管理员权限。
	ErrWorkspaceAdminAlreadyExists = errors.New("workspace admin already exists")
	// ErrWorkspaceAdminNotFound 表示当前用户并不拥有空间管理员权限。
	ErrWorkspaceAdminNotFound = errors.New("workspace admin not found")
	// ErrTempAdminAlreadyExists 表示已存在未回收的临时管理员授权。
	ErrTempAdminAlreadyExists = errors.New("temporary admin already exists")
)

// Service 提供平台管理页下统一的 workspace admin 能力。
type Service struct {
	workspaceStore bkmsworkspace.WorkspaceStore
	recordStore    Store
	permMgr        perm.Manager
	grantDuration  time.Duration
}

// NewService 创建 workspace admin 服务。
func NewService(workspaceStore bkmsworkspace.WorkspaceStore, recordStore Store, permMgr perm.Manager) *Service {
	return &Service{
		workspaceStore: workspaceStore,
		recordStore:    recordStore,
		permMgr:        permMgr,
		grantDuration:  defaultGrantDuration,
	}
}

// GetRoleStatus 返回指定用户在目标空间是否拥有指定角色。
func (s *Service) GetRoleStatus(
	ctx context.Context,
	workspaceID, roleCode, username string,
) (*RoleStatus, error) {
	if _, err := s.workspaceStore.Get(ctx, workspaceID); err != nil {
		return nil, errors.Wrap(err, "get workspace")
	}
	_, hasRole, err := s.loadRoleState(ctx, workspaceID, roleCode, username)
	if err != nil {
		return nil, err
	}
	return &RoleStatus{HasRole: hasRole}, nil
}

// GrantAdmin 为当前用户授予 workspace admin。
func (s *Service) GrantAdmin(
	ctx context.Context,
	workspaceID, username string,
	isTemporary bool,
) (*RoleStatus, error) {
	if _, err := s.workspaceStore.Get(ctx, workspaceID); err != nil {
		return nil, errors.Wrap(err, "get workspace")
	}

	roleID, hasAdminRole, record, err := s.loadAdminRoleAndRecord(ctx, workspaceID, username)
	if err != nil {
		return nil, err
	}
	if hasAdminRole {
		return nil, ErrWorkspaceAdminAlreadyExists
	}
	if isTemporary && record != nil {
		log.Errorf(
			ctx,
			"workspace temp admin record exists without remote admin role: workspaceID=%s username=%s expiresAt=%s",
			workspaceID,
			username,
			record.ExpiresAt.Format(time.RFC3339),
		)
		return nil, errors.Wrap(ErrTempAdminAlreadyExists, "temporary admin record already exists")
	}

	if err := s.permMgr.AddRoleForUsers(ctx, roleID, []string{username}); err != nil {
		return nil, err
	}
	if !isTemporary {
		return &RoleStatus{HasRole: true}, nil
	}

	// 创建临时管理员记录
	now := time.Now()
	record = &WorkspaceTempAdmin{
		WorkspaceID: workspaceID,
		Username:    username,
		ExpiresAt:   now.Add(s.grantDuration),
		IsRecycled:  false,
		Creator:     username,
		CreatedAt:   now,
		UpdatedAt:   now,
		Updater:     username,
	}
	if err := s.recordStore.Create(ctx, record); err != nil {
		if errors.Is(err, ErrRecordAlreadyExists) {
			return nil, ErrTempAdminAlreadyExists
		}
		return nil, errors.Wrap(err, "create temporary admin record")
	}

	return &RoleStatus{HasRole: true}, nil
}

// RevokeAdmin 统一回收目标空间管理员权限。
func (s *Service) RevokeAdmin(ctx context.Context, workspaceID, username string) (*RoleStatus, error) {
	if _, err := s.workspaceStore.Get(ctx, workspaceID); err != nil {
		return nil, errors.Wrap(err, "get workspace")
	}

	roleID, hasAdminRole, record, err := s.loadAdminRoleAndRecord(ctx, workspaceID, username)
	if err != nil {
		return nil, err
	}
	if !hasAdminRole {
		return nil, ErrWorkspaceAdminNotFound
	}

	if err := s.permMgr.DeleteRoleForUsers(ctx, roleID, []string{username}); err != nil {
		return nil, err
	}

	// 变更临时管理员记录
	if record != nil {
		record.IsRecycled = true
		record.Updater = username
		if err := s.recordStore.Update(ctx, record); err != nil {
			return nil, err
		}
	}

	return &RoleStatus{HasRole: false}, nil
}

// loadAdminRoleAndRecord 加载当前用户在目标空间的管理员角色与活跃临时授权记录。
//
// 返回值依次为：
//   - roleID：目标空间管理员角色 ID
//   - hasAdminRole：当前用户是否已拥有管理员角色
//   - record：当前未回收的临时管理员授权记录；不存在时为 nil
func (s *Service) loadAdminRoleAndRecord(
	ctx context.Context,
	workspaceID, username string,
) (string, bool, *WorkspaceTempAdmin, error) {
	roleID, hasAdminRole, err := s.loadRoleState(ctx, workspaceID, perm.RoleCodeAdmin, username)
	if err != nil {
		return "", false, nil, err
	}
	record, err := s.recordStore.GetLatestActiveGrant(ctx, workspaceID, username)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return roleID, hasAdminRole, nil, nil
		}
		return "", false, nil, err
	}
	return roleID, hasAdminRole, record, nil
}

// loadRoleState 加载目标角色的角色 ID，并判断指定用户是否已属于该角色。
func (s *Service) loadRoleState(
	ctx context.Context,
	workspaceID, roleCode, username string,
) (string, bool, error) {
	role, err := s.permMgr.GetRole(ctx, workspaceID, roleCode)
	if err != nil {
		return "", false, err
	}
	members, err := s.permMgr.ListRoleMembers(ctx, role.ID)
	if err != nil {
		return "", false, err
	}
	hasRole := lo.Contains(members, username)
	return role.ID, hasRole, nil
}
