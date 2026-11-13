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

// Package installtask 定义集群 Addon 的异步安装任务记录。
//
// 独立于 clusteraddon 包，是因为 clusteraddon 经 deploy/helm 依赖 addon/polaris，
// 而 addon/polaris/accesspoint 需要读取任务记录，放在 clusteraddon 内会形成循环依赖。
// 任务的执行（helm 安装、加锁）由 clusteraddon 负责，本包只有模型与存储。
//
// TODO(北极星组件安装子需求): 目前只有模型与存储，还没有写入方。需在 clusteraddon 中补齐：
//   - 按 clusterID 安装/升级的入口（环境入口复用它），创建任务记录并投递异步任务；
//   - 异步执行时持有 clusteraddon.NewInstallLock，结束后写回 status、message、finishedAt；
//   - 超时（10 分钟）置为失败，服务重启后不允许任务永久停留在 running。
package installtask

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Status 安装任务状态
type Status string

const (
	// StatusRunning 运行中，同一集群命名空间内的同一 Addon 同时至多一个
	StatusRunning Status = "running"
	// StatusSucceeded 成功
	StatusSucceeded Status = "succeeded"
	// StatusFailed 失败
	StatusFailed Status = "failed"
)

// InstallTask 在集群上异步安装或更新 Addon 的任务记录。
//
// 执行 helm 安装时必须持有 clusteraddon.NewInstallLock，与环境页同步安装同一 Addon 的路径互斥
type InstallTask struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// ClusterID 目标集群 ID
	ClusterID string `bson:"clusterID"`
	// Namespace Addon 安装的命名空间
	Namespace string `bson:"namespace"`
	// AddonName Addon 名称，即 ClusterAddonDef.Name
	AddonName string `bson:"addonName"`
	// ChartVersion 安装的 chart 版本
	ChartVersion string `bson:"chartVersion"`
	// Status 任务状态
	Status Status `bson:"status"`
	// Message 失败原因等补充信息
	Message string `bson:"message"`
	// Operator 发起人
	Operator string `bson:"operator"`
	// StartedAt 开始时间
	StartedAt time.Time `bson:"startedAt"`
	// FinishedAt 结束时间，运行中为零值
	FinishedAt time.Time `bson:"finishedAt"`

	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}
