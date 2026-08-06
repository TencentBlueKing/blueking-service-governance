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

// Package redis provide redis cache implementation
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"

	bkmscache "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/cache"
	bkmsredis "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
)

// CacheKeyPrefix 模块名对应的 Cache Key 前缀
const CacheKeyPrefix = "bkms-srv"

// Cache 缓存实例
type Cache struct {
	name      string        // 缓存名称，用于区分不同的缓存资源
	keyPrefix string        // 缓存键前缀，用于区分不同业务模块
	cli       *redis.Client // redis client
	codec     *cache.Cache  // go-redis cache
}

// NewCache 新建 cache 实例
func NewCache(name string) *Cache {
	cli := bkmsredis.Client()
	// key: {cache_key_prefix}:{cache_name}:{raw_key}
	keyPrefix := fmt.Sprintf("%s:%s", CacheKeyPrefix, name)
	codec := cache.New(&cache.Options{Redis: cli})

	return &Cache{
		name:      name,
		keyPrefix: keyPrefix,
		cli:       cli,
		codec:     codec,
	}
}

// Get 从 redis 中获取值，并存储到 value 中，如果获取不到，返回 error
func (c *Cache) Get(ctx context.Context, key bkmscache.Key, value any) error {
	k := c.genKey(key.Key())
	return c.codec.Get(ctx, k, value)
}

// Set 将 value 存储到 redis 中（键为 key 值），若 duration 为 0，则使用默认值（Cache.exp）
func (c *Cache) Set(ctx context.Context, key bkmscache.Key, value any, duration time.Duration) error {
	k := c.genKey(key.Key())
	return c.codec.Set(&cache.Item{
		Ctx:   ctx,
		Key:   k,
		Value: value,
		TTL:   duration,
	})
}

// Exists 检查 key 在 redis 中是否存在
func (c *Cache) Exists(ctx context.Context, key bkmscache.Key) bool {
	k := c.genKey(key.Key())
	return c.codec.Exists(ctx, k)
}

// Delete 从 redis 中删除指定的键
func (c *Cache) Delete(ctx context.Context, key bkmscache.Key) error {
	k := c.genKey(key.Key())
	return c.codec.Delete(ctx, k)
}

func (c *Cache) genKey(key string) string {
	return c.keyPrefix + ":" + key
}
