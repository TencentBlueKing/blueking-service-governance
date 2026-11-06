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
	"time"

	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

// Initialization 工作空间初始化的 asynq 任务类型
var Initialization *taskq.TaskType[InitializationArgs]

const (
	// initPollInterval BKM 项目就绪状态的轮询固定间隔
	initPollInterval = 15 * time.Second
	// initPollMaxRetry 轮询窗口 ~26min, 需略大于 maxWaitDuration, 以保证超时判断先于重试耗尽生效
	initPollMaxRetry = 104
)

// InitializationArgs 工作空间初始化任务参数
type InitializationArgs struct {
	WorkspaceID string `json:"workspaceID"`
}

func init() {
	Initialization = taskq.NewTaskType[InitializationArgs](
		"workspace.init",
		initHandler,
		asynq.MaxRetry(initPollMaxRetry),
	).
		WithFixedRetryInterval(initPollInterval).
		OnExhausted(func(ctx context.Context, args InitializationArgs, lastErr error) {
			onExhausted(ctx, args.WorkspaceID, lastErr)
		})
}
