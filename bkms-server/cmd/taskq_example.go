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

package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/example"
)

// NewTaskqExampleCmd 构造 taskq 端到端验证子命令。
//
// 用法:
//
//	bkms-server taskq-example --srvCfg=<path>           # 投递一条正常成功的示例任务
//	bkms-server taskq-example --srvCfg=<path> --err_fixed_retry    # 投递一条反复失败的任务,
//	                                                               # 用于验证 失败→固定间隔重试→耗尽全链路
//	bkms-server taskq-example --srvCfg=<path> --err_stop_retry    # 投递一条返回 ErrStopRetry 的任务,
//	                                                              # 用于验证 不可恢复错误→立即停止重试
//
// 消费端需先启动 `bkms-server worker`(worker 挂载任务 handler 并拉起 taskq.Server)。
func NewTaskqExampleCmd() *cobra.Command {
	var srvCfg string
	var errFixedRetry bool
	var errStopRetry bool

	cmd := cobra.Command{
		Use:   "taskq-example",
		Short: "Enqueue a taskq example task to verify the async task framework end-to-end.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				logging.Fatalf("failed to load config: %s", err)
			}

			redis.InitClient(ctx, cfg.Redis)
			taskq.InitClient(ctx, cfg.Asynq)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = taskq.Enqueue(ctx, example.ExampleTask.NewTask(example.Args{
				Msg:           "hello from taskq-example",
				ErrFixedRetry: errFixedRetry,
				ErrStopRetry:  errStopRetry,
			}))
			if err != nil {
				logging.Fatalf("failed to enqueue example task: %s", err)
			}
			logging.Infof(ctx, "taskq example task enqueued (err_fixed_retry=%t err_stop_retry=%t), "+
				"check worker logs for handler execution", errFixedRetry, errStopRetry)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().BoolVar(&errFixedRetry, "err_fixed_retry", false,
		"enqueue a task that keeps failing, to verify failure->fixed-interval-retry->exhausted flow")
	cmd.Flags().BoolVar(&errStopRetry, "err_stop_retry", false,
		"enqueue a task that returns ErrStopRetry, to verify unrecoverable-error->stop-retry flow")

	return &cmd
}

func init() {
	rootCmd.AddCommand(NewTaskqExampleCmd())
}
