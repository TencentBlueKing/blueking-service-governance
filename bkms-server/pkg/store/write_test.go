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

			Expect(w.ID).NotTo(BeZero())
			Expect(w.CreatedAt).To(BeTemporally("~", time.Now(), time.Second))
			Expect(w.UpdatedAt).To(Equal(w.CreatedAt))

			got, err := f.widgets.Get(f.ctx, store.Where("_id", w.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
			Expect(got.CreatedAt).To(beStoredTime(w.CreatedAt))
		})

		It("should keep a preset createdAt and use it as updatedAt", func() {
			w := f.newWidget("w1")
			w.CreatedAt = longAgo
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			got := f.mustGetWidget("w1")
			Expect(got.CreatedAt).To(beStoredTime(longAgo))
			Expect(got.UpdatedAt).To(beStoredTime(longAgo))
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
			Expect(n.CreatedAt).NotTo(BeZero())
			Expect(n.UpdatedAt).To(Equal(n.CreatedAt))
		})

		It("should wrap both the domain sentinel and store.ErrDuplicate on unique key conflict", func() {
			f.mustCreateWidget("w1")
			err := f.widgets.Create(f.ctx, f.newWidget("w1"))
			Expect(err).To(MatchError(ErrWidgetExists))
			Expect(err).To(MatchError(store.ErrDuplicate))

			// 仍可取到驱动的原始错误，以便查看冲突的键
			var we mongo.WriteException
			Expect(errors.As(err, &we)).To(BeTrue())
		})

		It("should fall back to store.ErrDuplicate when meta has no Duplicate", func() {
			Expect(f.items.Create(f.ctx, &Item{ID: "dup"})).To(Succeed())
			Expect(f.items.Create(f.ctx, &Item{ID: "dup"})).To(MatchError(store.ErrDuplicate))
		})
	})

	Describe("CreateMany", func() {
		It("should insert all objects and fill IDs and timestamps", func() {
			ws := []*Widget{f.newWidget("w1"), f.newWidget("w2")}
			Expect(f.widgets.CreateMany(f.ctx, ws)).To(Succeed())

			for _, w := range ws {
				Expect(w.ID).NotTo(BeZero())
				Expect(w.CreatedAt).NotTo(BeZero())
			}
			Expect(f.countWidgets()).To(Equal(int64(2)))
		})

		It("should do nothing for an empty slice", func() {
			Expect(f.widgets.CreateMany(f.ctx, nil)).To(Succeed())
		})

		It("should stop at the first conflict and keep the documents inserted before it", func() {
			f.mustCreateWidget("w2")
			err := f.widgets.CreateMany(f.ctx, []*Widget{f.newWidget("w1"), f.newWidget("w2"), f.newWidget("w3")})
			Expect(err).To(MatchError(ErrWidgetExists))

			Expect(f.widgetExists("w1")).To(BeTrue())
			Expect(f.widgetExists("w3")).To(BeFalse())
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
			Expect(f.items.Exists(f.ctx, store.Where("_id", "e2"))).To(BeTrue())
		})

		It("should do nothing for an empty slice", func() {
			Expect(f.items.CreateManyIfAbsent(f.ctx, nil)).To(Succeed())
		})
	})

	Describe("Replace", func() {
		It("should replace the whole document and refresh updatedAt", func() {
			created := f.mustCreateStaleWidget("w1")

			got := f.mustGetWidget("w1")
			got.Priority = 8
			got.Tags = nil
			Expect(f.widgets.Replace(f.ctx, got, store.Where("_id", got.ID))).To(Succeed())
			Expect(got.UpdatedAt).To(BeTemporally("~", time.Now(), time.Second))

			replaced := f.mustGetWidget("w1")
			Expect(replaced.Priority).To(Equal(8))
			Expect(replaced.Tags).To(BeEmpty())
			Expect(replaced.CreatedAt).To(beStoredTime(created.CreatedAt))
			Expect(replaced.UpdatedAt).To(beStoredTime(got.UpdatedAt))
		})

		It("should ignore the _id of the object and keep the stored one", func() {
			created := f.mustCreateWidget("w1")
			w := f.newWidget("w1")
			w.ID = bson.NewObjectID()
			Expect(f.widgets.Replace(f.ctx, w, f.byName("w1"))).To(Succeed())

			Expect(f.mustGetWidget("w1").ID).To(Equal(created.ID))
		})

		It("should replace documents of models without timestamps", func() {
			Expect(f.items.Create(f.ctx, &Item{ID: "e1", Value: "old"})).To(Succeed())
			Expect(f.items.Replace(f.ctx, &Item{Value: "new"}, store.Where("_id", "e1"))).To(Succeed())

			got, err := f.items.Get(f.ctx, store.Where("_id", "e1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Value).To(Equal("new"))
		})

		It("should return NotFound when nothing matches", func() {
			err := f.widgets.Replace(f.ctx, f.newWidget("missing"), f.byName("missing"))
			Expect(err).To(MatchError(ErrWidgetNotFound))
		})
	})
})

