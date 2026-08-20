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

package alertstrategysync

import (
	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

const (
	syncName    = "taskq.alertStrategy.sync"
	cleanupName = "taskq.alertStrategy.cleanup"
	maxRetry    = 5
)

// SyncTask 部署成功后同步告警策略到蓝鲸监控远端
var SyncTask *taskq.TaskType[Args]

// CleanupTask 卸载后清理告警策略远端引用
var CleanupTask *taskq.TaskType[Args]

func init() {
	SyncTask = taskq.NewTaskType[Args](syncName, handleSync, asynq.MaxRetry(maxRetry)).
		OnExhausted(onSyncExhausted)
	CleanupTask = taskq.NewTaskType[Args](cleanupName, handleCleanup, asynq.MaxRetry(maxRetry)).
		OnExhausted(onCleanupExhausted)
}
