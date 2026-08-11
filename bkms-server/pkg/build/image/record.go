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

package build

import (
	"time"
)

// Status 构建状态
type Status string

const (
	// StatusRunning 构建中
	StatusRunning Status = "running"
	// StatusSuccess 构建成功
	StatusSuccess Status = "success"
	// StatusFailed 构建失败
	StatusFailed Status = "failed"
	// StatusCanceled 已取消
	StatusCanceled Status = "canceled"
	// StatusUnknown 未知状态
	StatusUnknown Status = "unknown"
	// StatusPollingTimeout 轮询超时
	StatusPollingTimeout Status = "pollingTimeout"
	// StatusPollingBroken 轮询中断
	StatusPollingBroken Status = "pollingBroken"
)

// IsTerminated 判断状态是否为终态
func (s Status) IsTerminated() bool {
	switch s {
	case StatusSuccess, StatusFailed, StatusCanceled:
		return true
	case StatusPollingTimeout, StatusPollingBroken:
		return true
	default:
		return false
	}
}

// TriggerType 构建的触发方式
type TriggerType string

const (
	// TriggerTypeManual 手动触发，即前端「执行构建」按钮或 bkms-cli
	TriggerTypeManual TriggerType = "manual"
	// TriggerTypeAuto 自动触发，由触发策略经蓝盾回调发起
	TriggerTypeAuto TriggerType = "auto"
)

// Record 构建记录
type Record struct {
	// WorkspaceID 工作空间名称
	WorkspaceID string `bson:"workspaceID"`
	// AppID 应用 ID
	AppID string `bson:"appID"`
	// PipelineID 流水线 ID
	PipelineID string `bson:"pipelineID"`
	// BuildID 构建 ID
	BuildID string `bson:"buildID"`
	// Num 构建序号
	Num int64 `bson:"num"`
	// Params 构建参数
	Params map[string]string `bson:"params"`
	// Status 构建状态
	Status Status `bson:"status"`
	// Artifact 构建产物（镜像）
	Artifact string `bson:"artifact"`
	// Operator 操作人
	Operator string `bson:"operator"`
	// Extras 额外信息
	Extras map[string]string `bson:"extras"`
	// TriggerType 触发方式
	TriggerType TriggerType `bson:"triggerType,omitempty"`
	// TriggerPolicyID 自动触发时关联的触发策略 ID，手动触发为空
	TriggerPolicyID string `bson:"triggerPolicyID,omitempty"`
	// StartedAt 开始时间
	StartedAt time.Time `bson:"startedAt"`
	// EndedAt 结束时间
	EndedAt time.Time `bson:"endedAt"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
