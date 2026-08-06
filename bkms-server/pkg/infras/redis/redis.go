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

// Package redis 提供 redis 客户端实现
package redis

import (
	"context"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

var (
	// global redis client instance
	rds      *redis.Client
	initOnce sync.Once
)

const (
	// dialTimeout unit: s
	dialTimeout = 5
	// readTimeout unit: s
	readTimeout = 2
	// writeTimeout unit: s
	writeTimeout = 2
	// poolSizeMultiple * NumCPU
	poolSizeMultiple = 20
	// minIdleConnsMultiple * NumCPU
	minIdleConnsMultiple = 10
	// connMaxIdleTime unit: s
	connMaxIdleTime = 3 * 60
)

// Client 获取默认 Redis 客户端
func Client() *redis.Client {
	if rds == nil {
		log.Fatal("redis client not initialized")
	}
	return rds
}

// InitClient 初始化
func InitClient(ctx context.Context, cfg config.RedisConfig) {
	if rds != nil {
		return
	}
	initOnce.Do(func() {
		rds = newStandaloneClient(cfg)
		if _, err := rds.Ping(ctx).Result(); err != nil {
			log.Fatalf("failed to ping redis: %v", err)
		} else {
			log.Infof(ctx, "redis: %s/%d connected", net.JoinHostPort(cfg.Host, cfg.Port), cfg.DB)
		}
		// 自动为所有 Redis 操作生成 OTel 子 Span
		if err := redisotel.InstrumentTracing(rds); err != nil {
			log.Warnf(ctx, "failed to instrument redis tracing: %v", err)
		}
	})
}

// InitClientForTest 初始化单元测试用的 Redis 客户端
func InitClientForTest() {
	if rds != nil {
		return
	}
	initOnce.Do(func() {
		miniRds, err := miniredis.Run()
		if err != nil {
			log.Fatalf("failed to init mini redis: %v", err)
		}

		rds = redis.NewClient(&redis.Options{Addr: miniRds.Addr()})
	})
}

func newStandaloneClient(cfg config.RedisConfig) *redis.Client {
	opt := &redis.Options{
		Addr:     net.JoinHostPort(cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	// 默认配置
	opt.DialTimeout = time.Duration(dialTimeout) * time.Second
	opt.ReadTimeout = time.Duration(readTimeout) * time.Second
	opt.WriteTimeout = time.Duration(writeTimeout) * time.Second
	opt.PoolSize = poolSizeMultiple * runtime.NumCPU()
	opt.MinIdleConns = minIdleConnsMultiple * runtime.NumCPU()
	opt.ConnMaxIdleTime = time.Duration(connMaxIdleTime) * time.Second

	// 若指定配置中指定，则使用
	if cfg.DialTimeout > 0 {
		opt.DialTimeout = time.Duration(cfg.DialTimeout) * time.Second
	}
	if cfg.ReadTimeout > 0 {
		opt.ReadTimeout = time.Duration(cfg.ReadTimeout) * time.Second
	}
	if cfg.WriteTimeout > 0 {
		opt.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Second
	}

	if cfg.PoolSize > 0 {
		opt.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		opt.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.ConnMaxIdleTime > 0 {
		opt.ConnMaxIdleTime = time.Duration(cfg.ConnMaxIdleTime) * time.Second
	}

	return redis.NewClient(opt)
}
