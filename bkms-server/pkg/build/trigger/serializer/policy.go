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

package serializer

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger"
)

// TriggerHealth 触发器与流水线的健康状态。
// 属 P1 功能，数据来源（主动巡检蓝盾还是回调异常时被动标记）尚未确定，
// 契约先固定枚举，未接入前一律返回 TriggerHealthUnknown
type TriggerHealth string

const (
	// TriggerHealthUnknown 状态未知，健康状态能力未接入时的默认值
	TriggerHealthUnknown TriggerHealth = "unknown"
	// TriggerHealthHealthy 流水线与触发器正常
	TriggerHealthHealthy TriggerHealth = "healthy"
	// TriggerHealthUnauthorized 代码库授权失效，需重新授权
	TriggerHealthUnauthorized TriggerHealth = "unauthorized"
)

// VersionRuleInput is the JSON representation of an image version rule.
type VersionRuleInput struct {
	// 版本号规则类型：custom 自定义版本，semver 语义化版本
	Type string `json:"type" binding:"required,oneof=custom semver"`
	// 自定义前缀，仅 custom 类型使用
	Prefix string `json:"prefix" binding:"omitempty,max=16,trigger_version_prefix"`
	// 版本号是否拼接分支名，仅 custom 类型使用
	WithBranch bool `json:"withBranch"`
}

// ToModel converts the input to a version rule model.
func (i VersionRuleInput) ToModel() trigger.VersionRule {
	return trigger.VersionRule{
		Type:       trigger.VersionRuleType(i.Type),
		Prefix:     i.Prefix,
		WithBranch: i.WithBranch,
	}
}

// PolicyFormInput is the JSON body shared by creating, updating and conflict-checking a policy.
type PolicyFormInput struct {
	// 策略名称，应用内唯一，由汉字、大小写字母、数字、- 与 _ 组成
	Name string `json:"name" binding:"required,min=1,max=32,trigger_policy_name"`
	// 触发事件，本期仅支持 push（推送分支）
	Event string `json:"event" binding:"required,oneof=push"`
	// 分支匹配方式：eq 等于，prefix 前缀，all 全部
	BranchMatchMode string `json:"branchMatchMode" binding:"required,oneof=eq prefix all"`
	// 分支匹配值，多值以英文逗号分隔；匹配方式为 all 时必须留空
	BranchMatchValue string `json:"branchMatchValue" binding:"max=512"`
	// 文件路径条件，留空表示全匹配
	PathFilter string `json:"pathFilter" binding:"max=512"`
	// 镜像版本号规则
	VersionRule *VersionRuleInput `json:"versionRule" binding:"required"`
}

// PatchPolicyStatusInput is the JSON body for enabling or disabling a policy.
type PatchPolicyStatusInput struct {
	// 是否启用；用指针以区分「未传」与「显式传 false」
	Enabled *bool `json:"enabled" binding:"required"`
}

// Status maps the enabled flag to a policy status.
func (i PatchPolicyStatusInput) Status() trigger.Status {
	if i.Enabled != nil && *i.Enabled {
		return trigger.StatusEnabled
	}
	return trigger.StatusDisabled
}

// VersionRuleOutput is the JSON representation of an image version rule.
type VersionRuleOutput struct {
	// 版本号规则类型
	Type string `json:"type"`
	// 自定义前缀
	Prefix string `json:"prefix"`
	// 版本号是否拼接分支名
	WithBranch bool `json:"withBranch"`
}

// PolicyOutputObj is the JSON representation of one trigger policy.
// 该结构不包含回调凭证的任何形态，凭证由蓝盾凭证管理持有，不在响应中回显
type PolicyOutputObj struct {
	// 策略 ID
	ID string `json:"id"`
	// 所属应用 ID
	AppID string `json:"appID"`
	// 策略名称
	Name string `json:"name"`
	// 触发事件
	Event string `json:"event"`
	// 分支匹配方式
	BranchMatchMode string `json:"branchMatchMode"`
	// 分支匹配值
	BranchMatchValue string `json:"branchMatchValue"`
	// 文件路径条件
	PathFilter string `json:"pathFilter"`
	// 镜像版本号规则
	VersionRule VersionRuleOutput `json:"versionRule"`
	// 启停状态：enabled 生效中，disabled 已停用
	Status string `json:"status"`
	// 关联的蓝盾触发专用流水线 ID
	PipelineID string `json:"pipelineID"`
	// 关联的蓝盾触发器标识
	TriggerID string `json:"triggerID"`
	// 流水线与触发器健康状态：unknown / healthy / unauthorized
	Health string `json:"health"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from a trigger policy model.
// Health 不属于策略实体，需由调用方在健康状态能力接入后覆写
func (o *PolicyOutputObj) FromModel(p trigger.Policy) *PolicyOutputObj {
	*o = PolicyOutputObj{
		ID:               p.ID,
		AppID:            p.AppID,
		Name:             p.Name,
		Event:            string(p.Event),
		BranchMatchMode:  string(p.BranchMatchMode),
		BranchMatchValue: p.BranchMatchValue,
		PathFilter:       p.PathFilter,
		VersionRule: VersionRuleOutput{
			Type:       string(p.VersionRule.Type),
			Prefix:     p.VersionRule.Prefix,
			WithBranch: p.VersionRule.WithBranch,
		},
		Status:     string(p.Status),
		PipelineID: p.PipelineID,
		TriggerID:  p.TriggerID,
		Health:     string(TriggerHealthUnknown),
		Creator:    p.Creator,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
	return o
}

// PolicyListOutputObjs is the trigger policy list payload.
// 单应用策略上限为 MaxPoliciesPerApp，无需分页，一次返回全部
type PolicyListOutputObjs struct {
	// 策略总数，生效中与已停用合并计数
	Count int64 `json:"count,string"`
	// 全部策略
	Results []*PolicyOutputObj `json:"results"`
}

// ListPoliciesOutput is the JSON response for listing trigger policies.
type ListPoliciesOutput struct {
	// 触发策略列表
	Data *PolicyListOutputObjs `json:"data"`
}

// PolicyOutput is the JSON response for APIs returning a single trigger policy.
type PolicyOutput struct {
	// 触发策略
	Data *PolicyOutputObj `json:"data"`
}
