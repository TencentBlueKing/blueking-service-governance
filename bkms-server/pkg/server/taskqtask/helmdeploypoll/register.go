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

package helmdeploypoll

import (
	"time"

	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

const (
	// name asynq 任务类型名
	name = "taskq.pollingHelmDeployStatus"
	// tickMaxRetry 单次 tick 意外失败的 asynq 重试上限，不含轮询续跑
	tickMaxRetry = 10
	// totalFailureRetryCount 查 Release 状态连续失败次数上限，耗尽后标 StatusPollingBroken；
	// 查到状态即复位（见 Handle），故约束的是连续失败而非整轮累计失败
	totalFailureRetryCount = 10
	// saveStatusTimeout 状态落库的独立超时，避免 handler ctx 取消导致写不进去
	saveStatusTimeout = 10 * time.Second
)

// Task 部署状态轮询任务；init 赋值避免与 enqueueNext 引用形成包初始化环
var Task *taskq.TaskType[Args]

func init() {
	Task = taskq.NewTaskType[Args](name, handle, asynq.MaxRetry(tickMaxRetry)).OnExhausted(onExhausted)
}

// PollingInterval 读 TaskPoller.DeployStatus.Interval（秒），作为下一 tick 的 ProcessIn 延迟
func PollingInterval() time.Duration {
	return time.Duration(config.G.TaskPoller.DeployStatus.Interval) * time.Second
}

// pollingTimeout 读 TaskPoller.DeployStatus.Timeout（秒），从 record.StartedAt 起算轮询窗口
func pollingTimeout() time.Duration {
	return time.Duration(config.G.TaskPoller.DeployStatus.Timeout) * time.Second
}
