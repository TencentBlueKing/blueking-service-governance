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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrWorkspaceNotFound 工作空间未找到时, 返回固定错误
var ErrWorkspaceNotFound = errors.New("workspace not found")

// WorkspaceStore 工作空间存储
type WorkspaceStore = store.GenericStore[Workspace]

// Meta 工作空间的存储元数据。模型没有 _id 字段，工作空间 ID 存在 id 字段中，按 ID 查询须用 ByWorkspaceID。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id
var Meta = store.Meta[Workspace]{
	Collection: "workspaces",
	NotFound:   ErrWorkspaceNotFound,
}

// ByWorkspaceID 按工作空间 ID 匹配
func ByWorkspaceID(id string) store.Opt {
	return store.Where("id", id)
}

// InState 按工作空间状态匹配
func InState(state State) store.Opt {
	return store.Where("state", state)
}

// KeywordMatches 模糊搜索，匹配 id、displayName，忽视大小写；关键字为空时不过滤
func KeywordMatches(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "id", "displayName")
}

// DefaultOrder 默认工作空间排序：按更新时间降序，相同则按 ID 升序（保证分页稳定）
func DefaultOrder() store.Opt {
	return store.Combine(store.Desc("updatedAt"), store.Asc("id"))
}

// Update 更新已经存在的工作空间的可变字段，不存在时返回 ErrWorkspaceNotFound
func Update(ctx context.Context, workspaces *WorkspaceStore, ws *Workspace) error {
	mutable := store.UpdateSelected("displayName", "description", "bkSystems", "imageRegistryType", "state", "updater")
	if err := workspaces.Update(ctx, ws, mutable, ByWorkspaceID(ws.ID)); err != nil {
		return errors.Wrapf(err, "update workspace %s", ws.ID)
	}
	return nil
}

// Delete 依次执行已注册的删除前钩子并删除工作空间，工作空间不存在时不报错
func Delete(ctx context.Context, workspaces *WorkspaceStore, id string) error {
	if err := runPreDeleteHooks(ctx, id); err != nil {
		return errors.Wrapf(err, "before deleting workspace %s", id)
	}
	if err := workspaces.DeleteIfExists(ctx, ByWorkspaceID(id)); err != nil {
		return errors.Wrapf(err, "delete workspace %s", id)
	}
	return nil
}

type stateCount struct {
	State State `bson:"_id"`
	Count int64 `bson:"count"`
}

// CountByState 按生命周期状态统计满足 opts 条件的工作空间数量。结果固定包含已定义的三个状态（可为 0），
// 其它状态值忽略。必须用一次聚合完成：分状态多次计数时，期间切换状态的工作空间会被重复计入或漏计
func CountByState(ctx context.Context, workspaces *WorkspaceStore, opts ...store.Opt) (map[State]int64, error) {
	rows, err := store.Aggregate[stateCount](ctx, workspaces, mongo.Pipeline{
		{{Key: "$group", Value: bson.M{"_id": "$state", "count": bson.M{"$sum": 1}}}},
	}, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "count workspaces by state")
	}
	counts := map[State]int64{StateReady: 0, StateProcessing: 0, StateDisabled: 0}
	for _, r := range rows {
		if _, ok := counts[r.State]; ok {
			counts[r.State] = r.Count
		}
	}
	return counts, nil
}
