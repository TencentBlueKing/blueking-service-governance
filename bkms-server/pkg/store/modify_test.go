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

var _ = Describe("GenericStore Modify", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	// mustModify 修改当前 owner 下名为 b1 的 Board 并返回结果
	mustModify := func(update bson.M, opts ...store.Opt) store.ModifyResult {
		res, err := f.boards.Modify(f.ctx, update, append(opts, f.byName("b1"))...)
		Expect(err).NotTo(HaveOccurred())
		return res
	}

	BeforeEach(func() { f.mustCreateBoard("b1") })

	Describe("array and counter operators", func() {
		It("should push an element and refresh updatedAt", func() {
			res := mustModify(bson.M{"$push": bson.M{"cards": Card{Name: "c2", Title: "second"}}})
			Expect(res).To(Equal(store.ModifyResult{Matched: 1, Modified: 1}))

			got := f.mustGetBoard("b1")
			Expect(got.Cards).To(Equal([]Card{{Name: "c1", Title: "first"}, {Name: "c2", Title: "second"}}))
			Expect(got.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(got.CreatedAt).To(beStoredTime(longAgo))
		})

		It("should report no modification when $addToSet finds the element with KeepUpdatedAt", func() {
			res := mustModify(
				bson.M{"$addToSet": bson.M{"cards": Card{Name: "c1", Title: "first"}}},
				store.KeepUpdatedAt(),
			)
			Expect(res).To(Equal(store.ModifyResult{Matched: 1, Modified: 0}))
			Expect(f.mustGetBoard("b1").UpdatedAt).To(beStoredTime(longAgo))
		})

		It("should pull matching elements", func() {
			res := mustModify(bson.M{"$pull": bson.M{"cards": bson.M{"name": "c1"}}}, store.KeepUpdatedAt())
			Expect(res.Modified).To(Equal(int64(1)))
			Expect(f.mustGetBoard("b1").Cards).To(BeEmpty())

			res = mustModify(bson.M{"$pull": bson.M{"cards": bson.M{"name": "c1"}}}, store.KeepUpdatedAt())
			Expect(res.Modified).To(BeZero())
		})

		It("should increase a counter given by its Go field name", func() {
			mustModify(bson.M{"$inc": bson.M{"Hits": 2}})
			mustModify(bson.M{"$inc": bson.M{"hits": 3}})
			Expect(f.mustGetBoard("b1").Hits).To(Equal(int64(5)))
		})

		It("should combine several operators in one call", func() {
			mustModify(bson.M{
				"$inc":  bson.M{"hits": 1},
				"$push": bson.M{"cards": Card{Name: "c2"}},
			})
			got := f.mustGetBoard("b1")
			Expect(got.Hits).To(Equal(int64(1)))
			Expect(got.Cards).To(HaveLen(2))
		})
	})

	Describe("positional array paths", func() {
		It("should set a field of the elements matched by array filters", func() {
			mustModify(bson.M{"$push": bson.M{"cards": Card{Name: "c2", Title: "second"}}})

			mustModify(
				bson.M{"$set": bson.M{"Cards.$[c].Title": "renamed"}},
				store.ArrayFilters(bson.M{"c.name": "c2"}),
			)

			got := f.mustGetBoard("b1")
			Expect(got.Cards).To(Equal([]Card{{Name: "c1", Title: "first"}, {Name: "c2", Title: "renamed"}}))
			Expect(got.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should set the element matched by the filter with the $ operator", func() {
			mustModify(bson.M{"$set": bson.M{"cards.$.title": "matched"}}, store.Where("cards.name", "c1"))
			Expect(f.mustGetBoard("b1").Cards[0].Title).To(Equal("matched"))
		})

		It("should unset a field of all elements", func() {
			mustModify(bson.M{"$unset": bson.M{"cards.$[].title": ""}})
			raw, err := f.boards.Collection().FindOne(f.ctx, bson.M{"ownerID": f.owner, "name": "b1"}).Raw()
			Expect(err).NotTo(HaveOccurred())
			Expect(rawHas(raw, "cards", "0", "title")).To(BeFalse())
		})
	})

	It("should return NotFound when nothing matches", func() {
		_, err := f.boards.Modify(f.ctx, bson.M{"$inc": bson.M{"hits": 1}}, f.byName("missing"))
		Expect(err).To(MatchError(store.ErrNotFound))
	})

	DescribeTable("should reject invalid updates without writing",
		func(update bson.M, msg string) {
			_, err := f.boards.Modify(f.ctx, update, f.byName("b1"))
			Expect(err).To(MatchError(ContainSubstring(msg)))
			Expect(f.mustGetBoard("b1").UpdatedAt).To(beStoredTime(longAgo))
		},
		Entry("empty update", bson.M{}, store.ErrNoFieldsToUpdate.Error()),
		Entry("plain field", bson.M{"name": "x"}, "only accepts update operators"),
		Entry("$set of a plain field", bson.M{"$set": bson.M{"name": "x"}}, "use Update"),
		Entry("$unset of a plain field", bson.M{"$unset": bson.M{"states": ""}}, "use Update"),
		Entry("$setOnInsert", bson.M{"$setOnInsert": bson.M{"name": "x"}}, "not supported by Modify"),
		Entry("$rename", bson.M{"$rename": bson.M{"name": "title"}}, "not supported by Modify"),
		Entry("unknown field", bson.M{"$inc": bson.M{"missing": 1}}, "unknown field"),
		Entry("_id", bson.M{"$inc": bson.M{"_id": 1}}, "cannot be updated"),
		Entry("value that is not a document", bson.M{"$inc": 1}, "non-empty document"),
	)

	It("should require a filter", func() {
		_, err := f.boards.Modify(f.ctx, bson.M{"$inc": bson.M{"hits": 1}})
		Expect(err).To(MatchError(ContainSubstring("filter is required")))
	})

	It("should not accept ArrayFilters in Update", func() {
		err := f.boards.Update(f.ctx, bson.M{"hits": 1}, f.byName("b1"), store.ArrayFilters(bson.M{"c.name": "x"}))
		Expect(err).To(MatchError(ContainSubstring("array filters")))
	})

	Describe("ModifyMany", func() {
		It("should modify all matching documents", func() {
			f.mustCreateBoard("b2")

			res, err := f.boards.ModifyMany(f.ctx, bson.M{"$pull": bson.M{"cards": bson.M{"name": "c1"}}}, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(store.ModifyResult{Matched: 2, Modified: 2}))
			Expect(f.mustGetBoard("b1").Cards).To(BeEmpty())
			Expect(f.mustGetBoard("b2").Cards).To(BeEmpty())
		})

		It("should not return an error when nothing matches", func() {
			res, err := f.boards.ModifyMany(f.ctx, bson.M{"$inc": bson.M{"hits": 1}}, f.byName("missing"))
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(store.ModifyResult{}))
		})
	})
})