var _ = Describe("GenericStore delete operations", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	Describe("Delete", func() {
		It("should remove the matching document", func() {
			f.mustCreateWidget("w1")
			Expect(f.widgets.Delete(f.ctx, f.byName("w1"))).To(Succeed())
			Expect(f.widgetExists("w1")).To(BeFalse())
		})

		It("should remove only one document when several match", func() {
			f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")
			Expect(f.widgets.Delete(f.ctx, f.byOwner())).To(Succeed())
			Expect(f.countWidgets()).To(Equal(int64(1)))
		})

		It("should return NotFound when nothing matches", func() {
			err := f.widgets.Delete(f.ctx, f.byName("missing"))
			Expect(err).To(MatchError(ErrWidgetNotFound))
			Expect(err).To(MatchError(store.ErrNotFound))
		})
	})

	Describe("DeleteIfExists", func() {
		It("should remove the matching document", func() {
			f.mustCreateWidget("w1")
			Expect(f.widgets.DeleteIfExists(f.ctx, f.byName("w1"))).To(Succeed())
			Expect(f.widgetExists("w1")).To(BeFalse())
		})

		It("should succeed silently when nothing matches", func() {
			Expect(f.widgets.DeleteIfExists(f.ctx, f.byName("missing"))).To(Succeed())
		})
	})

	Describe("DeleteMany", func() {
		It("should remove all matching documents and report the count", func() {
			f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")
			f.mustCreateWidget("other")

			n, err := f.widgets.DeleteMany(f.ctx, f.byOwner(), store.Where("name", bson.M{"$in": bson.A{"w1", "w2"}}))
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))
			Expect(f.widgetExists("other")).To(BeTrue())
		})

		It("should report zero when nothing matches", func() {
			Expect(f.widgets.DeleteMany(f.ctx, f.byName("missing"))).To(BeZero())
		})
	})

	Describe("GetAndDelete", func() {
		It("should remove the matching document and return it decoded", func() {
			w := f.newSecretWidget("w1")
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			deleted, err := f.widgets.GetAndDelete(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.ID).To(Equal(w.ID))
			Expect(deleted.Secret.Password).To(Equal(plainPassword))
			Expect(f.widgetExists("w1")).To(BeFalse())
		})

		It("should delete the first document in the given sort order", func() {
			for i, name := range []string{"a", "b"} {
				w := f.newWidget(name)
				w.Priority = i
				Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
			}

			deleted, err := f.widgets.GetAndDelete(f.ctx, f.byOwner(), store.Desc("priority"))
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.Name).To(Equal("b"))
			Expect(f.widgetExists("a")).To(BeTrue())
		})

		It("should return NotFound when nothing matches", func() {
			_, err := f.widgets.GetAndDelete(f.ctx, f.byName("missing"))
			Expect(err).To(MatchError(ErrWidgetNotFound))
			Expect(err).To(MatchError(store.ErrNotFound))
		})
	})
})
