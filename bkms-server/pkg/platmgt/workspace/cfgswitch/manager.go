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

// Package cfgswitch provides platform administrators with workspace cfg switch management,
// used by the card page in platform workspace detail.
package cfgswitch

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Manager 平台管理员视角下的空间配置开关查看与设置
type Manager struct {
	db         *mongo.Database
	workspaces *bkmsworkspace.WorkspaceStore
	switches   *model.WorkspaceCfgSwitchStore
}

// NewManager 创建 Manager
func NewManager(db *mongo.Database) *Manager {
	return &Manager{
		db:         db,
		workspaces: store.Of(db, bkmsworkspace.Meta),
		switches:   store.Of(db, model.WorkspaceCfgSwitchMeta),
	}
}

// scopeToWorkspace 跨租户查出目标空间，返回以该空间所属租户为当前租户的 ctx，
// 后续配置开关的读写都落在这个租户下
func (m *Manager) scopeToWorkspace(
	ctx context.Context, workspaceID string,
) (context.Context, *bkmsworkspace.Workspace, error) {
	ws, err := m.workspaces.Get(ctx, bkmsworkspace.ByWorkspaceID(workspaceID), store.TenantAll())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get workspace %s", workspaceID)
	}
	return store.ScopeToTenant(ctx, ws.TenantID), ws, nil
}

// List 按定义顺序返回空间的全部配置开关，从未设置过的开关为关闭状态
//
// 涉及的模型：workspaces、workspace_cfg_switches
func (m *Manager) List(ctx context.Context, workspaceID string) ([]model.WorkspaceCfgSwitch, error) {
	ctx, ws, err := m.scopeToWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return model.ListOfWorkspace(ctx, m.db, ws.ID)
}

// Update 设置空间的一项配置开关，取值没有变化时不写库。开关标识未定义时返回 ErrUnknownSwitch
//
// 涉及的模型：workspaces、workspace_cfg_switches
func (m *Manager) Update(
	ctx context.Context, workspaceID string, key model.SwitchKey, enabled bool,
) (*model.WorkspaceCfgSwitch, error) {
	if _, ok := model.GetSwitchDef(key); !ok {
		return nil, errors.Wrapf(model.ErrUnknownSwitch, "key %q", key)
	}
	ctx, ws, err := m.scopeToWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	current, err := m.get(ctx, ws.ID, key)
	if err != nil {
		return nil, err
	}
	if current.Enabled == enabled {
		return current, nil
	}

	target := &model.WorkspaceCfgSwitch{
		WorkspaceID: ws.ID,
		Key:         key,
		Enabled:     enabled,
		Updater:     auth.MustGetUser(ctx).ID,
	}
	if _, err = m.switches.Upsert(
		ctx, target, model.ByWorkspaceAndKey(ws.ID, key), store.UpdateSelected("enabled", "updater"),
	); err != nil {
		return nil, errors.Wrapf(err, "save cfg switch %s of workspace %s", key, ws.ID)
	}
	return m.get(ctx, ws.ID, key)
}

// get 读取空间的一项配置开关，没有记录时返回关闭状态
func (m *Manager) get(
	ctx context.Context, workspaceID string, key model.SwitchKey,
) (*model.WorkspaceCfgSwitch, error) {
	s, err := m.switches.Get(ctx, model.ByWorkspaceAndKey(workspaceID, key))
	if errors.Is(err, model.ErrCfgSwitchNotFound) {
		return &model.WorkspaceCfgSwitch{WorkspaceID: workspaceID, Key: key}, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get cfg switch %s of workspace %s", key, workspaceID)
	}
	return s, nil
}
