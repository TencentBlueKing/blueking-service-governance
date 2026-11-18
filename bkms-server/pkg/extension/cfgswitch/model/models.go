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

// Package model 定义空间级配置开关（cfgswitch）的数据模型。
//
// 配置开关决定某个空间是否开放 BSCP 配置管理、是否展示七彩石，由平台管理员按空间逐项设置；
// 前端据此决定是否展示对应 tab，后端据此拦截未开放的接口与部署注入。
package model

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// SwitchKey 配置开关的标识
type SwitchKey string

const (
	// SwitchBscp 配置管理（BSCP）
	SwitchBscp SwitchKey = "bscp"
	// SwitchRainbowTConf 七彩石（Rainbow TConf）。关闭时应用的七彩石页面绑定接口不可用，见 extension/rainbowtconf
	SwitchRainbowTConf SwitchKey = "rainbowTConf"
)

// SwitchDef 配置开关定义
type SwitchDef struct {
	// Key 开关标识
	Key SwitchKey
	// Name 展示名称
	Name string
}

// SwitchDefs 全部配置开关定义，顺序即平台管理中的展示顺序
var SwitchDefs = []SwitchDef{
	{Key: SwitchBscp, Name: "配置管理（BSCP）"},
	{Key: SwitchRainbowTConf, Name: "七彩石"},
}

// GetSwitchDef 按标识查找配置开关定义
func GetSwitchDef(key SwitchKey) (SwitchDef, bool) {
	return lo.Find(SwitchDefs, func(d SwitchDef) bool { return d.Key == key })
}

var (
	// ErrCfgSwitchNotFound 空间的配置开关没有记录。读取请使用 ListOfWorkspace / IsEnabled，它们会把缺失视为关闭
	ErrCfgSwitchNotFound = errors.New("workspace cfg switch not found")
	// ErrCfgSwitchAlreadyExists 同一空间的同一配置开关已有记录
	ErrCfgSwitchAlreadyExists = errors.New("workspace cfg switch already exists")
	// ErrUnknownSwitch 开关标识不在 SwitchDefs 中
	ErrUnknownSwitch = errors.New("unknown cfg switch")
)

// WorkspaceCfgSwitch 空间的一项配置开关。
//
// 一个空间的每个开关至多一条记录，没有记录时视为关闭，因此新建空间无需初始化
type WorkspaceCfgSwitch struct {
	// WorkspaceID 所属工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// Key 开关标识
	Key SwitchKey `bson:"key"`
	// Enabled 是否开启
	Enabled bool `bson:"enabled"`
	// Updater 最近修改人
	Updater string `bson:"updater"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// WorkspaceCfgSwitchStore 空间配置开关存储
type WorkspaceCfgSwitchStore = store.GenericStore[WorkspaceCfgSwitch]

// WorkspaceCfgSwitchMeta 空间配置开关的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + key
var WorkspaceCfgSwitchMeta = store.Meta[WorkspaceCfgSwitch]{
	Collection:   "workspace_cfg_switches",
	TenantScoped: true,
	NotFound:     ErrCfgSwitchNotFound,
	Duplicate:    ErrCfgSwitchAlreadyExists,
	BeforeSave:   []func(*WorkspaceCfgSwitch) error{validateKey},
}

func validateKey(s *WorkspaceCfgSwitch) error {
	if _, ok := GetSwitchDef(s.Key); !ok {
		return errors.Wrapf(ErrUnknownSwitch, "key %q", s.Key)
	}
	return nil
}

// ByWorkspace 按工作空间匹配
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByWorkspaceAndKey 按工作空间与开关标识匹配
func ByWorkspaceAndKey(workspaceID string, key SwitchKey) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.Where("key", key))
}

// ListOfWorkspace 按 SwitchDefs 的顺序返回空间的全部配置开关，没有记录的开关取关闭状态（UpdatedAt 为零值）。
// scope 选择租户范围，不传时使用 ctx 中的租户
func ListOfWorkspace(
	ctx context.Context, db *mongo.Database, workspaceID string, scope ...store.TenantOpt,
) ([]WorkspaceCfgSwitch, error) {
	opts := append(store.ScopeOpts(scope), ByWorkspace(workspaceID))
	saved, err := store.Of(db, WorkspaceCfgSwitchMeta).List(ctx, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "list cfg switches of workspace %s", workspaceID)
	}
	byKey := lo.KeyBy(saved, func(s WorkspaceCfgSwitch) SwitchKey { return s.Key })

	return lo.Map(SwitchDefs, func(d SwitchDef, _ int) WorkspaceCfgSwitch {
		if s, ok := byKey[d.Key]; ok {
			return s
		}
		return WorkspaceCfgSwitch{WorkspaceID: workspaceID, Key: d.Key}
	}), nil
}

// IsEnabled 空间是否开启了指定配置开关，没有记录时视为关闭
func IsEnabled(ctx context.Context, db *mongo.Database, workspaceID string, key SwitchKey) (bool, error) {
	s, err := store.Of(db, WorkspaceCfgSwitchMeta).Get(ctx, ByWorkspaceAndKey(workspaceID, key))
	if errors.Is(err, ErrCfgSwitchNotFound) {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrapf(err, "get cfg switch %s of workspace %s", key, workspaceID)
	}
	return s.Enabled, nil
}
