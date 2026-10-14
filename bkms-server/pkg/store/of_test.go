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

package store_test

import (
	"sync"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

type badTimestampModel struct {
	CreatedAt int64 `bson:"createdAt"`
}

var _ = Describe("Of", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	It("should return a store sharing data with the store provided through fx", func() {
		s := store.Of(f.db, widgetMeta)
		Expect(s.Collection().Name()).To(Equal(widgetCollection))

		Expect(s.Create(f.ctx, f.newSecretWidget("w1"))).To(Succeed())

		Expect(f.mustGetWidget("w1").Secret.Password).To(Equal(plainPassword))
		Expect(f.rawPassword("w1")).To(Equal(encPrefix + plainPassword))
	})

	It("should not share hooks between metas of the same model", func() {
		plainMeta := widgetMeta
		plainMeta.Encode, plainMeta.Decode = nil, nil
		// 先用带钩子的 Meta 构造一次，确保模型的反射分析结果已被缓存
		_ = store.Of(f.db, widgetMeta)

		Expect(store.Of(f.db, plainMeta).Create(f.ctx, f.newSecretWidget("w1"))).To(Succeed())
		Expect(f.rawPassword("w1")).To(Equal(plainPassword))
	})

	It("should panic on invalid meta", func() {
		Expect(func() { store.Of(f.db, store.Meta[badTimestampModel]{Collection: "store_test_bad"}) }).
			To(PanicWith(ContainSubstring("createdAt")))
	})
})

var _ = Describe("New", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	DescribeTable("should reject invalid meta",
		func(newStore func() error, want string) {
			Expect(newStore()).To(MatchError(ContainSubstring(want)))
		},
		Entry("empty collection", func() error {
			_, err := store.New(f.db, store.Meta[badTimestampModel]{})
			return err
		}, "collection"),
		Entry("non-struct model", func() error {
			_, err := store.New(f.db, store.Meta[string]{Collection: "store_test_str"})
			return err
		}, "struct"),
		Entry("timestamp field of wrong type", func() error {
			_, err := store.New(f.db, store.Meta[badTimestampModel]{Collection: "store_test_bad"})
			return err
		}, "createdAt"),
	)
})

var _ = Describe("Counter", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	It("should increase from 1 independently per key", func() {
		c := store.NewCounter(f.db, "store_test_counters")
		a, b := "a-"+stringx.Random(8), "b-"+stringx.Random(8)

		Expect(c.Next(f.ctx, a)).To(Equal(int64(1)))
		Expect(c.Next(f.ctx, a)).To(Equal(int64(2)))
		Expect(c.Next(f.ctx, b)).To(Equal(int64(1)))
	})

	It("should hand out unique values under concurrency", func() {
		c := store.NewCounter(f.db, "store_test_counters")
		key := "c-" + stringx.Random(8)

		const n = 20
		var (
			wg  sync.WaitGroup
			mu  sync.Mutex
			got []int64
		)
		for range n {
			wg.Go(func() {
				defer GinkgoRecover()
				v, err := c.Next(f.ctx, key)
				Expect(err).NotTo(HaveOccurred())
				mu.Lock()
				got = append(got, v)
				mu.Unlock()
			})
		}
		wg.Wait()
		Expect(lo.Uniq(got)).To(HaveLen(n))
		Expect(lo.Max(got)).To(Equal(int64(n)))
	})
})
