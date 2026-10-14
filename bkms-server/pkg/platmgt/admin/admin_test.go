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

package admin

import (
	"context"
	"regexp"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Platform administrators", func() {
	var (
		ctx          context.Context
		roleBindings *Service
		bindingStore *Store
		testPrefix   string
	)

	BeforeEach(func() {
		ctx = context.Background()
		testPrefix = "plat-admin-service-test-" + stringx.Random(6)

		roleBindings = NewService(testutil.NewDB(GinkgoT()))
		bindingStore = testutil.NewStore(GinkgoT(), RoleBindingMeta)

		Expect(bindingStore.CreateMany(ctx, []*RoleBinding{
			{
				Username: testPrefix + "-b",
				RoleCode: RoleCodeAdmin,
				Creator:  "bootstrap-admin",
				Updater:  "bootstrap-admin",
			},
			{
				Username: testPrefix + "-a",
				RoleCode: RoleCodeAdmin,
				Creator:  "bootstrap-admin",
			},
		})).To(Succeed())
	})

	AfterEach(func() {
		_, err := bindingStore.DeleteMany(ctx, store.Where("username", bson.M{
			"$regex": "^" + regexp.QuoteMeta(testPrefix),
		}))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should list role bindings sorted by username with keyword filtering", func() {
		results, err := roleBindings.List(ctx, " "+testPrefix+" ")
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(2))
		Expect(results[0].Username).To(Equal(testPrefix + "-a"))
		Expect(results[1].Username).To(Equal(testPrefix + "-b"))
		Expect(results[0].CreatedAt).NotTo(BeZero())
		Expect(results[0].UpdatedAt).To(Equal(results[0].CreatedAt))
		// 未指定更新人时以创建人作为更新人
		Expect(results[0].Updater).To(Equal("bootstrap-admin"))

		results, err = roleBindings.List(ctx, testPrefix+"-A")
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(1))
		Expect(results[0].Username).To(Equal(testPrefix + "-a"))
	})

	It("should report platform administrator role", func() {
		roleCode, ok, err := roleBindings.GetRole(ctx, testPrefix+"-a")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(roleCode).To(Equal(RoleCodeAdmin))

		roleCode, ok, err = roleBindings.GetRole(ctx, testPrefix+"-ordinary-user")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
		Expect(roleCode).To(BeEmpty())
	})

	It("should assign platform administrator roles in batch and skip existing usernames", func() {
		err := roleBindings.AssignRoles(
			ctx, []string{testPrefix + "-c", testPrefix + "-a", testPrefix + "-c"}, RoleCodeAdmin, "admin-a",
		)
		Expect(err).NotTo(HaveOccurred())

		results, err := roleBindings.List(ctx, testPrefix)
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(3))

		// 已存在的角色绑定保持不变
		existing, err := bindingStore.Get(ctx, ByUsername(testPrefix+"-a"))
		Expect(err).NotTo(HaveOccurred())
		Expect(existing.Creator).To(Equal("bootstrap-admin"))
		created, err := bindingStore.Get(ctx, ByUsername(testPrefix+"-c"))
		Expect(err).NotTo(HaveOccurred())
		Expect(created.Creator).To(Equal("admin-a"))
	})

	It("should reject invalid role codes", func() {
		err := roleBindings.AssignRoles(ctx, []string{testPrefix + "-c"}, RoleCode("invalid"), "admin-a")
		Expect(err).To(MatchError(ErrInvalidRoleCode))
	})

	It("should revoke platform administrator roles idempotently", func() {
		targetUsername := testPrefix + "-b"
		Expect(roleBindings.RevokeRole(ctx, targetUsername)).To(Succeed())
		Expect(roleBindings.RevokeRole(ctx, targetUsername)).To(Succeed())
		_, err := bindingStore.Get(ctx, ByUsername(targetUsername))
		Expect(err).To(MatchError(ErrRoleBindingNotFound))
	})
})
