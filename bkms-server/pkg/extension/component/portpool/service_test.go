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

package portpool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("validateItemNamesUnique", func() {
	It("should pass when item names are unique", func() {
		items := []PoolItem{
			{ItemName: "item-0", StartPort: 30000, EndPort: 30100},
			{ItemName: "item-1", StartPort: 31000, EndPort: 31100},
		}
		Expect(validateItemNamesUnique(items)).To(Succeed())
	})

	It("should fail when item names are duplicated", func() {
		items := []PoolItem{
			{ItemName: "item-0", StartPort: 30000, EndPort: 30100},
			{ItemName: "item-0", StartPort: 31000, EndPort: 31100},
		}
		err := validateItemNamesUnique(items)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("item-0 already exists"))
	})
})

var _ = Describe("validateEndPortNotShrunk", func() {
	var oldItems []PoolItem

	BeforeEach(func() {
		oldItems = []PoolItem{
			{ItemName: "item-0", EndPort: 30100},
			{ItemName: "item-1", EndPort: 31100},
		}
	})

	It("should pass when endPort is not shrunk", func() {
		newItems := []PoolItem{
			{ItemName: "item-0", EndPort: 30200},
			{ItemName: "item-1", EndPort: 31100},
		}
		Expect(validateItemUpdate(oldItems, newItems)).To(Succeed())
	})

	It("should fail when endPort is shrunk", func() {
		newItems := []PoolItem{
			{ItemName: "item-0", EndPort: 30000},
		}
		err := validateItemUpdate(oldItems, newItems)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("must not be less than"))
	})

	It("should pass for new items not in old list", func() {
		newItems := []PoolItem{
			{ItemName: "item-2", EndPort: 100},
		}
		Expect(validateItemUpdate(oldItems, newItems)).To(Succeed())
	})

	It("should pass when old item is removed", func() {
		newItems := []PoolItem{
			{ItemName: "item-0", EndPort: 30200},
		}
		Expect(validateItemUpdate(oldItems, newItems)).To(Succeed())
	})
})

var _ = Describe("fillItemNames", func() {
	It("should auto-fill itemName based on existing max index", func() {
		items := []PoolItem{
			{ItemName: "item-0", StartPort: 30000, EndPort: 30100},
			{ItemName: "item-1", StartPort: 31000, EndPort: 31100},
			{StartPort: 32000, EndPort: 32100},
		}
		fillEmptyItemNames(items)
		Expect(items[2].ItemName).To(Equal("item-2"))
	})

	It("should auto-fill with next index when existing items have gaps", func() {
		items := []PoolItem{
			{ItemName: "item-0", StartPort: 30000, EndPort: 30100},
			{ItemName: "item-3", StartPort: 31000, EndPort: 31100},
			{StartPort: 32000, EndPort: 32100},
		}
		fillEmptyItemNames(items)
		Expect(items[2].ItemName).To(Equal("item-4"))
	})

	It("should auto-fill with next index when existing items have gaps", func() {
		items := []PoolItem{
			{ItemName: "item-0", StartPort: 30000, EndPort: 30100},
			{StartPort: 32000, EndPort: 32100},
			{ItemName: "item-3", StartPort: 31000, EndPort: 31100},
		}
		fillEmptyItemNames(items)
		Expect(items[1].ItemName).To(Equal("item-4"))
	})

	It("should auto-fill sequential indices for multiple empty-name items", func() {
		items := []PoolItem{
			{ItemName: "item-1", StartPort: 30000, EndPort: 30100},
			{StartPort: 31000, EndPort: 31100},
			{StartPort: 32000, EndPort: 32100},
		}
		fillEmptyItemNames(items)
		Expect(items[1].ItemName).To(Equal("item-2"))
		Expect(items[2].ItemName).To(Equal("item-3"))
	})
})
