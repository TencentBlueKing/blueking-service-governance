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
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore read operations", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	Describe("Get", func() {
		It("should return the document matching the filter", func() {
			created := f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(created.ID))
			Expect(got.Name).To(Equal("w1"))
		})

		It("should wrap both the domain sentinel and store.ErrNotFound when missing", func() {
			_, err := f.widgets.Get(f.ctx, f.byName("missing"))
			Expect(err).To(MatchError(ErrWidgetNotFound))
			Expect(err).To(MatchError(store.ErrNotFound))
			Expect(err).To(MatchError(ContainSubstring(widgetCollection)))
		})

		It("should fall back to store.ErrNotFound when meta has no NotFound", func() {
			_, err := f.items.Get(f.ctx, store.Where("_id", "missing"))
			Expect(err).To(MatchError(store.ErrNotFound))
		})
	})

	Describe("GetByID", func() {
		It("should return the document with the given ObjectID", func() {
			created := f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")

			got, err := f.widgets.GetByID(f.ctx, created.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
		})

		It("should return the document with the given string ID", func() {
			Expect(f.items.Create(f.ctx, &Item{ID: "e1", Value: "v"})).To(Succeed())

			got, err := f.items.GetByID(f.ctx, "e1")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Value).To(Equal("v"))
		})

		It("should return NotFound when the ID type differs from the model", func() {
			created := f.mustCreateWidget("w1")

			_, err := f.widgets.GetByID(f.ctx, created.ID.Hex())
			Expect(err).To(MatchError(ErrWidgetNotFound))
			Expect(err).To(MatchError(store.ErrNotFound))
		})
	})

	Describe("ContainsIgnoreCase", func() {
		BeforeEach(func() {
			for name, color := range map[string]string{"Alpha-Prod": "red", "beta": "Crimson-RED", "a.c": "blue"} {
				w := f.newWidget(name)
				w.Spec.Color = color
				Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
			}
		})

		listNames := func(opts ...store.Opt) []string {
			list, err := f.widgets.List(f.ctx, append([]store.Opt{f.byOwner()}, opts...)...)
			Expect(err).NotTo(HaveOccurred())
			return widgetNames(list)
		}

		It("should match a substring regardless of case", func() {
			Expect(listNames(store.ContainsIgnoreCase(" PROD ", "name"))).To(ConsistOf("Alpha-Prod"))
		})

		It("should match regex metacharacters literally", func() {
			// 未转义时 "a.c" 会被当作正则，"." 可以匹配任意字符
			Expect(listNames(store.ContainsIgnoreCase(".", "name"))).To(ConsistOf("a.c"))
		})

		It("should match documents where any of the fields contains the keyword", func() {
			// 字段既可以写 bson 路径，也可以写 Go 字段名
			opt := store.ContainsIgnoreCase("red", "name", "Spec.Color")
			Expect(listNames(opt)).To(ConsistOf("Alpha-Prod", "beta"))
		})

		It("should keep both conditions when combined with another one on the same fields", func() {
			opts := []store.Opt{
				store.ContainsIgnoreCase("red", "name", "spec.color"),
				store.ContainsIgnoreCase("prod", "name", "spec.color"),
			}
			Expect(listNames(opts...)).To(ConsistOf("Alpha-Prod"))
		})

		It("should not filter when the keyword is blank", func() {
			Expect(listNames(store.ContainsIgnoreCase("  ", "name"))).To(HaveLen(3))
		})

		It("should not count as a filter for write operations when the keyword is blank", func() {
			_, err := f.widgets.DeleteMany(f.ctx, store.ContainsIgnoreCase("", "name"))
			Expect(err).To(MatchError(ContainSubstring("filter is required")))
		})
	})

	Describe("ByID", func() {
		It("should narrow the match when combined with other conditions", func() {
			created := f.mustCreateWidget("w1")

			Expect(f.widgets.Exists(f.ctx, store.ByID(created.ID), f.byOwner())).To(BeTrue())
			Expect(f.widgets.Exists(f.ctx, store.ByID(created.ID), store.Where("ownerID", "nobody"))).To(BeFalse())
		})

		It("should work as the filter of write operations", func() {
			created := f.mustCreateWidget("w1")

			Expect(f.widgets.Update(f.ctx, map[string]any{"priority": 9}, store.ByID(created.ID))).To(Succeed())
			got, err := f.widgets.GetByID(f.ctx, created.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Priority).To(Equal(9))

			Expect(f.widgets.Delete(f.ctx, store.ByID(created.ID))).To(Succeed())
			Expect(f.widgets.Exists(f.ctx, store.ByID(created.ID))).To(BeFalse())
		})
	})

	Describe("List", func() {
		// 名称顺序 c a b，priority 依次为 3 2 1，便于区分按名称与按 priority 排序的结果
		BeforeEach(func() {
			for i, name := range []string{"c", "a", "b"} {
				w := f.newWidget(name)
				w.Priority = 3 - i
				Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
			}
		})

		It("should return all documents matching the filter", func() {
			list, err := f.widgets.List(f.ctx, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(list)).To(ConsistOf("a", "b", "c"))
		})

		DescribeTable("should sort by the given keys in order",
			func(expected []string, sorts ...store.Opt) {
				list, err := f.widgets.List(f.ctx, append([]store.Opt{f.byOwner()}, sorts...)...)
				Expect(err).NotTo(HaveOccurred())
				Expect(widgetNames(list)).To(Equal(expected))
			},
			Entry("ascending", []string{"a", "b", "c"}, store.Asc("name")),
			Entry("descending", []string{"c", "a", "b"}, store.Desc("priority")),
		)

		It("should use the next sort key when the previous one ties", func() {
			// d 与 c 的 priority 相同，由第二个排序键决定先后
			d := f.newWidget("d")
			d.Priority = 3
			Expect(f.widgets.Create(f.ctx, d)).To(Succeed())

			list, err := f.widgets.List(f.ctx, f.byOwner(), store.Desc("priority"), store.Desc("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(list)).To(Equal([]string{"d", "c", "a", "b"}))
		})

		It("should limit the number of results", func() {
			list, err := f.widgets.List(f.ctx, f.byOwner(), store.Asc("name"), store.Limit(2))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(list)).To(Equal([]string{"a", "b"}))
		})

		It("should not limit the results when the limit is not positive", func() {
			list, err := f.widgets.List(f.ctx, f.byOwner(), store.Limit(0))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(3))
		})

		It("should return an empty non-nil slice when nothing matches", func() {
			list, err := f.widgets.List(f.ctx, store.Where("ownerID", "nobody"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).NotTo(BeNil())
			Expect(list).To(BeEmpty())
		})

		It("should treat a nil opt as no condition", func() {
			list, err := f.widgets.List(f.ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(3))
		})
	})

	Describe("OmitFields", func() {
		BeforeEach(func() {
			w := f.newSecretWidget("w1")
			w.Spec = WidgetSpec{Color: "red", Size: 3}
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
		})

		It("should leave the omitted fields zero in Get and List", func() {
			got, err := f.widgets.Get(f.ctx, f.byName("w1"), store.OmitFields("tags", "Spec.Color"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
			Expect(got.Tags).To(BeNil())
			Expect(got.Spec).To(Equal(WidgetSpec{Size: 3}))

			list, err := f.widgets.List(f.ctx, f.byOwner(), store.OmitFields("spec"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Spec).To(BeZero())
			Expect(list[0].Tags).To(Equal([]string{"a"}))
		})

		It("should not affect which documents match", func() {
			// 被跳过的字段即使出现在过滤条件中，也照常参与匹配
			got, err := f.widgets.Get(f.ctx, f.byOwner(), store.Where("spec.color", "red"), store.OmitFields("spec"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
		})

		It("should still run Decode hooks", func() {
			got, err := f.widgets.Get(f.ctx, f.byName("w1"), store.OmitFields("tags"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Secret.Password).To(Equal(plainPassword))

			// 跳过需要解码的字段时，钩子收到的是零值
			got, err = f.widgets.Get(f.ctx, f.byName("w1"), store.OmitFields("secret"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Secret).To(BeNil())
		})

		It("should reject fields not defined in the model", func() {
			_, err := f.widgets.List(f.ctx, f.byOwner(), store.OmitFields("manifest"))
			Expect(err).To(MatchError(ContainSubstring("unknown field")))
		})

		It("should reject conflicting parent and child fields", func() {
			_, err := f.widgets.List(f.ctx, f.byOwner(), store.OmitFields("spec", "spec.color"))
			Expect(err).To(MatchError(ContainSubstring("conflicting fields")))
		})

		It("should not be accepted by Count or write operations", func() {
			_, err := f.widgets.Count(f.ctx, f.byOwner(), store.OmitFields("tags"))
			Expect(err).To(MatchError(ContainSubstring("projection")))
			_, err = f.widgets.GetAndDelete(f.ctx, f.byName("w1"), store.OmitFields("tags"))
			Expect(err).To(MatchError(ContainSubstring("projection")))
		})
	})

	Describe("SelectFields", func() {
		var created *Widget

		BeforeEach(func() {
			created = f.newSecretWidget("w1")
			created.Spec = WidgetSpec{Color: "red", Size: 3}
			Expect(f.widgets.Create(f.ctx, created)).To(Succeed())
		})

		It("should only read the selected fields and _id in Get and List", func() {
			got, err := f.widgets.Get(f.ctx, f.byName("w1"), store.SelectFields("name", "Spec.Color"))
			Expect(err).NotTo(HaveOccurred())
			Expect(*got).To(Equal(Widget{ID: created.ID, Name: "w1", Spec: WidgetSpec{Color: "red"}}))

			// 多个 SelectFields 会合并
			list, err := f.widgets.List(f.ctx, f.byOwner(), store.SelectFields("name"), store.SelectFields("priority"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(Equal([]Widget{{ID: created.ID, Name: "w1", Priority: 1}}))
		})

		It("should not affect which documents match or how they are sorted", func() {
			mustCreate := func(name string, priority int) {
				w := f.newWidget(name)
				w.Priority = priority
				Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
			}
			mustCreate("w2", 3)
			mustCreate("w3", 2)

			list, err := f.widgets.List(f.ctx, f.byOwner(), store.Where("priority", bson.M{"$gt": 1}),
				store.Desc("priority"), store.SelectFields("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(list)).To(Equal([]string{"w2", "w3"}))
			Expect(list[0].Priority).To(BeZero())
		})

		It("should apply to ListPage and ListLatestBy", func() {
			page, err := f.widgets.ListPage(f.ctx, 1, 10, f.byOwner(), store.SelectFields("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Items).To(Equal([]Widget{{ID: created.ID, Name: "w1"}}))

			latest, err := f.widgets.ListLatestBy(f.ctx, []string{"spec.color"},
				f.byOwner(), store.Desc("priority"), store.SelectFields("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(Equal([]Widget{{ID: created.ID, Name: "w1"}}))
		})

		It("should still run Decode hooks", func() {
			got, err := f.widgets.Get(f.ctx, f.byName("w1"), store.SelectFields("secret"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Secret.Password).To(Equal(plainPassword))
		})

		DescribeTable("should reject invalid fields",
			func(opts []store.Opt, msg string) {
				_, err := f.widgets.List(f.ctx, append([]store.Opt{f.byOwner()}, opts...)...)
				Expect(err).To(MatchError(ContainSubstring(msg)))
			},
			Entry("a field not defined in the model",
				[]store.Opt{store.SelectFields("manifest")}, "unknown field"),
			Entry("conflicting parent and child fields",
				[]store.Opt{store.SelectFields("spec", "spec.color")}, "conflicting fields"),
			Entry("combined with OmitFields",
				[]store.Opt{store.SelectFields("name"), store.OmitFields("tags")}, "cannot be combined"),
		)
	})

	Describe("Exists", func() {
		It("should report whether any document matches", func() {
			f.mustCreateWidget("w1")
			Expect(f.widgets.Exists(f.ctx, f.byName("w1"))).To(BeTrue())
			Expect(f.widgets.Exists(f.ctx, f.byName("missing"))).To(BeFalse())
		})
	})

	Describe("Count", func() {
		It("should count the matching documents", func() {
			f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")
			Expect(f.widgets.Count(f.ctx, f.byOwner())).To(Equal(int64(2)))
			Expect(f.widgets.Count(f.ctx, f.byName("w1"))).To(Equal(int64(1)))
		})
	})
})
