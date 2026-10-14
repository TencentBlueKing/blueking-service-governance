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
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("GenericStore aggregation", func() {
	var f *storeFixture

	BeforeEach(func() { f = setUpStores() })
	AfterEach(func() { f.tearDown() })

	// mustCreateColored 创建一个带凭证、指定颜色与优先级的 Widget
	mustCreateColored := func(name, color string, priority int) {
		w := f.newSecretWidget(name)
		w.Spec.Color = color
		w.Priority = priority
		Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
	}

	Describe("ListLatestBy", func() {
		BeforeEach(func() {
			mustCreateColored("r1", "red", 1)
			mustCreateColored("r3", "red", 3)
			mustCreateColored("b2", "blue", 2)
			mustCreateColored("b1", "blue", 1)
		})

		It("should return the first document of each group in sort order", func() {
			got, err := f.widgets.ListLatestBy(f.ctx, []string{"spec.color"},
				f.byOwner(), store.Asc("spec.color"), store.Desc("priority"))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(got)).To(Equal([]string{"b2", "r3"}))
			// 结果经过 Decode
			Expect(got[0].Secret.Password).To(Equal(plainPassword))
		})

		It("should accept Go field names and several group fields", func() {
			got, err := f.widgets.ListLatestBy(f.ctx, []string{"Spec.Color", "Priority"},
				f.byOwner(), store.Desc("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(got)).To(ConsistOf("r1", "r3", "b2", "b1"))
		})

		It("should only group the documents matching the filter", func() {
			got, err := f.widgets.ListLatestBy(f.ctx, []string{"spec.color"},
				f.byOwner(), store.Where("priority", bson.M{"$lt": 3}), store.Desc("priority"))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(got)).To(ConsistOf("b2", "r1"))
		})

		It("should group documents without the field together", func() {
			f.mustCreateWidget("plain")

			got, err := f.widgets.ListLatestBy(f.ctx, []string{"secret.username"}, f.byOwner(), store.Asc("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(widgetNames(got)).To(Equal([]string{"b1", "plain"}))
		})

		It("should skip the omitted fields", func() {
			got, err := f.widgets.ListLatestBy(f.ctx, []string{"spec.color"},
				f.byOwner(), store.Desc("priority"), store.OmitFields("secret"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(2))
			Expect(got[0].Secret).To(BeNil())
		})

		It("should return an empty slice when nothing matches", func() {
			got, err := f.widgets.ListLatestBy(f.ctx, []string{"spec.color"},
				f.byName("missing"), store.Desc("priority"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got).NotTo(BeNil())
			Expect(got).To(BeEmpty())
		})

		DescribeTable("should reject invalid arguments",
			func(groupBy []string, opts []store.Opt, msg string) {
				_, err := f.widgets.ListLatestBy(f.ctx, groupBy, opts...)
				Expect(err).To(MatchError(ContainSubstring(msg)))
			},
			Entry("without a sort option", []string{"name"}, []store.Opt{store.All()}, "sort option is required"),
			Entry("without group fields", nil, []store.Opt{store.Desc("name")}, "fields to group by are required"),
			Entry("with a limit", []string{"name"}, []store.Opt{store.Desc("name"), store.Limit(1)}, "limit"),
		)
	})

	Describe("Aggregate", func() {
		type colorCount struct {
			Color string `bson:"_id"`
			Count int64  `bson:"count"`
		}

		It("should decode the results into the given type", func() {
			mustCreateColored("r1", "red", 1)
			mustCreateColored("r2", "red", 2)
			mustCreateColored("b1", "blue", 1)

			rows, err := store.Aggregate[colorCount](f.ctx, f.widgets, mongo.Pipeline{
				{{Key: "$match", Value: bson.M{"ownerID": f.owner}}},
				{{Key: "$group", Value: bson.M{"_id": "$spec.color", "count": bson.M{"$sum": 1}}}},
				{{Key: "$sort", Value: bson.M{"_id": 1}}},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).To(Equal([]colorCount{{Color: "blue", Count: 1}, {Color: "red", Count: 2}}))
		})

		It("should return an empty slice when there are no results", func() {
			rows, err := store.Aggregate[colorCount](f.ctx, f.widgets, mongo.Pipeline{
				{{Key: "$match", Value: bson.M{"ownerID": "nobody"}}},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).NotTo(BeNil())
			Expect(rows).To(BeEmpty())
		})

		It("should apply the conditions in opts before the pipeline", func() {
			mustCreateColored("r1", "red", 1)
			mustCreateColored("r3", "red", 3)
			mustCreateColored("b1", "blue", 1)

			// 条件中使用 Go 字段名，验证字段名按模型解析
			rows, err := store.Aggregate[colorCount](f.ctx, f.widgets, mongo.Pipeline{
				{{Key: "$group", Value: bson.M{"_id": "$spec.color", "count": bson.M{"$sum": 1}}}},
				{{Key: "$sort", Value: bson.M{"_id": 1}}},
			}, f.byOwner(), store.Where("Priority", bson.M{"$lt": 3}))
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).To(Equal([]colorCount{{Color: "blue", Count: 1}, {Color: "red", Count: 1}}))
		})

		DescribeTable("should reject options other than conditions",
			func(opt store.Opt, msg string) {
				_, err := store.Aggregate[colorCount](f.ctx, f.widgets, mongo.Pipeline{}, f.byOwner(), opt)
				Expect(err).To(MatchError(ContainSubstring(msg)))
			},
			Entry("with a sort option", store.Asc("name"), "sort"),
			Entry("with a limit", store.Limit(1), "limit"),
		)
	})
})
