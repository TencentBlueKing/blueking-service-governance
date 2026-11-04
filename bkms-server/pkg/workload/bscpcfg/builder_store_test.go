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

package bscpcfg_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	cfgswitchmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

var _ = Describe("BuildFromStore", func() {
	var (
		ctx       context.Context
		db        *mongo.Database
		switches  *cfgswitchmodel.WorkspaceCfgSwitchStore
		metas     *model.MetadataStore
		bindings  *model.EnvBindingStore
		ws        *workspace.Workspace
		appID     string
		oldBSCPCf svccfg.BSCPConfig
	)

	setBscpEnabled := func(enabled bool) {
		key := cfgswitchmodel.SwitchBscp
		target := &cfgswitchmodel.WorkspaceCfgSwitch{WorkspaceID: ws.ID, Key: key, Enabled: enabled}
		_, err := switches.Upsert(
			ctx, target, cfgswitchmodel.ByWorkspaceAndKey(ws.ID, key), store.UpdateSelected("enabled"),
		)
		Expect(err).NotTo(HaveOccurred())
	}

	createSnapshot := func() {
		Expect(metas.Create(ctx, &model.Metadata{
			AppID:        appID,
			MountPath:    "/data/bscp",
			WorkloadName: "test-workload",
			WorkloadKind: "Deployment",
		})).To(Succeed())
		Expect(bindings.Create(ctx, &model.EnvBinding{
			AppID:       appID,
			EnvName:     "dev",
			BscpEnvID:   "1",
			BscpEnvName: "dev",
			BscpAppID:   "1001",
		})).To(Succeed())
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		switches = testutil.NewStore(GinkgoT(), cfgswitchmodel.WorkspaceCfgSwitchMeta)
		metas = testutil.NewStore(GinkgoT(), model.MetadataMeta)
		bindings = testutil.NewStore(GinkgoT(), model.EnvBindingMeta)
		// BuildFromStore 只读取空间 ID 与 BkSystems，无需落库
		ws = &workspace.Workspace{
			ID: "ws-" + stringx.Random(6),
			BkSystems: workspace.BkSystems{
				BkCCBizID:        "100",
				BscpToken:        "test-token",
				BkBSCPProjectKey: "BK-BSCP-1",
			},
		}
		appID = "test-app-" + stringx.Random(5)

		oldBSCPCf = svccfg.G.BSCP
		svccfg.G.BSCP = svccfg.BSCPConfig{
			FeedAddr:     "bscp-feed.example.com",
			InitImage:    "bscp-init:test",
			SidecarImage: "bscp-sidecar:test",
		}
	})

	AfterEach(func() {
		svccfg.G.BSCP = oldBSCPCf
		_, _ = bindings.DeleteMany(ctx, model.ByApp(appID))
		_ = metas.DeleteIfExists(ctx, model.ByApp(appID))
		Expect(testutil.CleanupMetaColl(cfgswitchmodel.WorkspaceCfgSwitchMeta)).To(Succeed())
	})

	Context("when the workspace has never enabled bscp", func() {
		It("should not inject even if the app is configured", func() {
			createSnapshot()

			fragment, err := bscpcfg.BuildFromStore(ctx, db, appID, "dev", ws)
			Expect(err).NotTo(HaveOccurred())
			Expect(fragment).To(BeNil())
		})
	})

	Context("when the workspace has disabled bscp", func() {
		It("should not inject even if the app is configured", func() {
			createSnapshot()
			setBscpEnabled(false)

			fragment, err := bscpcfg.BuildFromStore(ctx, db, appID, "dev", ws)
			Expect(err).NotTo(HaveOccurred())
			Expect(fragment).To(BeNil())
		})
	})

	Context("when the workspace has enabled bscp", func() {
		BeforeEach(func() {
			setBscpEnabled(true)
		})

		It("should not inject when the app is not configured", func() {
			fragment, err := bscpcfg.BuildFromStore(ctx, db, appID, "dev", ws)
			Expect(err).NotTo(HaveOccurred())
			Expect(fragment).To(BeNil())
		})

		It("should build the fragment from the snapshot", func() {
			createSnapshot()

			fragment, err := bscpcfg.BuildFromStore(ctx, db, appID, "dev", ws)
			Expect(err).NotTo(HaveOccurred())
			Expect(fragment).NotTo(BeNil())
			Expect(fragment.WorkloadName).To(Equal("test-workload"))
			Expect(fragment.WorkloadKind).To(Equal("Deployment"))
			Expect(fragment.InitContainers).To(HaveLen(1))
			Expect(fragment.Containers).To(HaveLen(1))
			Expect(fragment.MainContainerVolumeMounts[0].MountPath).To(Equal("/data/bscp"))
		})
	})
})
