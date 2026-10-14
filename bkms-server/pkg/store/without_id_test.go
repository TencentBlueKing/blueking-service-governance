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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore with a model without _id", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	byKey := func(key string) store.Opt { return store.Where("key", key) }

	// rawID 读取库中文档由 MongoDB 生成的 _id
	rawID := func(key string) bson.ObjectID {
		raw, err := f.profiles.Collection().FindOne(f.ctx, bson.M{"key": key}).Raw()
		Expect(err).NotTo(HaveOccurred())
		return raw.Lookup("_id").ObjectID()
	}

	mustGet := func(key string) *Profile {
		p, err := f.profiles.Get(f.ctx, byKey(key))
		Expect(err).NotTo(HaveOccurred())
		return p
	}

	It("should create and read documents, leaving _id to MongoDB", func() {
		p := &Profile{Key: "p1", Title: "first"}
		Expect(f.profiles.Create(f.ctx, p)).To(Succeed())
		Expect(p.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
		Expect(p.UpdatedAt).To(Equal(p.CreatedAt))
		Expect(rawID("p1").IsZero()).To(BeFalse())

		Expect(mustGet("p1").Title).To(Equal("first"))
		list, err := f.profiles.List(f.ctx, store.All())
		Expect(err).NotTo(HaveOccurred())
		Expect(list).To(HaveLen(1))
	})

	It("should report a duplicate on the unique business key", func() {
		Expect(f.profiles.Create(f.ctx, &Profile{Key: "p1"})).To(Succeed())
		Expect(f.profiles.Create(f.ctx, &Profile{Key: "p1"})).To(MatchError(store.ErrDuplicate))
	})

	It("should update and replace by the business key without changing _id or createdAt", func() {
		created := &Profile{Key: "p1", Title: "first", CreatedAt: longAgo}
		Expect(f.profiles.Create(f.ctx, created)).To(Succeed())
		id := rawID("p1")

		Expect(f.profiles.Update(f.ctx, map[string]any{"title": "second"}, byKey("p1"))).To(Succeed())
		got := mustGet("p1")
		Expect(got.Title).To(Equal("second"))
		Expect(got.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))

		got.Title = "third"
		Expect(f.profiles.Replace(f.ctx, got, byKey("p1"))).To(Succeed())
		replaced := mustGet("p1")
		Expect(replaced.Title).To(Equal("third"))
		Expect(replaced.CreatedAt).To(beStoredTime(longAgo))
		Expect(rawID("p1")).To(Equal(id))
	})

	It("should upsert by the business key", func() {
		inserted, err := f.profiles.Upsert(f.ctx, &Profile{Key: "p1", Title: "first"}, byKey("p1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(inserted).To(BeTrue())
		createdAt := mustGet("p1").CreatedAt
		id := rawID("p1")

		inserted, err = f.profiles.Upsert(f.ctx, &Profile{Key: "p1", Title: "second"}, byKey("p1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(inserted).To(BeFalse())

		got := mustGet("p1")
		Expect(got.Title).To(Equal("second"))
		Expect(got.CreatedAt).To(Equal(createdAt))
		Expect(rawID("p1")).To(Equal(id))
		Expect(f.profiles.Count(f.ctx, store.All())).To(Equal(int64(1)))
	})

	It("should delete by the business key and report NotFound afterwards", func() {
		Expect(f.profiles.Create(f.ctx, &Profile{Key: "p1", Title: "first"})).To(Succeed())

		deleted, err := f.profiles.GetAndDelete(f.ctx, byKey("p1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(deleted.Title).To(Equal("first"))

		_, err = f.profiles.Get(f.ctx, byKey("p1"))
		Expect(err).To(MatchError(ErrProfileNotFound))
		Expect(f.profiles.Delete(f.ctx, byKey("p1"))).To(MatchError(store.ErrNotFound))
	})
})
