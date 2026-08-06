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
	"errors"
	"time"

	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// Server 异步任务的消费者 Server，底层封装 asynq
type Server struct {
	inner *asynq.Server
	mux   *asynq.ServeMux
}

// NewServer 构造消费端 server。
//
//   - cfg  asynq 完整配置(含 Redis 连接 + 行为参数), cfg.Redis.Host 必须非空。
//   - mux  业务侧构建并注册各任务 handler 的路由(mux.Handle(task.Name(), task.Handler()))。
func NewServer(
	ctx context.Context, cfg config.AsynqConfig, mux *asynq.ServeMux,
) Server {
	if cfg.Redis.Host == "" {
		panic("taskq.NewServer: asynq.redis.host is required — " +
			"please configure the `asynq.redis` section in config")
	}
	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 10
		log.Infof(ctx, "taskq server concurrency is not positive, set to 10")
	}

	interval := time.Duration(defaultRetryInterval) * time.Second
	if cfg.RetryInterval > 0 {
		interval = time.Duration(cfg.RetryInterval) * time.Second
		log.Infof(ctx, "taskq server retry interval is %d, set to %d", cfg.RetryInterval, interval)
	}

	queue := cfg.Queue
	if queue == "" {
		queue = defaultQueue
		log.Infof(ctx, "taskq server queue is empty, set to %s", defaultQueue)
	}

	inner := asynq.NewServer(redisConnOpt(cfg.Redis), asynq.Config{
		Concurrency:    concurrency,
		Queues:         map[string]int{queue: 1},
		RetryDelayFunc: retryDelayFunc(interval),
		ErrorHandler:   errorHandler(),
	})
	return Server{inner: inner, mux: mux}
}

// Start 启动消费(非阻塞)。
func (s Server) Start() error {
	log.InfoNoContext("taskq server starting")
	return s.inner.Start(s.mux)
}

// Shutdown 优雅停止消费(阻塞直至在途任务完成或内部超时)。
func (s Server) Shutdown() {
	s.inner.Shutdown()
}

// retryDelayFunc 决定重试延迟: 错误链含 ErrFixedRetry 时用固定间隔, 否则默认退避。
func retryDelayFunc(fixed time.Duration) asynq.RetryDelayFunc {
	return func(n int, e error, t *asynq.Task) time.Duration {
		if errors.Is(e, ErrFixedRetry) {
			return fixed
		}
		return asynq.DefaultRetryDelayFunc(n, e, t)
	}
}

// errorHandler 构造 server 级错误处理器: 在重试耗尽(非 StopRetry)时记录错误日志。
func errorHandler() asynq.ErrorHandler {
	return asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
		if errors.Is(err, asynq.SkipRetry) {
			// 业务主动停止重试的终态已在 handler 内处理, 不重复记录。
			return
		}
		retried, ok1 := asynq.GetRetryCount(ctx)
		maxRetry, ok2 := asynq.GetMaxRetry(ctx)
		if !ok1 || !ok2 || retried < maxRetry {
			return // 仍有重试机会。
		}
		log.Errorf(ctx, "taskq task exhausted after %d retries: type=%s err=%v", retried, t.Type(), err)
	})
}

// wrapStopRetry 包装一个错误使底层停止重试, 同时保留原始错误信息。
func wrapStopRetry(err error) error {
	return errors.Join(err, asynq.SkipRetry)
}
