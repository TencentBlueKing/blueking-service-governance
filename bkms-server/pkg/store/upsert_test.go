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
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore Upsert", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	It("should insert with generated _id and timestamps when absent and fill them back", func() {
		w := f.newWidget("w1")
		inserted, err := f.widgets.Upsert(f.ctx, w, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(inserted).To(BeTrue())

		got, err := f.widgets.Get(f.ctx, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID.IsZero()).To(BeFalse())
		Expect(got.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
		Expect(got.UpdatedAt).To(BeTemporally("~", got.CreatedAt, time.Millisecond))

		Expect(w.ID).To(Equal(got.ID))
		Expect(w.CreatedAt).To(BeTemporally("~", got.CreatedAt, time.Millisecond))
	})

	It("should overwrite fields but keep _id and createdAt when present", func() {
		created := f.mustCreateWidget("w1")
		time.Sleep(10 * time.Millisecond)

		w := f.newWidget("w1")
		w.Priority = 7
		inserted, err := f.widgets.Upsert(f.ctx, w, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(inserted).To(BeFalse())
		// 更新时不知道库中已有的 _id 与 createdAt，不做回填
		Expect(w.ID.IsZero()).To(BeTrue())
		Expect(w.CreatedAt.IsZero()).To(BeTrue())

		got, err := f.widgets.Get(f.ctx, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID).To(Equal(created.ID))
		Expect(got.Priority).To(Equal(7))
		Expect(got.CreatedAt).To(BeTemporally("~", created.CreatedAt, time.Millisecond))
		Expect(got.UpdatedAt).To(BeTemporally(">", created.UpdatedAt))

		n, err := f.widgets.Count(f.ctx, f.byOwner())
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(int64(1)))
	})

	It("should replace a nested document as a whole", func() {
		w := f.newWidget("w1")
		w.Secret = &Credential{Username: "u", Password: "p"}
		Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

		upd := f.newWidget("w1")
		upd.Secret = &Credential{Username: "u2"}
		_, err := f.widgets.Upsert(f.ctx, upd, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())

		got, err := f.widgets.Get(f.ctx, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Secret).To(Equal(&Credential{Username: "u2"}))
	})

	Describe("fields omitted from the encoded object", func() {
		// newSecretWidget 创建一个带凭证的 Widget，并返回一个同名、Secret 为 nil 的待写入对象
		newSecretWidget := func(name string) *Widget {
			w := f.newWidget(name)
			w.Secret = &Credential{Username: "u", Password: "p"}
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
			return f.newWidget(name)
		}

		It("should keep the stored value by default", func() {
			upd := newSecretWidget("w1")
			_, err := f.widgets.Upsert(f.ctx, upd, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Secret).To(Equal(&Credential{Username: "u", Password: "p"}))
		})

		It("should unset them with UnsetOmitted", func() {
			upd := newSecretWidget("w1")
			inserted, err := f.widgets.Upsert(f.ctx, upd, f.byName("w1"), store.UnsetOmitted())
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeFalse())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Secret).To(BeNil())
			_, err = f.rawWidget("w1").LookupErr("secret")
			Expect(err).To(HaveOccurred())
		})

		It("should keep _id and createdAt with UnsetOmitted", func() {
			created := f.mustCreateWidget("w1")
			// 待写入对象的 _id（omitempty）与 createdAt 均为零值
			_, err := f.widgets.Upsert(f.ctx, f.newWidget("w1"), f.byName("w1"), store.UnsetOmitted())
			Expect(err).NotTo(HaveOccurred())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(created.ID))
			Expect(got.CreatedAt).To(BeTemporally("~", created.CreatedAt, time.Millisecond))
		})

		It("should insert normally with UnsetOmitted", func() {
			w := f.newWidget("w1")
			inserted, err := f.widgets.Upsert(f.ctx, w, f.byName("w1"), store.UnsetOmitted())
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeTrue())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(w.ID))
			Expect(got.Secret).To(BeNil())
		})

		It("should keep stored fields that are not defined in the model even with UnsetOmitted", func() {
			f.mustCreateWidget("w1")
			// 模拟模型删除字段后库中残留的旧字段
			_, err := f.widgets.Collection().UpdateOne(f.ctx,
				bson.M{"ownerID": f.owner, "name": "w1"}, bson.M{"$set": bson.M{"legacy": "x"}})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.widgets.Upsert(f.ctx, f.newWidget("w1"), f.byName("w1"), store.UnsetOmitted())
			Expect(err).NotTo(HaveOccurred())

			Expect(rawString(f.rawWidget("w1"), "legacy")).To(Equal("x"))
		})
	})

	Describe("_id of the inserted document", func() {
		// rawItemID 读取库中 value 匹配的 Item 原始的 _id，用于断言其 BSON 类型
		rawItemID := func(value string) bson.RawValue {
			raw, err := f.items.Collection().FindOne(f.ctx, bson.M{"value": value}).Raw()
			Expect(err).NotTo(HaveOccurred())
			return raw.Lookup("_id")
		}

		It("should generate a string _id for string-keyed models", func() {
			item := &Item{Value: "only"}
			inserted, err := f.items.Upsert(f.ctx, item, store.Where("value", "only"))
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeTrue())

			got, err := f.items.Get(f.ctx, store.Where("value", "only"))
			Expect(err).NotTo(HaveOccurred())
			_, err = bson.ObjectIDFromHex(got.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.ID).To(Equal(got.ID))
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
			created := f.mustCreateWidget("w1")
			upd := f.newWidget("w1")
			upd.ID = bson.NewObjectID()

			_, err := f.widgets.Upsert(f.ctx, upd, f.byName("w1"))
			Expect(errors.Is(err, ErrWidgetExists)).To(BeTrue())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(created.ID))
		})

		DescribeTable("should reject an _id filter when the object already has an ID",
			func(opt store.Opt) {
				item := &Item{ID: "new-id", Value: "v"}
				_, err := f.items.Upsert(f.ctx, item, opt)
				Expect(err).To(MatchError(ContainSubstring(
					"_id is specified in both object and filter",
				)))

				n, err := f.items.Count(f.ctx, store.All())
				Expect(err).NotTo(HaveOccurred())
				Expect(n).To(BeZero())
			},
			Entry("plain value", store.Where("_id", "new-id")),
			Entry("$ne", store.Where("_id", bson.M{"$ne": "excluded-id"})),
			Entry("multi-value $in", store.Where("_id", bson.M{"$in": bson.A{"a", "b"}})),
			Entry("inside $and", store.Filter(bson.M{
				"$and": bson.A{bson.M{"_id": "new-id"}},
			})),
		)

		It("should generate a string _id for a non-equality _id condition when the object has no ID", func() {
			item := &Item{Value: "v"}
			_, err := f.items.Upsert(f.ctx, item, store.Where("_id", bson.M{"$ne": "excluded-id"}))
			Expect(err).NotTo(HaveOccurred())

			Expect(rawItemID("v").Type).To(Equal(bson.TypeString))
			items, err := f.items.List(f.ctx, store.Where("value", "v"))
			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveLen(1))
			Expect(items[0].ID).To(Equal(item.ID))
		})

		It("should not use an _id equality from the filter as the inserted ID", func() {
			item := &Item{Value: "v"}
			_, err := f.items.Upsert(f.ctx, item, store.Where("_id", "env-filter"))
			Expect(err).To(MatchError(ContainSubstring("immutable field '_id'")))

			n, err := f.items.Count(f.ctx, store.All())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeZero())
		})
	})
})
