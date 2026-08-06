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

package worker

import (
	"context"

	"github.com/pkg/errors"
)

// ApplyTask 下发异步任务
func ApplyTask(ctx context.Context, uri, queue string, taskName taskName, args any) (string, error) {
	// 初始化任务管理器（ApplyTask 只是发送消息，不需要 prefetch 控制，也不需要 context mutators）
	worker, err := New(uri, queue, 0, nil)
	if err != nil {
		return "", errors.Wrap(err, "new task worker")
	}
	defer worker.Close()

	// 下发异步任务
	taskID, err := worker.apply(ctx, taskName, args)
	if err != nil {
		reportTaskEnqueue(taskName, statusErr)
		return "", errors.Wrap(err, "apply task")
	}

	reportTaskEnqueue(taskName, statusOK)
	return taskID, nil
}

// globalRegistry 全局任务注册表实例
var globalRegistry = &registry{mapping: make(map[taskName]definition)}

// RegisterTask 注册任务
func RegisterTask[Args, Result any](name taskName, taskFunc taskFunc[Args, Result]) {
	def := definition{
		NewArgs:   func() any { return new(Args) },
		NewResult: func() any { return new(Result) },
		ExecFunc: func(ctx context.Context, raw any) (any, error) {
			// 检查输入参数的类型是否符合要求
			args, ok := raw.(*Args)
			if !ok {
				return nil, errors.Errorf("invalid args type for task '%s’", name)
			}
			// 调用任务函数执行
			return taskFunc(ctx, *args)
		},
	}

	globalRegistry.set(name, def)
}
