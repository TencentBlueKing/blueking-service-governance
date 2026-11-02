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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
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
	})
})

var _ = Describe("MultiRedisLock", func() {
	var (
		ctx        context.Context
		keyA, keyB string
	)

	exists := func(key string) bool {
		n, err := redis.Client().Exists(ctx, key).Result()
		Expect(err).NotTo(HaveOccurred())
		return n == 1
	}

	BeforeEach(func() {
		redis.InitClientForTest()
		ctx = context.Background()
		keyA = "test-multi-lock-a" + stringx.Random(8)
		keyB = "test-multi-lock-b" + stringx.Random(8)
		DeferCleanup(func() { redis.Client().Del(ctx, keyA, keyB) })
	})

	It("should hold all keys after acquiring and release all of them", func() {
		multiLock := lock.NewMultiRedisLock(10, keyA, keyB)
		Expect(multiLock.Acquire(ctx)).To(BeTrue())
		Expect(exists(keyA)).To(BeTrue())
		Expect(exists(keyB)).To(BeTrue())
		Expect(multiLock.Acquire(ctx)).To(BeFalse())

		multiLock.Release(ctx)
		Expect(exists(keyA)).To(BeFalse())
		Expect(exists(keyB)).To(BeFalse())
	})

	It("should roll back acquired keys when a later key is held", func() {
		Expect(lock.NewRedisLock(keyB, 10).Acquire(ctx)).To(BeTrue())

		Expect(lock.NewMultiRedisLock(10, keyA, keyB).Acquire(ctx)).To(BeFalse())
		Expect(exists(keyA)).To(BeFalse())
	})
})

var _ = Describe("NewTenantRedisLock", func() {
	var (
		ctx       context.Context
		key       string
		scopedKey string
	)

	exists := func(k string) bool {
		n, err := redis.Client().Exists(ctx, k).Result()
		Expect(err).NotTo(HaveOccurred())
		return n == 1
	}

	BeforeEach(func() {
		redis.InitClientForTest()
		ctx = tenant.WithTenantID(context.Background(), "tenant-a")
		key = "lock:test-tenant-lock:" + stringx.Random(8)
		scopedKey = "tenant:tenant-a:" + key
		DeferCleanup(func() { redis.Client().Del(context.Background(), key, scopedKey) })
	})

	It("should hold both the legacy key and the tenant scoped key", func() {
		tenantLock, err := lock.NewTenantRedisLock(ctx, key, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(tenantLock.Acquire(ctx)).To(BeTrue())
		Expect(exists(key)).To(BeTrue())
		Expect(exists(scopedKey)).To(BeTrue())

		tenantLock.Release(ctx)
		Expect(exists(key)).To(BeFalse())
		Expect(exists(scopedKey)).To(BeFalse())
	})

	It("should be rejected by a legacy lock held by an old version instance", func() {
		Expect(lock.NewRedisLock(key, 10).Acquire(ctx)).To(BeTrue())

		tenantLock, err := lock.NewTenantRedisLock(ctx, key, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(tenantLock.Acquire(ctx)).To(BeFalse())
		Expect(exists(scopedKey)).To(BeFalse())
	})

	It("should block an old version instance that only uses the legacy key", func() {
		tenantLock, err := lock.NewTenantRedisLock(ctx, key, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(tenantLock.Acquire(ctx)).To(BeTrue())

		Expect(lock.NewRedisLock(key, 10).Acquire(ctx)).To(BeFalse())
	})

	It("should release the legacy key when the tenant scoped key is held", func() {
		Expect(lock.NewRedisLock(scopedKey, 10).Acquire(ctx)).To(BeTrue())

		tenantLock, err := lock.NewTenantRedisLock(ctx, key, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(tenantLock.Acquire(ctx)).To(BeFalse())
		Expect(exists(key)).To(BeFalse())
	})

	It("should return an error when ctx has no tenant", func() {
		_, err := lock.NewTenantRedisLock(context.Background(), key, 10)
		Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
	})
})
