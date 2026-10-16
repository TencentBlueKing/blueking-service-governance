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

// 字段选择规则（UpdateSelected / UpdateOmit 的组合与非法输入）在 field_selection_test.go 中
// 逐条覆盖，这里只验证它们与数据库写入的配合
var _ = Describe("GenericStore Update", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	// mustCreateFullWidget 创建一个 Spec 与 Secret 均有值的 Widget，用于断言未更新的字段保持不变
	mustCreateFullWidget := func(name string) *Widget {
		w := f.newSecretWidget(name)
		w.Spec = WidgetSpec{Color: "red", Size: 1}
		Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
		return w
	}

	Describe("with a map", func() {
		It("should $set the given fields and refresh updatedAt only", func() {
			w := f.mustCreateStaleWidget("w1")

			Expect(f.widgets.Update(f.ctx, bson.M{"priority": 5}, f.byName("w1"))).To(Succeed())

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(Equal(5))
			Expect(got.Tags).To(Equal(w.Tags))
			Expect(got.CreatedAt).To(beStoredTime(longAgo))
			Expect(got.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should accept Go field names and nested paths, keeping sibling fields", func() {
			mustCreateFullWidget("w1")

			err := f.widgets.Update(f.ctx, map[string]any{"Priority": 3, "Spec.Color": "blue"}, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(Equal(3))
			Expect(got.Spec).To(Equal(WidgetSpec{Color: "blue", Size: 1}))
		})

		It("should write zero values", func() {
			mustCreateFullWidget("w1")

			Expect(f.widgets.Update(f.ctx, map[string]any{"priority": 0, "spec.color": ""}, f.byName("w1"))).
				To(Succeed())

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(BeZero())
			Expect(got.Spec).To(Equal(WidgetSpec{Size: 1}))
		})

		It("should encode a nested field and keep its siblings", func() {
			mustCreateFullWidget("w1")

			Expect(f.widgets.Update(f.ctx, map[string]any{"secret.password": "new"}, f.byName("w1"))).To(Succeed())

			Expect(f.rawPassword("w1")).To(Equal(encPrefix + "new"))
			Expect(rawString(f.rawWidget("w1"), "secret", "username")).To(Equal("u"))
		})

		It("should convert values to the field types", func() {
			f.mustCreateWidget("w1")

			fields := map[string]any{"priority": int64(7), "tags": bson.A{"x", "y"}}
			Expect(f.widgets.Update(f.ctx, fields, f.byName("w1"))).To(Succeed())

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(Equal(7))
			Expect(got.Tags).To(Equal([]string{"x", "y"}))
		})

		It("should unset an omitempty field whose value is nil", func() {
			mustCreateFullWidget("w1")

			Expect(f.widgets.Update(f.ctx, map[string]any{"secret": nil}, f.byName("w1"))).To(Succeed())

			Expect(rawHas(f.rawWidget("w1"), "secret")).To(BeFalse())
		})

		It("should keep updatedAt when it is given explicitly", func() {
			f.mustCreateWidget("w1")

			Expect(f.widgets.Update(f.ctx, map[string]any{"updatedAt": longAgo}, f.byName("w1"))).To(Succeed())

			Expect(f.mustGetWidget("w1").UpdatedAt).To(beStoredTime(longAgo))
		})

		DescribeTable("should reject invalid fields without writing",
			func(m map[string]any, msg string) {
				w := f.mustCreateWidget("w1")
				Expect(f.widgets.Update(f.ctx, m, f.byName("w1"))).To(MatchError(ContainSubstring(msg)))
				Expect(f.mustGetWidget("w1").UpdatedAt).To(beStoredTime(w.UpdatedAt))
			},
			Entry("unknown field", map[string]any{"nonexistent": 1}, `has no field "nonexistent": unknown field`),
			Entry("unknown nested field", map[string]any{"spec.missing": 1},
				`resolve field path "spec.missing": store_test.WidgetSpec has no field "missing": unknown field`),
			Entry("update operator", map[string]any{"$inc": bson.M{"priority": 1}}, "update operator"),
			Entry("createdAt by Go name", map[string]any{"CreatedAt": time.Now()}, "cannot be updated"),
			Entry("path into an array", map[string]any{"tags.0": "x"}, "cannot be set from a map"),
			Entry("conflicting paths", map[string]any{"spec": WidgetSpec{}, "spec.color": "x"}, "conflicting"),
			Entry("value of a wrong type", map[string]any{"priority": "high"}, `set field "priority"`),
		)

		It("should return ErrNoFieldsToUpdate for an empty map", func() {
			Expect(f.widgets.Update(f.ctx, bson.M{}, f.byName("w1"))).To(MatchError(store.ErrNoFieldsToUpdate))
		})

		It("should not accept UpdateOpt", func() {
			err := f.widgets.Update(f.ctx, bson.M{"priority": 1}, f.byName("w1"), store.UpdateOmit("tags"))
			Expect(err).To(MatchError(ContainSubstring("not accepted when the target is a map")))
		})

		It("should return NotFound when nothing matches", func() {
			err := f.widgets.Update(f.ctx, bson.M{"priority": 5}, f.byName("missing"))
			Expect(err).To(MatchError(ErrWidgetNotFound))
		})

		It("should return a duplicate error on unique key conflict", func() {
			f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")
			err := f.widgets.Update(f.ctx, bson.M{"name": "w1"}, f.byName("w2"))
			Expect(err).To(MatchError(ErrWidgetExists))
		})
	})

	Describe("with an object", func() {
		It("should require an UpdateOpt", func() {
			f.mustCreateWidget("w1")
			err := f.widgets.Update(f.ctx, f.newWidget("w1"), f.byName("w1"))
			Expect(err).To(MatchError(ContainSubstring("specify the fields to update")))
		})

		It("should only update the selected fields and fill updatedAt back", func() {
			f.mustCreateStaleWidget("w1")

			patch := f.newWidget("w1")
			patch.Priority = 9
			patch.Tags = []string{"changed"}
			Expect(f.widgets.Update(f.ctx, patch, store.UpdateSelected("priority"), f.byName("w1"))).To(Succeed())
			Expect(patch.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(Equal(9))
			Expect(got.Tags).To(Equal([]string{"a"}))
			Expect(got.UpdatedAt).To(beStoredTime(patch.UpdatedAt))
		})

		It("should support nested paths and Go field names in UpdateSelected", func() {
			mustCreateFullWidget("w1")

			patch := &Widget{Priority: 2, Spec: WidgetSpec{Color: "blue", Size: 2}}
			Expect(f.widgets.Update(f.ctx, patch, store.UpdateSelected("spec.color", "Priority"), f.byName("w1"))).
				To(Succeed())

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(Equal(2))
			Expect(got.Spec).To(Equal(WidgetSpec{Color: "blue", Size: 1}))
		})

		It("should overwrite all fields except _id and createdAt with UpdateSelected(\"*\")", func() {
			created := mustCreateFullWidget("w1")

			patch := f.newWidget("w1")
			patch.Priority = 3
			Expect(f.widgets.Update(f.ctx, patch, store.UpdateSelected("*"), f.byName("w1"))).To(Succeed())

			got := f.mustGetWidget("w1")
			Expect(got.ID).To(Equal(created.ID))
			Expect(got.CreatedAt).To(beStoredTime(created.CreatedAt))
			Expect(got.Priority).To(Equal(3))
			Expect(got.Spec).To(Equal(WidgetSpec{}))
			// omitempty 的空值从库中删除，而不是写入 null
			Expect(rawHas(f.rawWidget("w1"), "secret")).To(BeFalse())
		})

		It("should skip the omitted fields", func() {
			mustCreateFullWidget("w1")

			patch := f.newWidget("w1")
			patch.Priority = 4
			Expect(f.widgets.Update(f.ctx, patch, store.UpdateOmit("secret", "Spec"), f.byName("w1"))).To(Succeed())

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(Equal(4))
			Expect(got.Spec).To(Equal(WidgetSpec{Color: "red", Size: 1}))
			Expect(got.Secret).To(Equal(&Credential{Username: "u", Password: plainPassword}))
		})

		It("should return ErrNoFieldsToUpdate when no field is left", func() {
			f.mustCreateWidget("w1")
			err := f.widgets.Update(f.ctx, &Widget{}, store.UpdateSelected(), f.byName("w1"))
			Expect(err).To(MatchError(store.ErrNoFieldsToUpdate))
		})

		It("should report invalid field selections with the collection name", func() {
			err := f.widgets.Update(f.ctx, f.newWidget("w1"), store.UpdateSelected("nonexistent"), f.byName("w1"))
			Expect(
				err,
			).To(MatchError(ContainSubstring(`update ` + widgetCollection + `: resolve field path "nonexistent"`)))
			Expect(err).To(MatchError(ContainSubstring("unknown field")))
		})

		It("should return NotFound when nothing matches", func() {
			err := f.widgets.Update(f.ctx, f.newWidget("x"), store.UpdateSelected("priority"), f.byName("missing"))
			Expect(err).To(MatchError(ErrWidgetNotFound))
		})
	})

	Describe("map entries", func() {
		BeforeEach(func() { f.mustCreateBoard("b1") })

		It("should set one entry and keep the others", func() {
			err := f.boards.Update(f.ctx, map[string]any{"states.test": BoardState{Status: "new"}}, f.byName("b1"))
			Expect(err).NotTo(HaveOccurred())

			Expect(f.mustGetBoard("b1").States).To(Equal(map[string]BoardState{
				"prod": {Status: "ok"},
				"test": {Status: "new"},
			}))
		})

		It("should remove an entry whose value is nil", func() {
			Expect(f.boards.Update(f.ctx, map[string]any{"states.prod": nil}, f.byName("b1"))).To(Succeed())

			raw, err := f.boards.Collection().FindOne(f.ctx, bson.M{"ownerID": f.owner, "name": "b1"}).Raw()
			Expect(err).NotTo(HaveOccurred())
			Expect(rawHas(raw, "states", "prod")).To(BeFalse())
		})
	})

	Describe("with KeepUpdatedAt", func() {
		It("should not refresh updatedAt of the document or the object", func() {
			f.mustCreateStaleWidget("w1")

			Expect(f.widgets.Update(f.ctx, bson.M{"priority": 5}, f.byName("w1"), store.KeepUpdatedAt())).To(Succeed())

			patch := &Widget{Priority: 6}
			Expect(f.widgets.Update(f.ctx, patch, store.UpdateSelected("priority"), store.KeepUpdatedAt(),
				f.byName("w1"))).To(Succeed())
			Expect(patch.UpdatedAt).To(BeZero())

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(Equal(6))
			Expect(got.UpdatedAt).To(beStoredTime(longAgo))
		})

		It("should not be accepted by Upsert", func() {
			_, err := f.widgets.Upsert(f.ctx, bson.M{"priority": 5}, f.byName("w1"), store.KeepUpdatedAt())
			Expect(err).To(MatchError(ContainSubstring("keep updatedAt")))
		})
	})

	Describe("UpdateMany", func() {
		It("should update all matching documents and return the matched count", func() {
			f.mustCreateStaleWidget("w1")
			f.mustCreateStaleWidget("w2")
			f.mustCreateWidget("other")

			n, err := f.widgets.UpdateMany(f.ctx, bson.M{"priority": 8},
				f.byOwner(), store.Where("name", bson.M{"$in": bson.A{"w1", "w2"}}))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))

			for _, name := range []string{"w1", "w2"} {
				got := f.mustGetWidget(name)
				Expect(got.Priority).To(Equal(8))
				Expect(got.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))
			}
			Expect(f.mustGetWidget("other").Priority).To(Equal(1))
		})

		It("should encode the written fields", func() {
			f.mustCreateWidget("w1")

			_, err := f.widgets.UpdateMany(f.ctx, bson.M{"secret": Credential{Password: "p"}}, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(f.rawPassword("w1")).To(Equal(encPrefix + "p"))
		})

		It("should not return an error when nothing matches", func() {
			n, err := f.widgets.UpdateMany(f.ctx, bson.M{"priority": 8}, f.byName("missing"))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeZero())
		})

		It("should require a filter", func() {
			_, err := f.widgets.UpdateMany(f.ctx, bson.M{"priority": 8})
			Expect(err).To(MatchError(ContainSubstring("filter is required")))
		})
	})

	Describe("IgnoreNotFound", func() {
		It("should drop NotFound errors only", func() {
			Expect(store.IgnoreNotFound(f.widgets.Update(f.ctx, bson.M{"priority": 1}, f.byName("missing")))).
				To(Succeed())

			f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")
			err := store.IgnoreNotFound(f.widgets.Update(f.ctx, bson.M{"name": "w1"}, f.byName("w2")))
			Expect(err).To(MatchError(ErrWidgetExists))
		})
	})

	It("should not write updatedAt for models without it", func() {
		Expect(f.items.Create(f.ctx, &Item{ID: "e1", Value: "old"})).To(Succeed())

		Expect(f.items.Update(f.ctx, map[string]any{"value": "new"}, store.Where("_id", "e1"))).To(Succeed())

		raw, err := f.items.Collection().FindOne(f.ctx, bson.M{"_id": "e1"}).Raw()
		Expect(err).NotTo(HaveOccurred())
		Expect(rawString(raw, "value")).To(Equal("new"))
		Expect(rawHas(raw, "updatedAt")).To(BeFalse())
	})

	DescribeTable("should reject invalid targets of Update and Upsert",
		func(target any, msg string) {
			Expect(f.widgets.Update(f.ctx, target, store.UpdateSelected("priority"), f.byName("w1"))).
				To(MatchError(ContainSubstring(msg)))
			_, err := f.widgets.Upsert(f.ctx, target, f.byName("w1"))
			Expect(err).To(MatchError(ContainSubstring(msg)))
		},
		Entry("nil object", (*Widget)(nil), "target object is nil"),
		Entry("struct value instead of a pointer", Widget{}, "unsupported target type store_test.Widget"),
		Entry("pointer to another model", &Item{}, "unsupported target type *store_test.Item"),
		Entry("nil", nil, "unsupported target type <nil>"),
	)
})
