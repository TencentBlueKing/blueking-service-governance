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

package taskq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// 内建哨兵错误: 业务 handler 直接返回(或用 %w 包装)即可控制重试行为,
//
// 其余错误(未命中任一哨兵)一律走默认指数退避重试。
var (
	// ErrStopRetry 标记不可恢复的失败, 框架停止重试(置为终态失败)。
	// 业务用法: return fmt.Errorf("invalid arg: %w", taskq.ErrStopRetry)。
	ErrStopRetry = errors.New("taskq: stop retry")
	// ErrFixedRetry 标记"任务仍在进行中", 框架以固定间隔重试(而非指数退避)。
	// 间隔取 config.Asynq.RetryInterval(全局默认)。
	// 业务用法: return fmt.Errorf("still provisioning: %w", taskq.ErrFixedRetry)。
	ErrFixedRetry = errors.New("taskq: retry with fixed interval")
)

// TaskType 是一种强类型异步任务的定义(模板), 由 NewTaskType 构造。
//
// 它对应 asynq 概念中的"任务类型(task type)", 而非单次执行的 asynq.Task 实例:
// 一个 TaskType 可被多次 Enqueue, 每次投递才生成一个具体的 asynq.Task。
//
// 它把"任务名 + 强类型 handler + 默认投递选项"收敛在一处, 对外提供:
//   - Enqueue: 类型安全投递(自动 JSON 序列化 Args), 每次调用投递一个该类型的任务实例。
//   - Name / Handler: 供消费侧显式挂载到 asynq.ServeMux(mux.Handle(t.Name(), t.Handler()))。
type TaskType[Args any] struct {
	name        string
	defaultOpts []asynq.Option
	handler     HandlerFunc[Args]
}

// HandlerFunc 强类型业务处理函数。
//
// 返回 nil 表示任务成功结束; 返回 error 时, 框架据错误链决定重试行为(默认指数退避)。
// 框架自动完成 payload 反序列化, 业务直接拿到 Args。
type HandlerFunc[Args any] func(ctx context.Context, args Args) error

// NewTaskType 定义一种强类型任务类型(模板)。
// 注意: 当前 server 仅消费单一队列, 请勿在 opts 中指定 asynq.Queue。
func NewTaskType[Args any](name string, h HandlerFunc[Args], opts ...asynq.Option) *TaskType[Args] {
	return &TaskType[Args]{name: name, defaultOpts: opts, handler: h}
}

// Name 返回任务名(用于 mux.Handle 与投递)。
func (t *TaskType[Args]) Name() string { return t.name }

// Handler 返回可挂载到 asynq.ServeMux 的底层处理函数。
//
// 它负责: 反序列化 Args -> 调用业务 handler ->
// 把 ErrStopRetry / 反序列化失败翻译为停止重试(其余错误交由重试策略)。
func (t *TaskType[Args]) Handler() asynq.HandlerFunc {
	return func(ctx context.Context, task *asynq.Task) error {
		var args Args
		if err := json.Unmarshal(task.Payload(), &args); err != nil {
			// 负载无法解析属不可恢复, 停止重试。
			return wrapStopRetry(fmt.Errorf("unmarshal args for task %q: %w", t.name, err))
		}
		err := t.handler(ctx, args)
		if err == nil {
			return nil
		}
		// 业务显式要求停止重试。
		if errors.Is(err, ErrStopRetry) {
			return wrapStopRetry(err)
		}
		// 其余错误直接抛给上层，由上层决定重试策略
		return err
	}
}

// NewTask 产出一个可投递的任务实例。
func (t *TaskType[Args]) NewTask(args Args) *Task {
	payload, err := json.Marshal(args)
	if err != nil {
		logging.ErrorNoContextf("taskq: marshal args for task %s: %v", t.name, err)
		return nil
	}
	return &Task{name: t.name, payload: payload, defaultOpts: t.defaultOpts}
}
