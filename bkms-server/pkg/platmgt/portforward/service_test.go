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

package portforward

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Service", func() {
	var (
		ctx     context.Context
		service *Service
		store   *StoreMongo
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		store, err = NewStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		// Service 的 envStore 在 CheckPermission 中不使用，仅在 Add 的 validateEnvIDs 中使用。
		// 此处测试 CheckPermission 逻辑，envStore 传 nil（不会被调用）。
		service = NewService(store, nil)
	})

	AfterEach(func() {
		_, _ = store.collection.DeleteMany(ctx, bson.M{})
	})

	Describe("CheckPermission", func() {
		BeforeEach(func() {
			// 添加测试白名单记录
			Expect(store.Add(ctx, []string{"env-id-aaa", "env-id-bbb"})).To(Succeed())
		})

		It("should allow access when envID is in whitelist", func() {
			err := service.CheckPermission(ctx, "env-id-aaa")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow access for another allowed envID", func() {
			err := service.CheckPermission(ctx, "env-id-bbb")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should deny access when envID not in whitelist", func() {
			err := service.CheckPermission(ctx, "env-id-ccc")
			Expect(err).To(Equal(ErrPermissionDenied))
		})

		It("should deny access when whitelist is empty", func() {
			// 清空白名单
			Expect(store.Remove(ctx, []string{"env-id-aaa", "env-id-bbb"})).To(Succeed())

			err := service.CheckPermission(ctx, "env-id-aaa")
			Expect(err).To(Equal(ErrPermissionDenied))
		})
	})

	Describe("List", func() {
		It("should return empty when no env IDs added", func() {
			envIDs, err := service.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(BeEmpty())
		})

		It("should return all env IDs after add", func() {
			Expect(store.Add(ctx, []string{"env-1", "env-2"})).To(Succeed())

			envIDs, err := service.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1", "env-2"))
		})
	})

	Describe("Remove", func() {
		It("should remove env IDs via service", func() {
			Expect(store.Add(ctx, []string{"env-1", "env-2", "env-3"})).To(Succeed())

			err := service.Remove(ctx, []string{"env-2"})
			Expect(err).NotTo(HaveOccurred())

			envIDs, err := service.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1", "env-3"))
		})
	})
})
