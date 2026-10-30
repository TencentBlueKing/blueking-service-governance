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

package releasestrategy

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Store 发布策略存储
type Store = store.GenericStore[ReleaseStrategy]

// Meta 发布策略的存储元数据。不存在（含 ID 不合法）时返回 ErrStrategyNotFound，
// 同一空间内名称重复时返回 ErrStrategyNameConflict。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + name，约束同一空间内策略名称唯一
var Meta = store.Meta[ReleaseStrategy]{
	Collection: "release_strategies",
	NotFound:   ErrStrategyNotFound,
	Duplicate:  ErrStrategyNameConflict,
}

// ByWorkspace 匹配空间内的全部发布策略
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// OldestFirst 按创建时间升序
func OldestFirst() store.Opt {
	return store.Asc("createdAt")
}

// byWorkspaceAndID 匹配空间内指定 ID 的发布策略，id 不是合法的 ObjectID 时返回 ErrStrategyNotFound
func byWorkspaceAndID(workspaceID, id string) (store.Opt, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrapf(ErrStrategyNotFound, "invalid release strategy id %q", id)
	}
	return store.Combine(ByWorkspace(workspaceID), store.ByID(objID)), nil
}

// ListStrategies 返回空间可用的全部发布策略：先是平台内置策略（按批次数升序），再是空间自定义策略（按创建时间升序）。
// 内置策略的 WorkspaceID 填为查询所在的空间
func ListStrategies(ctx context.Context, strategies *Store, workspaceID string) ([]ReleaseStrategy, error) {
	custom, err := strategies.List(ctx, ByWorkspace(workspaceID), OldestFirst())
	if err != nil {
		return nil, errors.Wrapf(err, "list release strategies in workspace %s", workspaceID)
	}
	builtins := BuiltinStrategies()
	for i := range builtins {
		builtins[i].WorkspaceID = workspaceID
	}
	return append(builtins, custom...), nil
}

// CreateStrategy 新建空间发布策略，名称与平台内置策略或空间内已有策略重复时返回 ErrStrategyNameConflict
func CreateStrategy(ctx context.Context, strategies *Store, strategy *ReleaseStrategy) error {
	if IsBuiltinName(strategy.Name) {
		return errors.Wrapf(ErrStrategyNameConflict, "name %q is used by a builtin release strategy", strategy.Name)
	}
	err := strategies.Create(ctx, strategy)
	return errors.Wrapf(err, "create release strategy in workspace %s", strategy.WorkspaceID)
}

// GetStrategy 获取空间可用的指定 ID 的发布策略，平台内置策略在所有空间可见，其 WorkspaceID 填为查询所在的空间；
// 不存在（含 ID 不合法）时返回 ErrStrategyNotFound
func GetStrategy(ctx context.Context, strategies *Store, workspaceID, id string) (*ReleaseStrategy, error) {
	if builtin, ok := getBuiltinStrategy(id); ok {
		builtin.WorkspaceID = workspaceID
		return builtin, nil
	}
	opt, err := byWorkspaceAndID(workspaceID, id)
	if err != nil {
		return nil, err
	}
	strategy, err := strategies.Get(ctx, opt)
	if err != nil {
		return nil, errors.Wrapf(err, "get release strategy %s in workspace %s", id, workspaceID)
	}
	return strategy, nil
}

// UpdateStrategy 整体替换发布策略的可编辑内容（名称、描述、批次）及更新人。
// 平台内置策略返回 ErrBuiltinStrategyReadOnly，名称与平台内置策略或空间内已有策略重复时返回 ErrStrategyNameConflict，
// 不存在返回 ErrStrategyNotFound
func UpdateStrategy(ctx context.Context, strategies *Store, strategy *ReleaseStrategy) error {
	if strategy.IsBuiltin() {
		return errors.Wrapf(ErrBuiltinStrategyReadOnly, "update release strategy %s", strategy.ID.Hex())
	}
	if IsBuiltinName(strategy.Name) {
		return errors.Wrapf(ErrStrategyNameConflict, "name %q is used by a builtin release strategy", strategy.Name)
	}
	err := strategies.Update(ctx, strategy,
		store.UpdateSelected("name", "description", "batches", "updater"),
		ByWorkspace(strategy.WorkspaceID),
		store.ByID(strategy.ID),
	)
	return errors.Wrapf(err, "update release strategy %s in workspace %s", strategy.ID.Hex(), strategy.WorkspaceID)
}

// DeleteStrategy 删除并返回空间内指定 ID 的发布策略，平台内置策略返回 ErrBuiltinStrategyReadOnly，
// 不存在（含 ID 不合法）时返回 ErrStrategyNotFound
func DeleteStrategy(ctx context.Context, strategies *Store, workspaceID, id string) (*ReleaseStrategy, error) {
	if _, ok := getBuiltinStrategy(id); ok {
		return nil, errors.Wrapf(ErrBuiltinStrategyReadOnly, "delete release strategy %s", id)
	}
	opt, err := byWorkspaceAndID(workspaceID, id)
	if err != nil {
		return nil, err
	}
	strategy, err := strategies.GetAndDelete(ctx, opt)
	if err != nil {
		return nil, errors.Wrapf(err, "delete release strategy %s in workspace %s", id, workspaceID)
	}
	return strategy, nil
}
