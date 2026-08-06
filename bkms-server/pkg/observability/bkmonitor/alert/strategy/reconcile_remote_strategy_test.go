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
	"testing"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

type fakeReconcileMonitorClient struct {
	*bkmapi.StubClient
	saveReqs     []*bkmapi.SaveAlarmStrategyReq
	saveFunc     func(req *bkmapi.SaveAlarmStrategyReq) (*bkmapi.SaveAlarmStrategyResp, error)
	deleteReqs   []*bkmapi.DeleteAlarmStrategyReq
	deleteCalled int
}

func (c *fakeReconcileMonitorClient) SaveAlarmStrategy(
	_ context.Context,
	req *bkmapi.SaveAlarmStrategyReq,
) (*bkmapi.SaveAlarmStrategyResp, error) {
	c.saveReqs = append(c.saveReqs, req)
	if c.saveFunc != nil {
		return c.saveFunc(req)
	}
	if req.ID > 0 {
		return &bkmapi.SaveAlarmStrategyResp{ID: req.ID}, nil
	}
	return &bkmapi.SaveAlarmStrategyResp{ID: 1001}, nil
}

func (c *fakeReconcileMonitorClient) DeleteAlarmStrategy(
	_ context.Context,
	req *bkmapi.DeleteAlarmStrategyReq,
) error {
	c.deleteCalled++
	c.deleteReqs = append(c.deleteReqs, req)
	return nil
}

func TestReconcileRemoteStrategyRecreatesMissingRemoteStrategyOnce(t *testing.T) {
	t.Helper()

	client := &fakeReconcileMonitorClient{
		StubClient: bkmapi.NewStub("tester"),
		saveFunc: func(req *bkmapi.SaveAlarmStrategyReq) (*bkmapi.SaveAlarmStrategyResp, error) {
			if req.ID > 0 {
				return nil, errors.New("api error, code: 3313003, message: 策略配置不存在, request_id: ")
			}
			return &bkmapi.SaveAlarmStrategyResp{ID: 2001}, nil
		},
	}
	svc := &Service{
		newClient: func(string) (bkmapi.MonitorClient, error) {
			return client, nil
		},
	}
	ws := &workspace.Workspace{ID: "test-ws"}
	ws.BkSystems.BkMonitorProjectID = "-100"
	envID := bson.NewObjectID()
	strategyID := bson.NewObjectID()
	strategy := &AlertStrategy{
		ID:            strategyID,
		WorkspaceID:   "test-ws",
		AppID:         "app-1",
		AppName:       "trpc-test-app",
		StrategyCode:  "memory_limit_usage_high",
		DisplayName:   "Memory Limit Usage High",
		MonitorMetric: "container_memory_working_set_bytes",
		Threshold:     ThresholdConfig{Method: "gte", Value: 80},
		Enabled:       true,
		RemoteRefs: []RemoteStrategyRef{{
			EnvID:            envID,
			EnvName:          "stage",
			RemoteStrategyID: 101,
		}},
	}
	targets := []remoteTargetContext{
		workspaceTarget(envID, "stage", "BCS-K8S-100018", "ieg-bkms-pd-stage"),
	}

	refs, err := svc.reconcileRemoteStrategy(context.Background(), ws, strategy, targets, "tester")
	if err != nil {
		t.Fatalf("reconcileRemoteStrategy() unexpected error: %v", err)
	}
	if len(client.saveReqs) != 2 {
		t.Fatalf("expected 2 save attempts, got %d", len(client.saveReqs))
	}
	if client.saveReqs[0].ID != 101 {
		t.Fatalf("expected first save to reuse remote ID 101, got %d", client.saveReqs[0].ID)
	}
	if client.saveReqs[1].ID != 0 {
		t.Fatalf("expected second save to recreate with empty ID, got %d", client.saveReqs[1].ID)
	}
	if client.deleteCalled != 0 {
		t.Fatalf("expected no delete calls during recreate flow, got %d", client.deleteCalled)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 remote ref, got %d", len(refs))
	}
	if refs[0].RemoteStrategyID != 2001 {
		t.Fatalf("expected recreated remote strategy ID 2001, got %d", refs[0].RemoteStrategyID)
	}
}

func TestReconcileRemoteStrategyDoesNotRetryNonMissingError(t *testing.T) {
	t.Helper()

	client := &fakeReconcileMonitorClient{
		StubClient: bkmapi.NewStub("tester"),
		saveFunc: func(req *bkmapi.SaveAlarmStrategyReq) (*bkmapi.SaveAlarmStrategyResp, error) {
			return nil, errors.New("api error, code: 5001001, message: arbitrary upstream failure, request_id: ")
		},
	}
	svc := &Service{
		newClient: func(string) (bkmapi.MonitorClient, error) {
			return client, nil
		},
	}
	ws := &workspace.Workspace{ID: "test-ws"}
	ws.BkSystems.BkMonitorProjectID = "-100"
	envID := bson.NewObjectID()
	strategy := &AlertStrategy{
		ID:            bson.NewObjectID(),
		WorkspaceID:   "test-ws",
		AppID:         "app-1",
		AppName:       "trpc-test-app",
		StrategyCode:  "memory_limit_usage_high",
		DisplayName:   "Memory Limit Usage High",
		MonitorMetric: "container_memory_working_set_bytes",
		Threshold:     ThresholdConfig{Method: "gte", Value: 80},
		Enabled:       true,
		RemoteRefs: []RemoteStrategyRef{{
			EnvID:            envID,
			EnvName:          "stage",
			RemoteStrategyID: 101,
		}},
	}
	targets := []remoteTargetContext{
		workspaceTarget(envID, "stage", "BCS-K8S-100018", "ieg-bkms-pd-stage"),
	}

	_, err := svc.reconcileRemoteStrategy(context.Background(), ws, strategy, targets, "tester")
	if err == nil {
		t.Fatal("reconcileRemoteStrategy() expected error, got nil")
	}
	if len(client.saveReqs) != 1 {
		t.Fatalf("expected 1 save attempt for non-missing error, got %d", len(client.saveReqs))
	}
	if client.saveReqs[0].ID != 101 {
		t.Fatalf("expected save attempt to reuse remote ID 101, got %d", client.saveReqs[0].ID)
	}
}

func workspaceTarget(envID bson.ObjectID, name, clusterID, namespace string) remoteTargetContext {
	return remoteTargetContext{
		Env: envmodel.Environment{
			ID:   envID,
			Name: name,
			Cluster: envmodel.BizCluster{
				ClusterID: clusterID,
				Namespace: namespace,
			},
		},
		Workloads: []string{"trpc-test-app"},
	}
}
