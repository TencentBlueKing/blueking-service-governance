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

// Package lock provide distributed lock implementation
package lock

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
)

// Locker 分布式锁，RedisLock 与 MultiRedisLock 均实现该接口
type Locker interface {
	Acquire(ctx context.Context) bool
	Release(ctx context.Context)
}

// RedisLock redis 锁，用于控制并发操作
// 目前基于最简单需求实现，仅支持加锁，释放锁的基础功能
// 暂不支持续期，检查所属，阻塞等待等高级功能，后续有需要再扩展
// 也可以直接改成使用开源库 github.com/go-redsync/redsync
type RedisLock struct {
	key     string        // 锁的 key
	timeout time.Duration // 超时时间，单位：秒
}

// NewRedisLock 创建 RedisLock 实例，key 推荐有一定的可读性，timeout 单位为秒
func NewRedisLock(key string, timeout int) *RedisLock {
	// 转换超时时间为 time.Duration 类型
	return &RedisLock{key: key, timeout: time.Duration(timeout) * time.Second}
}

// Acquire 尝试获取锁
func (l *RedisLock) Acquire(ctx context.Context) bool {
	// 使用 SET key value NX EX 命令尝试加锁，并设置超时
	args := goredis.SetArgs{Mode: "NX", TTL: l.timeout}
	result, err := redis.Client().SetArgs(ctx, l.key, "locked", args).Result()
	if err != nil {
		log.Errorf(ctx, "error occurred when acquire lock %s: %v", l.key, err)
		return false
	}
	return result == "OK"
}

// Release 释放锁
func (l *RedisLock) Release(ctx context.Context) {
	// 删除键以释放锁
	redis.Client().Del(ctx, l.key)
}

// MultiRedisLock 由多个 key 组成的锁：按顺序逐个获取，全部成功才算加锁成功。
// 每个 key 单独 SET NX，不使用 Lua 脚本，因此 Redis 集群模式下 key 分属不同 slot 也可用。
type MultiRedisLock struct {
	locks []*RedisLock
}

// NewMultiRedisLock 创建 MultiRedisLock 实例，keys 的顺序即加锁顺序，所有并发方须使用相同顺序
func NewMultiRedisLock(timeout int, keys ...string) *MultiRedisLock {
	locks := make([]*RedisLock, 0, len(keys))
	for _, key := range keys {
		locks = append(locks, NewRedisLock(key, timeout))
	}
	return &MultiRedisLock{locks: locks}
}

// Acquire 按顺序获取全部 key，任一失败时释放本次已获取的 key 并返回 false
func (l *MultiRedisLock) Acquire(ctx context.Context) bool {
	for i, lk := range l.locks {
		if lk.Acquire(ctx) {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			l.locks[j].Release(ctx)
		}
		return false
	}
	return true
}

// Release 按加锁的逆序释放全部 key
func (l *MultiRedisLock) Release(ctx context.Context) {
	for i := len(l.locks) - 1; i >= 0; i-- {
		l.locks[i].Release(ctx)
	}
}
