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

			It("should reject nested fields", func() {
				_, err := f.widgets.Upsert(
					f.ctx,
					f.newWidget("w1"),
					store.UpdateSelected("spec.color"),
					f.byName("w1"),
				)
				Expect(err).To(MatchError(ContainSubstring("nested field")))
			})
		})
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
			_, err := f.widgets.Upsert(f.ctx, map[string]any{"priority": 5}, store.UpdateNonZero(), f.byName("w1"))
			Expect(err).To(MatchError(ContainSubstring("not accepted when the target is a map")))
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
