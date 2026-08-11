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

// Package trigger 自动触发镜像构建的触发策略与触发记录领域模型。
// 契约详见 design_notes/build_trigger_contract.md
package trigger

import (
	"time"
)

// Event 触发事件类型
type Event string

// EventPush 推送分支时触发，本期唯一取值
const EventPush Event = "push"

// BranchMatchMode 分支匹配方式
type BranchMatchMode string

const (
	// BranchMatchModeEq 分支名等于匹配值
	BranchMatchModeEq BranchMatchMode = "eq"
	// BranchMatchModePrefix 分支名以匹配值为前缀
	BranchMatchModePrefix BranchMatchMode = "prefix"
	// BranchMatchModeAll 匹配全部分支，此时匹配值为空
	BranchMatchModeAll BranchMatchMode = "all"
)

// Status 策略启停状态
type Status string

const (
	// StatusEnabled 生效中
	StatusEnabled Status = "enabled"
	// StatusDisabled 已停用，回调仍会到达但不发起构建
	StatusDisabled Status = "disabled"
)

// Result 触发记录的处理结果
type Result string

const (
	// ResultBuilt 已构建，关联构建号
	ResultBuilt Result = "built"
	// ResultSkipped 已跳过，仅覆盖「策略已停用」与「同 commit 已构建去重」两类
	ResultSkipped Result = "skipped"
	// ResultFailed 触发失败，即已收到回调但发起构建失败
	ResultFailed Result = "failed"
)

const (
	// PolicyNameMinLen 策略名称最小长度
	PolicyNameMinLen = 1
	// PolicyNameMaxLen 策略名称最大长度
	PolicyNameMaxLen = 32
	// MaxPoliciesPerApp 单应用策略数量上限，生效中与已停用合并计数
	MaxPoliciesPerApp = 5
)

// Policy 触发策略。
// 一个应用最多 MaxPoliciesPerApp 条策略，同应用的多条策略映射到同一条触发专用流水线上的多个触发器。
// 自动触发的镜像 tag 规则不落在策略上，统一使用应用 buildConfig.tagConfig
type Policy struct {
	// ID 策略 ID，全局唯一
	ID string `bson:"id"`
	// AppID 所属应用
	AppID string `bson:"appID"`
	// Name 策略名称，应用内唯一
	Name string `bson:"name"`
	// Event 触发事件
	Event Event `bson:"event"`
	// BranchMatchMode 分支匹配方式
	BranchMatchMode BranchMatchMode `bson:"branchMatchMode"`
	// BranchMatchValue 分支匹配值，多值以英文逗号分隔；匹配方式为 all 时为空
	BranchMatchValue string `bson:"branchMatchValue,omitempty"`
	// PathFilter 文件路径条件，留空表示全匹配
	PathFilter string `bson:"pathFilter,omitempty"`
	// Status 启停状态
	Status Status `bson:"status"`
	// PipelineID 关联的蓝盾触发专用流水线 ID
	PipelineID string `bson:"pipelineID,omitempty"`
	// TriggerID 关联的蓝盾触发器元素标识，供触发器同步增删改时定位蓝盾侧节点
	TriggerID string `bson:"triggerID,omitempty"`
	// Creator 创建人
	Creator string `bson:"creator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// Record 触发记录。
// 全量落库长期保留，本期不做条数上限与过期清理。
// 分支 / 路径未命中的事件被蓝盾触发器过滤，不会回调 bkms，因此不产生记录
type Record struct {
	// PolicyID 归属策略
	PolicyID string `bson:"policyID"`
	// AppID 归属应用
	AppID string `bson:"appID"`
	// TriggeredAt 触发时间
	TriggeredAt time.Time `bson:"triggeredAt"`
	// Event 事件类型
	Event Event `bson:"event"`
	// Branch 分支名
	Branch string `bson:"branch"`
	// CommitID commit 哈希
	CommitID string `bson:"commitID"`
	// CommitAuthor commit 作者
	CommitAuthor string `bson:"commitAuthor"`
	// Result 处理结果
	Result Result `bson:"result"`
	// BuildID 结果为 built 时关联的构建号，其余为空
	BuildID string `bson:"buildID,omitempty"`
	// Reason 跳过或失败原因，结果为 built 时为空
	Reason string `bson:"reason,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
}
