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

package strategy

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("AlertStrategyStore", func() {
	var (
		ctx        context.Context
		strategies *Store
	)

	newStoreRule := func(workspaceID, appID, appName, strategyCode string) *AlertStrategy {
		return &AlertStrategy{
			WorkspaceID:   workspaceID,
			AppID:         appID,
			AppName:       appName,
			StrategyCode:  strategyCode,
			DisplayName:   strategyCode + "-display",
			MonitorMetric: "cpu_usage",
			Severity:      AlertSeverityInfo,
			Threshold:     ThresholdConfig{Method: "gte", Value: 80},
			EffectiveScope: EffectiveScope{
				Type: EffectiveScopeAll,
			},
			Creator: "tester",
			Updater: "tester",
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		strategies = testutil.NewStore(GinkgoT(), Meta)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(Meta)).To(Succeed())
	})

	Describe("Create and Get", func() {
		It("should store empty slices instead of null for remoteRefs and noticeGroupIDs", func() {
			rule := newStoreRule("ws-1", "app-1", "app-one", "empty_slices")
			Expect(strategies.Create(ctx, rule)).To(Succeed())

			raw, err := strategies.Collection().FindOne(ctx, bson.M{"_id": rule.ID}).Raw()
			Expect(err).NotTo(HaveOccurred())
			Expect(raw.Lookup("remoteRefs").Type).To(Equal(bson.TypeArray))
			Expect(raw.Lookup("noticeGroupIDs").Type).To(Equal(bson.TypeArray))
		})

		It("should reject strategies missing required fields", func() {
			rule := newStoreRule("ws-1", "app-1", "app-one", "missing_app_name")
			rule.AppName = ""
			Expect(strategies.Create(ctx, rule)).To(MatchError(ContainSubstring("AlertStrategy validation failed")))
		})

		It("should reject strategies with an invalid effective scope", func() {
			rule := newStoreRule("ws-1", "app-1", "app-one", "invalid_scope")
			rule.EffectiveScope = EffectiveScope{Type: EffectiveScopeEnvType}
			Expect(strategies.Create(ctx, rule)).To(
				MatchError(ContainSubstring("AlertStrategy effectiveScope validation failed")),
			)
		})

		It("should return ErrNotFound when the strategy does not exist", func() {
			_, err := strategies.GetByID(ctx, bson.NewObjectID())
			Expect(err).To(MatchError(ErrNotFound))
		})
	})

	Describe("List", func() {
		It("should list rules by workspace", func() {
			for _, code := range []string{"rule_1", "rule_2", "rule_3"} {
				Expect(strategies.Create(ctx, newStoreRule("ws-list", "app-1", "app-one", code))).To(Succeed())
			}
			Expect(strategies.Create(ctx, newStoreRule("ws-other", "app-2", "app-two", "rule_x"))).To(Succeed())

			rules, err := strategies.List(ctx, ByWorkspace("ws-list"), NewestFirst())
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(HaveLen(3))
			// 按创建时间倒序，最后创建的排在最前
			Expect(rules[0].StrategyCode).To(Equal("rule_3"))
		})

		It("should list rules by app", func() {
			Expect(strategies.Create(ctx, newStoreRule("ws-list", "app-a", "app-a", "rule_1"))).To(Succeed())
			Expect(strategies.Create(ctx, newStoreRule("ws-list", "app-b", "app-b", "rule_1"))).To(Succeed())

			rules, err := strategies.List(ctx, ByApp("ws-list", "app-a"), NewestFirst())
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(HaveLen(1))
			Expect(rules[0].AppID).To(Equal("app-a"))
		})

		It("should list strategies by app and remote env lane", func() {
			targetEnvID := bson.NewObjectID()
			otherEnvID := bson.NewObjectID()

			matched := newStoreRule("ws-list", "app-a", "app-a", "rule_match")
			matched.RemoteRefs = []RemoteStrategyRef{
				{EnvID: targetEnvID, TrafficLaneName: "feature-a", RemoteStrategyID: 1001},
			}
			Expect(strategies.Create(ctx, matched)).To(Succeed())

			otherEnv := newStoreRule("ws-list", "app-a", "app-a", "rule_other_env")
			otherEnv.RemoteRefs = []RemoteStrategyRef{
				{EnvID: otherEnvID, TrafficLaneName: "feature-a", RemoteStrategyID: 1002},
			}
			Expect(strategies.Create(ctx, otherEnv)).To(Succeed())

			otherApp := newStoreRule("ws-list", "app-b", "app-b", "rule_other_app")
			otherApp.RemoteRefs = []RemoteStrategyRef{
				{EnvID: targetEnvID, TrafficLaneName: "feature-a", RemoteStrategyID: 1003},
			}
			Expect(strategies.Create(ctx, otherApp)).To(Succeed())

			otherLane := newStoreRule("ws-list", "app-a", "app-a", "rule_other_lane")
			otherLane.RemoteRefs = []RemoteStrategyRef{
				{EnvID: targetEnvID, TrafficLaneName: "", RemoteStrategyID: 1004},
			}
			Expect(strategies.Create(ctx, otherLane)).To(Succeed())

			got, err := strategies.List(ctx, ByRemoteEnv("ws-list", "app-a", targetEnvID, "feature-a"), NewestFirst())
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(1))
			Expect(got[0].StrategyCode).To(Equal("rule_match"))
		})

		It("should list enabled strategies by app matching env scope", func() {
			targetEnvID := bson.NewObjectID()

			matchedAll := newStoreRule("ws-list", "app-a", "app-a", "rule_all")
			matchedAll.Enabled = true
			Expect(strategies.Create(ctx, matchedAll)).To(Succeed())

			matchedEnvType := newStoreRule("ws-list", "app-a", "app-a", "rule_env_type")
			matchedEnvType.Enabled = true
			matchedEnvType.EffectiveScope = EffectiveScope{
				Type:     EffectiveScopeEnvType,
				EnvTypes: []string{"production"},
			}
			Expect(strategies.Create(ctx, matchedEnvType)).To(Succeed())

			matchedSpecificEnv := newStoreRule("ws-list", "app-a", "app-a", "rule_specific_env")
			matchedSpecificEnv.Enabled = true
			matchedSpecificEnv.EffectiveScope = EffectiveScope{
				Type:   EffectiveScopeSpecificEnvs,
				EnvIDs: []bson.ObjectID{targetEnvID},
			}
			Expect(strategies.Create(ctx, matchedSpecificEnv)).To(Succeed())

			disabledAll := newStoreRule("ws-list", "app-a", "app-a", "rule_disabled")
			disabledAll.Enabled = false
			Expect(strategies.Create(ctx, disabledAll)).To(Succeed())

			otherType := newStoreRule("ws-list", "app-a", "app-a", "rule_other_type")
			otherType.Enabled = true
			otherType.EffectiveScope = EffectiveScope{
				Type:     EffectiveScopeEnvType,
				EnvTypes: []string{"development"},
			}
			Expect(strategies.Create(ctx, otherType)).To(Succeed())

			otherApp := newStoreRule("ws-list", "app-b", "app-b", "rule_other_app")
			otherApp.Enabled = true
			Expect(strategies.Create(ctx, otherApp)).To(Succeed())

			got, err := strategies.List(
				ctx, EnabledMatchingEnv("ws-list", "app-a", "production", targetEnvID), NewestFirst(),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(HaveLen(3))
			Expect(got[0].StrategyCode).To(Equal("rule_specific_env"))
			Expect(got[1].StrategyCode).To(Equal("rule_env_type"))
			Expect(got[2].StrategyCode).To(Equal("rule_all"))
		})

		It("should detect duplicate displayName in same app", func() {
			first := newStoreRule("ws-list", "app-a", "app-a", "rule_a")
			first.DisplayName = "Same Name"
			Expect(strategies.Create(ctx, first)).To(Succeed())

			second := newStoreRule("ws-list", "app-a", "app-a", "rule_b")
			second.DisplayName = "Same Name"
			err := strategies.Create(ctx, second)
			Expect(err).To(MatchError(ErrDisplayNameAlreadyExists))
			Expect(err).To(MatchError(store.ErrDuplicate))

			exists, err := strategies.Exists(ctx, ByDisplayName("ws-list", "app-a", "Same Name", nil))
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			exists, err = strategies.Exists(ctx, ByDisplayName("ws-list", "app-a", "Same Name", &first.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("Repeated strategyCode", func() {
		It("should allow duplicate strategyCode for different apps in same workspace", func() {
			Expect(strategies.Create(ctx, newStoreRule("ws-dup", "app-a", "app-a", "same_code"))).To(Succeed())

			second := newStoreRule("ws-dup", "app-b", "app-b", "same_code")
			second.Severity = AlertSeverityWarning
			Expect(strategies.Create(ctx, second)).To(Succeed())
		})

		It("should allow duplicate strategyCode in the same app", func() {
			Expect(strategies.Create(ctx, newStoreRule("ws-a", "app-a", "app-a", "same_code"))).To(Succeed())

			second := newStoreRule("ws-a", "app-a", "app-a", "same_code")
			second.DisplayName = "same_code-display-2"
			Expect(strategies.Create(ctx, second)).To(Succeed())
		})
	})

	Describe("Indexes", func() {
		It("should create the expected compound indexes", func() {
			cursor, err := strategies.Collection().Indexes().List(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer cursor.Close(ctx) // nolint

			type indexDoc struct {
				Name   string `bson:"name"`
				Unique bool   `bson:"unique,omitempty"`
			}
			var indexes []indexDoc
			err = cursor.All(ctx, &indexes)
			Expect(err).NotTo(HaveOccurred())

			indexByName := make(map[string]indexDoc, len(indexes))
			for _, idx := range indexes {
				indexByName[idx.Name] = idx
			}

			Expect(indexByName).To(HaveKey("workspaceID_1_appID_1_enabled_1"))
			Expect(indexByName).To(HaveKey("workspaceID_1_appID_1_displayName_1"))
			Expect(indexByName["workspaceID_1_appID_1_displayName_1"].Unique).To(BeTrue())
			Expect(indexByName).NotTo(HaveKey("workspaceID_1_enabled_1"))
		})
	})
})
