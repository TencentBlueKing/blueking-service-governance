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

package model_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("WorkspaceCfgSwitch", func() {
	var (
		ctx      context.Context
		db       *mongo.Database
		switches *model.WorkspaceCfgSwitchStore
		wsID     string
	)

	saveSwitch := func(workspaceID string, key model.SwitchKey, enabled bool) {
		_, err := switches.Upsert(ctx, &model.WorkspaceCfgSwitch{
			WorkspaceID: workspaceID, Key: key, Enabled: enabled, Updater: "tester",
		}, model.ByWorkspaceAndKey(workspaceID, key), store.UpdateSelected("enabled", "updater"))
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		switches = testutil.NewStore(GinkgoT(), model.WorkspaceCfgSwitchMeta)
		wsID = "ws-" + stringx.Random(6)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(model.WorkspaceCfgSwitchMeta)).To(Succeed())
	})

	Describe("ListOfWorkspace", func() {
		It("should return all switches disabled in definition order when nothing is saved", func() {
			list, err := model.ListOfWorkspace(ctx, db, wsID)
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(len(model.SwitchDefs)))
			for i, def := range model.SwitchDefs {
				Expect(list[i].Key).To(Equal(def.Key))
				Expect(list[i].WorkspaceID).To(Equal(wsID))
				Expect(list[i].Enabled).To(BeFalse())
				Expect(list[i].UpdatedAt.IsZero()).To(BeTrue())
			}
		})

		It("should merge saved switches with defaults", func() {
			saveSwitch(wsID, model.SwitchRainbowTConf, true)

			list, err := model.ListOfWorkspace(ctx, db, wsID)
			Expect(err).NotTo(HaveOccurred())
			Expect(list[0].Key).To(Equal(model.SwitchBscp))
			Expect(list[0].Enabled).To(BeFalse())
			Expect(list[1].Key).To(Equal(model.SwitchRainbowTConf))
			Expect(list[1].Enabled).To(BeTrue())
			Expect(list[1].Updater).To(Equal("tester"))
		})

		It("should not read switches of another tenant", func() {
			saveSwitch(wsID, model.SwitchBscp, true)

			list, err := model.ListOfWorkspace(ctx, db, wsID, store.TenantOf("another-tenant"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list[0].Enabled).To(BeFalse())
		})
	})

	Describe("IsEnabled", func() {
		It("should treat a missing record as disabled", func() {
			enabled, err := model.IsEnabled(ctx, db, wsID, model.SwitchBscp)
			Expect(err).NotTo(HaveOccurred())
			Expect(enabled).To(BeFalse())
		})

		It("should reflect the saved value", func() {
			saveSwitch(wsID, model.SwitchBscp, true)
			enabled, err := model.IsEnabled(ctx, db, wsID, model.SwitchBscp)
			Expect(err).NotTo(HaveOccurred())
			Expect(enabled).To(BeTrue())

			saveSwitch(wsID, model.SwitchBscp, false)
			enabled, err = model.IsEnabled(ctx, db, wsID, model.SwitchBscp)
			Expect(err).NotTo(HaveOccurred())
			Expect(enabled).To(BeFalse())
		})
	})

	It("should reject an unknown switch key on save", func() {
		err := switches.Create(ctx, &model.WorkspaceCfgSwitch{WorkspaceID: wsID, Key: "unknown", Enabled: true})
		Expect(err).To(MatchError(model.ErrUnknownSwitch))
	})

	Describe("pre-delete hook", func() {
		var workspaces *bkmsworkspace.WorkspaceStore

		BeforeEach(func() {
			bkmsworkspace.ResetLifecycleHooksForTest()
			model.RegisterPreDeleteHooks(db)
			workspaces = testutil.NewStore(GinkgoT(), bkmsworkspace.Meta)
		})

		AfterEach(func() {
			bkmsworkspace.ResetLifecycleHooksForTest()
			Expect(testutil.CleanupMetaColl(bkmsworkspace.Meta)).To(Succeed())
		})

		It("should remove the switches of the deleted workspace only", func() {
			deleted := dbfactory.Workspace(ctx, workspaces)
			kept := dbfactory.Workspace(ctx, workspaces)
			saveSwitch(deleted.ID, model.SwitchBscp, true)
			saveSwitch(deleted.ID, model.SwitchRainbowTConf, true)
			saveSwitch(kept.ID, model.SwitchBscp, true)

			Expect(bkmsworkspace.Delete(ctx, workspaces, deleted.ID)).To(Succeed())

			Expect(switches.Count(ctx, model.ByWorkspace(deleted.ID))).To(BeZero())
			Expect(switches.Count(ctx, model.ByWorkspace(kept.ID))).To(BeEquivalentTo(1))
		})
	})
})
