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

package depsvcredis

import (
	"context"
	"time"

	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

var (
	// CreateTask 创建 Redis 实例的 asynq 任务类型
	CreateTask *taskq.TaskType[CreateArgs]
	// DisableTask 禁用 Redis 实例的 asynq 任务类型
	DisableTask *taskq.TaskType[DisableArgs]
	// DestroyTask 销毁 Redis 实例的 asynq 任务类型
	DestroyTask *taskq.TaskType[DestroyArgs]
)

const (
	// redisPollInterval Redis 工单轮询固定间隔
	redisPollInterval = 30 * time.Second
	// redisPollMaxRetry 轮询窗口：30s × 5760 ≈ 48h，覆盖 DBM 审批+部署最长等待。
	redisPollMaxRetry = 5760
)

// Init 注册 Redis 生命周期相关的任务
func init() {
	CreateTask = taskq.NewTaskType[CreateArgs](
		"depservice.redis.create",
		createHandler,
		asynq.MaxRetry(redisPollMaxRetry),
	).
		WithFixedRetryInterval(redisPollInterval).
		OnExhausted(func(ctx context.Context, args CreateArgs, lastErr error) {
			failOnExhausted(ctx, args.InstanceID, model.CreateFailedStatus, lastErr)
		})

	DisableTask = taskq.NewTaskType[DisableArgs](
		"depservice.redis.disable",
		disableHandler,
		asynq.MaxRetry(redisPollMaxRetry),
	).
		WithFixedRetryInterval(redisPollInterval).
		OnExhausted(func(ctx context.Context, args DisableArgs, lastErr error) {
			failOnExhausted(ctx, args.InstanceID, model.DeleteFailedStatus, lastErr)
		})

	DestroyTask = taskq.NewTaskType[DestroyArgs](
		"depservice.redis.destroy",
		destroyHandler,
		asynq.MaxRetry(redisPollMaxRetry),
	).
		WithFixedRetryInterval(redisPollInterval).
		OnExhausted(func(ctx context.Context, args DestroyArgs, lastErr error) {
			failOnExhausted(ctx, args.InstanceID, model.DeleteFailedStatus, lastErr)
		})
}
