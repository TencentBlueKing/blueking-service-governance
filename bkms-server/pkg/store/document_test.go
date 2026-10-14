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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ = Describe("Mongo document utilities", func() {
	DescribeTable("toDoc",
		func(v any, expected bson.D, ok bool) {
			doc, isDoc := toDoc(v)
			Expect(isDoc).To(Equal(ok))
			Expect(doc).To(Equal(expected))
		},
		Entry(
			"keeps bson.D as is",
			bson.D{
				{Key: "b", Value: 1},
				{Key: "a", Value: 2},
			},
			bson.D{{Key: "b", Value: 1}, {Key: "a", Value: 2}},
			true,
		),
		Entry("sorts bson.M by key",
			bson.M{"b": 1, "a": 2}, bson.D{{Key: "a", Value: 2}, {Key: "b", Value: 1}}, true,
		),
		Entry("sorts map[string]any by key",
			map[string]any{"b": 1, "a": 2}, bson.D{{Key: "a", Value: 2}, {Key: "b", Value: 1}}, true,
		),
		Entry("rejects a scalar", "a", bson.D(nil), false),
		Entry("rejects an array", bson.A{bson.M{"a": 1}}, bson.D(nil), false),
	)

	DescribeTable("toArray",
		func(v any, expected []any, ok bool) {
			items, isArray := toArray(v)
			Expect(isArray).To(Equal(ok))
			Expect(items).To(Equal(expected))
		},
		Entry("accepts bson.A", bson.A{1, "a"}, []any{1, "a"}, true),
		Entry("accepts typed slices", []bson.M{{"a": 1}}, []any{bson.M{"a": 1}}, true),
		Entry("rejects bson.D, which is a document", bson.D{{Key: "a", Value: 1}}, []any(nil), false),
		Entry("rejects []byte", []byte("ab"), []any(nil), false),
		Entry("rejects nil", nil, []any(nil), false),
	)

	DescribeTable("documentHasKey",
		func(doc bson.D, key string, expected bool) {
			Expect(documentHasKey(doc, key)).To(Equal(expected))
		},
		Entry("finds a direct field",
			bson.D{{Key: "_id", Value: "id"}}, "_id", true,
		),
		Entry("finds a field inside $and encoded with bson.M",
			bson.D{{Key: "$and", Value: bson.A{bson.M{"_id": "id"}}}}, "_id", true,
		),
		Entry("finds a field inside $and encoded with a typed slice",
			bson.D{{Key: "$and", Value: []bson.M{{"_id": "id"}}}}, "_id", true,
		),
		Entry("finds a field inside nested operators encoded with bson.D",
			bson.D{{
				Key: "$or",
				Value: bson.A{bson.D{{
					Key:   "$and",
					Value: bson.A{bson.D{{Key: "_id", Value: "id"}}},
				}}},
			}}, "_id", true,
		),
		Entry("ignores the same key in an ordinary nested document",
			bson.D{{Key: "payload", Value: bson.M{"_id": "nested"}}}, "_id", false,
		),
		Entry("ignores the same key below an ordinary field inside an operator",
			bson.D{{
				Key: "$and",
				Value: bson.A{bson.M{
					"payload": bson.M{"_id": "nested"},
				}},
			}}, "_id", false,
		),
		Entry("returns false when the key is absent",
			bson.D{{Key: "name", Value: "demo"}}, "_id", false,
		),
	)

	DescribeTable("isOperatorDoc",
		func(v any, expected bool) {
			Expect(isOperatorDoc(v)).To(Equal(expected))
		},
		Entry("bson.M with an operator", bson.M{"$in": bson.A{1}}, true),
		Entry("bson.D with an operator", bson.D{{Key: "$gt", Value: 1}}, true),
		Entry("map[string]any with an operator", map[string]any{"$exists": true}, true),
		Entry("an ordinary embedded document", bson.M{"color": "red"}, false),
		Entry("a scalar", "red", false),
		Entry("nil", nil, false),
	)

	DescribeTable("equalityKeys",
		func(conds bson.D, expected map[string]bool) {
			Expect(equalityKeys(conds)).To(Equal(expected))
		},
		Entry("collects plain values and embedded documents",
			bson.D{{Key: "name", Value: "a"}, {Key: "spec", Value: bson.M{"color": "red"}}},
			map[string]bool{"name": true, "spec": true},
		),
		Entry("skips operator expressions",
			bson.D{{Key: "name", Value: "a"}, {Key: "priority", Value: bson.M{"$gt": 1}}},
			map[string]bool{"name": true},
		),
		Entry("collects keys inside $and of any document form",
			bson.D{{Key: "$and", Value: bson.A{
				bson.D{{Key: "name", Value: "a"}},
				bson.M{"kind": "b"},
				bson.M{"priority": bson.M{"$lt": 3}},
			}}},
			map[string]bool{"name": true, "kind": true},
		),
		Entry("skips $or and $nor, whose values are not written on insert",
			bson.D{{Key: "$or", Value: bson.A{bson.M{"name": "a"}}}},
			map[string]bool{},
		),
	)

	Describe("mapFilterKeys", func() {
		upper := func(k string) (string, error) { return strings.ToUpper(k), nil }

		It("maps field keys and keeps operator values as is", func() {
			elemMatch := bson.M{"$elemMatch": bson.M{"name": "x"}}
			got, err := mapFilterKeys(bson.D{
				{Key: "name", Value: "a"},
				{Key: "items", Value: elemMatch},
				{Key: "$expr", Value: bson.M{"$eq": bson.A{"$a", "$b"}}},
			}, upper)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.D{
				{Key: "NAME", Value: "a"},
				{Key: "ITEMS", Value: elemMatch},
				{Key: "$expr", Value: bson.M{"$eq": bson.A{"$a", "$b"}}},
			}))
		})

		It("maps keys inside logical operators recursively", func() {
			got, err := mapFilterKeys(bson.D{{Key: "$or", Value: []bson.M{
				{"name": "a"},
				{"$and": bson.A{bson.D{{Key: "kind", Value: "b"}}}},
			}}}, upper)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.D{{Key: "$or", Value: bson.A{
				bson.D{{Key: "NAME", Value: "a"}},
				bson.D{{Key: "$and", Value: bson.A{bson.D{{Key: "KIND", Value: "b"}}}}},
			}}}))
		})

		It("keeps an unrecognized logical operator value as is", func() {
			value := bson.A{"not a document"}
			got, err := mapFilterKeys(bson.D{{Key: "$and", Value: value}}, upper)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.D{{Key: "$and", Value: value}}))
		})

		It("returns the error of fn, including those from nested conditions", func() {
			fail := func(k string) (string, error) { return "", errors.Errorf("bad key %s", k) }
			_, err := mapFilterKeys(bson.D{{Key: "$nor", Value: bson.A{bson.M{"name": "a"}}}}, fail)
			Expect(err).To(MatchError("bad key name"))
		})
	})

	DescribeTable("updateDoc",
		func(set, unset, onInsert, expected bson.M) {
			Expect(updateDoc(set, unset, onInsert)).To(Equal(expected))
		},
		Entry("includes all non-empty operators",
			bson.M{"a": 1}, bson.M{"b": ""}, bson.M{"c": 2},
			bson.M{"$set": bson.M{"a": 1}, "$unset": bson.M{"b": ""}, "$setOnInsert": bson.M{"c": 2}},
		),
		Entry("skips empty and nil operators",
			bson.M{"a": 1}, bson.M{}, nil,
			bson.M{"$set": bson.M{"a": 1}},
		),
	)
})
