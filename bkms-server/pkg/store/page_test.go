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
	"fmt"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore ListPage", func() {
	var f *storeFixture

	BeforeEach(func() {
		f = setUpStores()
		// w1 ~ w5，按名称升序即创建顺序
		for i := 1; i <= 5; i++ {
			f.mustCreateWidget(fmt.Sprintf("w%d", i))
		}
	})
	AfterEach(func() { f.tearDown() })

	listPage := func(page, size int64, opts ...store.Opt) *store.Page[Widget] {
		opts = append([]store.Opt{f.byOwner(), store.Asc("name")}, opts...)
		result, err := f.widgets.ListPage(f.ctx, page, size, opts...)
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	DescribeTable("should return the requested page with the total count",
		func(page, size int64, names []string, hasNext bool) {
			result := listPage(page, size)
			Expect(widgetNames(result.Items)).To(Equal(names))
			Expect(result.Total).To(Equal(int64(5)))
			Expect(result.HasNext).To(Equal(hasNext))
		},
		Entry("first page", int64(1), int64(2), []string{"w1", "w2"}, true),
		Entry("middle page", int64(2), int64(2), []string{"w3", "w4"}, true),
		Entry("last partial page", int64(3), int64(2), []string{"w5"}, false),
		Entry("last full page", int64(1), int64(5), []string{"w1", "w2", "w3", "w4", "w5"}, false),
		Entry("page beyond the end", int64(4), int64(2), []string{}, false),
	)

	DescribeTable("should report HasNext without counting the total when WithoutTotal is given",
		func(page, size int64, names []string, hasNext bool) {
			result := listPage(page, size, store.WithoutTotal())
			Expect(widgetNames(result.Items)).To(Equal(names))
			Expect(result.Total).To(BeZero())
			Expect(result.HasNext).To(Equal(hasNext))
		},
		Entry("first page", int64(1), int64(2), []string{"w1", "w2"}, true),
		Entry("last partial page", int64(3), int64(2), []string{"w5"}, false),
		Entry("last full page", int64(1), int64(5), []string{"w1", "w2", "w3", "w4", "w5"}, false),
		Entry("page beyond the end", int64(4), int64(2), []string{}, false),
	)

	It("should order items by several sort keys across pages", func() {
		Expect(f.widgets.UpdateMany(f.ctx, map[string]any{"priority": 2},
			f.byOwner(), store.Where("name", bson.M{"$in": bson.A{"w2", "w4"}}),
		)).To(Equal(int64(2)))

		var names []string
		for page := int64(1); page <= 3; page++ {
			result, err := f.widgets.ListPage(f.ctx, page, 2, f.byOwner(), store.Desc("priority"), store.Asc("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Total).To(Equal(int64(5)))
			Expect(result.HasNext).To(Equal(page < 3))
			names = append(names, widgetNames(result.Items)...)
		}
		Expect(names).To(Equal([]string{"w2", "w4", "w1", "w3", "w5"}))
	})

	It("should return an empty non-nil page when nothing matches", func() {
		result, err := f.widgets.ListPage(f.ctx, 1, 10, store.Where("ownerID", "nobody"))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Items).NotTo(BeNil())
		Expect(result.Items).To(BeEmpty())
		Expect(result.Total).To(BeZero())
		Expect(result.HasNext).To(BeFalse())
	})

	It("should apply OmitFields to the items", func() {
		result := listPage(1, 2, store.OmitFields("tags"))
		Expect(result.Items).To(HaveLen(2))
		Expect(result.Items[0].Tags).To(BeNil())
		Expect(result.Items[0].Name).To(Equal("w1"))
	})

	DescribeTable("should reject invalid page or size",
		func(page, size int64) {
			_, err := f.widgets.ListPage(f.ctx, page, size, f.byOwner())
			Expect(err).To(HaveOccurred())
		},
		Entry("zero page", int64(0), int64(10)),
		Entry("negative size", int64(1), int64(-1)),
		Entry("overflowing skip", int64(math.MaxInt64), int64(2)),
		Entry("overflowing size", int64(1), int64(math.MaxInt64)),
	)

	It("should reject Limit", func() {
		_, err := f.widgets.ListPage(f.ctx, 1, 10, f.byOwner(), store.Limit(1))
		Expect(err).To(MatchError(ContainSubstring("limit")))
	})
})
