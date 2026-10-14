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

package releasestrategy_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
)

func newStrategy(workspaceID, name string) *releasestrategy.ReleaseStrategy {
	return &releasestrategy.ReleaseStrategy{
		WorkspaceID: workspaceID,
		Name:        name,
		Description: "careful production release",
		Batches: []releasestrategy.Batch{
			{Target: "1", PauseMode: releasestrategy.PauseModePermanent},
			{Target: "25%", PauseMode: releasestrategy.PauseModeNone},
			{Target: "50%", PauseMode: releasestrategy.PauseModeDuration, PauseSeconds: 1800},
		},
		Creator: "admin",
		Updater: "admin",
	}
}

var _ = Describe("Release strategy store", func() {
	var ctx context.Context
	var store *releasestrategy.Store

	BeforeEach(func() {
		ctx = context.Background()
		store = testutil.NewStore(GinkgoT(), releasestrategy.Meta)
	})

	AfterEach(func() {
		// 仅清空文档，保留 migration 创建的唯一索引
		Expect(testutil.CleanupMetaColl(releasestrategy.Meta)).To(Succeed())
	})

	It("supports workspace scoped CRUD operations", func() {
		created := newStrategy("workspace-a", "生产谨慎发布")
		Expect(store.Create(ctx, created)).To(Succeed())
		Expect(created.ID.IsZero()).To(BeFalse())
		// createdAt 入库只有毫秒精度，连续创建可能相同，列表顺序会不稳定
		time.Sleep(5 * time.Millisecond)
		Expect(store.Create(ctx, newStrategy("workspace-a", "fast"))).To(Succeed())
		Expect(store.Create(ctx, newStrategy("workspace-b", "other"))).To(Succeed())

		strategies, err := store.List(ctx, releasestrategy.ByWorkspace("workspace-a"), releasestrategy.OldestFirst())
		Expect(err).NotTo(HaveOccurred())
		Expect(strategies).To(HaveLen(2))
		Expect(strategies[0].Name).To(Equal("生产谨慎发布"))
		Expect(strategies[1].Name).To(Equal("fast"))

		stored, err := releasestrategy.GetStrategy(ctx, store, "workspace-a", created.ID.Hex())
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.Batches).To(Equal(created.Batches))
		Expect(stored.Creator).To(Equal("admin"))

		updated := *stored
		updated.Name = "renamed"
		updated.Description = ""
		updated.Batches = []releasestrategy.Batch{{Target: "100%", PauseMode: releasestrategy.PauseModeNone}}
		updated.Updater = "operator"
		Expect(releasestrategy.UpdateStrategy(ctx, store, &updated)).To(Succeed())

		stored, err = releasestrategy.GetStrategy(ctx, store, "workspace-a", created.ID.Hex())
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.Name).To(Equal("renamed"))
		Expect(stored.Description).To(BeEmpty())
		Expect(stored.Batches).To(Equal(updated.Batches))
		Expect(stored.Creator).To(Equal("admin"))
		Expect(stored.Updater).To(Equal("operator"))

		deleted, err := releasestrategy.DeleteStrategy(ctx, store, "workspace-a", created.ID.Hex())
		Expect(err).NotTo(HaveOccurred())
		Expect(deleted.ID).To(Equal(created.ID))
		_, err = releasestrategy.GetStrategy(ctx, store, "workspace-a", created.ID.Hex())
		Expect(errors.Is(err, releasestrategy.ErrStrategyNotFound)).To(BeTrue())
	})

	It("rejects duplicate names only within the same workspace", func() {
		Expect(store.Create(ctx, newStrategy("workspace-a", "生产谨慎发布"))).To(Succeed())

		err := store.Create(ctx, newStrategy("workspace-a", "生产谨慎发布"))
		Expect(errors.Is(err, releasestrategy.ErrStrategyNameConflict)).To(BeTrue())

		Expect(store.Create(ctx, newStrategy("workspace-b", "生产谨慎发布"))).To(Succeed())
	})

	It("rejects renaming to an existing name in the same workspace", func() {
		Expect(store.Create(ctx, newStrategy("workspace-a", "first"))).To(Succeed())
		second := newStrategy("workspace-a", "second")
		Expect(store.Create(ctx, second)).To(Succeed())

		second.Name = "first"
		err := releasestrategy.UpdateStrategy(ctx, store, second)
		Expect(errors.Is(err, releasestrategy.ErrStrategyNameConflict)).To(BeTrue())
	})

	It("returns not found for missing, invalid or cross-workspace IDs", func() {
		created := newStrategy("workspace-a", "scoped")
		Expect(store.Create(ctx, created)).To(Succeed())

		for _, id := range []string{bson.NewObjectID().Hex(), "not-an-object-id"} {
			_, err := releasestrategy.GetStrategy(ctx, store, "workspace-a", id)
			Expect(errors.Is(err, releasestrategy.ErrStrategyNotFound)).To(BeTrue())
			_, err = releasestrategy.DeleteStrategy(ctx, store, "workspace-a", id)
			Expect(errors.Is(err, releasestrategy.ErrStrategyNotFound)).To(BeTrue())
		}

		_, err := releasestrategy.GetStrategy(ctx, store, "workspace-b", created.ID.Hex())
		Expect(errors.Is(err, releasestrategy.ErrStrategyNotFound)).To(BeTrue())

		crossWorkspace := *created
		crossWorkspace.WorkspaceID = "workspace-b"
		err = releasestrategy.UpdateStrategy(ctx, store, &crossWorkspace)
		Expect(errors.Is(err, releasestrategy.ErrStrategyNotFound)).To(BeTrue())

		_, err = releasestrategy.DeleteStrategy(ctx, store, "workspace-b", created.ID.Hex())
		Expect(errors.Is(err, releasestrategy.ErrStrategyNotFound)).To(BeTrue())
		_, err = releasestrategy.GetStrategy(ctx, store, "workspace-a", created.ID.Hex())
		Expect(err).NotTo(HaveOccurred())
	})
})
