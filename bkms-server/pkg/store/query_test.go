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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore query options", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	// createWithPriority 创建一组 Widget，名称为 w<priority>
	createWithPriority := func(priorities ...int) {
		for _, p := range priorities {
			w := f.newWidget("w" + string(rune('0'+p)))
			w.Priority = p
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
		}
	}
	namesOf := func(list []Widget) []string {
		return lo.Map(list, func(w Widget, _ int) string { return w.Name })
	}

	Describe("filter composition", func() {
		It("should AND conditions from multiple opts", func() {
			createWithPriority(1, 2)

			list, err := f.widgets.List(f.ctx, f.byOwner(), store.Where("priority", 2))
			Expect(err).NotTo(HaveOccurred())
			Expect(namesOf(list)).To(Equal([]string{"w2"}))
		})

		It("should accept a raw bson.M through Filter", func() {
			createWithPriority(1, 2)

			list, err := f.widgets.List(f.ctx, store.Filter(bson.M{"ownerID": f.owner, "priority": 1}))
			Expect(err).NotTo(HaveOccurred())
			Expect(namesOf(list)).To(Equal([]string{"w1"}))
		})

		It("should keep every condition when the same field appears more than once", func() {
			createWithPriority(1, 2, 3, 4)

			list, err := f.widgets.List(f.ctx,
				f.byOwner(),
				store.Where("priority", bson.M{"$gte": 2}),
				store.Where("priority", bson.M{"$lte": 3}),
				store.Asc("priority"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(namesOf(list)).To(Equal([]string{"w2", "w3"}))
		})

		It("should ignore nil opts", func() {
			createWithPriority(1)

			got, err := f.widgets.Get(f.ctx, nil, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
		})

		It("should match all documents for reads without any condition", func() {
			createWithPriority(1, 2)

			n, err := f.widgets.Count(f.ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))
		})

		It("should include the conditions in the NotFound error", func() {
			_, err := f.widgets.Get(f.ctx, f.byName("missing"))
			Expect(err).To(MatchError(ContainSubstring("missing")))
			Expect(err).To(MatchError(ContainSubstring(f.owner)))
		})
	})

	Describe("sort", func() {
		It("Get should pick the first document by the given order", func() {
			createWithPriority(1, 3, 2)

			got, err := f.widgets.Get(f.ctx, f.byOwner(), store.Desc("priority"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w3"))
		})

		It("should reject sort and limit for operations other than Get and List", func() {
			createWithPriority(1)
			w := f.newWidget("w1")

			_, err := f.widgets.Exists(f.ctx, f.byOwner(), store.Limit(1))
			Expect(err).To(MatchError(ContainSubstring("only supported by Get and List")))
			_, err = f.widgets.Count(f.ctx, f.byOwner(), store.Asc("name"))
			Expect(err).To(MatchError(ContainSubstring("only supported by Get and List")))
			_, err = f.widgets.DeleteMany(f.ctx, f.byOwner(), store.Limit(1))
			Expect(err).To(MatchError(ContainSubstring("only supported by Get and List")))
			Expect(f.widgets.Update(f.ctx, bson.M{"priority": 9}, f.byOwner(), store.Desc("name"))).
				To(MatchError(ContainSubstring("only supported by Get and List")))
			_, err = f.widgets.Upsert(f.ctx, w, f.byName("w1"), store.Desc("name"))
			Expect(err).To(MatchError(ContainSubstring("only supported by Get and List")))

			n, err := f.widgets.Count(f.ctx, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(1)))
		})
	})

	Describe("write without conditions", func() {
		It("should be rejected unless store.All is given", func() {
			createWithPriority(1)
			w := f.newWidget("w1")
			const msg = "filter is required for write operations"

			_, err := f.widgets.DeleteMany(f.ctx)
			Expect(err).To(MatchError(ContainSubstring(msg)))
			Expect(f.widgets.Delete(f.ctx)).To(MatchError(ContainSubstring(msg)))
			Expect(f.widgets.DeleteIfExists(f.ctx)).To(MatchError(ContainSubstring(msg)))
			Expect(f.widgets.Update(f.ctx, bson.M{"priority": 2})).To(MatchError(ContainSubstring(msg)))
			Expect(f.widgets.UpdateFields(f.ctx, w, []string{"priority"})).To(MatchError(ContainSubstring(msg)))
			_, err = f.widgets.Upsert(f.ctx, w)
			Expect(err).To(MatchError(ContainSubstring(msg)))
			Expect(f.widgets.Replace(f.ctx, w)).To(MatchError(ContainSubstring(msg)))

			// 仅包含排序、数量限制等非过滤条件的 Opt 同样视为没有过滤条件
			_, err = f.widgets.DeleteMany(f.ctx, store.Asc("name"))
			Expect(err).To(MatchError(ContainSubstring(msg)))

			n, err := f.widgets.Count(f.ctx, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(1)))
		})

		It("should apply to all documents with store.All", func() {
			createWithPriority(1, 2)

			Expect(f.widgets.Update(f.ctx, bson.M{"tags": []string{"x"}}, store.All())).To(Succeed())
			n, err := f.widgets.Count(f.ctx, store.Where("tags", "x"))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(1)), "Update only modifies one document")

			deleted, err := f.widgets.DeleteMany(f.ctx, store.All())
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted).To(Equal(int64(2)))
		})
	})
})
