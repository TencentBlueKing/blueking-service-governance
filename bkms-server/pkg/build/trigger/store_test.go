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

package trigger

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("PolicyStoreMongo", func() {
	var (
		store PolicyStore
		ctx   context.Context
		appID string
	)

	BeforeEach(func() {
		var err error
		store, err = NewPolicyStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
		appID = "trigger-app-" + stringx.Random(8)
	})

	It("creates, gets and lists policies by createdAt ascending", func() {
		first := persistTestPolicy(ctx, store, appID, "first", "master")
		time.Sleep(5 * time.Millisecond)
		second := persistTestPolicy(ctx, store, appID, "second", "develop")

		got, err := store.Get(ctx, appID, first.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Name).To(Equal("first"))
		Expect(got.Creator).To(Equal("dbfactory"))

		listed, err := store.List(ctx, appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(listed).To(HaveLen(2))
		Expect(listed[0].ID).To(Equal(first.ID))
		Expect(listed[1].ID).To(Equal(second.ID))
	})

	It("rejects duplicated policy names in the same app", func() {
		persistTestPolicy(ctx, store, appID, "dup", "master")
		err := store.Create(ctx, &Policy{
			ID:              PolicyIDPrefix + stringx.Random(8),
			AppID:           appID,
			Name:            "dup",
			Event:           EventPush,
			BranchMatchMode: BranchMatchModeEq,
			Status:          StatusEnabled,
			Creator:         "tester",
		})
		Expect(err).To(MatchError(ErrPolicyNameDuplicated))
	})

	It("updates form fields and status", func() {
		policy := persistTestPolicy(ctx, store, appID, "editable", "master")
		policy.Name = "renamed"
		policy.BranchMatchValue = "release"
		policy.PathFilter = "src/**"
		Expect(store.Update(ctx, policy)).To(Succeed())

		got, err := store.Get(ctx, appID, policy.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Name).To(Equal("renamed"))
		Expect(got.BranchMatchValue).To(Equal("release"))
		Expect(got.PathFilter).To(Equal("src/**"))
		Expect(got.Creator).To(Equal("dbfactory"))

		Expect(store.UpdateStatus(ctx, appID, policy.ID, StatusDisabled)).To(Succeed())
		got, err = store.Get(ctx, appID, policy.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Status).To(Equal(StatusDisabled))
	})

	It("returns not found for missing policies", func() {
		_, err := store.Get(ctx, appID, "btp-missing")
		Expect(err).To(MatchError(ErrPolicyNotFound))
		Expect(store.UpdateStatus(ctx, appID, "btp-missing", StatusDisabled)).To(MatchError(ErrPolicyNotFound))
		Expect(store.Delete(ctx, appID, "btp-missing")).To(MatchError(ErrPolicyNotFound))
	})

	It("deletes a policy without affecting others", func() {
		keep := persistTestPolicy(ctx, store, appID, "keep", "master")
		drop := persistTestPolicy(ctx, store, appID, "drop", "develop")
		Expect(store.Delete(ctx, appID, drop.ID)).To(Succeed())
		_, err := store.Get(ctx, appID, drop.ID)
		Expect(err).To(MatchError(ErrPolicyNotFound))
		_, err = store.Get(ctx, appID, keep.ID)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns empty list when the app has no policies", func() {
		listed, err := store.List(ctx, appID)
		Expect(err).NotTo(HaveOccurred())
		Expect(listed).To(BeEmpty())
	})
})

// persistTestPolicy 写入一条测试策略
// 本包测试不能导入 dbfactory（与 trigger 循环依赖），dbfactory.TriggerPolicy 留给外部测试包
func persistTestPolicy(ctx context.Context, store PolicyStore, appID, name, branch string) *Policy {
	policy := &Policy{
		ID:               PolicyIDPrefix + stringx.Random(8),
		AppID:            appID,
		Name:             name,
		Event:            EventPush,
		BranchMatchMode:  BranchMatchModeEq,
		BranchMatchValue: branch,
		Status:           StatusEnabled,
		Creator:          "dbfactory",
	}
	Expect(store.Create(ctx, policy)).To(Succeed())
	return policy
}
