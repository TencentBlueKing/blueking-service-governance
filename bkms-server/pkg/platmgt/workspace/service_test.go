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

package workspace

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
)

// MongoDB DateTime 精度为毫秒，测试更新时间排序时需要等待时间戳进入下一个精度窗口。
const workspaceTimestampPrecisionWait = 5 * time.Millisecond

var _ = Describe("PlatWorkspaceService", func() {
	var (
		ctx            context.Context
		diApp          *fxtest.App
		service        *Service
		workspaceStore bkmsworkspace.WorkspaceStore
		appStore       bkmsapp.ApplicationStore
		envSvc         *env.EnvService
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsworkspace.FxModule,
			bkmsapp.FxModule,
			env.FxModule,
			FxModule,
			fx.Populate(&service, &workspaceStore, &appStore, &envSvc),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	It("should list platform workspaces with aggregated counts", func() {
		wsA := dbfactory.Workspace(ctx, workspaceStore)
		wsB := dbfactory.Workspace(ctx, workspaceStore)
		defer func() {
			_ = workspaceStore.Delete(ctx, wsA.ID)
			_ = workspaceStore.Delete(ctx, wsB.ID)
		}()

		helmStores := &dbfactory.HelmApplicationStores{AppStore: appStore}
		appA1 := dbfactory.HelmApplication(ctx, helmStores, &dbfactory.HelmApplicationOpts{WorkspaceID: wsA.ID})
		appA2 := dbfactory.HelmApplication(ctx, helmStores, &dbfactory.HelmApplicationOpts{WorkspaceID: wsA.ID})
		appB1 := dbfactory.HelmApplication(ctx, helmStores, &dbfactory.HelmApplicationOpts{WorkspaceID: wsB.ID})
		defer func() {
			_ = appStore.DeleteAppByName(ctx, wsA.ID, appA1.Name)
			_ = appStore.DeleteAppByName(ctx, wsA.ID, appA2.Name)
			_ = appStore.DeleteAppByName(ctx, wsB.ID, appB1.Name)
		}()

		envA1 := dbfactory.Env(ctx, envSvc, wsA.ID)
		envB1 := dbfactory.Env(ctx, envSvc, wsB.ID)
		envB2 := dbfactory.Env(ctx, envSvc, wsB.ID)
		defer func() {
			_ = envSvc.Delete(ctx, envA1.ID)
			_ = envSvc.Delete(ctx, envB1.ID)
			_ = envSvc.Delete(ctx, envB2.ID)
		}()

		time.Sleep(workspaceTimestampPrecisionWait)
		wsB.Description = "workspace b updated"
		Expect(workspaceStore.Update(ctx, wsB)).To(Succeed())

		items, err := service.List(ctx, WorkspaceListOptions{
			Page:     1,
			PageSize: 10,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(items.Count).To(Equal(int64(2)))
		Expect(items.Results).To(HaveLen(2))

		Expect(items.Results[0].ID).To(Equal(wsB.ID))
		Expect(items.Results[0].AppCount).To(Equal(1))
		Expect(items.Results[0].EnvCount).To(Equal(2))

		Expect(items.Results[1].ID).To(Equal(wsA.ID))
		Expect(items.Results[1].AppCount).To(Equal(2))
		Expect(items.Results[1].EnvCount).To(Equal(1))
	})

	It("should filter by keyword and state and paginate results", func() {
		wsA := dbfactory.Workspace(ctx, workspaceStore)
		wsB := dbfactory.Workspace(ctx, workspaceStore)
		defer func() {
			_ = workspaceStore.Delete(ctx, wsA.ID)
			_ = workspaceStore.Delete(ctx, wsB.ID)
		}()

		items, err := service.List(ctx, WorkspaceListOptions{
			Keyword:  wsA.ID,
			State:    string(bkmsworkspace.StateReady),
			Page:     1,
			PageSize: 1,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(items.Count).To(Equal(int64(1)))
		Expect(items.Results).To(HaveLen(1))
		Expect(items.Results[0].ID).To(Equal(wsA.ID))
	})

	It("should return filtered state summary independent of pagination", func() {
		wsReady := dbfactory.Workspace(ctx, workspaceStore)
		wsDisabled := dbfactory.Workspace(ctx, workspaceStore)
		wsOther := dbfactory.Workspace(ctx, workspaceStore)
		defer func() {
			_ = workspaceStore.Delete(ctx, wsReady.ID)
			_ = workspaceStore.Delete(ctx, wsDisabled.ID)
			_ = workspaceStore.Delete(ctx, wsOther.ID)
		}()

		wsReady.DisplayName = "alpha ready workspace"
		Expect(workspaceStore.Update(ctx, wsReady)).To(Succeed())

		wsDisabled.DisplayName = "alpha disabled workspace"
		wsDisabled.State = bkmsworkspace.StateDisabled
		Expect(workspaceStore.Update(ctx, wsDisabled)).To(Succeed())

		wsOther.DisplayName = "beta processing workspace"
		wsOther.State = bkmsworkspace.StateProcessing
		Expect(workspaceStore.Update(ctx, wsOther)).To(Succeed())

		items, err := service.List(ctx, WorkspaceListOptions{
			Keyword:  "alpha",
			Page:     1,
			PageSize: 1,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(items.Count).To(Equal(int64(2)))
		Expect(items.Results).To(HaveLen(1))
		Expect(items.Statistics.TotalCount).To(Equal(int64(2)))
		Expect(items.Statistics.ReadyCount).To(Equal(int64(1)))
		Expect(items.Statistics.DisabledCount).To(Equal(int64(1)))
		Expect(items.Statistics.ProcessingCount).To(Equal(int64(0)))
	})

	It("should keep state summary grouped by keyword when state filter is selected", func() {
		wsReady := dbfactory.Workspace(ctx, workspaceStore)
		wsDisabled := dbfactory.Workspace(ctx, workspaceStore)
		wsProcessing := dbfactory.Workspace(ctx, workspaceStore)
		defer func() {
			_ = workspaceStore.Delete(ctx, wsReady.ID)
			_ = workspaceStore.Delete(ctx, wsDisabled.ID)
			_ = workspaceStore.Delete(ctx, wsProcessing.ID)
		}()

		wsReady.DisplayName = "alpha ready workspace"
		Expect(workspaceStore.Update(ctx, wsReady)).To(Succeed())

		wsDisabled.DisplayName = "alpha disabled workspace"
		wsDisabled.State = bkmsworkspace.StateDisabled
		Expect(workspaceStore.Update(ctx, wsDisabled)).To(Succeed())

		wsProcessing.DisplayName = "alpha processing workspace"
		wsProcessing.State = bkmsworkspace.StateProcessing
		Expect(workspaceStore.Update(ctx, wsProcessing)).To(Succeed())

		items, err := service.List(ctx, WorkspaceListOptions{
			Keyword:  "alpha",
			State:    string(bkmsworkspace.StateDisabled),
			Page:     1,
			PageSize: 10,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(items.Count).To(Equal(int64(1)))
		Expect(items.Results).To(HaveLen(1))
		Expect(items.Results[0].ID).To(Equal(wsDisabled.ID))
		Expect(items.Statistics.TotalCount).To(Equal(int64(3)))
		Expect(items.Statistics.ReadyCount).To(Equal(int64(1)))
		Expect(items.Statistics.DisabledCount).To(Equal(int64(1)))
		Expect(items.Statistics.ProcessingCount).To(Equal(int64(1)))
	})

	It("should sort workspace list by updated at in ascending order", func() {
		wsA := dbfactory.Workspace(ctx, workspaceStore)
		wsB := dbfactory.Workspace(ctx, workspaceStore)
		defer func() {
			_ = workspaceStore.Delete(ctx, wsA.ID)
			_ = workspaceStore.Delete(ctx, wsB.ID)
		}()

		time.Sleep(workspaceTimestampPrecisionWait)
		wsB.Description = "workspace b updated"
		Expect(workspaceStore.Update(ctx, wsB)).To(Succeed())

		items, err := service.List(ctx, WorkspaceListOptions{
			SortBy:    "updatedAt",
			SortOrder: "asc",
			Page:      1,
			PageSize:  10,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(items.Results).To(HaveLen(2))
		Expect(items.Results[0].ID).To(Equal(wsA.ID))
		Expect(items.Results[1].ID).To(Equal(wsB.ID))
	})

	It("should sort workspace list by display name in descending order", func() {
		wsA := dbfactory.Workspace(ctx, workspaceStore)
		wsB := dbfactory.Workspace(ctx, workspaceStore)
		defer func() {
			_ = workspaceStore.Delete(ctx, wsA.ID)
			_ = workspaceStore.Delete(ctx, wsB.ID)
		}()

		items, err := service.List(ctx, WorkspaceListOptions{
			SortBy:    "displayName",
			SortOrder: "desc",
			Page:      1,
			PageSize:  10,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(items.Results).To(HaveLen(2))
		Expect(items.Results[0].DisplayName > items.Results[1].DisplayName).To(BeTrue())
	})

	It("should get one platform workspace info", func() {
		ws := dbfactory.Workspace(ctx, workspaceStore)
		defer func() { _ = workspaceStore.Delete(ctx, ws.ID) }()

		item, err := service.Get(ctx, ws.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(item.ID).To(Equal(ws.ID))
		Expect(item.DisplayName).To(Equal(ws.DisplayName))
		Expect(item.State).To(Equal(string(ws.State)))
		Expect(item.CreatedAt.IsZero()).To(BeFalse())
		Expect(item.UpdatedAt.IsZero()).To(BeFalse())
	})

	It("should return not found when target workspace does not exist", func() {
		_, err := service.Get(ctx, "plat-ws-nonexistent")
		Expect(err).To(Equal(bkmsworkspace.ErrWorkspaceNotFound))
	})

	It("should return aggregated workspace state statistics", func() {
		// 先读取当前统计值，再校验本用例新增的 3 个 workspace 对各状态计数带来的增量
		baselineStats, err := service.GetStateStatistics(ctx)
		Expect(err).NotTo(HaveOccurred())

		wsReady := dbfactory.Workspace(ctx, workspaceStore)
		wsProcessing := dbfactory.Workspace(ctx, workspaceStore)
		wsDisabled := dbfactory.Workspace(ctx, workspaceStore)
		defer func() {
			_ = workspaceStore.Delete(ctx, wsReady.ID)
			_ = workspaceStore.Delete(ctx, wsProcessing.ID)
			_ = workspaceStore.Delete(ctx, wsDisabled.ID)
		}()

		wsProcessing.State = bkmsworkspace.StateProcessing
		Expect(workspaceStore.Update(ctx, wsProcessing)).To(Succeed())

		wsDisabled.State = bkmsworkspace.StateDisabled
		Expect(workspaceStore.Update(ctx, wsDisabled)).To(Succeed())

		stats, err := service.GetStateStatistics(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(stats.ReadyCount - baselineStats.ReadyCount).To(Equal(int64(1)))
		Expect(stats.ProcessingCount - baselineStats.ProcessingCount).To(Equal(int64(1)))
		Expect(stats.DisabledCount - baselineStats.DisabledCount).To(Equal(int64(1)))
		Expect(stats.TotalCount - baselineStats.TotalCount).To(Equal(int64(3)))
	})
})
