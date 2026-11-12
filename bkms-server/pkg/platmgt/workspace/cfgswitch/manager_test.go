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

package cfgswitch

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Manager", func() {
	var (
		ctx            context.Context
		platCtx        context.Context
		diApp          *fxtest.App
		manager        *Manager
		workspaceStore *bkmsworkspace.WorkspaceStore
		switchStore    *model.WorkspaceCfgSwitchStore
		ws             *bkmsworkspace.Workspace
	)

	BeforeEach(func() {
		ctx = auth.WithUser(testutil.NewDefaultTenantContext(), auth.User{ID: "plat-admin"})
		// 与平台管理请求一致：操作者在另一个租户，且 ctx 要求显式指定租户范围
		platCtx = store.RequireExplicitTenantScope(tenant.WithTenantID(ctx, "plat-admin-tenant"))

		diApp = fxtest.New(GinkgoT(), FxModule, fx.Populate(&manager))
		diApp.RequireStart()
		workspaceStore = testutil.NewStore(GinkgoT(), bkmsworkspace.Meta)
		switchStore = testutil.NewStore(GinkgoT(), model.WorkspaceCfgSwitchMeta)
		ws = dbfactory.Workspace(ctx, workspaceStore)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(model.WorkspaceCfgSwitchMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(bkmsworkspace.Meta)).To(Succeed())
		diApp.RequireStop()
	})

	Describe("List", func() {
		It("should list all switches disabled when never set", func() {
			list, err := manager.List(platCtx, ws.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(len(model.SwitchDefs)))
			for _, s := range list {
				Expect(s.Enabled).To(BeFalse())
			}
		})

		It("should return ErrWorkspaceNotFound for unknown workspace", func() {
			_, err := manager.List(platCtx, "ws-not-exists")
			Expect(err).To(MatchError(bkmsworkspace.ErrWorkspaceNotFound))
		})
	})

	Describe("Update", func() {
		It("should save the switch into the tenant of the workspace and keep others unchanged", func() {
			f, err := manager.Update(platCtx, ws.ID, model.SwitchBscp, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.Enabled).To(BeTrue())
			Expect(f.Updater).To(Equal("plat-admin"))
			Expect(f.UpdatedAt.IsZero()).To(BeFalse())

			stored, err := switchStore.Get(ctx, model.ByWorkspaceAndKey(ws.ID, model.SwitchBscp))
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.TenantID).To(Equal(ws.TenantID))
			Expect(stored.Enabled).To(BeTrue())

			list, err := manager.List(platCtx, ws.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(list[0].Enabled).To(BeTrue())
			Expect(list[1].Enabled).To(BeFalse())
		})

		It("should turn a switch off", func() {
			_, err := manager.Update(platCtx, ws.ID, model.SwitchRainbowTConf, true)
			Expect(err).NotTo(HaveOccurred())
			f, err := manager.Update(platCtx, ws.ID, model.SwitchRainbowTConf, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.Enabled).To(BeFalse())

			enabled, err := model.IsEnabled(ctx, testutil.NewDB(GinkgoT()), ws.ID, model.SwitchRainbowTConf)
			Expect(err).NotTo(HaveOccurred())
			Expect(enabled).To(BeFalse())
		})

		It("should not write when nothing changes", func() {
			// 从未设置过时默认关闭，提交关闭不应产生记录
			_, err := manager.Update(platCtx, ws.ID, model.SwitchBscp, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(switchStore.Exists(ctx, model.ByWorkspace(ws.ID))).To(BeFalse())

			first, err := manager.Update(platCtx, ws.ID, model.SwitchBscp, true)
			Expect(err).NotTo(HaveOccurred())
			second, err := manager.Update(platCtx, ws.ID, model.SwitchBscp, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(second.UpdatedAt).To(Equal(first.UpdatedAt))
		})

		It("should reject an unknown switch key", func() {
			_, err := manager.Update(platCtx, ws.ID, "unknown", true)
			Expect(err).To(MatchError(model.ErrUnknownSwitch))
		})

		It("should return ErrWorkspaceNotFound for unknown workspace", func() {
			_, err := manager.Update(platCtx, "ws-not-exists", model.SwitchBscp, true)
			Expect(err).To(MatchError(bkmsworkspace.ErrWorkspaceNotFound))
		})
	})
})
