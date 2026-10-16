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

var _ = Describe("GenericStore Upsert", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	// mustUpsert 执行 Upsert 并断言是否发生了插入
	mustUpsert := func(target any, inserted bool, opts ...store.Opt) {
		got, err := f.widgets.Upsert(f.ctx, target, opts...)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(inserted), "inserted")
	}

	Describe("with an object", func() {
		It("should insert with generated _id and timestamps when absent and fill them back", func() {
			w := f.newWidget("w1")
			mustUpsert(w, true, f.byName("w1"))

			got := f.mustGetWidget("w1")
			Expect(got.ID).NotTo(BeZero())
			Expect(got.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(got.UpdatedAt).To(beStoredTime(got.CreatedAt))

			Expect(w.ID).To(Equal(got.ID))
			Expect(w.CreatedAt).To(beStoredTime(got.CreatedAt))
		})

		It("should keep a preset createdAt on insert", func() {
			w := f.newWidget("w1")
			w.CreatedAt = longAgo
			mustUpsert(w, true, f.byName("w1"))

			Expect(f.mustGetWidget("w1").CreatedAt).To(beStoredTime(longAgo))
		})

		It("should overwrite fields but keep _id and createdAt when present", func() {
			created := f.mustCreateStaleWidget("w1")

			w := f.newWidget("w1")
			w.Priority = 7
			w.CreatedAt = time.Now()
			mustUpsert(w, false, f.byName("w1"))
			// 更新时不知道库中已有的 _id，不做回填
			Expect(w.ID).To(BeZero())

			got := f.mustGetWidget("w1")
			Expect(got.ID).To(Equal(created.ID))
			Expect(got.Priority).To(Equal(7))
			Expect(got.CreatedAt).To(beStoredTime(longAgo))
			Expect(got.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(f.countWidgets()).To(Equal(int64(1)))
		})

		It("should replace a nested document as a whole", func() {
			Expect(f.widgets.Create(f.ctx, f.newSecretWidget("w1"))).To(Succeed())

			upd := f.newWidget("w1")
			upd.Secret = &Credential{Username: "u2"}
			mustUpsert(upd, false, f.byName("w1"))

			Expect(f.mustGetWidget("w1").Secret).To(Equal(&Credential{Username: "u2"}))
		})

		Describe("fields omitted from the encoded object", func() {
			BeforeEach(func() {
				Expect(f.widgets.Create(f.ctx, f.newSecretWidget("w1"))).To(Succeed())
			})

			It("should unset them by default", func() {
				mustUpsert(f.newWidget("w1"), false, f.byName("w1"))
				Expect(rawHas(f.rawWidget("w1"), "secret")).To(BeFalse())
			})

			It("should keep the stored value when the field is omitted", func() {
				mustUpsert(f.newWidget("w1"), false, f.byName("w1"), store.UpdateOmit("secret"))
				Expect(f.mustGetWidget("w1").Secret).To(Equal(&Credential{Username: "u", Password: plainPassword}))
			})

			It("should keep stored fields that are not defined in the model", func() {
				// 模拟模型删除字段后库中残留的旧字段
				_, err := f.widgets.Collection().UpdateOne(f.ctx,
					bson.M{"ownerID": f.owner, "name": "w1"}, bson.M{"$set": bson.M{"legacy": "x"}})
				Expect(err).NotTo(HaveOccurred())

				mustUpsert(f.newWidget("w1"), false, f.byName("w1"))

				Expect(rawString(f.rawWidget("w1"), "legacy")).To(Equal("x"))
			})
		})

		Describe("with UpdateSelected", func() {
			It("should only update the selected fields of an existing document", func() {
				f.mustCreateWidget("w1")

				upd := f.newWidget("w1")
				upd.Priority = 9
				upd.Tags = []string{"b"}
				mustUpsert(upd, false, store.UpdateSelected("priority"), f.byName("w1"))

				got := f.mustGetWidget("w1")
				Expect(got.Priority).To(Equal(9))
				Expect(got.Tags).To(Equal([]string{"a"}))
			})

			It("should insert all fields of the object when absent", func() {
				w := f.newWidget("w1")
				w.Priority = 4
				w.Tags = []string{"b"}
				mustUpsert(w, true, store.UpdateSelected("priority"), f.byName("w1"))

				got := f.mustGetWidget("w1")
				Expect(got.Priority).To(Equal(4))
				Expect(got.Tags).To(Equal([]string{"b"}))
				// BeforeSave 补全的默认值同样写入
				Expect(got.Spec.Color).To(Equal(defaultColor))
			})

			It("should take equality fields from the filter instead of zero values of the object on insert", func() {
				// 对象中 ownerID 为空，插入的文档取过滤条件中的值
				mustUpsert(&Widget{Name: "w1", Priority: 2}, true, store.UpdateSelected("priority"), f.byName("w1"))

				got := f.mustGetWidget("w1")
				Expect(got.OwnerID).To(Equal(f.owner))
				Expect(got.Priority).To(Equal(2))
			})

			Describe("nested fields", func() {
				newColored := func() *Widget {
					w := f.newWidget("w1")
					w.Priority = 5
					w.Spec = WidgetSpec{Color: "blue", Size: 3}
					return w
				}

				It("should only update the nested field of an existing document", func() {
					w := f.newWidget("w1")
					w.Spec = WidgetSpec{Color: "red", Size: 1}
					Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

					mustUpsert(newColored(), false, store.UpdateSelected("spec.color"), f.byName("w1"))

					got := f.mustGetWidget("w1")
					Expect(got.Spec).To(Equal(WidgetSpec{Color: "blue", Size: 1}))
					Expect(got.Priority).To(Equal(1))
				})

				It("should insert the parent with the selected nested field only", func() {
					mustUpsert(newColored(), true, store.UpdateSelected("spec.color"), f.byName("w1"))

					got := f.mustGetWidget("w1")
					Expect(got.Spec).To(Equal(WidgetSpec{Color: "blue"}))
					Expect(got.Priority).To(Equal(5))
				})
			})
		})
	})

	Describe("with a map of nested fields", func() {
		It("should insert a document containing only the given map entry", func() {
			inserted, err := f.boards.Upsert(
				f.ctx,
				map[string]any{"states.prod": BoardState{Status: "ok"}},
				f.byName("b1"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeTrue())

			got := f.mustGetBoard("b1")
			Expect(got.States).To(Equal(map[string]BoardState{"prod": {Status: "ok"}}))
			Expect(got.OwnerID).To(Equal(f.owner))
		})

		It("should keep the other entries of an existing document", func() {
			f.mustCreateBoard("b1")

			inserted, err := f.boards.Upsert(
				f.ctx,
				map[string]any{"states.test": BoardState{Status: "new"}},
				f.byName("b1"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeFalse())

			got := f.mustGetBoard("b1")
			Expect(got.States).To(HaveKeyWithValue("prod", BoardState{Status: "ok"}))
			Expect(got.States).To(HaveKeyWithValue("test", BoardState{Status: "new"}))
			Expect(got.Cards).To(HaveLen(1))
		})

		It("should not write the parent of a nested equality field from the filter on insert", func() {
			mustUpsert(map[string]any{"priority": 2}, true, f.byName("w1"), store.Where("spec.color", "green"))

			got := f.mustGetWidget("w1")
			Expect(got.Spec.Color).To(Equal("green"))
			Expect(got.Priority).To(Equal(2))
		})
	})

	Describe("UpsertMany", func() {
		byObjectName := func(w *Widget) store.Opt { return f.byName(w.Name) }

		It("should insert absent objects, update present ones and fill back inserted IDs", func() {
			existing := f.mustCreateStaleWidget("w1")

			upd, absent := f.newSecretWidget("w1"), f.newSecretWidget("w2")
			upd.Priority, absent.Priority = 7, 8
			n, err := f.widgets.UpsertMany(
				f.ctx,
				[]*Widget{upd, absent},
				byObjectName,
				store.UpdateSelected("priority"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(1)))

			Expect(upd.ID).To(BeZero())
			Expect(absent.ID).NotTo(BeZero())

			got := f.mustGetWidget("w1")
			Expect(got.ID).To(Equal(existing.ID))
			Expect(got.Priority).To(Equal(7))
			// 已存在的文档只更新选中的字段
			Expect(got.Secret).To(BeNil())
			Expect(got.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))

			inserted := f.mustGetWidget("w2")
			Expect(inserted.ID).To(Equal(absent.ID))
			Expect(inserted.Priority).To(Equal(8))
			// 插入的文档包含全部字段，且经过 BeforeSave 与 Encode
			Expect(inserted.Spec.Color).To(Equal(defaultColor))
			Expect(f.rawPassword("w2")).To(Equal(encPrefix + plainPassword))
		})

		It("should do nothing for an empty list", func() {
			n, err := f.widgets.UpsertMany(f.ctx, nil, byObjectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeZero())
		})

		It("should return a duplicate error on unique key conflict", func() {
			byPriority := func(w *Widget) store.Opt {
				return store.Combine(f.byOwner(), store.Where("priority", w.Priority))
			}
			a, b := f.newWidget("same"), f.newWidget("same")
			a.Priority, b.Priority = 1, 2

			_, err := f.widgets.UpsertMany(f.ctx, []*Widget{a, b}, byPriority)
			Expect(err).To(MatchError(ErrWidgetExists))
			Expect(f.countWidgets()).To(Equal(int64(1)))
		})

		DescribeTable(
			"should reject invalid input before writing",
			func(objs func() []*Widget, key func(*Widget) store.Opt, msg string) {
				_, err := f.widgets.UpsertMany(f.ctx, objs(), key)
				Expect(err).To(MatchError(ContainSubstring(msg)))
				Expect(f.countWidgets()).To(BeZero())
			},
			Entry("nil object", func() []*Widget { return []*Widget{f.newWidget("w1"), nil} },
				byObjectName, "object 1 is nil"),
			Entry("key that is not a condition", func() []*Widget { return []*Widget{f.newWidget("w1")} },
				func(*Widget) store.Opt { return store.Asc("name") }, "must be a condition"),
			Entry(
				"object failing BeforeSave",
				func() []*Widget { return []*Widget{f.newWidget("w1"), {OwnerID: "x"}} },
				func(w *Widget) store.Opt { return store.Where("ownerID", w.OwnerID) },
				"object 1",
			),
		)
	})

	Describe("with a map", func() {
		It("should insert the given fields, zero values and the equality fields from the filter", func() {
			mustUpsert(map[string]any{"priority": 5}, true, f.byName("w1"))

			got := f.mustGetWidget("w1")
			Expect(got.ID).NotTo(BeZero())
			Expect(got.OwnerID).To(Equal(f.owner))
			Expect(got.Name).To(Equal("w1"))
			Expect(got.Priority).To(Equal(5))
			Expect(got.Tags).To(BeNil())
			// 不执行 BeforeSave，因此没有默认颜色
			Expect(got.Spec).To(Equal(WidgetSpec{}))
			Expect(got.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should only update the given fields of an existing document", func() {
			f.mustCreateWidget("w1")

			mustUpsert(map[string]any{"Tags": []string{"x"}}, false, f.byName("w1"))

			got := f.mustGetWidget("w1")
			Expect(got.Priority).To(Equal(1))
			Expect(got.Tags).To(Equal([]string{"x"}))
		})

		It("should not accept UpdateOpt", func() {
			_, err := f.widgets.Upsert(f.ctx, map[string]any{"priority": 5}, store.UpdateOmit("tags"), f.byName("w1"))
			Expect(err).To(MatchError(ContainSubstring("not accepted when the target is a map")))
		})
	})

	// 乐观锁写法：过滤条件除唯一键外还带有版本等守卫条件，守卫不满足时 Upsert 会转为插入并撞上唯一键
	Describe("with a guard condition on top of the unique key", func() {
		guarded := func(w *Widget, maxPriority int) (bool, error) {
			return f.widgets.Upsert(f.ctx, w, f.byName(w.Name), store.Where("priority", bson.M{"$lte": maxPriority}))
		}

		It("should update the document when the guard matches", func() {
			f.mustCreateWidget("w1")
			w := f.newWidget("w1")
			w.Priority = 5

			Expect(guarded(w, 1)).To(BeFalse())
			Expect(f.mustGetWidget("w1").Priority).To(Equal(5))
		})

		It("should return a duplicate error without writing when the guard misses", func() {
			f.mustCreateWidget("w1")
			w := f.newWidget("w1")
			w.Priority = 5

			_, err := guarded(w, 0)
			Expect(err).To(MatchError(ErrWidgetExists))
			Expect(err).To(MatchError(store.ErrDuplicate))
			Expect(f.mustGetWidget("w1").Priority).To(Equal(1))
			Expect(f.countWidgets()).To(Equal(int64(1)))
		})
	})

	Describe("_id of the inserted document", func() {
		It("should generate a string _id for string-keyed models", func() {
			item := &Item{Value: "only"}
			inserted, err := f.items.Upsert(f.ctx, item, store.Where("value", "only"))
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeTrue())

			raw, err := f.items.Collection().FindOne(f.ctx, bson.M{"value": "only"}).Raw()
			Expect(err).NotTo(HaveOccurred())
			Expect(raw.Lookup("_id").Type).To(Equal(bson.TypeString))
			_, err = bson.ObjectIDFromHex(item.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(rawString(raw, "_id")).To(Equal(item.ID))
		})

		It("should use the object's ID as the filter when no condition is given", func() {
			item := &Item{ID: "env-9", Value: "v1"}
			inserted, err := f.items.Upsert(f.ctx, item)
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeTrue())

			item.Value = "v2"
			inserted, err = f.items.Upsert(f.ctx, item)
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeFalse())

			got, err := f.items.Get(f.ctx, store.Where("_id", "env-9"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Value).To(Equal("v2"))
		})

		It("should combine the object's ID with the explicit filter", func() {
			// 过滤条件匹配 w1，但对象的 _id 不同，因此转为插入并触发 ownerID + name 唯一索引冲突
			created := f.mustCreateWidget("w1")
			upd := f.newWidget("w1")
			upd.ID = bson.NewObjectID()

			_, err := f.widgets.Upsert(f.ctx, upd, f.byName("w1"))
			Expect(err).To(MatchError(ErrWidgetExists))
			Expect(f.mustGetWidget("w1").ID).To(Equal(created.ID))
		})

		DescribeTable("should reject an _id filter when the object already has an ID",
			func(opt store.Opt) {
				_, err := f.items.Upsert(f.ctx, &Item{ID: "new-id", Value: "v"}, opt)
				Expect(err).To(MatchError(ContainSubstring("_id is specified in both object and filter")))
				Expect(f.items.Count(f.ctx, store.All())).To(BeZero())
			},
			Entry("plain value", store.Where("_id", "new-id")),
			Entry("$ne", store.Where("_id", bson.M{"$ne": "excluded-id"})),
			Entry("multi-value $in", store.Where("_id", bson.M{"$in": bson.A{"a", "b"}})),
			Entry("inside $and", store.Filter(bson.M{"$and": bson.A{bson.M{"_id": "new-id"}}})),
			Entry("inside $and given as a typed slice", store.Filter(bson.M{"$and": []bson.M{{"_id": "new-id"}}})),
		)

		It("should generate a string _id for a non-equality _id condition when the object has no ID", func() {
			item := &Item{Value: "v"}
			_, err := f.items.Upsert(f.ctx, item, store.Where("_id", bson.M{"$ne": "excluded-id"}))
			Expect(err).NotTo(HaveOccurred())

			items, err := f.items.List(f.ctx, store.Where("value", "v"))
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].ID).To(Equal(item.ID))
		})

		It("should not use an _id equality from the filter as the inserted ID", func() {
			// 插入所需的 _id 按模型类型生成，与过滤条件中的 _id 冲突时由 MongoDB 拒绝
			_, err := f.items.Upsert(f.ctx, &Item{Value: "v"}, store.Where("_id", "env-filter"))
			Expect(err).To(MatchError(ContainSubstring("immutable field '_id'")))
			Expect(f.items.Count(f.ctx, store.All())).To(BeZero())
		})
	})
})
