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

// Package taskqtask 集中挂载 taskq 异步任务框架的所有业务任务 handler。
//
// 各任务的名称、Args、handler 与默认投递配置收敛在各自的子包(如 example),
// 以 taskq.TaskType 对外暴露。消费进程(worker)调用 RegisterAll(mux) 完成显式挂载,
// 与 asynq 官方 mux 用法一致(mux.Handle(name, handler))。
//
// 新增业务任务:
//  1. 参照 example 子包新建自己的子包, 用 taskq.NewTaskType 定义并暴露任务类型。
//  2. 在 RegisterAll 中追加一行 mux.Handle(<pkg>.<Type>.Name(), <pkg>.<Type>.Handler())。
package taskqtask

import (
	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/example"
)

// RegisterAll 把所有业务任务的 handler 挂载到给定的 mux。
func RegisterAll(mux *asynq.ServeMux) {
	mux.Handle(example.ExampleTask.Name(), example.ExampleTask.Handler())
}
