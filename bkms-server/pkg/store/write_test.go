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
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore write operations", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	Describe("Create", func() {
		It("should fill ObjectID and timestamps back into the object", func() {
			w := f.newWidget("w1")
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			Expect(w.ID.IsZero()).To(BeFalse())
			Expect(w.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(w.UpdatedAt).To(Equal(w.CreatedAt))

			got, err := f.widgets.Get(f.ctx, store.Where("_id", w.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
			Expect(got.CreatedAt).To(BeTemporally("~", w.CreatedAt, time.Millisecond))
		})

		It("should keep a preset CreatedAt", func() {
			preset := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
			w := f.newWidget("w1")
			w.CreatedAt = preset
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.CreatedAt.Equal(preset)).To(BeTrue())
		})

		It("should generate a hex string _id when the string ID is empty", func() {
			e := &Item{Value: "v"}
			Expect(f.items.Create(f.ctx, e)).To(Succeed())
			_, err := bson.ObjectIDFromHex(e.ID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should keep a preset string _id", func() {
			Expect(f.items.Create(f.ctx, &Item{ID: "env-1", Value: "v"})).To(Succeed())
			got, err := f.items.Get(f.ctx, store.Where("_id", "env-1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Value).To(Equal("v"))
		})

		It("should recognize timestamps embedded inline", func() {
			n := &Note{Body: "hello"}
			Expect(f.notes.Create(f.ctx, n)).To(Succeed())
			Expect(n.CreatedAt.IsZero()).To(BeFalse())
			Expect(n.UpdatedAt.IsZero()).To(BeFalse())
		})

		It("should wrap both the domain sentinel and store.ErrDuplicate on unique key conflict", func() {
			f.mustCreateWidget("w1")
			err := f.widgets.Create(f.ctx, f.newWidget("w1"))
			Expect(errors.Is(err, ErrWidgetExists)).To(BeTrue())
			Expect(errors.Is(err, store.ErrDuplicate)).To(BeTrue())

			// 仍可取到驱动的原始错误，以便查看冲突的键
			var we mongo.WriteException
			Expect(errors.As(err, &we)).To(BeTrue())
		})

		It("should fall back to store.ErrDuplicate when meta has no Duplicate", func() {
			Expect(f.items.Create(f.ctx, &Item{ID: "dup"})).To(Succeed())
			err := f.items.Create(f.ctx, &Item{ID: "dup"})
			Expect(errors.Is(err, store.ErrDuplicate)).To(BeTrue())
		})
	})

	Describe("CreateMany", func() {
		It("should insert all objects and fill IDs and timestamps", func() {
			ws := []*Widget{f.newWidget("w1"), f.newWidget("w2")}
			Expect(f.widgets.CreateMany(f.ctx, ws)).To(Succeed())

			for _, w := range ws {
				Expect(w.ID.IsZero()).To(BeFalse())
				Expect(w.CreatedAt.IsZero()).To(BeFalse())
			}
			n, err := f.widgets.Count(f.ctx, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))
		})

		It("should do nothing for an empty slice", func() {
			Expect(f.widgets.CreateMany(f.ctx, nil)).To(Succeed())
		})

		It("should return a duplicate error on unique key conflict", func() {
			f.mustCreateWidget("w1")
			err := f.widgets.CreateMany(f.ctx, []*Widget{f.newWidget("w1")})
			Expect(errors.Is(err, ErrWidgetExists)).To(BeTrue())
		})
	})

	Describe("CreateManyIfAbsent", func() {
		It("should skip existing documents and insert the rest", func() {
			Expect(f.items.Create(f.ctx, &Item{ID: "e1", Value: "old"})).To(Succeed())

			err := f.items.CreateManyIfAbsent(f.ctx, []*Item{
				{ID: "e1", Value: "new"}, {ID: "e2", Value: "new"},
			})
			Expect(err).NotTo(HaveOccurred())

			e1, err := f.items.Get(f.ctx, store.Where("_id", "e1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(e1.Value).To(Equal("old"))
			_, err = f.items.Get(f.ctx, store.Where("_id", "e2"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Update", func() {
		It("should $set the given fields and refresh updatedAt only", func() {
			w := f.mustCreateWidget("w1")
			time.Sleep(10 * time.Millisecond)

			Expect(f.widgets.Update(f.ctx, bson.M{"priority": 5}, f.byName("w1"))).To(Succeed())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Priority).To(Equal(5))
			Expect(got.Tags).To(Equal([]string{"a"}))
			Expect(got.CreatedAt).To(BeTemporally("~", w.CreatedAt, time.Millisecond))
			Expect(got.UpdatedAt).To(BeTemporally(">", w.UpdatedAt))
		})

		It("should return NotFound when nothing matches", func() {
			err := f.widgets.Update(f.ctx, bson.M{"priority": 5}, f.byName("missing"))
			Expect(errors.Is(err, ErrWidgetNotFound)).To(BeTrue())
		})

		It("should return a duplicate error on unique key conflict", func() {
			f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")
			err := f.widgets.Update(f.ctx, bson.M{"name": "w1"}, f.byName("w2"))
			Expect(errors.Is(err, ErrWidgetExists)).To(BeTrue())
		})

		It("should be a no-op for an empty set", func() {
			Expect(f.widgets.Update(f.ctx, bson.M{}, f.byName("missing"))).To(Succeed())
		})
	})

	Describe("UpdateFields", func() {
		It("should only update the selected fields from the object", func() {
			f.mustCreateWidget("w1")

			patch := f.newWidget("w1")
			patch.Priority = 9
			patch.Tags = []string{"changed"}
			Expect(f.widgets.UpdateFields(f.ctx, patch, []string{"priority"}, f.byName("w1"))).To(Succeed())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Priority).To(Equal(9))
			Expect(got.Tags).To(Equal([]string{"a"}))
		})

		It("should support nested field paths", func() {
			w := f.newWidget("w1")
			w.Spec = WidgetSpec{Color: "red", Size: 1}
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			patch := &Widget{Spec: WidgetSpec{Color: "blue", Size: 2}}
			Expect(f.widgets.UpdateFields(f.ctx, patch, []string{"spec.color"}, f.byName("w1"))).To(Succeed())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Spec).To(Equal(WidgetSpec{Color: "blue", Size: 1}))
		})

		It("should unset a field that is omitted as empty", func() {
			w := f.newWidget("w1")
			w.Secret = &Credential{Username: "u", Password: "p"}
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			Expect(f.widgets.UpdateFields(f.ctx, &Widget{}, []string{"secret"}, f.byName("w1"))).To(Succeed())

			_, err := f.rawWidget("w1").LookupErr("secret")
			Expect(err).To(HaveOccurred())
		})

		It("should reject unknown field paths", func() {
			f.mustCreateWidget("w1")
			err := f.widgets.UpdateFields(f.ctx, f.newWidget("w1"), []string{"nonexistent"}, f.byName("w1"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nonexistent"))
		})

		It("should return NotFound when nothing matches", func() {
			err := f.widgets.UpdateFields(f.ctx, f.newWidget("x"), []string{"priority"}, f.byName("missing"))
			Expect(errors.Is(err, ErrWidgetNotFound)).To(BeTrue())
		})
	})

	Describe("Replace", func() {
		It("should replace the whole document and refresh updatedAt", func() {
			created := f.mustCreateWidget("w1")
			time.Sleep(10 * time.Millisecond)

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			got.Priority = 8
			got.Tags = nil
			Expect(f.widgets.Replace(f.ctx, got, store.Where("_id", got.ID))).To(Succeed())

			replaced, err := f.widgets.Get(f.ctx, store.Where("_id", created.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(replaced.Priority).To(Equal(8))
			Expect(replaced.Tags).To(BeEmpty())
			Expect(replaced.CreatedAt).To(BeTemporally("~", created.CreatedAt, time.Millisecond))
			Expect(replaced.UpdatedAt).To(BeTemporally(">", created.UpdatedAt))
		})

		It("should ignore the _id of the object and keep the stored one", func() {
			created := f.mustCreateWidget("w1")
			w := f.newWidget("w1")
			w.ID = bson.NewObjectID()
			Expect(f.widgets.Replace(f.ctx, w, f.byName("w1"))).To(Succeed())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(created.ID))
		})

		It("should return NotFound when nothing matches", func() {
			err := f.widgets.Replace(f.ctx, f.newWidget("missing"), f.byName("missing"))
			Expect(errors.Is(err, ErrWidgetNotFound)).To(BeTrue())
		})
	})
})

var _ = Describe("GenericStore delete operations", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	It("Delete should remove the matching document", func() {
		f.mustCreateWidget("w1")
		Expect(f.widgets.Delete(f.ctx, f.byName("w1"))).To(Succeed())

		ok, err := f.widgets.Exists(f.ctx, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	It("Delete should return NotFound when nothing matches", func() {
		err := f.widgets.Delete(f.ctx, f.byName("missing"))
		Expect(errors.Is(err, ErrWidgetNotFound)).To(BeTrue())
		Expect(errors.Is(err, store.ErrNotFound)).To(BeTrue())
	})

	It("DeleteIfExists should succeed silently when nothing matches", func() {
		Expect(f.widgets.DeleteIfExists(f.ctx, f.byName("missing"))).To(Succeed())
	})

	It("DeleteIfExists should remove the matching document", func() {
		f.mustCreateWidget("w1")
		Expect(f.widgets.DeleteIfExists(f.ctx, f.byName("w1"))).To(Succeed())

		ok, err := f.widgets.Exists(f.ctx, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	It("DeleteMany should remove all matching documents and report the count", func() {
		f.mustCreateWidget("w1")
		f.mustCreateWidget("w2")

		n, err := f.widgets.DeleteMany(f.ctx, f.byOwner())
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(int64(2)))
	})
})
