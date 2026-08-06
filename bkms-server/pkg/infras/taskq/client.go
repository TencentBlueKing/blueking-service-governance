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
	"net"
	"sync"

	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

const (
	// defaultQueue 默认队列名(任务与投递均未指定 asynq.Queue 时使用)。
	defaultQueue = "default"
	// defaultMaxRetry 默认最大重试次数(任务与投递均未指定 asynq.MaxRetry 且配置未给时使用)。
	defaultMaxRetry = 10
	// defaultRetryInterval ErrFixedRetry 的默认固定间隔(秒)。
	defaultRetryInterval = 5
)

var (
	// client 投递端单例。投递进程(webserver)与消费进程(worker)均在启动时 Init。
	client     *asynq.Client
	clientOnce sync.Once
	// globalDefaults 在 InitClient 时从配置读取, 作为任务未覆盖时的兜底。
	globalDefaults struct {
		queue    string
		maxRetry int
	}
)

// InitClient 初始化投递端 client 单例。多次调用仅首次生效。
// cfg.Redis.Host 必须非空，否则 panic（配置缺失属于编程 / 部署错误，不应静默忽略）。
func InitClient(ctx context.Context, cfg config.AsynqConfig) {
	if cfg.Redis.Host == "" {
		panic("taskq.InitClient: asynq.redis.host is required — " +
			"please configure the `asynq.redis` section in config")
	}
	clientOnce.Do(func() {
		client = asynq.NewClient(redisConnOpt(cfg.Redis))
		globalDefaults.queue = defaultQueue
		if cfg.Queue != "" {
			globalDefaults.queue = cfg.Queue
		}
		globalDefaults.maxRetry = defaultMaxRetry
		if cfg.MaxRetry > 0 {
			globalDefaults.maxRetry = cfg.MaxRetry
		}
		logging.Infof(ctx, "taskq client initialized, defaultQueue=%s defaultMaxRetry=%d",
			globalDefaults.queue, globalDefaults.maxRetry)
	})
}

// CloseClient 关闭投递端 client(优雅退出时调用)。
func CloseClient(ctx context.Context) {
	if client != nil {
		if err := client.Close(); err != nil {
			logging.Errorf(ctx, "taskq close client: %v", err)
		}
	}
}

// Task 是一次可投递的任务实例, 由 TaskType.NewTask 产出。
//
// 它对应 asynq 概念中的一次具体任务(asynq.Task),
// 携带已序列化的 payload 和来自 TaskType 的默认投递选项。
// 通过包级 Enqueue 函数投递。
type Task struct {
	name        string
	payload     []byte
	defaultOpts []asynq.Option
}

// Enqueue 投递一个 Task 实例(由 TaskType.NewTask 产出)。
//
// 投递选项按 asynq"后者覆盖前者"的语义分三层拼接:
//  1. 全局兜底(队列、最大重试, 来自配置) —— 最前, 仅在业务未指定时生效;
//  2. Task 自带默认(来自 TaskType 定义时传入的 asynq.Option);
//  3. 本次调用 opts —— 最后, 覆盖以上。
//
// 去重冲突(同名 + 同负载在途)视为正常, 返回 nil。
func Enqueue(ctx context.Context, t *Task, opts ...asynq.Option) error {
	if t == nil {
		return errors.New("taskq: task is nil (likely due to args serialization failure)")
	}
	if client == nil {
		return errors.New("taskq client not initialized")
	}

	enqOpts := make([]asynq.Option, 0, len(t.defaultOpts)+len(opts)+2)
	enqOpts = append(enqOpts,
		asynq.Queue(globalDefaults.queue),
		asynq.MaxRetry(globalDefaults.maxRetry),
	)
	enqOpts = append(enqOpts, t.defaultOpts...)
	enqOpts = append(enqOpts, opts...)

	task := asynq.NewTask(t.name, t.payload)
	_, err := client.EnqueueContext(ctx, task, enqOpts...)
	if errors.Is(err, asynq.ErrDuplicateTask) {
		logging.Infof(ctx, "taskq enqueue skipped, duplicate in-flight task: %s", t.name)
		return nil
	}
	return err
}

// redisConnOpt 从 taskq 独立的 Redis 配置构造底层连接选项。
func redisConnOpt(cfg config.RedisConfig) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     net.JoinHostPort(cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	}
}
