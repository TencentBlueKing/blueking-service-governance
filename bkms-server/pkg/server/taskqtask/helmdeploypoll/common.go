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

package helmdeploypoll

import (
	"context"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// triggerTopologyRefresh 按 ReleaseName 刷拓扑资源范围
// 由 Handle 在 goroutine 中调用，releaseName 为空则跳过；失败只打日志，不影响本 tick
func triggerTopologyRefresh(ctx context.Context, args Args, record *helmdeploy.Record) {
	if record.ReleaseName == "" {
		log.Warnf(ctx, "skip topology refresh (helm): releaseName is empty")
		return
	}

	store, err := topology.NewResourceSnapshotStoreMongo(database.Client(), database.Name())
	if err != nil {
		log.Errorf(ctx, "topology refresh (helm): create store: %v", err)
		return
	}
	topology.NewRefresher(store).TriggerRefresh(ctx, topology.RefreshArgs{
		AppID:           args.AppID,
		EnvName:         args.EnvName,
		TrafficLaneName: args.TrafficLaneName,
		ClusterID:       record.ClusterID,
		Namespace:       record.Namespace,
		ReleaseName:     record.ReleaseName,
	})
}

// handleDeploySucceeded 部署成功后的后置动作：记录应用与环境关联，并异步同步该环境告警策略
// 仅 StatusDeployed 时由 onStable 调用；任一步失败只打日志，不改部署结果、不让本 tick 失败
func handleDeploySucceeded(ctx context.Context, args Args, record *helmdeploy.Record) {
	reg := storereg.G()
	if reg == nil {
		log.Errorf(ctx, "post-deploy hooks: registry is not initialized")
		return
	}

	log.Infof(
		ctx, "deploy succeeded, start post-deploy hooks for workspace=%s app=%s env=%s lane=%s operator=%s",
		args.WorkspaceID, args.AppID, args.EnvName, args.TrafficLaneName, record.Operator,
	)
	// 记录应用到环境的部署关联，envStore 未初始化时只打日志，不阻断后续告警同步
	if reg.EnvStore == nil {
		log.Errorf(ctx, "track env add app: env store is not initialized")
	} else {
		deploy.TrackEnvAddApp(ctx, reg.EnvStore, args.WorkspaceID, args.EnvName, args.AppID)
	}

	// 异步把应用关联的告警策略同步到当前环境，失败只打日志，不改部署结果、不让本 tick 失败
	deploy.SyncAlertStrategiesAfterDeploy(
		ctx, args.WorkspaceID, args.AppID, args.EnvName, args.TrafficLaneName, record.Operator,
	)
}
