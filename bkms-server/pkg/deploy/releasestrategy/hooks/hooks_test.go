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

package hooks_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	releasestrategyhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy/hooks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

func newStrategy(workspaceID, name string) *releasestrategy.ReleaseStrategy {
	return &releasestrategy.ReleaseStrategy{
		WorkspaceID: workspaceID,
		Name:        name,
		Batches:     []releasestrategy.Batch{{Target: "1", PauseMode: releasestrategy.PauseModePermanent}},
	}
}

var _ = Describe("Release strategy lifecycle hooks", func() {
	var ctx context.Context
	var strategyStore *releasestrategy.Store
	var workspaceStore *bkmsworkspace.WorkspaceStore
	var workspaces []*bkmsworkspace.Workspace

	BeforeEach(func() {
		ctx = context.Background()
		bkmsworkspace.ResetLifecycleHooksForTest()

		db := testutil.NewDB(GinkgoT())
		strategyStore = store.Of(db, releasestrategy.Meta)
		workspaceStore = store.Of(db, bkmsworkspace.Meta)
		releasestrategyhooks.RegisterPreDeleteHooks(db)
		workspaces = nil
	})

	AfterEach(func() {
		for _, ws := range workspaces {
			Expect(bkmsworkspace.Delete(ctx, workspaceStore, ws.ID)).To(Succeed())
		}
		Expect(testutil.CleanupMetaColl(releasestrategy.Meta)).To(Succeed())
		bkmsworkspace.ResetLifecycleHooksForTest()
	})

	It("removes only the deleted workspace's strategies and frees their names", func() {
		deleted := dbfactory.Workspace(ctx, workspaceStore)
		kept := dbfactory.Workspace(ctx, workspaceStore)
		workspaces = append(workspaces, kept)
		Expect(strategyStore.Create(ctx, newStrategy(deleted.ID, "生产谨慎发布"))).To(Succeed())
		Expect(strategyStore.Create(ctx, newStrategy(deleted.ID, "fast"))).To(Succeed())
		Expect(strategyStore.Create(ctx, newStrategy(kept.ID, "生产谨慎发布"))).To(Succeed())

		Expect(bkmsworkspace.Delete(ctx, workspaceStore, deleted.ID)).To(Succeed())

		strategies, err := strategyStore.List(ctx, releasestrategy.ByWorkspace(deleted.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(strategies).To(BeEmpty())
		strategies, err = strategyStore.List(ctx, releasestrategy.ByWorkspace(kept.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(strategies).To(HaveLen(1))
		// 空间内名称唯一索引不再被旧记录占用
		Expect(strategyStore.Create(ctx, newStrategy(deleted.ID, "生产谨慎发布"))).To(Succeed())
	})
})
