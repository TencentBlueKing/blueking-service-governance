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
	stderrors "errors"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/docs/apis" // register swagger docs
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

// apmShutdownTimeout 限制进程退出时等待 APM tracer provider 优雅关闭的最长时间
//
// 使用独立超时的 context 触发 shutdown，避免复用已经 Done 的 cmd.Context() 导致 span 未刷完就返回
const apmShutdownTimeout = 5 * time.Second

// NewWebServerCmd ...
func NewWebServerCmd() *cobra.Command {
	var srvCfg string

	wsCmd := cobra.Command{
		Use:   "webserver",
		Short: "Start the Gin HTTP server.",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			// 初始化北极星 SDK 客户端
			polarisInfra.MustInitClient(ctx, cfg.Polaris.JoinPoint, cfg.Polaris.Address)
			// 初始化所有 store 实例（必须在 database 初始化之后）
			storereg.Init(ctx)
			reg := storereg.G()
			router := server.RegisterRouter(ctx, *cfg, cmd.Name())
			workload.InitPlugin(
				reg.AppConfigFileStore,
				reg.PolarisConfigStore,
			)

			// 加载流水线模板
			// FIXME 建议该 Reload 逻辑放在 migration 中完成
			//
			// 使用 Errorf + return 而不是 log.Fatalf → os.Exit，确保上方注册的 shutdownAPM defer 能够被执行，
			// 从而把已缓冲的 span 完整刷到蓝鲸监控采集端
			if err = bkci.ReloadPipelineTemplates(cmd.Context()); err != nil {
				log.Errorf(ctx, "failed to reload pipeline templates: %v", err)
				return err
			}

			// 加载内置集群 Addon 定义
			//
			// 与 ReloadPipelineTemplates 保持一致，避免绕过 shutdownAPM defer
			if err = clusteraddon.ReloadBuiltinClusterAddons(cmd.Context()); err != nil {
				log.Errorf(ctx, "failed to reload builtin cluster addons: %v", err)
				return err
			}

			// 启动阶段主动初始化权限管理器，提前暴露 IAM client、角色存储等构造问题，避免延迟到请求首次鉴权时才失败。
			_ = perm.NewManager()

			return serveHTTP(ctx, cfg.HTTPServer, router)
		},
	}

	// 配置文件路径
	wsCmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")

	return &wsCmd
}

// serveHTTP 提供 HTTP 服务
func serveHTTP(ctx context.Context, cfg config.HTTPServerConfig, handler http.Handler) error {
	addr := net.JoinHostPort(cfg.Address, strconv.Itoa(int(cfg.Port))) //nolint:gosec // G115: integer overflow
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
		ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Infof(ctx, "http server starting on %s", addr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "serve http server")
		}
		return nil
	case <-ctx.Done():
		log.Info(ctx, "http server received shutdown signal, shutting down...")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), time.Duration(cfg.ShutdownTimeout)*time.Second,
	)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		shutdownErr := errors.Wrap(err, "shutdown http server")
		select {
		case serveErr := <-errCh:
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				return stderrors.Join(shutdownErr, errors.Wrap(serveErr, "serve http server"))
			}
		default:
		}
		return shutdownErr
	}

	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "serve http server")
	}

	log.Info(ctx, "http server stopped")
	return nil
}

func init() {
	rootCmd.AddCommand(NewWebServerCmd())
}
