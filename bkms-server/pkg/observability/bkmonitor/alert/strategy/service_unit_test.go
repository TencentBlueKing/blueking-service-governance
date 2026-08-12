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

type captureAlertStrategyClient struct {
	*bkmapi.StubClient
	saveReqs   []*bkmapi.SaveAlarmStrategyReq
	deleteReqs []*bkmapi.DeleteAlarmStrategyReq
}

func (c *captureAlertStrategyClient) SaveAlarmStrategy(
	_ context.Context,
	req *bkmapi.SaveAlarmStrategyReq,
) (*bkmapi.SaveAlarmStrategyResp, error) {
	c.saveReqs = append(c.saveReqs, req)
	if req.ID > 0 {
		return &bkmapi.SaveAlarmStrategyResp{ID: req.ID}, nil
	}
	return &bkmapi.SaveAlarmStrategyResp{ID: int64(len(c.saveReqs) + 1000)}, nil
}

func (c *captureAlertStrategyClient) DeleteAlarmStrategy(
	_ context.Context,
	req *bkmapi.DeleteAlarmStrategyReq,
) error {
	c.deleteReqs = append(c.deleteReqs, req)
	return nil
}

type recreateOnMissingRemoteClient struct {
	*bkmapi.StubClient
	saveReqs []*bkmapi.SaveAlarmStrategyReq
}

func (c *recreateOnMissingRemoteClient) SaveAlarmStrategy(
	_ context.Context,
	req *bkmapi.SaveAlarmStrategyReq,
) (*bkmapi.SaveAlarmStrategyResp, error) {
	c.saveReqs = append(c.saveReqs, req)
	if len(c.saveReqs) == 1 && req.ID > 0 {
		return nil, errors.New("api error, code: 3313003, message: 策略配置不存在, request_id: ")
	}
	if req.ID > 0 {
		return &bkmapi.SaveAlarmStrategyResp{ID: req.ID}, nil
	}
	return &bkmapi.SaveAlarmStrategyResp{ID: 2001}, nil
}

