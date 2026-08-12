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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

type failAlertStrategyClient struct {
	*bkmapi.StubClient
	saveErr     error
	switchErr   error
	deleteErr   error
	deleteCalls int
}

func (c *failAlertStrategyClient) SaveAlarmStrategy(
	_ context.Context,
	_ *bkmapi.SaveAlarmStrategyReq,
) (*bkmapi.SaveAlarmStrategyResp, error) {
	if c.saveErr != nil {
		return nil, c.saveErr
	}
	return &bkmapi.SaveAlarmStrategyResp{ID: 1001}, nil
}

func (c *failAlertStrategyClient) SwitchAlarmStrategy(
	_ context.Context,
	_ *bkmapi.SwitchAlarmStrategyReq,
) error {
	return c.switchErr
}

func (c *failAlertStrategyClient) DeleteAlarmStrategy(
	_ context.Context,
	_ *bkmapi.DeleteAlarmStrategyReq,
) error {
	c.deleteCalls++
	return c.deleteErr
}

type failOnNthDeleteAlertStrategyClient struct {
	*bkmapi.StubClient
	failAtCall int
	deleteReqs []*bkmapi.DeleteAlarmStrategyReq
	deleteErr  error
	callCount  int
}

func (c *failOnNthDeleteAlertStrategyClient) DeleteAlarmStrategy(
	_ context.Context,
	req *bkmapi.DeleteAlarmStrategyReq,
) error {
	c.callCount++
	c.deleteReqs = append(c.deleteReqs, req)
	if c.callCount == c.failAtCall {
		return c.deleteErr
	}
	return nil
}

type failingCreateStore struct {
	Store
	createErr error
}

func (s *failingCreateStore) Create(_ context.Context, _ *AlertStrategy) (bson.ObjectID, error) {
	return bson.NilObjectID, s.createErr
}

