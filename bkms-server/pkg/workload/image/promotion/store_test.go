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

package promotion

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("PromotionStoreMongo", func() {
	var ctx context.Context
	var mongoClient *mongo.Client
	var store PromotionStore
	var dbName string

	BeforeEach(func() {
		var err error
		mongoClient, dbName = database.Client(), database.Name()
		ctx = context.Background()
		store, err = NewPromotionStoreMongo(mongoClient, dbName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := store.DeleteAll(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Upsert", func() {
		It("should create a new promotion record", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())

			results, err := store.ListByAppAndRepoKey(ctx, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].AppID).To(Equal("app1"))
			Expect(results[0].RepoKey).To(Equal("repokey1"))
			Expect(results[0].Tag).To(Equal("v1.0.0"))
			Expect(results[0].PromotedBy).To(Equal("alice"))
			Expect(results[0].PromotedAt).NotTo(BeZero())
		})

		It("should update promotedBy and promotedAt on repeated upsert (idempotent)", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())

			results, err := store.ListByAppAndRepoKey(ctx, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			originalPromotedAt := results[0].PromotedAt

			// 等待一小段时间，确保时间戳不同
			time.Sleep(10 * time.Millisecond)

			// 第二次 upsert（不同操作人），应覆盖为最后一个操作人
			err = store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "bob")
			Expect(err).NotTo(HaveOccurred())

			results2, err := store.ListByAppAndRepoKey(ctx, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(results2).To(HaveLen(1))
			Expect(results2[0].PromotedBy).To(Equal("bob"))
			Expect(results2[0].PromotedAt.After(originalPromotedAt)).To(BeTrue())
		})
	})

	Describe("ListByAppAndRepoKey", func() {
		It("should return all promotions for the given app and repoKey", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())
			err = store.Upsert(ctx, "app1", "repokey1", "v2.0.0", "bob")
			Expect(err).NotTo(HaveOccurred())
			err = store.Upsert(ctx, "app1", "repokey1", "v3.0.0", "charlie")
			Expect(err).NotTo(HaveOccurred())

			results, err := store.ListByAppAndRepoKey(ctx, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(3))

			tags := make([]string, 0, len(results))
			for _, r := range results {
				tags = append(tags, r.Tag)
			}
			Expect(tags).To(ContainElements("v1.0.0", "v2.0.0", "v3.0.0"))
		})

		It("should return empty for non-existent app", func() {
			results, err := store.ListByAppAndRepoKey(ctx, "nonexistent", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("ListTagsByAppAndRepoKey", func() {
		It("should return all promoted tags for the given app and repoKey", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())
			err = store.Upsert(ctx, "app1", "repokey1", "v2.0.0", "bob")
			Expect(err).NotTo(HaveOccurred())
			err = store.Upsert(ctx, "app1", "repokey1", "v3.0.0", "charlie")
			Expect(err).NotTo(HaveOccurred())

			tags, err := store.ListTagsByAppAndRepoKey(ctx, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(HaveLen(3))
			Expect(tags).To(ContainElements("v1.0.0", "v2.0.0", "v3.0.0"))
		})

		It("should return empty list when no promotions exist", func() {
			tags, err := store.ListTagsByAppAndRepoKey(ctx, "nonexistent-app", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(BeEmpty())
		})

		It("should only return tags for the specified app, not other apps", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())
			err = store.Upsert(ctx, "app2", "repokey1", "v2.0.0", "bob")
			Expect(err).NotTo(HaveOccurred())

			tags, err := store.ListTagsByAppAndRepoKey(ctx, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(HaveLen(1))
			Expect(tags).To(ContainElement("v1.0.0"))
			Expect(tags).NotTo(ContainElement("v2.0.0"))
		})
	})

	Describe("IsTagPromoted", func() {
		It("should return true when promotion record exists", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())

			exists, err := store.IsTagPromoted(ctx, "app1", "repokey1", "v1.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should return false when promotion record does not exist", func() {
			exists, err := store.IsTagPromoted(ctx, "nonexistent", "repokey1", "v1.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should return false for wrong tag even if other tags exist", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())

			exists, err := store.IsTagPromoted(ctx, "app1", "repokey1", "v2.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("DeleteTag", func() {
		It("should delete only the specified promotion record", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())
			err = store.Upsert(ctx, "app1", "repokey1", "v2.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())
			err = store.Upsert(ctx, "app2", "repokey1", "v1.0.0", "bob")
			Expect(err).NotTo(HaveOccurred())

			err = store.DeleteTag(ctx, "app1", "repokey1", "v1.0.0")
			Expect(err).NotTo(HaveOccurred())

			tags, err := store.ListTagsByAppAndRepoKey(ctx, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(HaveLen(1))
			Expect(tags).To(ContainElement("v2.0.0"))

			exists, err := store.IsTagPromoted(ctx, "app2", "repokey1", "v1.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should be a no-op when the promotion record does not exist", func() {
			err := store.Upsert(ctx, "app1", "repokey1", "v1.0.0", "alice")
			Expect(err).NotTo(HaveOccurred())

			err = store.DeleteTag(ctx, "app1", "repokey1", "not-found")
			Expect(err).NotTo(HaveOccurred())

			tags, err := store.ListTagsByAppAndRepoKey(ctx, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(HaveLen(1))
			Expect(tags).To(ContainElement("v1.0.0"))
		})
	})
})
