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
	"strings"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrWorkspaceNotFound 工作空间未找到时, 返回固定错误
var ErrWorkspaceNotFound = errors.New("workspace not found")

// ErrBCSProjectAlreadyBound BCS 项目已被其他工作空间绑定
var ErrBCSProjectAlreadyBound = errors.New("bcs project already bound")

// ErrBCSProjectAlreadyExists 自动创建使用的确定性 BCS 项目已存在，且不能作为本次重试复用
var ErrBCSProjectAlreadyExists = errors.New("bcs project already exists")

// ErrIndependentWorkspaceBizRequired 独立 BCS 模式下新建容器项目必须提供业务 ID
var ErrIndependentWorkspaceBizRequired = errors.New("bizID is required when creating an independent bcs project")

// ErrIndependentWorkspaceBizInvalid 独立 BCS 模式下业务不存在或当前用户无权访问
var ErrIndependentWorkspaceBizInvalid = errors.New("independent workspace biz is invalid")

// bcsProjectCodeIndex BCS 项目 code 唯一索引名，用于从唯一键冲突中区分出 BCS 项目重复绑定
const bcsProjectCodeIndex = "bkSystems.bkBcsProjectCode_1"

// WorkspaceStore 工作空间存储
type WorkspaceStore = store.GenericStore[Workspace]

// Meta 工作空间的存储元数据。模型没有 _id 字段，工作空间 ID 存在 id 字段中，按 ID 查询须用 ByWorkspaceID。
//
// 集合按租户划分：租户默认取自 ctx，平台管理与运维命令用 TenantAll / TenantOf 改变范围。
// id 目前仍是全局唯一，即不同租户不能使用同一个工作空间 ID。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id
// - 唯一（partial）：bkSystems.bkBcsProjectCode。存的是 BCS 返回的 projectCode，本身全局唯一（多租户下为 {tenantID}-{code}），
// 索引不需要带 tenantID
// - 普通：tenantID
var Meta = store.Meta[Workspace]{
	Collection:   "workspaces",
	TenantScoped: true,
	NotFound:     ErrWorkspaceNotFound,
}

// ByWorkspaceID 按工作空间 ID 匹配
func ByWorkspaceID(id string) store.Opt {
	return store.Where("id", id)
}

// ByBCSProject 按绑定的 BCS 项目匹配，projectID 与 projectCode 任一命中即可，为空的一方不参与匹配。
// 两者都为空时不匹配任何工作空间。
//
// BCS 项目跨租户唯一，检查是否已被绑定时调用方应传 store.TenantAll()，否则其他租户的绑定会被漏掉
func ByBCSProject(projectID, projectCode string) store.Opt {
	conds := make(bson.A, 0, 2)
	if projectID != "" {
		conds = append(conds, bson.M{"bkSystems.bkBcsProjectID": projectID})
	}
	if projectCode != "" {
		conds = append(conds, bson.M{"bkSystems.bkBcsProjectCode": projectCode})
	}
	if len(conds) == 0 {
		return store.Where("bkSystems.bkBcsProjectCode", bson.M{"$in": bson.A{}})
	}
	return store.Where("$or", conds)
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

// Create 创建工作空间。绑定的 BCS 项目已被其他工作空间占用时返回 ErrBCSProjectAlreadyBound
func Create(ctx context.Context, workspaces *WorkspaceStore, ws *Workspace) error {
	err := workspaces.Create(ctx, ws)
	if errors.Is(err, store.ErrDuplicate) && strings.Contains(err.Error(), bcsProjectCodeIndex) {
		return errors.Wrapf(
			ErrBCSProjectAlreadyBound,
			"bcs project(%s) already bound by another workspace", ws.BkSystems.BkBCSProjectCode,
		)
	}
	if err != nil {
		return errors.Wrapf(err, "create workspace %s", ws.ID)
	}
	return nil
}

// Update 更新已经存在的工作空间的可变字段，不存在时返回 ErrWorkspaceNotFound。
// scope 选择租户范围，不传时使用 ctx 中的租户；运维命令跨租户更新时传 store.TenantAll()。
// 写操作只接受 TenantAll / TenantOf，传 TenantWithPlatform 会返回错误
func Update(ctx context.Context, workspaces *WorkspaceStore, ws *Workspace, scope ...store.TenantOpt) error {
	mutable := store.UpdateSelected("displayName", "description", "bkSystems", "imageRegistryType", "state", "updater")
	if err := workspaces.Update(ctx, ws, append(store.ScopeOpts(scope), mutable, ByWorkspaceID(ws.ID))...); err != nil {
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
