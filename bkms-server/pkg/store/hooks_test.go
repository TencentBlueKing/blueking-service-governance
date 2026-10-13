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
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore hooks", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	newSecretWidget := func(name string) *Widget {
		w := f.newWidget(name)
		w.Secret = &Credential{Username: "u", Password: "plain"}
		return w
	}

	rawPassword := func(name string) string {
		return rawString(f.rawWidget(name), "secret", "password")
	}

	Describe("BeforeSave", func() {
		It("should apply defaults to the caller's object and the stored document", func() {
			w := f.newWidget("w1")
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			Expect(w.Spec.Color).To(Equal(defaultColor))
			Expect(rawString(f.rawWidget("w1"), "spec", "color")).To(Equal(defaultColor))
		})

		It("should abort the write when it fails", func() {
			err := f.widgets.Create(f.ctx, &Widget{OwnerID: f.owner})
			Expect(errors.Is(err, errNameRequired)).To(BeTrue())

			n, err := f.widgets.Count(f.ctx, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeZero())
		})

		It("should run on Upsert and Replace as well", func() {
			_, err := f.widgets.Upsert(f.ctx, &Widget{OwnerID: f.owner}, f.byName("w1"))
			Expect(err).To(MatchError(errNameRequired))

			f.mustCreateWidget("w1")
			Expect(f.widgets.Replace(f.ctx, &Widget{OwnerID: f.owner}, f.byName("w1"))).
				To(MatchError(errNameRequired))
		})
	})

	Describe("Encode", func() {
		It("should encode the stored document without touching the caller's object", func() {
			w := newSecretWidget("w1")
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			Expect(w.Secret.Password).To(Equal("plain"))
			Expect(rawPassword("w1")).To(Equal(encPrefix + "plain"))
		})

		It("should encode on CreateMany, CreateManyIfAbsent, Upsert, Replace and UpdateFields", func() {
			Expect(f.widgets.CreateMany(f.ctx, []*Widget{newSecretWidget("many")})).To(Succeed())
			Expect(rawPassword("many")).To(Equal(encPrefix + "plain"))

			Expect(f.widgets.CreateManyIfAbsent(f.ctx, []*Widget{newSecretWidget("if-absent")})).To(Succeed())
			Expect(rawPassword("if-absent")).To(Equal(encPrefix + "plain"))

			_, err := f.widgets.Upsert(f.ctx, newSecretWidget("upsert"), f.byName("upsert"))
			Expect(err).NotTo(HaveOccurred())
			Expect(rawPassword("upsert")).To(Equal(encPrefix + "plain"))

			Expect(f.widgets.Replace(f.ctx, newSecretWidget("upsert"), f.byName("upsert"))).To(Succeed())
			Expect(rawPassword("upsert")).To(Equal(encPrefix + "plain"))

			patch := &Widget{Secret: &Credential{Username: "u2", Password: "new"}}
			Expect(f.widgets.UpdateFields(f.ctx, patch, []string{"secret"}, f.byName("upsert"))).To(Succeed())
			Expect(rawPassword("upsert")).To(Equal(encPrefix + "new"))
			Expect(patch.Secret.Password).To(Equal("new"))
		})
	})

	Describe("Decode", func() {
		It("should decode documents returned by Get and List", func() {
			Expect(f.widgets.Create(f.ctx, newSecretWidget("w1"))).To(Succeed())

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Secret.Password).To(Equal("plain"))

			list, err := f.widgets.List(f.ctx, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Secret.Password).To(Equal("plain"))
		})

		It("should propagate decode failures", func() {
			_, err := f.widgets.Collection().InsertOne(f.ctx, bson.M{
				"ownerID": f.owner, "name": "bad",
				"secret": bson.M{"username": "u", "password": "not-encrypted"},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.widgets.Get(f.ctx, f.byName("bad"))
			Expect(err).To(MatchError(ContainSubstring("password is not encrypted")))

			_, err = f.widgets.List(f.ctx, f.byOwner())
			Expect(err).To(MatchError(ContainSubstring("password is not encrypted")))
		})
	})

	It("should run multiple hooks of the same kind in declaration order", func() {
		var calls []string
		record := func(name string) func(*Widget) error {
			return func(*Widget) error {
				calls = append(calls, name)
				return nil
			}
		}
		meta := widgetMeta
		meta.Encode = []func(*Widget) error{record("encode-1"), record("encode-2")}
		meta.Decode = []func(*Widget) error{record("decode-1"), record("decode-2")}
		s := store.Of(f.db, meta)

		Expect(s.Create(f.ctx, f.newWidget("w1"))).To(Succeed())
		_, err := s.Get(f.ctx, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(calls).To(Equal([]string{"encode-1", "encode-2", "decode-1", "decode-2"}))
	})

	It("should not run Encode hooks on Update", func() {
		Expect(f.widgets.Create(f.ctx, newSecretWidget("w1"))).To(Succeed())

		Expect(f.widgets.Update(f.ctx, bson.M{"secret.password": "raw"}, f.byName("w1"))).To(Succeed())
		Expect(rawPassword("w1")).To(Equal("raw"))
	})
})
