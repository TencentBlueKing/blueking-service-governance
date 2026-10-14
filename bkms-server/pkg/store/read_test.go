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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore read operations", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	Describe("Get", func() {
		It("should return the document matching the filter", func() {
			created := f.mustCreateWidget("w1")

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ID).To(Equal(created.ID))
			Expect(got.Priority).To(Equal(1))
		})

		It("should wrap both the domain sentinel and store.ErrNotFound when missing", func() {
			_, err := f.widgets.Get(f.ctx, f.byName("missing"))
			Expect(errors.Is(err, ErrWidgetNotFound)).To(BeTrue())
			Expect(errors.Is(err, store.ErrNotFound)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring(widgetCollection))
		})

		It("should fall back to store.ErrNotFound when meta has no NotFound", func() {
			_, err := f.items.Get(f.ctx, store.Where("_id", "missing"))
			Expect(errors.Is(err, store.ErrNotFound)).To(BeTrue())
		})
	})

	Describe("List", func() {
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
			Expect(list).To(HaveLen(3))
		})

		It("should sort by the given keys in order", func() {
			list, err := f.widgets.List(f.ctx, f.byOwner(), store.Asc("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(list)).To(Equal([]string{"a", "b", "c"}))

			list, err = f.widgets.List(f.ctx, f.byOwner(), store.Desc("priority"))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(list)).To(Equal([]string{"c", "a", "b"}))
		})

		It("should support multiple sort keys", func() {
			w := f.newWidget("d")
			w.Priority = 3
			Expect(f.widgets.Create(f.ctx, w)).To(Succeed())

			list, err := f.widgets.List(
				f.ctx, f.byOwner(), store.Desc("priority"), store.Desc("name"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(list)).To(Equal([]string{"d", "c", "a", "b"}))
		})

		It("should limit the number of results", func() {
			list, err := f.widgets.List(f.ctx, f.byOwner(), store.Asc("name"), store.Limit(2))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(list)).To(Equal([]string{"a", "b"}))
		})

		It("should return an empty non-nil slice when nothing matches", func() {
			list, err := f.widgets.List(f.ctx, store.Where("ownerID", "nobody"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).NotTo(BeNil())
			Expect(list).To(BeEmpty())
		})

		It("should treat a nil filter as matching all documents", func() {
			list, err := f.widgets.List(f.ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(list)).To(BeNumerically(">=", 3))
		})
	})

	Describe("Exists and Count", func() {
		It("should report existence and count of matching documents", func() {
			f.mustCreateWidget("w1")
			f.mustCreateWidget("w2")

			ok, err := f.widgets.Exists(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())

			ok, err = f.widgets.Exists(f.ctx, f.byName("missing"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())

			n, err := f.widgets.Count(f.ctx, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))
		})
	})
})

func widgetNames(list []Widget) []string {
	names := make([]string, 0, len(list))
	for _, w := range list {
		names = append(names, w.Name)
	}
	return names
}