var _ = Describe("AlertStrategyService", func() {
	var (
		ctx      context.Context
		diApp    *fxtest.App
		store    Store
		envStore envmodel.EnvironmentStore
		svc      *Service
	)

	newCreateReq := func() *CreateReq {
		return &CreateReq{
			WorkspaceID:  "test-ws",
			AppID:        "app-1",
			AppName:      "demo-app",
			StrategyCode: "test_strategy",
			DisplayName:  "Test Strategy",
			Severity:     AlertSeverityInfo,
			Threshold:    ThresholdConfig{Method: "gte", Value: 50},
			EffectiveScope: EffectiveScope{
				Type: EffectiveScopeAll,
			},
			Operator: "test-user",
		}
	}

	createApp := func(workspaceID, appID, appName string) *bkmsapp.Application {
		return dbfactory.ApplicationWithOpts(ctx, svc.appStore, &dbfactory.ApplicationOpts{
			WorkspaceID: workspaceID,
			ID:          appID,
			Name:        appName,
		})
	}

	BeforeEach(func() {
		ctx = context.Background()

		err := testutil.CleanupCollection(collectionName)
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("environments")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("applications")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("resource_snapshots")
		Expect(err).NotTo(HaveOccurred())

		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			database.PrivateFxModule,
			fx.Provide(
				envmodel.NewEnvironmentStoreMongo,
				fx.Annotate(bkmsapp.NewApplicationStoreMongo, fx.As(new(bkmsapp.ApplicationStore))),
				topology.NewResourceSnapshotStoreMongo,
				func(store *topology.ResourceSnapshotStoreMongo) topology.ResourceSnapshotStore { return store },
			),
			fx.Populate(&svc, &store, &envStore),
		)
		diApp.RequireStart()

		svc = newServiceWithClientFactory(
			svc.store,
			svc.envStore,
			svc.appStore,
			svc.snapshotStore,
			func(_ string) (bkmapi.MonitorClient, error) {
				return bkmapi.NewStub("test-user"), nil
			},
		)
	})

	AfterEach(func() {
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	Describe("Create", func() {
		It("should create alert strategy successfully", func() {
			req := newCreateReq()
			req.AppName = "app-one"
			req.StrategyCode = "cpu_high"
			req.DisplayName = "CPU 过高"
			req.Severity = AlertSeverityWarning
			req.Threshold = ThresholdConfig{Method: "gte", Value: 80}
			req.Enabled = true

			rule, err := svc.Create(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(rule).NotTo(BeNil())
			Expect(rule.StrategyCode).To(Equal("cpu_high"))
			Expect(rule.AppID).To(Equal("app-1"))
			Expect(rule.ID).NotTo(Equal(bson.NilObjectID))
		})

		It("should fill monitorMetric from strategyCode when request omits it", func() {
			req := newCreateReq()
			req.StrategyCode = "cpu_limit_usage_high"

			rule, err := svc.Create(ctx, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(rule.MonitorMetric).To(Equal("container_cpu_usage_seconds_total"))
		})

		It("should allow duplicate strategyCode in same workspace for different apps", func() {
			firstReq := newCreateReq()
			firstReq.AppID = "app-a"
			firstReq.AppName = "app-a"
			firstReq.StrategyCode = "dup_code"
			firstReq.DisplayName = "First"

			_, err := svc.Create(ctx, firstReq)
			Expect(err).NotTo(HaveOccurred())

			secondReq := newCreateReq()
			secondReq.AppID = "app-b"
			secondReq.AppName = "app-b"
			secondReq.StrategyCode = "dup_code"
			secondReq.DisplayName = "Second"

			_, err = svc.Create(ctx, secondReq)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject duplicate displayName in same app", func() {
			firstReq := newCreateReq()
			firstReq.AppID = "app-a"
			firstReq.AppName = "app-a"
			firstReq.StrategyCode = "code_a"
			firstReq.DisplayName = "Same Name"

			_, err := svc.Create(ctx, firstReq)
			Expect(err).NotTo(HaveOccurred())

			secondReq := newCreateReq()
			secondReq.AppID = "app-a"
			secondReq.AppName = "app-a"
			secondReq.StrategyCode = "code_b"
			secondReq.DisplayName = "Same Name"

			_, err = svc.Create(ctx, secondReq)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("CreateAndSync", func() {
		It("should create local strategy and sync remote when enabled", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"
			createApp("test-ws", "app-1", "demo-app")

			_, err := envStore.Create(ctx, &envmodel.Environment{
				Name:        "create-env",
				Type:        "production",
				WorkspaceID: "test-ws",
				Kind:        envmodel.EnvironmentKindStandard,
				AppIDs:      []string{"app-1"},
				Cluster: envmodel.BizCluster{
					ClusterID: "BCS-K8S-00001",
					Namespace: "ns-create",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			req := newCreateReq()
			req.StrategyCode = "create_sync_test"
			req.DisplayName = "Create Sync Test"
			req.Enabled = true

			rule, err := svc.CreateAndSync(ctx, ws, req)

			Expect(err).NotTo(HaveOccurred())
			Expect(rule.ID).NotTo(Equal(bson.NilObjectID))

			stored, getErr := store.Get(ctx, rule.ID)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.RemoteRefs).To(HaveLen(1))
		})

		It("should not persist local strategy when remote sync fails", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"
			createApp("test-ws", "app-1", "demo-app")

			_, err := envStore.Create(ctx, &envmodel.Environment{
				Name:        "create-fail-env",
				Type:        "production",
				WorkspaceID: "test-ws",
				Kind:        envmodel.EnvironmentKindStandard,
				AppIDs:      []string{"app-1"},
				Cluster: envmodel.BizCluster{
					ClusterID: "BCS-K8S-00001",
					Namespace: "ns-create-fail",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			failSvc := newServiceWithClientFactory(
				svc.store,
				svc.envStore,
				svc.appStore,
				svc.snapshotStore,
				func(_ string) (bkmapi.MonitorClient, error) {
					return &failAlertStrategyClient{
						StubClient: bkmapi.NewStub("test-user"),
						saveErr:    bkmapi.ErrApmAppNotFound,
					}, nil
				},
			)

			req := newCreateReq()
			req.StrategyCode = "create_sync_fail_test"
			req.DisplayName = "Create Sync Fail Test"
			req.Enabled = true

			_, err = failSvc.CreateAndSync(ctx, ws, req)

			Expect(err).To(HaveOccurred())
			rules, listErr := store.ListByApp(ctx, "test-ws", "app-1")
			Expect(listErr).NotTo(HaveOccurred())
			Expect(rules).To(BeEmpty())
		})

		It("should return create error directly when local persistence fails", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"
			createApp("test-ws", "app-1", "demo-app")

			_, err := envStore.Create(ctx, &envmodel.Environment{
				Name:        "create-store-fail-env",
				Type:        "production",
				WorkspaceID: "test-ws",
				Kind:        envmodel.EnvironmentKindStandard,
				AppIDs:      []string{"app-1"},
				Cluster: envmodel.BizCluster{
					ClusterID: "BCS-K8S-00001",
					Namespace: "ns-create-store-fail",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			client := &failAlertStrategyClient{StubClient: bkmapi.NewStub("test-user")}
			failSvc := newServiceWithClientFactory(
				&failingCreateStore{Store: svc.store, createErr: errors.New("create failed")},
				svc.envStore,
				svc.appStore,
				svc.snapshotStore,
				func(_ string) (bkmapi.MonitorClient, error) {
					return client, nil
				},
			)

			req := newCreateReq()
			req.StrategyCode = "create_store_fail_test"
			req.DisplayName = "Create Store Fail Test"
			req.Enabled = true

			_, err = failSvc.CreateAndSync(ctx, ws, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("create alert strategy"))
			Expect(client.deleteCalls).To(Equal(0))
		})
	})

	Describe("resolveEffectiveEnvs", func() {
		var (
			envDev  envmodel.Environment
			envTest envmodel.Environment
			envProd envmodel.Environment
		)

		BeforeEach(func() {
			envDev = envmodel.Environment{
				Name:        "dev",
				Type:        "development",
				WorkspaceID: "test-ws",
				AppIDs:      []string{"app-1"},
			}
			envTest = envmodel.Environment{
				Name:        "test",
				Type:        "test",
				WorkspaceID: "test-ws",
				AppIDs:      []string{"app-2"},
			}
			envProd = envmodel.Environment{
				Name:        "prod",
				Type:        "production",
				WorkspaceID: "test-ws",
				AppIDs:      []string{"app-1"},
			}
			var err error
			envDev.ID, err = envStore.Create(ctx, &envDev)
			Expect(err).NotTo(HaveOccurred())
			envTest.ID, err = envStore.Create(ctx, &envTest)
			Expect(err).NotTo(HaveOccurred())
			envProd.ID, err = envStore.Create(ctx, &envProd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return all envs for scope=all", func() {
			rule := &AlertStrategy{
				WorkspaceID:    "test-ws",
				AppID:          "app-1",
				EffectiveScope: EffectiveScope{Type: EffectiveScopeAll},
			}
			envs, err := svc.resolveEffectiveEnvs(ctx, rule)
			Expect(err).NotTo(HaveOccurred())
			Expect(envs).To(HaveLen(2))
		})

		It("should filter envs by type for scope=env_type", func() {
			rule := &AlertStrategy{
				WorkspaceID: "test-ws",
				AppID:       "app-1",
				EffectiveScope: EffectiveScope{
					Type:     EffectiveScopeEnvType,
					EnvTypes: []string{"production"},
				},
			}
			envs, err := svc.resolveEffectiveEnvs(ctx, rule)
			Expect(err).NotTo(HaveOccurred())
			Expect(envs).To(HaveLen(1))
			Expect(envs[0].Name).To(Equal("prod"))
		})

		It("should filter envs by IDs for scope=specific_envs", func() {
			rule := &AlertStrategy{
				WorkspaceID: "test-ws",
				AppID:       "app-1",
				EffectiveScope: EffectiveScope{
					Type:   EffectiveScopeSpecificEnvs,
					EnvIDs: []bson.ObjectID{envDev.ID, envProd.ID},
				},
			}
			envs, err := svc.resolveEffectiveEnvs(ctx, rule)
			Expect(err).NotTo(HaveOccurred())
			Expect(envs).To(HaveLen(2))
			names := []string{envs[0].Name, envs[1].Name}
			Expect(names).To(ContainElements("dev", "prod"))
		})

		It("should return empty for scope=env_type with no matching types", func() {
			rule := &AlertStrategy{
				WorkspaceID: "test-ws",
				AppID:       "app-1",
				EffectiveScope: EffectiveScope{
					Type:     EffectiveScopeEnvType,
					EnvTypes: []string{"staging"},
				},
			}
			envs, err := svc.resolveEffectiveEnvs(ctx, rule)
			Expect(err).NotTo(HaveOccurred())
			Expect(envs).To(BeEmpty())
		})
	})

	Describe("SyncToRemote", func() {
		var ws *workspace.Workspace

		BeforeEach(func() {
			ws = &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"

			envDev := &envmodel.Environment{
				Name: "dev", Type: "development", WorkspaceID: "test-ws", AppIDs: []string{"app-1"},
				Cluster: envmodel.BizCluster{
					ClusterID: "BCS-K8S-00001",
					Namespace: "ns-dev",
				},
			}
			_, err := envStore.Create(ctx, envDev)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should sync rule to remote and update refs", func() {
			createApp("test-ws", "app-1", "order-svc")
			req := newCreateReq()
			req.AppName = "order-svc"
			req.StrategyCode = "sync_test"
			req.DisplayName = "Sync Test"
			req.Severity = AlertSeverityWarning
			req.Threshold = ThresholdConfig{Method: "gte", Value: 80}
			req.EffectiveTimeRange = EffectiveTimeRange{StartTime: "09:00:00", EndTime: "18:00:00"}
			req.NoticeGroupIDs = []int64{1001}
			req.Enabled = true

			rule, err := svc.Create(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			err = svc.SyncToRemote(ctx, ws, rule.ID, "test-user")
			Expect(err).NotTo(HaveOccurred())

			updated, err := store.Get(ctx, rule.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.RemoteRefs).To(HaveLen(1))
			Expect(updated.RemoteRefs[0].EnvName).To(Equal("dev"))
			Expect(updated.RemoteRefs[0].RemoteStrategyID).To(BeNumerically(">", 0))
			Expect(updated.RemoteRefs[0].RemoteStrategyName).To(ContainSubstring("order-svc"))
		})

		It("should mark failed when client returns error", func() {
			failSvc := newServiceWithClientFactory(
				svc.store,
				svc.envStore,
				svc.appStore,
				svc.snapshotStore,
				func(_ string) (bkmapi.MonitorClient, error) {
					return nil, bkmapi.ErrApmAppNotFound
				},
			)

			req := newCreateReq()
			req.AppName = "order-svc"
			req.StrategyCode = "fail_test"
			req.DisplayName = "Fail Test"
			req.Enabled = true

			rule, err := failSvc.Create(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			err = failSvc.SyncToRemote(ctx, ws, rule.ID, "test-user")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("InitDefaultAlertStrategiesForApp", func() {
		It("should create default rules enabled with scope=all for app (local only, no remote sync)", func() {
			err := svc.InitDefaultAlertStrategiesForApp(
				ctx, "default-ws", "app-1", "demo-app", "test-user", []int64{1001},
			)
			Expect(err).NotTo(HaveOccurred())

			rules, err := store.ListByApp(ctx, "default-ws", "app-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(HaveLen(len(defaultTemplates)))

			for _, r := range rules {
				Expect(r.AppID).To(Equal("app-1"))
				Expect(r.AppName).To(Equal("demo-app"))
				Expect(r.Enabled).To(BeTrue())
				Expect(r.EffectiveScope.Type).To(Equal(EffectiveScopeAll))
				Expect(r.EffectiveTimeRange.StartTime).To(Equal(defaultEffectiveStartTime))
				Expect(r.EffectiveTimeRange.EndTime).To(Equal(defaultEffectiveEndTime))
				Expect(r.NoticeGroupIDs).To(Equal([]int64{1001}))
				Expect(r.Creator).To(Equal("test-user"))
				Expect(r.RemoteRefs).To(BeEmpty())
			}
		})
	})

	Describe("SwitchEnabled", func() {
		It("should toggle enabled state", func() {
			mockey.PatchConvey("switch enabled", GinkgoT(), func() {
				mockClient := bkmapi.NewStub("test-user")
				mockey.Mock(bkmapi.NewMonitorClient).Return(mockClient, nil).Build()
				switchSvc := newServiceWithClientFactory(
					svc.store,
					svc.envStore,
					svc.appStore,
					svc.snapshotStore,
					bkmapi.NewMonitorClient,
				)

				ws := &workspace.Workspace{ID: "test-ws"}
				ws.BkSystems.BkMonitorProjectID = "-100"

				req := newCreateReq()
				req.StrategyCode = "switch_test"
				req.DisplayName = "Switch Test"
				req.Enabled = false

				rule, err := switchSvc.Create(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				err = switchSvc.SwitchEnabled(ctx, ws, rule.ID, true, "test-user")
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, rule.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Enabled).To(BeTrue())
			})
		})

		It("should auto sync to remote on first enable when remoteRefs is empty", func() {
			mockey.PatchConvey("first enable auto sync", GinkgoT(), func() {
				mockClient := bkmapi.NewStub("test-user")
				mockey.Mock(bkmapi.NewMonitorClient).Return(mockClient, nil).Build()
				switchSvc := newServiceWithClientFactory(
					svc.store,
					svc.envStore,
					svc.appStore,
					svc.snapshotStore,
					bkmapi.NewMonitorClient,
				)

				ws := &workspace.Workspace{ID: "test-ws"}
				ws.BkSystems.BkMonitorProjectID = "-100"
				createApp("test-ws", "app-1", "demo-app")

				_, envErr := envStore.Create(ctx, &envmodel.Environment{
					Name:        "switch-env",
					Type:        "production",
					WorkspaceID: "test-ws",
					Kind:        envmodel.EnvironmentKindStandard,
					AppIDs:      []string{"app-1"},
					Cluster: envmodel.BizCluster{
						ClusterID: "BCS-K8S-00001",
						Namespace: "ns-switch",
					},
				})
				Expect(envErr).NotTo(HaveOccurred())

				req := newCreateReq()
				req.StrategyCode = "cpu_limit_usage_high"
				req.DisplayName = "CPU Limit 使用率过高"
				req.Severity = AlertSeverityFatal
				req.Threshold = ThresholdConfig{Method: "gte", Value: 90}
				req.Enabled = false

				rule, err := switchSvc.Create(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(rule.RemoteRefs).To(BeEmpty())

				err = switchSvc.SwitchEnabled(ctx, ws, rule.ID, true, "test-user")
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, rule.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Enabled).To(BeTrue())
				Expect(updated.RemoteRefs).NotTo(BeEmpty())
			})
		})

		It("should keep local state unchanged when remote switch fails", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"
			failSvc := newServiceWithClientFactory(
				svc.store,
				svc.envStore,
				svc.appStore,
				svc.snapshotStore,
				func(_ string) (bkmapi.MonitorClient, error) {
					return &failAlertStrategyClient{
						StubClient: bkmapi.NewStub("test-user"),
						switchErr:  bkmapi.ErrApmAppNotFound,
					}, nil
				},
			)

			ruleID := bson.NewObjectID()
			_, err := store.Create(ctx, &AlertStrategy{
				ID:           ruleID,
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "switch_fail_test",
				DisplayName:  "Switch Fail Test",
				Threshold:    ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				Enabled: false,
				RemoteRefs: []RemoteStrategyRef{{
					EnvID:              bson.NewObjectID(),
					EnvName:            "prod",
					RemoteStrategyName: "demo",
					RemoteStrategyID:   1001,
				}},
				Creator: "tester",
				Updater: "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = failSvc.SwitchEnabled(ctx, ws, ruleID, true, "tester")

			Expect(err).To(HaveOccurred())
			stored, getErr := store.Get(ctx, ruleID)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.Enabled).To(BeFalse())
		})
	})

	Describe("UpdateAndSync", func() {
		It("should keep local fields unchanged when remote sync fails", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"
			createApp("test-ws", "app-1", "demo-app")

			_, err := envStore.Create(ctx, &envmodel.Environment{
				Name:        "update-env",
				Type:        "production",
				WorkspaceID: "test-ws",
				Kind:        envmodel.EnvironmentKindStandard,
				AppIDs:      []string{"app-1"},
				Cluster: envmodel.BizCluster{
					ClusterID: "BCS-K8S-00001",
					Namespace: "ns-update",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			failSvc := newServiceWithClientFactory(
				svc.store,
				svc.envStore,
				svc.appStore,
				svc.snapshotStore,
				func(_ string) (bkmapi.MonitorClient, error) {
					return &failAlertStrategyClient{
						StubClient: bkmapi.NewStub("test-user"),
						saveErr:    bkmapi.ErrApmAppNotFound,
					}, nil
				},
			)

			req := newCreateReq()
			req.StrategyCode = "update_sync_fail_test"
			req.DisplayName = "Before Update"
			req.Enabled = true
			rule, err := svc.Create(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			newName := "After Update"
			changed, err := failSvc.UpdateAndSync(ctx, ws, rule.ID, &UpdateReq{
				DisplayName: &newName,
				Operator:    "tester",
			})

			Expect(err).To(HaveOccurred())
			Expect(changed).To(BeFalse())
			stored, getErr := store.Get(ctx, rule.ID)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.DisplayName).To(Equal("Before Update"))
		})
	})

	Describe("Delete", func() {
		It("should delete rule and cleanup remote", func() {
			mockey.PatchConvey("delete with cleanup", GinkgoT(), func() {
				mockClient := bkmapi.NewStub("test-user")
				mockey.Mock(bkmapi.NewMonitorClient).Return(mockClient, nil).Build()
				deleteSvc := newServiceWithClientFactory(
					svc.store,
					svc.envStore,
					svc.appStore,
					svc.snapshotStore,
					bkmapi.NewMonitorClient,
				)

				ws := &workspace.Workspace{ID: "test-ws"}
				ws.BkSystems.BkMonitorProjectID = "-100"

				req := newCreateReq()
				req.StrategyCode = "delete_test"
				req.DisplayName = "Delete Test"

				rule, err := deleteSvc.Create(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				err = deleteSvc.Delete(ctx, ws, rule.ID, "test-user")
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Get(ctx, rule.ID)
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(ErrNotFound))
			})
		})

		It("should keep local strategy when remote cleanup fails", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"
			failSvc := newServiceWithClientFactory(
				svc.store,
				svc.envStore,
				svc.appStore,
				svc.snapshotStore,
				func(_ string) (bkmapi.MonitorClient, error) {
					return &failAlertStrategyClient{
						StubClient: bkmapi.NewStub("test-user"),
						deleteErr:  bkmapi.ErrApmAppNotFound,
					}, nil
				},
			)

			ruleID := bson.NewObjectID()
			_, err := store.Create(ctx, &AlertStrategy{
				ID:           ruleID,
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "delete_fail_test",
				DisplayName:  "Delete Fail Test",
				Threshold:    ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				RemoteRefs: []RemoteStrategyRef{{
					EnvID:              bson.NewObjectID(),
					EnvName:            "prod",
					RemoteStrategyName: "demo",
					RemoteStrategyID:   1001,
				}},
				Creator: "tester",
				Updater: "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = failSvc.Delete(ctx, ws, ruleID, "tester")

			Expect(err).To(HaveOccurred())
			stored, getErr := store.Get(ctx, ruleID)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(stored.DisplayName).To(Equal("Delete Fail Test"))
		})
	})

	Describe("DeleteByApp", func() {
		It("should delete all strategies for the app", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"

			firstID, err := store.Create(ctx, &AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "delete_app_first",
				DisplayName:  "Delete App First",
				Threshold:    ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				RemoteRefs: []RemoteStrategyRef{{
					EnvID:              bson.NewObjectID(),
					EnvName:            "prod",
					RemoteStrategyName: "demo-first",
					RemoteStrategyID:   1001,
				}},
				Creator: "tester",
				Updater: "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			secondID, err := store.Create(ctx, &AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "delete_app_second",
				DisplayName:  "Delete App Second",
				Threshold:    ThresholdConfig{Method: "gte", Value: 90},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				RemoteRefs: []RemoteStrategyRef{{
					EnvID:              bson.NewObjectID(),
					EnvName:            "stag",
					RemoteStrategyName: "demo-second",
					RemoteStrategyID:   1002,
				}},
				Creator: "tester",
				Updater: "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			err = svc.DeleteByApp(ctx, ws, "test-ws", "app-1", "tester")

			Expect(err).NotTo(HaveOccurred())
			_, err = store.Get(ctx, firstID)
			Expect(err).To(Equal(ErrNotFound))
			_, err = store.Get(ctx, secondID)
			Expect(err).To(Equal(ErrNotFound))
		})

		It("should continue deleting later strategies when one delete fails", func() {
			ws := &workspace.Workspace{ID: "test-ws"}
			ws.BkSystems.BkMonitorProjectID = "-100"
			failClient := &failOnNthDeleteAlertStrategyClient{
				StubClient: bkmapi.NewStub("test-user"),
				failAtCall: 1,
				deleteErr:  bkmapi.ErrApmAppNotFound,
			}
			failSvc := newServiceWithClientFactory(
				svc.store,
				svc.envStore,
				svc.appStore,
				svc.snapshotStore,
				func(_ string) (bkmapi.MonitorClient, error) {
					return failClient, nil
				},
			)

			_, err := store.Create(ctx, &AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "delete_app_keep",
				DisplayName:  "Delete App Keep",
				Threshold:    ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				RemoteRefs: []RemoteStrategyRef{{
					EnvID:              bson.NewObjectID(),
					EnvName:            "prod",
					RemoteStrategyName: "demo-keep",
					RemoteStrategyID:   1101,
				}},
				Creator: "tester",
				Updater: "tester",
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Create(ctx, &AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "delete_app_remove",
				DisplayName:  "Delete App Remove",
				Threshold:    ThresholdConfig{Method: "gte", Value: 90},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				RemoteRefs: []RemoteStrategyRef{{
					EnvID:              bson.NewObjectID(),
					EnvName:            "stag",
					RemoteStrategyName: "demo-remove",
					RemoteStrategyID:   1102,
				}},
				Creator: "tester",
				Updater: "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			ordered, err := store.ListByApp(ctx, "test-ws", "app-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(ordered).To(HaveLen(2))

			err = failSvc.DeleteByApp(ctx, ws, "test-ws", "app-1", "tester")

			Expect(err).To(HaveOccurred())
			Expect(failClient.deleteReqs).To(HaveLen(2))
			_, firstErr := store.Get(ctx, ordered[0].ID)
			Expect(firstErr).NotTo(HaveOccurred())
			_, secondErr := store.Get(ctx, ordered[1].ID)
			Expect(secondErr).To(Equal(ErrNotFound))
		})
	})
})
