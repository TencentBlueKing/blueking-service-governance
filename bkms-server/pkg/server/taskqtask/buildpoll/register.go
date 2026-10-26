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

package buildpoll

import (
	"time"

	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

const (
	// name asynq 任务类型名
	name = "taskq.pollingBuildStatus"
	// tickMaxRetry 单次 tick 意外失败的 asynq 重试上限，不含轮询续跑
	tickMaxRetry = 10
	// totalFailureRetryCount 查蓝盾连续失败次数上限，耗尽后标 StatusPollingBroken；
	// 查到状态即复位（见 runTick），故约束的是连续失败而非整轮累计失败
	totalFailureRetryCount = 10
	// saveStatusTimeout 状态落库的独立超时，避免 handler ctx 取消导致写不进去
	saveStatusTimeout = 10 * time.Second
	// pollingTimeout 从 record.StartedAt 起算的轮询窗口，超时标 StatusPollingTimeout
	pollingTimeout = 24 * time.Hour

	// imageDigestName 记录构建产物镜像 digest 的 asynq 任务类型名
	imageDigestName = "taskq.recordBuildImageDigest"
	// imageDigestMaxRetry 记录 digest 失败的重试上限；按 asynq 默认退避，全部重试落在 imageDigestRecordWindow 内
	imageDigestMaxRetry = 3
	// imageDigestRecordWindow 投递任务后允许记录 digest 的时长。digest 按 tag 回查，读得越晚越可能读到被覆盖后的镜像，
	// 超出则放弃。从投递时而非构建结束时起算，否则轮询间隔长的构建会一律记录不上
	imageDigestRecordWindow = 5 * time.Minute
)

// Task 构建状态轮询任务；init 赋值避免与 enqueueNext 引用形成包初始化环
var Task *taskq.TaskType[Args]

// ImageDigestTask 构建成功后记录产物镜像 digest 的任务
var ImageDigestTask = taskq.NewTaskType[ImageDigestArgs](
	imageDigestName, handleRecordImageDigest, asynq.MaxRetry(imageDigestMaxRetry),
)

func init() {
	Task = taskq.NewTaskType[Args](name, handle, asynq.MaxRetry(tickMaxRetry)).OnExhausted(onExhausted)
}
