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

// GetStrategy 获取空间内指定 ID 的发布策略，不存在（含 ID 不合法）时返回 ErrStrategyNotFound
func GetStrategy(ctx context.Context, strategies *Store, workspaceID, id string) (*ReleaseStrategy, error) {
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

// UpdateStrategy 整体替换发布策略的可编辑内容（名称、描述、批次）及更新人，
// 名称冲突返回 ErrStrategyNameConflict，不存在返回 ErrStrategyNotFound
func UpdateStrategy(ctx context.Context, strategies *Store, strategy *ReleaseStrategy) error {
	err := strategies.Update(ctx, strategy,
		store.UpdateSelected("name", "description", "batches", "updater"),
		ByWorkspace(strategy.WorkspaceID),
		store.ByID(strategy.ID),
	)
	return errors.Wrapf(err, "update release strategy %s in workspace %s", strategy.ID.Hex(), strategy.WorkspaceID)
}

// DeleteStrategy 删除并返回空间内指定 ID 的发布策略，不存在（含 ID 不合法）时返回 ErrStrategyNotFound
func DeleteStrategy(ctx context.Context, strategies *Store, workspaceID, id string) (*ReleaseStrategy, error) {
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
