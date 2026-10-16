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
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	releasestrategyhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy/hooks"
)

var _ = Describe("Release strategy lifecycle hooks", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var strategyStore releasestrategy.Store
	var workspaceStore bkmsworkspace.WorkspaceStore

	BeforeEach(func() {
		ctx = context.Background()
		bkmsworkspace.ResetLifecycleHooksForTest()

		diApp = fxtest.New(
			GinkgoT(),
			releasestrategy.FxModule,
			bkmsworkspace.FxModule,
			fx.Populate(&strategyStore, &workspaceStore),
		)
		diApp.RequireStart()
		releasestrategyhooks.RegisterPreDeleteHooks(strategyStore)
	})

	AfterEach(func() {
		Expect(testutil.CleanupCollection(releasestrategy.CollectionName)).To(Succeed())
		bkmsworkspace.ResetLifecycleHooksForTest()
		diApp.RequireStop()
	})

	It("removes strategies when their workspace is deleted", func() {
		workspaceID := dbfactory.Workspace(ctx, workspaceStore).ID
		Expect(strategyStore.Create(ctx, &releasestrategy.ReleaseStrategy{
			WorkspaceID: workspaceID,
			Name:        "生产谨慎发布",
			Batches:     []releasestrategy.Batch{{Target: "1", PauseMode: releasestrategy.PauseModePermanent}},
		})).To(Succeed())

		Expect(workspaceStore.Delete(ctx, workspaceID)).To(Succeed())

		strategies, err := strategyStore.List(ctx, workspaceID)
		Expect(err).NotTo(HaveOccurred())
		Expect(strategies).To(BeEmpty())
	})
})
