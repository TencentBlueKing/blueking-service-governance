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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
)

var _ = Describe("Promotion store", func() {
	var (
		ctx        context.Context
		promotions *Store
	)

	BeforeEach(func() {
		ctx = context.Background()
		promotions = testutil.NewStore(GinkgoT(), Meta)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(Meta)).To(Succeed())
	})

	promote := func(appID, repoKey, tag, operator string) {
		Expect(Promote(ctx, promotions, appID, repoKey, tag, operator)).To(Succeed())
	}

	Describe("Promote", func() {
		It("should create a new promotion record", func() {
			promote("app1", "repokey1", "v1.0.0", "alice")

			results, err := promotions.List(ctx, ByAppAndRepoKey("app1", "repokey1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].AppID).To(Equal("app1"))
			Expect(results[0].RepoKey).To(Equal("repokey1"))
			Expect(results[0].Tag).To(Equal("v1.0.0"))
			Expect(results[0].PromotedBy).To(Equal("alice"))
			Expect(results[0].PromotedAt).NotTo(BeZero())
			Expect(results[0].CreatedAt).NotTo(BeZero())
		})

		It("should update promotedBy and promotedAt on repeated promotion (idempotent)", func() {
			promote("app1", "repokey1", "v1.0.0", "alice")
			first, err := promotions.Get(ctx, ByTag("app1", "repokey1", "v1.0.0"))
			Expect(err).NotTo(HaveOccurred())

			// 等待一小段时间，确保时间戳不同
			time.Sleep(10 * time.Millisecond)

			// 第二次晋级（不同操作人），应覆盖为最后一个操作人
			promote("app1", "repokey1", "v1.0.0", "bob")

			results, err := promotions.List(ctx, ByAppAndRepoKey("app1", "repokey1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].PromotedBy).To(Equal("bob"))
			Expect(results[0].PromotedAt.After(first.PromotedAt)).To(BeTrue())
			Expect(results[0].CreatedAt).To(Equal(first.CreatedAt))
		})
	})

	Describe("ListPromotedTags", func() {
		It("should return all promoted tags for the given app and repoKey", func() {
			promote("app1", "repokey1", "v1.0.0", "alice")
			promote("app1", "repokey1", "v2.0.0", "bob")
			promote("app1", "repokey2", "v3.0.0", "charlie")
			promote("app2", "repokey1", "v4.0.0", "bob")

			tags, err := ListPromotedTags(ctx, promotions, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(ConsistOf("v1.0.0", "v2.0.0"))
		})

		It("should return empty list when no promotions exist", func() {
			tags, err := ListPromotedTags(ctx, promotions, "nonexistent-app", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(BeEmpty())
		})
	})

	Describe("ByTag", func() {
		It("should match only the specified promotion record", func() {
			promote("app1", "repokey1", "v1.0.0", "alice")
			promote("app1", "repokey1", "v2.0.0", "alice")
			promote("app2", "repokey1", "v1.0.0", "bob")

			Expect(promotions.DeleteIfExists(ctx, ByTag("app1", "repokey1", "v1.0.0"))).To(Succeed())

			tags, err := ListPromotedTags(ctx, promotions, "app1", "repokey1")
			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(ConsistOf("v2.0.0"))

			exists, err := promotions.Exists(ctx, ByTag("app2", "repokey1", "v1.0.0"))
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})
})
