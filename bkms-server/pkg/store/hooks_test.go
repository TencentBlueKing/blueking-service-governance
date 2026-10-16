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

var _ = Describe("GenericStore hooks", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	Describe("BeforeSave", func() {
		It("should apply defaults to the caller's object and the stored document", func() {
			w := f.newWidget("w1")
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			Expect(w.Spec.Color).To(Equal(defaultColor))
			Expect(rawString(f.rawWidget("w1"), "spec", "color")).To(Equal(defaultColor))
		})

		// 各写操作传入缺少名称的对象，requireWidgetName 会让 BeforeSave 失败
		DescribeTable("should run on writes of whole objects and abort them when failing",
			func(write func(nameless *Widget) error) {
				f.mustCreateWidget("w1")
				Expect(write(&Widget{OwnerID: f.owner})).To(MatchError(errNameRequired))
				Expect(f.countWidgets()).To(Equal(int64(1)))
			},
			Entry("Create", func(w *Widget) error { return f.widgets.Create(f.ctx, w) }),
			Entry("CreateMany", func(w *Widget) error { return f.widgets.CreateMany(f.ctx, []*Widget{w}) }),
			Entry("CreateManyIfAbsent", func(w *Widget) error {
				return f.widgets.CreateManyIfAbsent(f.ctx, []*Widget{w})
			}),
			Entry("Replace", func(w *Widget) error { return f.widgets.Replace(f.ctx, w, f.byName("w1")) }),
			Entry("Upsert with an object", func(w *Widget) error {
				_, err := f.widgets.Upsert(f.ctx, w, f.byName("w1"))
				return err
			}),
		)

		// 这些写操作的对象可能只填写了部分字段，执行 BeforeSave 会误判
		DescribeTable("should not run on partial writes",
			func(write func() error) {
				f.mustCreateWidget("w1")
				Expect(write()).To(Succeed())
			},
			Entry("Update with an object", func() error {
				return f.widgets.Update(f.ctx, &Widget{Priority: 2}, store.UpdateSelected("priority"), f.byName("w1"))
			}),
			Entry("Update with a map", func() error {
				return f.widgets.Update(f.ctx, map[string]any{"priority": 2}, f.byName("w1"))
			}),
			Entry("Upsert with a map that inserts", func() error {
				_, err := f.widgets.Upsert(f.ctx, map[string]any{"priority": 2}, f.byName("w2"))
				return err
			}),
		)
	})

	Describe("Encode", func() {
		// write 写入名为 w1、密码为 plainPassword 的凭证，返回传入的对象（传入 map 时为 nil），
		// 用于断言 Encode 只作用于副本
		DescribeTable("should encode the stored document on every write without touching the caller's object",
			func(write func() (*Widget, error)) {
				obj, err := write()
				Expect(err).NotTo(HaveOccurred())

				Expect(f.rawPassword("w1")).To(Equal(encPrefix + plainPassword))
				if obj != nil {
					Expect(obj.Secret.Password).To(Equal(plainPassword))
				}
			},
			Entry("Create", func() (*Widget, error) {
				w := f.newSecretWidget("w1")
				return w, f.widgets.Create(f.ctx, w)
			}),
			Entry("CreateMany", func() (*Widget, error) {
				w := f.newSecretWidget("w1")
				return w, f.widgets.CreateMany(f.ctx, []*Widget{w})
			}),
			Entry("CreateManyIfAbsent", func() (*Widget, error) {
				w := f.newSecretWidget("w1")
				return w, f.widgets.CreateManyIfAbsent(f.ctx, []*Widget{w})
			}),
			Entry("Replace", func() (*Widget, error) {
				f.mustCreateWidget("w1")
				w := f.newSecretWidget("w1")
				return w, f.widgets.Replace(f.ctx, w, f.byName("w1"))
			}),
			Entry("Upsert with an object", func() (*Widget, error) {
				w := f.newSecretWidget("w1")
				_, err := f.widgets.Upsert(f.ctx, w, f.byName("w1"))
				return w, err
			}),
			Entry("Upsert with a map", func() (*Widget, error) {
				secret := &Credential{Username: "u", Password: plainPassword}
				_, err := f.widgets.Upsert(f.ctx, map[string]any{"secret": secret}, f.byName("w1"))
				Expect(secret.Password).To(Equal(plainPassword))
				return nil, err
			}),
			Entry("Update with an object", func() (*Widget, error) {
				f.mustCreateWidget("w1")
				w := f.newSecretWidget("w1")
				return w, f.widgets.Update(f.ctx, w, store.UpdateSelected("secret"), f.byName("w1"))
			}),
			Entry("Update with a map", func() (*Widget, error) {
				f.mustCreateWidget("w1")
				return nil, f.widgets.Update(f.ctx, bson.M{"secret.password": plainPassword}, f.byName("w1"))
			}),
		)

		// map 局部更新时 Encode 作用于其余字段均为零值的对象，已加密的字段不应被覆盖、清除或重复加密
		DescribeTable("should leave encoded fields absent from a partial update untouched",
			func(fields map[string]any) {
				Expect(f.widgets.Create(f.ctx, f.newSecretWidget("w1"))).To(Succeed())

				Expect(f.widgets.Update(f.ctx, fields, f.byName("w1"))).To(Succeed())

				Expect(f.rawPassword("w1")).To(Equal(encPrefix + plainPassword))
				Expect(f.mustGetWidget("w1").Secret.Password).To(Equal(plainPassword))
			},
			Entry("an unrelated top-level field", map[string]any{"priority": 2}),
			Entry("a sibling of the encoded field", map[string]any{"secret.username": "u2"}),
		)
	})

	Describe("Decode", func() {
		It("should decode documents returned by Get, List and GetAndDelete", func() {
			Expect(f.widgets.Create(f.ctx, f.newSecretWidget("w1"))).To(Succeed())

			Expect(f.mustGetWidget("w1").Secret.Password).To(Equal(plainPassword))

			list, err := f.widgets.List(f.ctx, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Secret.Password).To(Equal(plainPassword))

			deleted, err := f.widgets.GetAndDelete(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(deleted.Secret.Password).To(Equal(plainPassword))
		})

		It("should propagate decode failures", func() {
			_, err := f.widgets.Collection().InsertOne(f.ctx, bson.M{
				"ownerID": f.owner, "name": "bad",
				"secret": bson.M{"username": "u", "password": "not-encrypted"},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = f.widgets.Get(f.ctx, f.byName("bad"))
			Expect(err).To(MatchError(errNotEncrypted))

			_, err = f.widgets.List(f.ctx, f.byOwner())
			Expect(err).To(MatchError(errNotEncrypted))

			_, err = f.widgets.GetAndDelete(f.ctx, f.byName("bad"))
			Expect(err).To(MatchError(errNotEncrypted))
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
		meta.BeforeSave = []func(*Widget) error{record("before-save-1"), record("before-save-2")}
		meta.Encode = []func(*Widget) error{record("encode-1"), record("encode-2")}
		meta.Decode = []func(*Widget) error{record("decode-1"), record("decode-2")}
		s := store.Of(f.db, meta)

		Expect(s.Create(f.ctx, f.newWidget("w1"))).To(Succeed())
		_, err := s.Get(f.ctx, f.byName("w1"))
		Expect(err).NotTo(HaveOccurred())
		Expect(calls).To(Equal([]string{
			"before-save-1", "before-save-2", "encode-1", "encode-2", "decode-1", "decode-2",
		}))
	})
})
