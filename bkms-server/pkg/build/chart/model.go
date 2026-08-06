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

// Package build 提供 Helm Chart 构建触发和构建记录管理功能
package build

import "time"

// Status Helm Chart 构建状态
type Status string

const (
	// StatusRunning 构建中
	StatusRunning Status = "running"
	// StatusSuccess 构建成功
	StatusSuccess Status = "success"
	// StatusFailed 构建失败
	StatusFailed Status = "failed"
	// StatusCanceled 构建已取消
	StatusCanceled Status = "canceled"
	// StatusPollingTimeout 轮询超时（蓝盾构建仍在运行，但平台侧轮询已超时）
	StatusPollingTimeout Status = "pollingTimeout"
	// StatusPollingBroken 轮询异常（查询蓝盾状态时发生错误）
	StatusPollingBroken Status = "pollingBroken"
)

// Record Helm Chart 构建记录
type Record struct {
	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// AppID 应用 ID
	AppID string `bson:"appID"`
	// Num 构建序号（每个 AppID 独立自增）
	Num int64 `bson:"num"`
	// BuildID 蓝盾构建 ID
	BuildID string `bson:"buildID"`
	// PipelineID 蓝盾流水线 ID
	PipelineID string `bson:"pipelineID"`
	// ChartVersion 本次构建的 Chart 版本号（semver 格式：major.minor.patch）
	ChartVersion string `bson:"chartVersion"`
	// Status 构建状态
	Status Status `bson:"status"`
	// Operator 触发人
	Operator string `bson:"operator"`
	// Params 构建启动参数（含代码库、分支等触发时已知信息）
	Params map[string]string `bson:"params,omitempty"`
	// Extras 构建额外信息（含 commit ID 等，由轮询任务回写）
	Extras map[string]string `bson:"extras,omitempty"`
	// StartedAt 构建开始时间
	StartedAt time.Time `bson:"startedAt"`
	// EndedAt 构建结束时间（终态时设置）
	EndedAt *time.Time `bson:"endedAt,omitempty"`
}

// IsTerminated 判断构建是否已处于终态
func (r *Record) IsTerminated() bool {
	switch r.Status {
	case StatusSuccess, StatusFailed, StatusCanceled, StatusPollingTimeout, StatusPollingBroken:
		return true
	default:
		return false
	}
}
