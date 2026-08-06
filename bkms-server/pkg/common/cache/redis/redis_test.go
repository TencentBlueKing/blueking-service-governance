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

package redis

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmscache "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/cache"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
)

var _ = Describe("Cache", func() {
	var (
		ctx    context.Context
		cache  *Cache
		expiry = time.Minute
	)

	BeforeEach(func() {
		ctx = context.Background()
		redis.InitClientForTest()
		cache = NewCache("test-cache")
	})

	Describe("Get", func() {
		Context("when the key exists", func() {
			It("should return the value", func() {
				key := bkmscache.NewStringKey(stringx.Random(10))
				val := stringx.Random(10)
				err := cache.Set(ctx, key, val, expiry)
				Expect(err).NotTo(HaveOccurred())

				var result string
				err = cache.Get(ctx, key, &result)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(val))
			})
		})

		Context("when the key does not exist", func() {
			It("should return an error", func() {
				var result string
				key := bkmscache.NewStringKey("non-exist-key")
				err := cache.Get(ctx, key, &result)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Set", func() {
		It("should store the value with the specified key", func() {
			key := bkmscache.NewStringKey(stringx.Random(10))
			val := stringx.Random(10)

			err := cache.Set(ctx, key, "aba", expiry)
			Expect(err).NotTo(HaveOccurred())

			err = cache.Set(ctx, key, val, expiry)
			Expect(err).NotTo(HaveOccurred())

			var result string
			err = cache.Get(ctx, key, &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(val))
		})
	})

	Describe("Exists", func() {
		Context("when the key exists", func() {
			It("should return true", func() {
				key := bkmscache.NewStringKey(stringx.Random(10))
				val := stringx.Random(10)

				err := cache.Set(ctx, key, val, expiry)
				Expect(err).NotTo(HaveOccurred())

				exists := cache.Exists(ctx, key)
				Expect(exists).To(BeTrue())
			})
		})

		Context("when the key does not exist", func() {
			It("should return false", func() {
				key := bkmscache.NewStringKey("non-exist-key")
				exists := cache.Exists(ctx, key)
				Expect(exists).To(BeFalse())
			})
		})
	})

	Describe("Delete", func() {
		It("should remove the key from the cache", func() {
			key := bkmscache.NewStringKey(stringx.Random(10))
			val := stringx.Random(10)

			err := cache.Set(ctx, key, val, expiry)
			Expect(err).NotTo(HaveOccurred())

			err = cache.Delete(ctx, key)
			Expect(err).NotTo(HaveOccurred())

			exists := cache.Exists(ctx, key)
			Expect(exists).To(BeFalse())
		})
	})
})
