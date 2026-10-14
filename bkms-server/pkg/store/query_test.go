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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore query options", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	// createWithPriority 创建一组 Widget，名称为 "w" + priority
	createWithPriority := func(priorities ...int) {
		for _, p := range priorities {
			w := f.newWidget("w" + strconv.Itoa(p))
			w.Priority = p
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
		}
	}
	listNames := func(opts ...store.Opt) []string {
		list, err := f.widgets.List(f.ctx, opts...)
		Expect(err).NotTo(HaveOccurred())
		return widgetNames(list)
	}

	Describe("filter composition", func() {
		It("should AND conditions from multiple opts", func() {
			createWithPriority(1, 2)
			Expect(listNames(f.byOwner(), store.Where("priority", 2))).To(Equal([]string{"w2"}))
		})

		It("should accept a raw bson.M through Filter", func() {
			createWithPriority(1, 2)
			Expect(listNames(store.Filter(bson.M{"ownerID": f.owner, "priority": 1}))).To(Equal([]string{"w1"}))
		})

		It("should keep every condition when the same field appears more than once", func() {
			createWithPriority(1, 2, 3, 4)

			names := listNames(
				f.byOwner(),
				store.Where("priority", bson.M{"$gte": 2}),
				store.Where("priority", bson.M{"$lte": 3}),
				store.Asc("priority"),
			)
			Expect(names).To(Equal([]string{"w2", "w3"}))
		})

		It("should ignore nil opts", func() {
			createWithPriority(1)

			got, err := f.widgets.Get(f.ctx, nil, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
		})

		It("should match all documents for reads without any condition", func() {
			createWithPriority(1, 2)
			Expect(f.widgets.Count(f.ctx)).To(Equal(int64(2)))
		})

		It("should include the conditions in the NotFound error", func() {
			_, err := f.widgets.Get(f.ctx, f.byName("missing"))
			Expect(err).To(MatchError(ContainSubstring("missing")))
			Expect(err).To(MatchError(ContainSubstring(f.owner)))
		})
	})

	Describe("field names", func() {
		It("should accept Go field names in conditions and sorts", func() {
			createWithPriority(1, 2, 3)

			names := listNames(
				store.Where("OwnerID", f.owner),
				store.Filter(bson.M{"$or": []bson.M{{"Name": "w1"}, {"Priority": 3}}}),
				store.Desc("Priority"),
			)
			Expect(names).To(Equal([]string{"w3", "w1"}))
		})

		It("should accept Go field names in nested paths", func() {
			createWithPriority(1)
			Expect(f.widgets.Count(f.ctx, f.byOwner(), store.Where("Spec.Color", defaultColor))).To(Equal(int64(1)))
		})

		It("should keep fields that are not defined in the model as they are", func() {
			_, err := f.widgets.Collection().InsertOne(f.ctx, bson.M{"ownerID": f.owner, "name": "w1", "legacy": "x"})
			Expect(err).NotTo(HaveOccurred())

			got, err := f.widgets.Get(f.ctx, f.byOwner(), store.Where("legacy", "x"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
		})

		It("should reject a field name that is ambiguous between a bson key and a Go field name", func() {
			ambiguous := store.Of(f.db, ambiguousMeta)
			_, err := ambiguous.List(f.ctx, store.Where("Label", "x"))
			Expect(err).To(MatchError(ContainSubstring("ambiguous field")))

			_, err = ambiguous.List(f.ctx, store.Asc("Label"))
			Expect(err).To(MatchError(ContainSubstring("ambiguous field")))

			// 不存在歧义的写法正常解析
			_, err = ambiguous.List(f.ctx, store.Where("title", "x"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("sort and limit", func() {
		It("Get should pick the first document by the given order", func() {
			createWithPriority(1, 3, 2)

			got, err := f.widgets.Get(f.ctx, f.byOwner(), store.Desc("priority"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w3"))
		})

		It("should honor sort and limit combined with conditions through Combine", func() {
			createWithPriority(1, 3, 2)
			names := listNames(store.Combine(f.byOwner(), store.Desc("priority")), store.Limit(2))
			Expect(names).To(Equal([]string{"w3", "w2"}))
		})
	})

	Describe("unsupported options", func() {
		const msg = "option not supported by this operation"

		DescribeTable("should be rejected without touching data",
			func(op func(w *Widget) error) {
				w := f.mustCreateWidget("w1")
				Expect(op(f.newWidget("w1"))).To(MatchError(ContainSubstring(msg)))
				Expect(f.mustGetWidget("w1").UpdatedAt).To(beStoredTime(w.UpdatedAt))
			},
			Entry("Limit on Get", func(*Widget) error {
				_, err := f.widgets.Get(f.ctx, f.byOwner(), store.Limit(1))
				return err
			}),
			Entry("UpdateOpt on List", func(*Widget) error {
				_, err := f.widgets.List(f.ctx, f.byOwner(), store.UpdateOmit("tags"))
				return err
			}),
			Entry("Limit on Exists", func(*Widget) error {
				_, err := f.widgets.Exists(f.ctx, f.byOwner(), store.Limit(1))
				return err
			}),
			Entry("sort on Count", func(*Widget) error {
				_, err := f.widgets.Count(f.ctx, f.byOwner(), store.Asc("name"))
				return err
			}),
			Entry("Limit on DeleteMany", func(*Widget) error {
				_, err := f.widgets.DeleteMany(f.ctx, f.byOwner(), store.Limit(1))
				return err
			}),
			Entry("Limit on GetAndDelete", func(*Widget) error {
				_, err := f.widgets.GetAndDelete(f.ctx, f.byOwner(), store.Limit(1))
				return err
			}),
			Entry("sort on Update", func(*Widget) error {
				return f.widgets.Update(f.ctx, bson.M{"priority": 9}, f.byOwner(), store.Desc("name"))
			}),
			Entry("Limit on Update", func(w *Widget) error {
				return f.widgets.Update(f.ctx, w, store.UpdateSelected("priority"), f.byOwner(), store.Limit(1))
			}),
			Entry("UpdateOpt on Replace", func(w *Widget) error {
				return f.widgets.Replace(f.ctx, w, f.byName("w1"), store.UpdateOmit("tags"))
			}),
			Entry("sort on Upsert", func(w *Widget) error {
				_, err := f.widgets.Upsert(f.ctx, w, f.byName("w1"), store.Desc("name"))
				return err
			}),
		)

		It("should be detected inside options built by Combine", func() {
			_, err := f.widgets.Count(f.ctx, store.Combine(f.byOwner(), store.Desc("name")))
			Expect(err).To(MatchError(ContainSubstring(msg)))
		})

		It("should name the rejected option kinds in the error", func() {
			_, err := f.widgets.Count(f.ctx, store.Combine(store.Asc("name"), store.Limit(1)))
			Expect(err).To(MatchError(ContainSubstring("count in " + widgetCollection + ": sort, limit")))
		})
	})

	Describe("writes without conditions", func() {
		const msg = "filter is required for write operations"

		DescribeTable("should be rejected unless store.All is given",
			func(op func(w *Widget) error) {
				f.mustCreateWidget("w1")
				Expect(op(f.newWidget("w1"))).To(MatchError(ContainSubstring(msg)))
				Expect(f.countWidgets()).To(Equal(int64(1)))
			},
			Entry("Delete", func(*Widget) error { return f.widgets.Delete(f.ctx) }),
			Entry("DeleteIfExists", func(*Widget) error { return f.widgets.DeleteIfExists(f.ctx) }),
			Entry("DeleteMany", func(*Widget) error {
				_, err := f.widgets.DeleteMany(f.ctx)
				return err
			}),
			Entry("GetAndDelete", func(*Widget) error {
				_, err := f.widgets.GetAndDelete(f.ctx)
				return err
			}),
			Entry("Update with a map", func(*Widget) error { return f.widgets.Update(f.ctx, bson.M{"priority": 2}) }),
			Entry("Update with an object", func(w *Widget) error {
				return f.widgets.Update(f.ctx, w, store.UpdateSelected("priority"))
			}),
			Entry("Replace", func(w *Widget) error { return f.widgets.Replace(f.ctx, w) }),
			Entry("Upsert", func(w *Widget) error {
				_, err := f.widgets.Upsert(f.ctx, w)
				return err
			}),
			// 仅包含非过滤条件的 Opt 同样视为没有过滤条件
			Entry("Upsert with only an UpdateOpt", func(w *Widget) error {
				_, err := f.widgets.Upsert(f.ctx, w, store.UpdateOmit("tags"))
				return err
			}),
		)

		It("should apply to all documents with store.All", func() {
			createWithPriority(1, 2)

			Expect(f.widgets.Update(f.ctx, bson.M{"tags": []string{"x"}}, store.All())).To(Succeed())
			Expect(f.widgets.Count(f.ctx, store.Where("tags", "x"))).
				To(Equal(int64(1)), "Update only modifies one document")

			Expect(f.widgets.DeleteMany(f.ctx, store.All())).To(Equal(int64(2)))
		})
	})
})
