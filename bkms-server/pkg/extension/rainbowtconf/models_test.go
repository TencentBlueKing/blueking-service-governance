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

package rainbowtconf_test

import (
	"context"
	"strings"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	cfgswitchmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/rainbowtconf"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const validURL = "https://rainbow.example.com/console/"

var _ = Describe("NormalizeURL", func() {
	It("should trim surrounding spaces", func() {
		u, err := rainbowtconf.NormalizeURL("  " + validURL + "  ")
		Expect(err).NotTo(HaveOccurred())
		Expect(u).To(Equal(validURL))
	})

	It("should accept query and fragment", func() {
		raw := "https://rainbow.example.com/console/#/project/1?env=prod"
		Expect(rainbowtconf.NormalizeURL(raw)).To(Equal(raw))
	})

	DescribeTable("rejects invalid urls",
		func(raw string) {
			_, err := rainbowtconf.NormalizeURL(raw)
			Expect(err).To(MatchError(rainbowtconf.ErrInvalidURL))
		},
		Entry("blank", "   "),
		Entry("http scheme", "http://rainbow.example.com/console/"),
		Entry("javascript scheme", "javascript:alert(1)"),
		Entry("no scheme", "rainbow.example.com/console/"),
		Entry("empty host", "https:///console/"),
		Entry("user info", "https://user:pass@rainbow.example.com/"),
		Entry("unparsable", "https://rainbow.example.com/%zz"),
		Entry("too long", "https://rainbow.example.com/"+strings.Repeat("a", 2048)),
	)
})

var _ = Describe("Entry", func() {
	var (
		ctx     context.Context
		db      *mongo.Database
		entries *rainbowtconf.EntryStore
		appID   string
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		entries = testutil.NewStore(GinkgoT(), rainbowtconf.EntryMeta)
		appID = "test-app-" + stringx.Random(6)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(rainbowtconf.EntryMeta)).To(Succeed())
	})

	Describe("GetEntry", func() {
		It("should return nil when not configured", func() {
			b, err := rainbowtconf.GetEntry(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(b).To(BeNil())
		})
	})

	Describe("SaveURL", func() {
		It("should create the entry on first save", func() {
			b, err := rainbowtconf.SaveURL(ctx, db, appID, " "+validURL+" ", "tester")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.URL).To(Equal(validURL))
			Expect(b.Updater).To(Equal("tester"))
			Expect(b.UpdatedAt.IsZero()).To(BeFalse())

			stored, err := entries.Get(ctx, rainbowtconf.ByApp(appID))
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.URL).To(Equal(validURL))
		})

		It("should update the url and updater", func() {
			_, err := rainbowtconf.SaveURL(ctx, db, appID, validURL, "tester")
			Expect(err).NotTo(HaveOccurred())

			newURL := "https://rainbow.example.com/console/#/project/2"
			b, err := rainbowtconf.SaveURL(ctx, db, appID, newURL, "another")
			Expect(err).NotTo(HaveOccurred())
			Expect(b.URL).To(Equal(newURL))
			Expect(b.Updater).To(Equal("another"))
			Expect(entries.Count(ctx, rainbowtconf.ByApp(appID))).To(BeEquivalentTo(1))
		})

		It("should not write when the url is unchanged", func() {
			first, err := rainbowtconf.SaveURL(ctx, db, appID, validURL, "tester")
			Expect(err).NotTo(HaveOccurred())
			second, err := rainbowtconf.SaveURL(ctx, db, appID, validURL, "another")
			Expect(err).NotTo(HaveOccurred())
			Expect(second.Updater).To(Equal("tester"))
			Expect(second.UpdatedAt).To(Equal(first.UpdatedAt))
		})

		It("should reject an invalid url without writing", func() {
			_, err := rainbowtconf.SaveURL(ctx, db, appID, "http://rainbow.example.com/", "tester")
			Expect(err).To(MatchError(rainbowtconf.ErrInvalidURL))
			Expect(entries.Exists(ctx, rainbowtconf.ByApp(appID))).To(BeFalse())
		})

		It("should keep entries of different tenants isolated", func() {
			_, err := rainbowtconf.SaveURL(ctx, db, appID, validURL, "tester")
			Expect(err).NotTo(HaveOccurred())
			Expect(entries.Exists(ctx, rainbowtconf.ByApp(appID), store.TenantOf("another-tenant"))).To(BeFalse())
		})
	})

	Describe("DeleteByApp", func() {
		It("should remove the entry of the app only", func() {
			otherAppID := "test-app-" + stringx.Random(6)
			_, err := rainbowtconf.SaveURL(ctx, db, appID, validURL, "tester")
			Expect(err).NotTo(HaveOccurred())
			_, err = rainbowtconf.SaveURL(ctx, db, otherAppID, validURL, "tester")
			Expect(err).NotTo(HaveOccurred())

			Expect(rainbowtconf.DeleteByApp(ctx, db, appID)).To(Succeed())
			Expect(entries.Exists(ctx, rainbowtconf.ByApp(appID))).To(BeFalse())
			Expect(entries.Exists(ctx, rainbowtconf.ByApp(otherAppID))).To(BeTrue())
		})

		It("should succeed when the app has no entry", func() {
			Expect(rainbowtconf.DeleteByApp(ctx, db, appID)).To(Succeed())
		})
	})
})

var _ = Describe("RequireEnabled", func() {
	var (
		ctx      context.Context
		db       *mongo.Database
		switches *cfgswitchmodel.WorkspaceCfgSwitchStore
		wsID     string
	)

	setEnabled := func(enabled bool) {
		key := cfgswitchmodel.SwitchRainbowTConf
		target := &cfgswitchmodel.WorkspaceCfgSwitch{WorkspaceID: wsID, Key: key, Enabled: enabled}
		_, err := switches.Upsert(
			ctx, target, cfgswitchmodel.ByWorkspaceAndKey(wsID, key), store.UpdateSelected("enabled"),
		)
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		switches = testutil.NewStore(GinkgoT(), cfgswitchmodel.WorkspaceCfgSwitchMeta)
		wsID = "ws-" + stringx.Random(6)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(cfgswitchmodel.WorkspaceCfgSwitchMeta)).To(Succeed())
	})

	It("should reject when the switch has never been set", func() {
		Expect(rainbowtconf.RequireEnabled(ctx, db, wsID)).To(MatchError(rainbowtconf.ErrNotEnabled))
	})

	It("should reject when the switch is disabled", func() {
		setEnabled(false)
		Expect(rainbowtconf.RequireEnabled(ctx, db, wsID)).To(MatchError(rainbowtconf.ErrNotEnabled))
	})

	It("should pass when the switch is enabled", func() {
		setEnabled(true)
		Expect(rainbowtconf.RequireEnabled(ctx, db, wsID)).To(Succeed())
	})

	It("should not be affected by the bscp switch", func() {
		key := cfgswitchmodel.SwitchBscp
		target := &cfgswitchmodel.WorkspaceCfgSwitch{WorkspaceID: wsID, Key: key, Enabled: true}
		_, err := switches.Upsert(
			ctx, target, cfgswitchmodel.ByWorkspaceAndKey(wsID, key), store.UpdateSelected("enabled"),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(rainbowtconf.RequireEnabled(ctx, db, wsID)).To(MatchError(rainbowtconf.ErrNotEnabled))
	})
})
