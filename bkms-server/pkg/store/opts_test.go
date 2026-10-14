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

package store

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ = Describe("Options", func() {
	apply := func(opts ...Opt) *query {
		q := &query{}
		for _, opt := range opts {
			opt.apply(q)
		}
		return q
	}

	Describe("query.filter", func() {
		It("should return an empty document without conditions", func() {
			Expect(apply().filter()).To(Equal(bson.D{}))
		})

		It("should keep conditions on distinct fields as they are", func() {
			q := apply(Where("a", 1), Filter(bson.M{"c": 3, "b": 2}))
			Expect(q.filter()).To(Equal(bson.D{{Key: "a", Value: 1}, {Key: "b", Value: 2}, {Key: "c", Value: 3}}))
		})

		It("should wrap all conditions in $and when a field appears more than once", func() {
			q := apply(Where("a", bson.M{"$gte": 1}), Where("b", 2), Where("a", bson.M{"$lte": 3}))
			Expect(q.filter()).To(Equal(bson.D{{Key: "$and", Value: []bson.D{
				{{Key: "a", Value: bson.M{"$gte": 1}}},
				{{Key: "b", Value: 2}},
				{{Key: "a", Value: bson.M{"$lte": 3}}},
			}}}))
		})
	})

	Describe("ContainsIgnoreCase", func() {
		match := bson.M{"$regex": `a\.b`, "$options": "i"}

		It("should build a single field condition with an escaped pattern", func() {
			Expect(apply(ContainsIgnoreCase(" a.b ", "name")).conds).To(Equal(bson.D{{Key: "name", Value: match}}))
		})

		It("should join multiple fields with $or", func() {
			q := apply(ContainsIgnoreCase("a.b", "name", "desc"))
			Expect(q.conds).To(Equal(bson.D{{Key: "$or", Value: []bson.D{
				{{Key: "name", Value: match}},
				{{Key: "desc", Value: match}},
			}}}))
		})

		It("should build no condition when the keyword is blank", func() {
			Expect(apply(ContainsIgnoreCase(" ", "name", "desc")).conds).To(BeEmpty())
		})
	})

	Describe("Combine", func() {
		It("should report the union of the kinds of its options and skip nil ones", func() {
			opt := Combine(Where("a", 1), nil, Asc("a"), Combine(Limit(1)))
			Expect(opt.kind()).To(Equal(kindCond | kindSort | kindLimit))
			Expect(opt.kind().String()).To(Equal("condition, sort, limit"))
		})

		It("should apply its options in order", func() {
			q := apply(Combine(Where("a", 1), All(), Desc("b"), Limit(5), UpdateOmit("c"), Combine(Asc("d"))))
			Expect(q.conds).To(Equal(bson.D{{Key: "a", Value: 1}}))
			Expect(q.matchAll).To(BeTrue())
			Expect(q.sort).To(Equal(bson.D{{Key: "b", Value: -1}, {Key: "d", Value: 1}}))
			Expect(q.limit).To(Equal(int64(5)))
			Expect(q.fields).To(Equal(&fieldSelection{omitted: []string{"c"}}))
		})
	})

	It("should merge OmitFields and WithoutTotal", func() {
		q := apply(OmitFields("a"), Combine(OmitFields("b", "c"), WithoutTotal()))
		Expect(q.omitted).To(Equal([]string{"a", "b", "c"}))
		Expect(q.withoutTotal).To(BeTrue())
		Expect(q.projection()).To(Equal(bson.D{{Key: "a", Value: 0}, {Key: "b", Value: 0}, {Key: "c", Value: 0}}))
		Expect(apply().projection()).To(BeNil())
	})

	It("should merge SelectFields into an inclusion projection", func() {
		q := apply(SelectFields("a"), Combine(SelectFields("b", "c")))
		Expect(q.selected).To(Equal([]string{"a", "b", "c"}))
		Expect(q.projection()).To(Equal(bson.D{{Key: "a", Value: 1}, {Key: "b", Value: 1}, {Key: "c", Value: 1}}))
	})

	It("should merge multiple UpdateOpts", func() {
		q := apply(UpdateSelected("a"), UpdateOmit("b"), UpdateSelected("c"))
		Expect(q.fields).To(Equal(&fieldSelection{
			selecting: true, selected: []string{"a", "c"}, omitted: []string{"b"},
		}))
	})
})