var _ = Describe("AlertStrategyServiceRealDB", func() {
	var (
		ctx           context.Context
		diApp         *fxtest.App
		store         Store
		envStore      envmodel.EnvironmentStore
		appStore      bkmsapp.ApplicationStore
		snapshotStore topology.ResourceSnapshotStore
		svc           *Service
		client        *captureAlertStrategyClient
	)

	buildService := func() *Service {
		return newServiceWithClientFactory(
			store,
			envStore,
			appStore,
			snapshotStore,
			func(_ string) (bkmapi.MonitorClient, error) {
				return client, nil
			},
		)
	}

	buildWorkspace := func(workspaceID string) *workspace.Workspace {
		ws := &workspace.Workspace{ID: workspaceID}
		ws.BkSystems.BkMonitorProjectID = "-100"
		return ws
	}

	createEnv := func(
		workspaceID, name, envType string,
		appIDs []string,
		cluster envmodel.BizCluster,
	) *envmodel.Environment {
		env := &envmodel.Environment{
			Name:        name,
			DisplayName: name,
			Type:        envType,
			WorkspaceID: workspaceID,
			Kind:        envmodel.EnvironmentKindStandard,
			AppIDs:      appIDs,
			Cluster:     cluster,
		}
		envID, err := envStore.Create(ctx, env)
		Expect(err).NotTo(HaveOccurred())
		created, err := envStore.Get(ctx, envID)
		Expect(err).NotTo(HaveOccurred())
		return created
	}

	createStrategy := func(strategy *AlertStrategy) bson.ObjectID {
		if strategy.DisplayName == "" {
			strategy.DisplayName = strategy.StrategyCode
		}
		if strategy.Threshold.Method == "" {
			strategy.Threshold = ThresholdConfig{Method: "gte", Value: 80}
		}
		if strategy.EffectiveScope.Type == "" {
			strategy.EffectiveScope = EffectiveScope{Type: EffectiveScopeAll}
		}
		id, err := store.Create(ctx, strategy)
		Expect(err).NotTo(HaveOccurred())
		return id
	}

	createApp := func(workspaceID, appID, appName string) *bkmsapp.Application {
		return dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
			WorkspaceID: workspaceID,
			ID:          appID,
			Name:        appName,
		})
	}

	BeforeEach(func() {
		ctx = context.Background()
		client = &captureAlertStrategyClient{StubClient: bkmapi.NewStub("test-user")}

		Expect(testutil.CleanupCollection(collectionName)).To(Succeed())
		Expect(testutil.CleanupCollection("environments")).To(Succeed())
		Expect(testutil.CleanupCollection("applications")).To(Succeed())
		Expect(testutil.CleanupCollection(topology.ResourceSnapshotsCollection)).To(Succeed())

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
			fx.Populate(&store, &envStore, &appStore, &snapshotStore),
		)
		diApp.RequireStart()
		svc = buildService()
	})

	AfterEach(func() {
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	Describe("SyncToRemote", func() {
		It("builds environment scoped per pod cpu limit strategy", func() {
			createApp("test-ws", "app-1", "order-svc")
			_ = createEnv(
				"test-ws",
				"prod",
				"production",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-00001", Namespace: "bkmonitor-operator"},
			)

			rule, err := svc.Create(ctx, &CreateReq{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "order-svc",
				StrategyCode: "cpu_limit_usage_high",
				DisplayName:  "CPU Limit 使用率过高",
				Severity:     AlertSeverityFatal,
				Threshold:    ThresholdConfig{Method: "gte", Value: 90},
				EffectiveScope: EffectiveScope{
					Type:     EffectiveScopeEnvType,
					EnvTypes: []string{"production"},
				},
				EffectiveTimeRange: EffectiveTimeRange{StartTime: "09:00:00", EndTime: "18:00:00"},
				NoticeGroupIDs:     []int64{1001},
				Enabled:            true,
				Operator:           "test-user",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(svc.SyncToRemote(ctx, buildWorkspace("test-ws"), rule.ID, "test-user")).To(Succeed())
			Expect(client.saveReqs).To(HaveLen(1))

			queryConfigs, ok := client.saveReqs[0].Items[0]["query_configs"].([]map[string]any)
			Expect(ok).To(BeTrue())
			Expect(queryConfigs).To(HaveLen(1))

			promql, ok := queryConfigs[0]["promql"].(string)
			Expect(ok).To(BeTrue())
			Expect(promql).To(ContainSubstring(`bcs_cluster_id="BCS-K8S-00001"`))
			Expect(promql).To(ContainSubstring(`namespace="bkmonitor-operator"`))
			Expect(
				promql,
			).To(ContainSubstring(`container_cpu_usage_seconds_total{bcs_cluster_id="BCS-K8S-00001",namespace="bkmonitor-operator",container_name!="POD",workload_name="order-svc"}`))
			Expect(
				promql,
			).To(ContainSubstring(`kube_pod_container_resource_limits_cpu_cores{bcs_cluster_id="BCS-K8S-00001",namespace="bkmonitor-operator",container_name!="POD"}`))
			Expect(promql).To(ContainSubstring(`kube_pod_container_resource_limits_cpu_cores`))
			Expect(promql).To(ContainSubstring(`sum by(pod_name, bcs_cluster_id, namespace)`))

			detectExpr, ok := client.saveReqs[0].Detects[0]["expression"].(string)
			Expect(ok).To(BeTrue())
			Expect(detectExpr).To(Equal(""))
			noticeOptions, ok := client.saveReqs[0].Notice["options"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(noticeOptions["start_time"]).To(Equal("09:00:00"))
			Expect(noticeOptions["end_time"]).To(Equal("18:00:00"))
			noticeConfig, ok := client.saveReqs[0].Notice["config"].(map[string]any)
			Expect(ok).To(BeTrue())
			templates, ok := noticeConfig["template"].([]map[string]any)
			Expect(ok).To(BeTrue())
			Expect(templates).To(ContainElement(map[string]any{
				"signal":       "abnormal",
				"message_tmpl": "",
				"title_tmpl":   "【告警触发】CPU Limit 使用率过高",
			}))
			Expect(templates).To(ContainElement(map[string]any{
				"signal":       "recovered",
				"message_tmpl": "",
				"title_tmpl":   "【告警恢复】CPU Limit 使用率过高",
			}))
		})

		It("builds memory limit promql without raw wrapper", func() {
			createApp("test-ws", "app-1", "trpc-test-app")
			_ = createEnv(
				"test-ws",
				"stage",
				"staging",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-100018", Namespace: "ieg-bkms-pd-stage"},
			)

			rule, err := svc.Create(ctx, &CreateReq{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "trpc-test-app",
				StrategyCode: "memory_limit_usage_high",
				DisplayName:  "Memory Limit 使用率过高",
				Severity:     AlertSeverityWarning,
				Threshold:    ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type:     EffectiveScopeEnvType,
					EnvTypes: []string{"staging"},
				},
				Enabled:  true,
				Operator: "test-user",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(svc.SyncToRemote(ctx, buildWorkspace("test-ws"), rule.ID, "test-user")).To(Succeed())
			Expect(client.saveReqs).To(HaveLen(1))

			queryConfigs, ok := client.saveReqs[0].Items[0]["query_configs"].([]map[string]any)
			Expect(ok).To(BeTrue())
			Expect(queryConfigs).To(HaveLen(1))

			promql, ok := queryConfigs[0]["promql"].(string)
			Expect(ok).To(BeTrue())
			Expect(promql).To(ContainSubstring(`container_memory_working_set_bytes{bcs_cluster_id="BCS-K8S-100018"`))
			Expect(promql).To(ContainSubstring(`workload_name="trpc-test-app"`))
			Expect(promql).To(ContainSubstring(`kube_pod_container_resource_limits_memory_bytes`))
			Expect(
				promql,
			).NotTo(ContainSubstring(`kube_pod_container_resource_limits_memory_bytes{bcs_cluster_id="BCS-K8S-100018",namespace="ieg-bkms-pd-stage",container_name!="POD",workload_name="trpc-test-app"}`))
			Expect(promql).NotTo(ContainSubstring(`raw(`))
			Expect(promql).NotTo(ContainSubstring(`rate(container_memory_working_set_bytes`))
		})

		It("builds pod restart promql with pod name prefix matcher", func() {
			createApp("test-ws", "app-1", "trpc-test-app")
			_ = createEnv(
				"test-ws",
				"stage",
				"staging",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-100018", Namespace: "ieg-bkms-pd-stage"},
			)

			rule, err := svc.Create(ctx, &CreateReq{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "trpc-test-app",
				StrategyCode: "pod_restart_frequent",
				DisplayName:  "Pod 重启频繁",
				Severity:     AlertSeverityWarning,
				Threshold:    ThresholdConfig{Method: "gte", Value: 3},
				EffectiveScope: EffectiveScope{
					Type:     EffectiveScopeEnvType,
					EnvTypes: []string{"staging"},
				},
				TriggerCondition: TriggerCondition{Count: 1, CheckWindow: 5},
				Enabled:          true,
				Operator:         "test-user",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(svc.SyncToRemote(ctx, buildWorkspace("test-ws"), rule.ID, "test-user")).To(Succeed())
			Expect(client.saveReqs).To(HaveLen(1))

			queryConfigs, ok := client.saveReqs[0].Items[0]["query_configs"].([]map[string]any)
			Expect(ok).To(BeTrue())
			Expect(queryConfigs).To(HaveLen(1))

			promql, ok := queryConfigs[0]["promql"].(string)
			Expect(ok).To(BeTrue())
			Expect(
				promql,
			).To(ContainSubstring(`increase(kube_pod_container_status_restarts_total{bcs_cluster_id="BCS-K8S-100018",namespace="ieg-bkms-pd-stage",pod_name=~"^trpc-test-app(-.*)?$"}[5m])`))
			Expect(promql).NotTo(ContainSubstring(`workload_name="trpc-test-app"`))
		})

		It("deletes stale remote refs when scope shrinks", func() {
			createApp("test-ws", "app-1", "demo-app")
			targetEnv := createEnv(
				"test-ws",
				"prod",
				"production",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-00001", Namespace: "ns-prod"},
			)
			staleEnvID := bson.NewObjectID()

			strategyID := createStrategy(&AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "cpu_high",
				DisplayName:  "CPU High",
				EffectiveScope: EffectiveScope{
					Type:   EffectiveScopeSpecificEnvs,
					EnvIDs: []bson.ObjectID{targetEnv.ID},
				},
				RemoteRefs: []RemoteStrategyRef{
					{EnvID: targetEnv.ID, EnvName: "prod", RemoteStrategyID: 101},
					{EnvID: staleEnvID, EnvName: "stale", RemoteStrategyID: 202},
				},
			})

			Expect(svc.SyncToRemote(ctx, buildWorkspace("test-ws"), strategyID, "test-user")).To(Succeed())
			Expect(client.deleteReqs).To(HaveLen(1))
			Expect(client.deleteReqs[0].IDs).To(Equal([]int64{202}))

			updated, err := store.Get(ctx, strategyID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.RemoteRefs).To(HaveLen(1))
			Expect(updated.RemoteRefs[0].EnvID).To(Equal(targetEnv.ID))
		})

		It("deletes existing ref when target env is temporarily invalid during full sync", func() {
			createApp("test-ws", "app-1", "demo-app")
			targetEnv := createEnv(
				"test-ws",
				"prod",
				"production",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-00001"},
			)

			strategyID := createStrategy(&AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "cpu_high",
				DisplayName:  "CPU High",
				EffectiveScope: EffectiveScope{
					Type:   EffectiveScopeSpecificEnvs,
					EnvIDs: []bson.ObjectID{targetEnv.ID},
				},
				RemoteRefs: []RemoteStrategyRef{
					{EnvID: targetEnv.ID, EnvName: "prod", RemoteStrategyID: 101},
				},
			})

			err := svc.SyncToRemote(ctx, buildWorkspace("test-ws"), strategyID, "test-user")
			Expect(err).NotTo(HaveOccurred())
			Expect(client.saveReqs).To(BeEmpty())
			Expect(client.deleteReqs).To(HaveLen(1))
			Expect(client.deleteReqs[0].IDs).To(Equal([]int64{101}))

			updated, getErr := store.Get(ctx, strategyID)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(updated.RemoteRefs).To(BeEmpty())
		})

		It("recreates remote strategy once when saved remote ref points to missing remote config", func() {
			createApp("test-ws", "app-1", "trpc-test-app")
			targetEnv := createEnv(
				"test-ws",
				"stage",
				"staging",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-100018", Namespace: "ieg-bkms-pd-stage"},
			)
			recoverClient := &recreateOnMissingRemoteClient{StubClient: bkmapi.NewStub("test-user")}
			recoverSvc := newServiceWithClientFactory(
				store,
				envStore,
				appStore,
				snapshotStore,
				func(_ string) (bkmapi.MonitorClient, error) {
					return recoverClient, nil
				},
			)
			strategyID := createStrategy(&AlertStrategy{
				WorkspaceID:   "test-ws",
				AppID:         "app-1",
				AppName:       "trpc-test-app",
				StrategyCode:  "memory_limit_usage_high",
				DisplayName:   "Memory Limit 使用率过高",
				MonitorMetric: "container_memory_working_set_bytes",
				Severity:      AlertSeverityWarning,
				Threshold:     ThresholdConfig{Method: "gte", Value: 80},
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeSpecificEnvs,
					EnvIDs: []bson.ObjectID{
						targetEnv.ID,
					},
				},
				Enabled: true,
				RemoteRefs: []RemoteStrategyRef{{
					EnvID:            targetEnv.ID,
					EnvName:          targetEnv.Name,
					RemoteStrategyID: 101,
				}},
			})

			err := recoverSvc.SyncToRemote(ctx, buildWorkspace("test-ws"), strategyID, "test-user")
			Expect(err).NotTo(HaveOccurred())
			Expect(recoverClient.saveReqs).To(HaveLen(2))
			Expect(recoverClient.saveReqs[0].ID).To(Equal(int64(101)))
			Expect(recoverClient.saveReqs[1].ID).To(BeZero())

			updated, getErr := store.Get(ctx, strategyID)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(updated.RemoteRefs).To(HaveLen(1))
			Expect(updated.RemoteRefs[0].RemoteStrategyID).To(Equal(int64(2001)))
			Expect(updated.RemoteRefs[0].EnvID).To(Equal(targetEnv.ID))
		})
	})

	Describe("resolveStrategyWorkloads", func() {
		It("prefers helm topology snapshot", func() {
			app := dbfactory.HelmApplication(ctx, &dbfactory.HelmApplicationStores{
				AppStore: appStore,
			}, &dbfactory.HelmApplicationOpts{WorkspaceID: "test-ws"})

			err := snapshotStore.UpsertWithVersion(ctx, &topology.ResourceSnapshot{
				AppID:           app.ID,
				EnvName:         "prod",
				TrafficLaneName: "",
				DataVersion:     1,
				RefreshStatus:   topology.RefreshStatusSuccess,
				Resources: []topology.ResourceEntry{
					{Kind: "Deployment", Name: "demo-web", IsManaged: true},
					{Kind: "StatefulSet", Name: "demo-worker", IsManaged: true},
					{Kind: "Deployment", Name: "demo-web", IsManaged: true},
					{Kind: "ConfigMap", Name: "demo-config", IsManaged: true},
					{Kind: "Deployment", Name: "foreign", IsManaged: false},
				},
			}, 0)
			Expect(err).NotTo(HaveOccurred())

			workloads, err := svc.resolveStrategyWorkloads(ctx, &AlertStrategy{
				AppID:   app.ID,
				AppName: app.Name,
			}, envmodel.Environment{Name: "prod"}, "")
			Expect(err).NotTo(HaveOccurred())

			Expect(workloads).To(Equal([]string{"demo-web", "demo-worker"}))
		})
	})

	Describe("buildRemoteStrategyName", func() {
		It("uses display name with app name suffix", func() {
			strategyID, err := bson.ObjectIDFromHex("64b64c8e5f627c5ef8a1bcde")
			Expect(err).NotTo(HaveOccurred())
			name := buildRemoteStrategyName(&AlertStrategy{
				ID:           strategyID,
				AppName:      "Order_Svc",
				DisplayName:  "CPU Limit",
				StrategyCode: "cpu_limit_usage_high",
			})

			Expect(name).To(Equal("CPU Limit【Order_Svc】"))
		})
	})

	Describe("CleanupStrategiesForAppInEnv", func() {
		It("cleans only refs for the target env lane", func() {
			createApp("test-ws", "app-1", "demo-app")
			env := createEnv(
				"test-ws",
				"prod",
				"production",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-00001", Namespace: "ns-prod"},
			)
			otherEnv := createEnv(
				"test-ws",
				"stage",
				"staging",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-00002", Namespace: "ns-stage"},
			)
			strategyID := createStrategy(&AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "cpu_high",
				DisplayName:  "CPU High",
				RemoteRefs: []RemoteStrategyRef{
					{EnvID: env.ID, EnvName: env.Name, TrafficLaneName: "", RemoteStrategyID: 101},
					{EnvID: env.ID, EnvName: env.Name, TrafficLaneName: "feature-a", RemoteStrategyID: 102},
					{EnvID: otherEnv.ID, EnvName: otherEnv.Name, TrafficLaneName: "", RemoteStrategyID: 103},
				},
			})

			svc.CleanupStrategiesForAppInEnv(ctx, buildWorkspace("test-ws"), "app-1", env.ID, "feature-a", "tester")

			Expect(client.deleteReqs).To(HaveLen(1))
			Expect(client.deleteReqs[0].IDs).To(Equal([]int64{102, 103}))

			updated, err := store.Get(ctx, strategyID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.RemoteRefs).To(HaveLen(2))
			Expect(updated.RemoteRefs).To(ContainElements(
				RemoteStrategyRef{
					EnvID:              env.ID,
					EnvName:            env.Name,
					TrafficLaneName:    "",
					RemoteStrategyID:   101,
					RemoteStrategyName: "CPU High【demo-app】",
				},
				RemoteStrategyRef{
					EnvID:              otherEnv.ID,
					EnvName:            otherEnv.Name,
					TrafficLaneName:    "",
					RemoteStrategyID:   101,
					RemoteStrategyName: "CPU High【demo-app】",
				},
			))
		})
	})

	Describe("SyncStrategiesForAppInEnv", func() {
		It("syncs only enabled strategies matching the target env scope", func() {
			createApp("test-ws", "app-1", "demo-app")
			env := createEnv(
				"test-ws",
				"prod",
				"production",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-00001", Namespace: "ns-prod"},
			)
			otherEnvID := bson.NewObjectID()

			createStrategy(&AlertStrategy{
				WorkspaceID: "test-ws", AppID: "app-1", AppName: "demo-app", StrategyCode: "scope_all", Enabled: true,
				EffectiveScope: EffectiveScope{Type: EffectiveScopeAll},
			})
			createStrategy(&AlertStrategy{
				WorkspaceID: "test-ws", AppID: "app-1", AppName: "demo-app", StrategyCode: "scope_env_type", Enabled: true,
				EffectiveScope: EffectiveScope{Type: EffectiveScopeEnvType, EnvTypes: []string{"production"}},
			})
			createStrategy(&AlertStrategy{
				WorkspaceID: "test-ws", AppID: "app-1", AppName: "demo-app", StrategyCode: "scope_specific_env", Enabled: true,
				EffectiveScope: EffectiveScope{Type: EffectiveScopeSpecificEnvs, EnvIDs: []bson.ObjectID{env.ID}},
			})
			createStrategy(&AlertStrategy{
				WorkspaceID: "test-ws", AppID: "app-1", AppName: "demo-app", StrategyCode: "disabled_all", Enabled: false,
				EffectiveScope: EffectiveScope{Type: EffectiveScopeAll},
			})
			createStrategy(&AlertStrategy{
				WorkspaceID: "test-ws", AppID: "app-1", AppName: "demo-app", StrategyCode: "other_env_type", Enabled: true,
				EffectiveScope: EffectiveScope{Type: EffectiveScopeEnvType, EnvTypes: []string{"development"}},
			})
			createStrategy(&AlertStrategy{
				WorkspaceID: "test-ws", AppID: "app-1", AppName: "demo-app", StrategyCode: "other_specific_env", Enabled: true,
				EffectiveScope: EffectiveScope{Type: EffectiveScopeSpecificEnvs, EnvIDs: []bson.ObjectID{otherEnvID}},
			})

			svc.SyncStrategiesForAppInEnv(ctx, buildWorkspace("test-ws"), "app-1", env.ID, "", "tester")

			Expect(client.saveReqs).To(HaveLen(3))
		})

		It("keeps baseline and feature lane refs independent", func() {
			createApp("test-ws", "app-1", "demo-app")
			env := createEnv(
				"test-ws",
				"prod",
				"production",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-00001", Namespace: "ns-prod"},
			)
			strategyID := createStrategy(&AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "scope_all",
				Enabled:      true,
				EffectiveScope: EffectiveScope{
					Type: EffectiveScopeAll,
				},
				RemoteRefs: []RemoteStrategyRef{
					{EnvID: env.ID, EnvName: env.Name, TrafficLaneName: "", RemoteStrategyID: 101},
				},
			})

			svc.SyncStrategiesForAppInEnv(ctx, buildWorkspace("test-ws"), "app-1", env.ID, "feature-a", "tester")

			updated, err := store.Get(ctx, strategyID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.RemoteRefs).To(HaveLen(2))
			Expect(updated.RemoteRefs).To(ContainElements(
				RemoteStrategyRef{
					EnvID:              env.ID,
					EnvName:            env.Name,
					TrafficLaneName:    "",
					RemoteStrategyName: "scope_all【demo-app】",
					RemoteStrategyID:   101,
				},
				RemoteStrategyRef{
					EnvID:              env.ID,
					EnvName:            env.Name,
					TrafficLaneName:    "feature-a",
					RemoteStrategyName: "scope_all【demo-app】",
					RemoteStrategyID:   101,
				},
			))
		})
	})

	Describe("ReconcileStrategiesForEnvTypeChange", func() {
		It("removes stale old-type refs and syncs newly matched strategies", func() {
			createApp("test-ws", "app-1", "demo-app")
			env := createEnv(
				"test-ws",
				"shared",
				"test",
				[]string{"app-1"},
				envmodel.BizCluster{ClusterID: "BCS-K8S-00001", Namespace: "ns-shared"},
			)
			before := *env

			oldTypeStrategyID := createStrategy(&AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "scope_old_type",
				DisplayName:  "Old Type Strategy",
				Enabled:      true,
				EffectiveScope: EffectiveScope{
					Type:     EffectiveScopeEnvType,
					EnvTypes: []string{"test"},
				},
				RemoteRefs: []RemoteStrategyRef{
					{EnvID: env.ID, EnvName: env.Name, RemoteStrategyID: 101},
				},
			})
			newTypeStrategyID := createStrategy(&AlertStrategy{
				WorkspaceID:  "test-ws",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "scope_new_type",
				DisplayName:  "New Type Strategy",
				Enabled:      true,
				EffectiveScope: EffectiveScope{
					Type:     EffectiveScopeEnvType,
					EnvTypes: []string{"staging"},
				},
			})

			newType := "staging"
			Expect(envStore.Update(ctx, env.ID, &envmodel.EnvironmentUpdateData{Type: &newType})).To(Succeed())
			after, err := envStore.Get(ctx, env.ID)
			Expect(err).NotTo(HaveOccurred())

			Expect(
				svc.ReconcileStrategiesForEnvTypeChange(
					ctx,
					buildWorkspace("test-ws"),
					before,
					*after,
					"tester",
				),
			).To(Succeed())

			Expect(client.deleteReqs).To(HaveLen(1))
			Expect(client.deleteReqs[0].IDs).To(Equal([]int64{101}))
			Expect(client.saveReqs).To(HaveLen(1))

			oldTypeUpdated, err := store.Get(ctx, oldTypeStrategyID)
			Expect(err).NotTo(HaveOccurred())
			Expect(oldTypeUpdated.RemoteRefs).To(BeEmpty())

			newTypeUpdated, err := store.Get(ctx, newTypeStrategyID)
			Expect(err).NotTo(HaveOccurred())
			Expect(newTypeUpdated.RemoteRefs).To(HaveLen(1))
			Expect(newTypeUpdated.RemoteRefs[0].EnvID).To(Equal(env.ID))
			Expect(newTypeUpdated.RemoteRefs[0].EnvName).To(Equal(env.Name))
		})
	})
})
