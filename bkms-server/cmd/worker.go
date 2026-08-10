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
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/worker"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

// workerShutdownTimeout 限制进程退出时等待 worker 优雅关闭的最长时间
const workerShutdownTimeout = 30 * time.Second

// NewWorkerCmd ...
func NewWorkerCmd() *cobra.Command {
	var srvCfg string

	workerCmd := cobra.Command{
		Use:   "worker",
		Short: "Start the async task worker to process tasks in rabbitmq.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 提前监听信号，避免初始化期间信号丢失
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			// 加载项目自定义配置
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				return errors.Wrap(err, "load config")
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				return errors.Wrap(err, "init logger")
			}

			// 初始化蓝鲸监控 APM，与 webserver 保持一致，覆盖 worker 内异步任务的链路上报
			shutdownAPM := apm.Setup(ctx, cfg.BkMonitor, cmd.Name())
			defer func() {
				// 使用基于 Background 的独立超时 context，避免复用已经 Done 的 cmd.Context() 导致 span 未刷完就退出
				shutdownCtx, cancel := context.WithTimeout(context.Background(), apmShutdownTimeout)
				defer cancel()
				if err = shutdownAPM(shutdownCtx); err != nil {
					log.Errorf(ctx, "shutdown APM: %v", err)
				}
			}()

			// 启动 Prometheus Metrics Server
			// metrics.StartServer 内部会监听 ctx.Done() 自行触发优雅关闭，无需在此重复调用 StopServer
			metrics.StartServer(ctx)

			// 初始化数据库客户端
			database.InitClient(ctx, cfg.Mongo)
			// 初始化 redis 客户端
			redis.InitClient(ctx, cfg.Redis)
			// 初始化通用任务框架投递端 client
			taskq.InitClient(ctx, cfg.Asynq)
			// 初始化 worker 侧可复用的 store registry
			storereg.Init(ctx)

			// 启动阶段主动初始化权限管理器，提前暴露 IAM client、角色存储等构造问题，避免延迟到异步任务首次鉴权时才失败
			_ = perm.NewManager()

			// 初始化 workload 插件（必须在 store 初始化之后）
			workload.InitPlugin(
				storereg.G().AppConfigFileStore,
				storereg.G().PolarisConfigStore,
			)

			// 初始化任务管理器
			wk, err := worker.New(
				cfg.RabbitMQ.GetURI(),
				cfg.RabbitMQ.Queue,
				cfg.RabbitMQ.Prefetch,
				nil,
			)
			if err != nil {
				return errors.Wrap(err, "new task worker")
			}
			defer wk.Close()

			// 启动消费者，开始监听队列
			if err = wk.Run(ctx); err != nil {
				return errors.Wrap(err, "start task consumer")
			}

			// 启动通用任务框架 server: 初始化任务依赖并挂载所有业务任务 handler
			mux := asynq.NewServeMux()
			if err = taskqtask.Setup(mux); err != nil {
				return errors.Wrap(err, "setup taskq tasks")
			}
			taskSrv := taskq.NewServer(ctx, cfg.Asynq, mux)
			if err = taskSrv.Start(); err != nil {
				log.Fatalf("failed to start taskq server: %v", err)
			}

			log.Info(ctx, "worker started, waiting for tasks...")

			// 等待信号（SIGINT / SIGTERM）触发 ctx.Done()
			<-ctx.Done()
			log.Info(ctx, "worker received shutdown signal, starting graceful shutdown...")

			// 优雅退出 worker
			gracefulShutdown(ctx, wk, taskSrv)
			return nil
		},
	}

	// 配置文件路径
	workerCmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")

	return &workerCmd
}

// gracefulShutdown 优雅停止 worker
//
// 使用独立超时的 Background context，避免复用已 Done 的 ctx 导致 Stop 立即返回
func gracefulShutdown(ctx context.Context, wk *worker.Worker, taskSrv taskq.Server) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), workerShutdownTimeout)
	defer cancel()

	// 停止通用任务框架 server(阻塞直至在途任务处理完或内部超时)
	taskSrv.Shutdown()
	taskq.CloseClient(ctx)
	log.Info(ctx, "taskq server stopped")

	done := make(chan error, 1)
	go func() {
		done <- wk.Stop(shutdownCtx)
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Errorf(ctx, "error stopping task worker: %v", err)
		} else {
			log.Info(ctx, "task worker stopped successfully")
		}
	case <-shutdownCtx.Done():
		log.Warn(ctx, "graceful shutdown timeout, forcing exit")
	}

	log.Info(ctx, "worker shutdown complete")
}

func init() {
	rootCmd.AddCommand(NewWorkerCmd())
}
