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
// 以 taskq.TaskType 对外暴露。消费进程(worker)调用 Setup(mux) 完成依赖初始化与
// handler 挂载, 与 asynq 官方 mux 用法一致(mux.Handle(name, handler))。
//
// 新增业务任务:
//  1. 参照 example 子包新建自己的子包, 用 taskq.NewTaskType 定义并暴露任务类型。
//  2. 在 Setup 中追加该子包的依赖初始化(如有)与 mux.Handle 挂载。
package taskqtask

import (
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"

	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/depsvcredis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/example"
)

// Setup 初始化各业务任务 handler 所需依赖, 并将 handler 挂载到给定的 mux。
//
// 由消费进程(worker)在 store registry 初始化之后、asynq server 启动之前调用一次;
// 投递进程只需构造 task, 无需调用。各任务子包无法自行从 store registry 取依赖
// (会与 registry 形成导入环), 因此统一在本聚合包完成注入。
func Setup(mux *asynq.ServeMux) error {
	// Redis 生命周期 tasks
	if err := depsvcredis.Init(storereg.G().DepSvcInstStore); err != nil {
		return errors.Wrap(err, "init depsvcredis")
	}
	mux.Handle(depsvcredis.CreateTask.Name(), depsvcredis.CreateTask.Handler())
	mux.Handle(depsvcredis.DisableTask.Name(), depsvcredis.DisableTask.Handler())
	mux.Handle(depsvcredis.DestroyTask.Name(), depsvcredis.DestroyTask.Handler())

	mux.Handle(example.ExampleTask.Name(), example.ExampleTask.Handler())
	return nil
}
