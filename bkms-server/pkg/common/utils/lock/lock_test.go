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

package lock_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/lock"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
)

var _ = Describe("RedisLock", func() {
	var (
		ctx     context.Context
		key     string
		rdsLock *lock.RedisLock
	)

	BeforeEach(func() {
		redis.InitClientForTest()

		ctx = context.Background()
		key = "test-lock" + stringx.Random(8)
		rdsLock = lock.NewRedisLock(key, 10)
	})

	Describe("Acquire", func() {
		It("should acquire the lock successfully", func() {
			ok := rdsLock.Acquire(ctx)
			Expect(ok).To(BeTrue())
		})

		It("should fail to acquire the lock if it is already held", func() {
			ok := rdsLock.Acquire(ctx)
			Expect(ok).To(BeTrue())

			ok = rdsLock.Acquire(ctx)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Release", func() {
		It("should release the lock successfully", func() {
			ok := rdsLock.Acquire(ctx)
			Expect(ok).To(BeTrue())

			rdsLock.Release(ctx)
			ok = rdsLock.Acquire(ctx)
			Expect(ok).To(BeTrue())
		})

		It("should not delete a lock re-acquired by someone else", func() {
			Expect(rdsLock.Acquire(ctx)).To(BeTrue())
			// 模拟本实例的锁已过期，另一个持有者重新拿到了同一把锁
			Expect(redis.Client().Del(ctx, key).Err()).NotTo(HaveOccurred())
			other := lock.NewRedisLock(key, 10)
			Expect(other.Acquire(ctx)).To(BeTrue())

			rdsLock.Release(ctx)

			third := lock.NewRedisLock(key, 10)
			Expect(third.Acquire(ctx)).To(BeFalse())
			other.Release(ctx)
			Expect(third.Acquire(ctx)).To(BeTrue())
		})
	})
})
