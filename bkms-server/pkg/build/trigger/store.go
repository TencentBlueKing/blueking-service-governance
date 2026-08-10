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

package trigger

import (
	"context"

	"github.com/pkg/errors"
)

// PolicyCollName 触发策略表名
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + name
// - 普通：appID
const PolicyCollName = "build_trigger_policies"

// RecordCollName 触发记录表名
//
// 索引（由 golang-migrate 维护）：
// - 普通：policyID + triggeredAt
// - 普通：policyID + commitID
//
// 注意 policyID + commitID 不能设唯一约束：去重规则是「同一策略下同一 commit 已成功构建过则跳过」，
// 同一 commit 仍可能产生多条 skipped / failed 记录，唯一约束会让这些记录写不进去
const RecordCollName = "build_trigger_records"

var (
	// ErrPolicyNotFound 触发策略不存在
	ErrPolicyNotFound = errors.New("build trigger policy not found")
	// ErrPolicyNameDuplicated 同应用下已存在同名触发策略
	ErrPolicyNameDuplicated = errors.New("build trigger policy name already exists in the app")
)

// PolicyStore 触发策略存储接口。
// Mongo 实现由「触发策略后端管理与冲突检测」子需求落地
type PolicyStore interface {
	// Create 创建触发策略，名称在应用内重复时返回 ErrPolicyNameDuplicated
	Create(ctx context.Context, policy *Policy) error

	// Update 更新触发策略，策略不存在时返回 ErrPolicyNotFound。
	// 仅更新策略表单字段与蓝盾关联标识，不改动 appID / creator / createdAt
	Update(ctx context.Context, policy *Policy) error

	// UpdateStatus 更新策略启停状态，策略不存在时返回 ErrPolicyNotFound
	UpdateStatus(ctx context.Context, appID, policyID string, status Status) error

	// Get 获取单条触发策略，不存在时返回 ErrPolicyNotFound
	Get(ctx context.Context, appID, policyID string) (*Policy, error)

	// List 获取应用下的全部触发策略。
	// 单应用策略上限为 MaxPoliciesPerApp，无需分页；生效中与已停用一并返回
	List(ctx context.Context, appID string) ([]Policy, error)

	// Delete 删除触发策略，策略不存在时返回 ErrPolicyNotFound。
	// 不级联删除触发记录与已产出的构建记录
	Delete(ctx context.Context, appID, policyID string) error
}

// RecordStore 触发记录存储接口。
// Mongo 实现由「自动触发记录存储与查询」子需求落地
type RecordStore interface {
	// Create 创建触发记录
	Create(ctx context.Context, record *Record) error

	// List 获取策略的触发记录列表（支持分页），按触发时间倒序。
	// result 为空表示不按结果筛选
	List(ctx context.Context, policyID string, result Result, page, pageSize int64) ([]Record, int64, error)

	// ExistsBuiltByCommit 判断该策略下指定 commit 是否已成功发起过构建，用于回调去重。
	// 只统计 result 为 ResultBuilt 的记录
	ExistsBuiltByCommit(ctx context.Context, policyID, commitID string) (bool, error)
}
