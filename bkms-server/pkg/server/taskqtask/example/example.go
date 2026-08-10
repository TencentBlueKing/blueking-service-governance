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

// Package example 提供 taskq 框架的示例任务, 用于端到端验证投递→消费→重试全链路。
//
// 它作为新业务任务的参考模板: 任务名、Args、handler、默认投递配置都收敛在本包,
// 对外暴露 TaskType(用于消费侧挂载与投递)。
package example

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

// name 任务名
const name = "taskq.example"

// maxRetry 任务最大重试次数
const maxRetry = 3

// Args 示例任务参数。
type Args struct {
	Msg string `json:"msg"`
	// ErrKeepRetry 为 true 时, handler 返回包装 ErrFixedRetry 的错误, 用于模拟"任务反复失败直至耗尽"。
	ErrFixedRetry bool `json:"err_fixed_retry"`
	// ErrStopRetry 为 true 时, handler 返回包装 ErrStopRetry 的错误, 用于模拟"遇到不可恢复错误立即停止重试"。
	ErrStopRetry bool `json:"err_stop_retry"`
}

// ExampleTask 示例任务类型。消费侧通过 taskqtask 聚合包挂载, 投递侧调用 ExampleTask.Enqueue。
var ExampleTask = taskq.NewTaskType[Args](name, handler, asynq.MaxRetry(maxRetry))

// handler 示例任务处理逻辑, 演示三种典型返回:
//
//   - Skip=true: 返回包装 ErrStopRetry 的错误。框架识别后立即停止重试(不再消耗 maxRetry),
//     适用于参数非法、资源已删除等不可恢复错误。优先级高于 Fail。
//   - Fail=true: 返回包装 ErrFixedRetry 的错误, 框架按固定间隔重试, 到 maxRetry 后耗尽,
//     用于验证 失败→固定间隔重试→耗尽 全链路。
//   - 均为 false: 打印日志并成功返回。
func handler(ctx context.Context, args Args) error {
	log.Infof(
		ctx, "taskq example handler received: msg=%s fail=%t skip=%t",
		args.Msg, args.ErrFixedRetry, args.ErrStopRetry,
	)
	switch {
	case args.ErrStopRetry:
		// 不可恢复错误: 包装 ErrStopRetry, 框架翻译为 asynq.SkipRetry, 立即终止不再重试。
		return errors.Wrapf(taskq.ErrStopRetry, "taskq example handler unrecoverable error: msg=%s", args.Msg)
	case args.ErrFixedRetry:
		return errors.Wrapf(taskq.ErrFixedRetry, "taskq example handler simulated failure: msg=%s", args.Msg)
	default:
		return nil
	}
}
