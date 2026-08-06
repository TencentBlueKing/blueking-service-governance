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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("AlertStrategyStoreMongo", func() {
	var (
		ctx   context.Context
		store Store
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

		err := testutil.CleanupCollection("bkmonitor_alert_strategy")
		Expect(err).NotTo(HaveOccurred())

		store, err = NewStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Create and Get", func() {
		It("should create and retrieve by ID", func() {
			rule := &AlertStrategy{
				WorkspaceID:   "ws-1",
				AppID:         "app-1",
				AppName:       "app-one",
				StrategyCode:  "cpu_high",
				DisplayName:   "CPU 过高",
				MonitorMetric: "cpu_usage",
				Severity:      AlertSeverityWarning,
				Threshold:     ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				Creator: "user-1",
				Updater: "user-1",
			}

			id, err := store.Create(ctx, rule)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(Equal(bson.NilObjectID))

			got, err := store.Get(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.StrategyCode).To(Equal("cpu_high"))
			Expect(got.DisplayName).To(Equal("CPU 过高"))
			Expect(got.WorkspaceID).To(Equal("ws-1"))
			Expect(got.AppID).To(Equal("app-1"))
		})
	})

	Describe("List", func() {
		It("should list rules by workspace", func() {
			for _, code := range []string{"rule_1", "rule_2", "rule_3"} {
				_, err := store.Create(ctx, newStoreRule("ws-list", "app-1", "app-one", code))
				Expect(err).NotTo(HaveOccurred())
			}

			_, err := store.Create(ctx, newStoreRule("ws-other", "app-2", "app-two", "rule_x"))
			Expect(err).NotTo(HaveOccurred())

			rules, err := store.ListByWorkspace(ctx, "ws-list")
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(HaveLen(3))
		})

		It("should list rules by app", func() {
			_, err := store.Create(ctx, newStoreRule("ws-list", "app-a", "app-a", "rule_1"))
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Create(ctx, newStoreRule("ws-list", "app-b", "app-b", "rule_1"))
			Expect(err).NotTo(HaveOccurred())

			rules, err := store.ListByApp(ctx, "ws-list", "app-a")
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
			_, err := store.Create(ctx, matched)
			Expect(err).NotTo(HaveOccurred())

			otherEnv := newStoreRule("ws-list", "app-a", "app-a", "rule_other_env")
			otherEnv.RemoteRefs = []RemoteStrategyRef{
				{EnvID: otherEnvID, TrafficLaneName: "feature-a", RemoteStrategyID: 1002},
			}
			_, err = store.Create(ctx, otherEnv)
			Expect(err).NotTo(HaveOccurred())

			otherApp := newStoreRule("ws-list", "app-b", "app-b", "rule_other_app")
			otherApp.RemoteRefs = []RemoteStrategyRef{
				{EnvID: targetEnvID, TrafficLaneName: "feature-a", RemoteStrategyID: 1003},
			}
			_, err = store.Create(ctx, otherApp)
			Expect(err).NotTo(HaveOccurred())

			otherLane := newStoreRule("ws-list", "app-a", "app-a", "rule_other_lane")
			otherLane.RemoteRefs = []RemoteStrategyRef{
				{EnvID: targetEnvID, TrafficLaneName: "", RemoteStrategyID: 1004},
			}
			_, err = store.Create(ctx, otherLane)
			Expect(err).NotTo(HaveOccurred())

			strategies, err := store.ListByAppAndRemoteEnv(ctx, "ws-list", "app-a", targetEnvID, "feature-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(strategies).To(HaveLen(1))
			Expect(strategies[0].StrategyCode).To(Equal("rule_match"))
		})

		It("should list enabled strategies by app matching env scope", func() {
			targetEnvID := bson.NewObjectID()

			matchedAll := newStoreRule("ws-list", "app-a", "app-a", "rule_all")
			matchedAll.Enabled = true
			_, err := store.Create(ctx, matchedAll)
			Expect(err).NotTo(HaveOccurred())

			matchedEnvType := newStoreRule("ws-list", "app-a", "app-a", "rule_env_type")
			matchedEnvType.Enabled = true
			matchedEnvType.EffectiveScope = EffectiveScope{
				Type:     EffectiveScopeEnvType,
				EnvTypes: []string{"production"},
			}
			_, err = store.Create(ctx, matchedEnvType)
			Expect(err).NotTo(HaveOccurred())

			matchedSpecificEnv := newStoreRule("ws-list", "app-a", "app-a", "rule_specific_env")
			matchedSpecificEnv.Enabled = true
			matchedSpecificEnv.EffectiveScope = EffectiveScope{
				Type:   EffectiveScopeSpecificEnvs,
				EnvIDs: []bson.ObjectID{targetEnvID},
			}
			_, err = store.Create(ctx, matchedSpecificEnv)
			Expect(err).NotTo(HaveOccurred())

			disabledAll := newStoreRule("ws-list", "app-a", "app-a", "rule_disabled")
			disabledAll.Enabled = false
			_, err = store.Create(ctx, disabledAll)
			Expect(err).NotTo(HaveOccurred())

			otherType := newStoreRule("ws-list", "app-a", "app-a", "rule_other_type")
			otherType.Enabled = true
			otherType.EffectiveScope = EffectiveScope{
				Type:     EffectiveScopeEnvType,
				EnvTypes: []string{"development"},
			}
			_, err = store.Create(ctx, otherType)
			Expect(err).NotTo(HaveOccurred())

			otherApp := newStoreRule("ws-list", "app-b", "app-b", "rule_other_app")
			otherApp.Enabled = true
			_, err = store.Create(ctx, otherApp)
			Expect(err).NotTo(HaveOccurred())

			strategies, err := store.ListEnabledByAppMatchingEnv(ctx, "ws-list", "app-a", "production", targetEnvID)
			Expect(err).NotTo(HaveOccurred())
			Expect(strategies).To(HaveLen(3))
			Expect(strategies[0].StrategyCode).To(Equal("rule_specific_env"))
			Expect(strategies[1].StrategyCode).To(Equal("rule_env_type"))
			Expect(strategies[2].StrategyCode).To(Equal("rule_all"))
		})
	})

	Describe("Update", func() {
		It("should update mutable fields", func() {
			rule := &AlertStrategy{
				WorkspaceID:   "ws-1",
				AppID:         "app-1",
				AppName:       "app-one",
				StrategyCode:  "update_me",
				DisplayName:   "Before",
				MonitorMetric: "cpu_usage",
				Severity:      AlertSeverityInfo,
				Threshold:     ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				Creator: "tester",
				Updater: "tester",
			}
			id, err := store.Create(ctx, rule)
			Expect(err).NotTo(HaveOccurred())

			rule.ID = id
			rule.DisplayName = "After"
			rule.Severity = AlertSeverityFatal

			err = store.Update(ctx, id, bson.M{
				"displayName": rule.DisplayName,
				"severity":    rule.Severity,
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := store.Get(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.DisplayName).To(Equal("After"))
			Expect(got.Severity).To(Equal(AlertSeverityFatal))
		})
	})

	Describe("Delete", func() {
		It("should delete by ID", func() {
			rule := &AlertStrategy{
				WorkspaceID:   "ws-1",
				AppID:         "app-1",
				AppName:       "app-one",
				StrategyCode:  "delete_me",
				DisplayName:   "Delete Me",
				MonitorMetric: "cpu_usage",
				Severity:      AlertSeverityInfo,
				Threshold:     ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				Creator: "tester",
				Updater: "tester",
			}
			id, err := store.Create(ctx, rule)
			Expect(err).NotTo(HaveOccurred())

			err = store.Delete(ctx, id)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Get(ctx, id)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(ErrNotFound))
		})
	})

	Describe("Repeated strategyCode", func() {
		It("should allow duplicate strategyCode for different apps in same workspace", func() {
			_, err := store.Create(ctx, newStoreRule("ws-dup", "app-a", "app-a", "same_code"))
			Expect(err).NotTo(HaveOccurred())

			second := newStoreRule("ws-dup", "app-b", "app-b", "same_code")
			second.Severity = AlertSeverityWarning
			_, err = store.Create(ctx, second)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow duplicate strategyCode in the same app", func() {
			_, err := store.Create(ctx, newStoreRule("ws-a", "app-a", "app-a", "same_code"))
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Create(ctx, newStoreRule("ws-a", "app-a", "app-a", "same_code"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Indexes", func() {
		It("should create the expected compound indexes", func() {
			storeMongo := store.(*StoreMongo)
			cursor, err := storeMongo.collection.Indexes().List(ctx)
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
			Expect(indexByName).NotTo(HaveKey("workspaceID_1_enabled_1"))
		})
	})
})
