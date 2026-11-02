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

package lock

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// NewTenantRedisLock 创建租户内资源的锁，key 带 ctx 租户前缀（见 tenant.ScopedKey），
// ctx 中没有租户时返回错误。加锁与释放须使用同一租户的 ctx，否则释放的不是同一个 key。
//
// TODO(tenant-lock-compat): 当前为多租户改造的升级兼容实现，后续版本移除。
// 多租户改造前 key 不带租户前缀，滚动升级期间新旧版本实例并存，若新版本只使用带前缀的 key，
// 新旧实例互不感知，同一资源可被并发操作，升级前已持有的旧锁也拦不住新请求。
// 因此过渡期同时持有两个 key，按「旧 key -> 新 key」的固定顺序加锁：
//   - 新旧实例之间通过旧 key 互斥，升级前遗留的旧锁同样会拒绝新请求；
//   - 释放时两个 key 都删除。旧版本实例只会删除旧 key，残留的新 key 最多在锁超时后过期，
//     期间该资源会被多锁一段时间，但不会出现并发操作。
//
// 移除条件：所有旧版本实例下线，且已超过一个锁超时时间。移除时改为只使用 tenant.ScopedKey 生成的单个 key，
// 返回 NewRedisLock，并删除 MultiRedisLock（若已无其他使用方）。
func NewTenantRedisLock(ctx context.Context, key string, timeout int) (Locker, error) {
	scopedKey, err := tenant.ScopedKey(ctx, key)
	if err != nil {
		return nil, errors.Wrapf(err, "build tenant lock key %s", key)
	}
	return NewMultiRedisLock(timeout, key, scopedKey), nil
}
