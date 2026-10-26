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

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
)

// releaseLockScript 只删除本实例持有的锁。锁过期后被别人重新拿到时，无条件 DEL 会把对方的锁删掉
var releaseLockScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

// RedisLock redis 锁，用于控制并发操作
// 加锁时写入随机 token，释放时只删除自己持有的 key
// 暂不支持续期与阻塞等待
type RedisLock struct {
	key     string        // 锁的 key
	token   string        // 持有者标识，释放时用来确认锁仍属于自己
	timeout time.Duration // 超时时间
}

// NewRedisLock 创建 RedisLock 实例，key 推荐有一定的可读性，timeout 单位为秒
func NewRedisLock(key string, timeout int) *RedisLock {
	return &RedisLock{
		key:     key,
		token:   uuid.NewString(),
		timeout: time.Duration(timeout) * time.Second,
	}
}

// Acquire 尝试获取锁
func (l *RedisLock) Acquire(ctx context.Context) bool {
	// 使用 SET key token NX EX 命令尝试加锁，并设置超时
	args := goredis.SetArgs{Mode: "NX", TTL: l.timeout}
	result, err := redis.Client().SetArgs(ctx, l.key, l.token, args).Result()
	if err != nil {
		log.Errorf(ctx, "error occurred when acquire lock %s: %v", l.key, err)
		return false
	}
	return result == "OK"
}

// Held 锁的当前值是否仍是本实例加锁时写入的 token
func (l *RedisLock) Held(ctx context.Context) bool {
	val, err := redis.Client().Get(ctx, l.key).Result()
	if err != nil {
		return false
	}
	return val == l.token
}

// Release 释放锁；key 已过期或已被其他持有者重新获取时不做删除
func (l *RedisLock) Release(ctx context.Context) {
	if err := releaseLockScript.Run(ctx, redis.Client(), []string{l.key}, l.token).Err(); err != nil {
		log.Errorf(ctx, "release lock %s: %v", l.key, err)
	}
}
