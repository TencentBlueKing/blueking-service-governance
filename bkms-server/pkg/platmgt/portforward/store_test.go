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

var _ = Describe("StoreMongo", func() {
	var (
		ctx   context.Context
		store *StoreMongo
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		store, err = NewStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// 清理所有白名单文档
		_, _ = store.collection.DeleteMany(ctx, bson.M{})
	})

	Describe("Add", func() {
		It("should add env IDs to whitelist", func() {
			err := store.Add(ctx, []string{"env-1", "env-2"})
			Expect(err).NotTo(HaveOccurred())

			envIDs, err := store.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1", "env-2"))
		})

		It("should deduplicate env IDs on add", func() {
			Expect(store.Add(ctx, []string{"env-1", "env-2"})).To(Succeed())
			Expect(store.Add(ctx, []string{"env-2", "env-3"})).To(Succeed())

			envIDs, err := store.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1", "env-2", "env-3"))
		})

		It("should handle empty input", func() {
			err := store.Add(ctx, []string{})
			Expect(err).NotTo(HaveOccurred())

			envIDs, err := store.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(BeEmpty())
		})
	})

	Describe("Remove", func() {
		It("should remove env IDs from whitelist", func() {
			Expect(store.Add(ctx, []string{"env-1", "env-2", "env-3"})).To(Succeed())

			err := store.Remove(ctx, []string{"env-2"})
			Expect(err).NotTo(HaveOccurred())

			envIDs, err := store.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1", "env-3"))
		})

		It("should not error when removing non-existent env ID", func() {
			Expect(store.Add(ctx, []string{"env-1"})).To(Succeed())

			err := store.Remove(ctx, []string{"env-nonexist"})
			Expect(err).NotTo(HaveOccurred())

			envIDs, err := store.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1"))
		})

		It("should handle empty input", func() {
			Expect(store.Add(ctx, []string{"env-1"})).To(Succeed())

			err := store.Remove(ctx, []string{})
			Expect(err).NotTo(HaveOccurred())

			envIDs, err := store.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1"))
		})
	})

	Describe("List", func() {
		It("should return empty slice when no documents exist", func() {
			envIDs, err := store.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(BeEmpty())
		})

		It("should return all env IDs", func() {
			Expect(store.Add(ctx, []string{"env-a", "env-b", "env-c"})).To(Succeed())

			envIDs, err := store.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-a", "env-b", "env-c"))
		})
	})

	Describe("Contains", func() {
		It("should return true when env ID exists", func() {
			Expect(store.Add(ctx, []string{"env-1", "env-2"})).To(Succeed())

			ok, err := store.Contains(ctx, "env-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("should return false when env ID does not exist", func() {
			Expect(store.Add(ctx, []string{"env-1"})).To(Succeed())

			ok, err := store.Contains(ctx, "env-nonexist")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("should return false when collection is empty", func() {
			ok, err := store.Contains(ctx, "env-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})
})
