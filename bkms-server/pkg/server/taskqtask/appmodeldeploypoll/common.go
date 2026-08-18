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

package appmodeldeploypoll

import (
	"context"
	"fmt"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// triggerTopologyRefresh 按部署记录的 ResourceKeys / LabelSelector 刷拓扑资源范围
// 由 runTick 在 goroutine 中调用，resourceKeys 为空则跳过；失败只打日志，不影响本 tick
func triggerTopologyRefresh(ctx context.Context, args Args, record *appmodeldeploy.Record) {
	var resourceKeys []topology.ResourceKeyEntry
	for _, rk := range record.ResourceKeys {
		resourceKeys = append(resourceKeys, topology.ResourceKeyEntry{Kind: rk.Kind, Name: rk.Name})
	}
	if len(resourceKeys) == 0 {
		log.Warnf(ctx, "skip topology refresh (app model): resourceKeys is empty")
		return
	}

	store, err := topology.NewResourceSnapshotStoreMongo(database.Client(), database.Name())
	if err != nil {
		log.Errorf(ctx, "topology refresh (app model): create store: %v", err)
		return
	}
	topology.NewRefresher(store).TriggerRefresh(ctx, topology.RefreshArgs{
		AppID:           args.AppID,
		EnvName:         args.EnvName,
		TrafficLaneName: args.TrafficLaneName,
		ClusterID:       record.ClusterID,
		Namespace:       record.Namespace,
		ResourceKeys:    resourceKeys,
		LabelSelector:   record.LabelSelector,
	})
}

// handleDeploySucceeded 部署成功后的后置动作：记录应用与环境关联，并异步同步该环境告警策略
// 仅 StatusDeployed 时由 onStable 调用；任一步失败只打日志，不改部署结果、不让本 tick 失败
func handleDeploySucceeded(ctx context.Context, args Args, record *appmodeldeploy.Record) {
	// 后置动作依赖全局 registry，未初始化则两步都做不了
	reg := storereg.G()
	if reg == nil {
		log.Errorf(ctx, "post-deploy hooks: registry is not initialized")
		return
	}

	log.Infof(
		ctx, "deploy succeeded, start post-deploy hooks for workspace=%s app=%s env=%s lane=%s operator=%s",
		args.WorkspaceID, args.AppID, args.EnvName, args.TrafficLaneName, record.Creator,
	)
	// 记录应用到环境的部署关联，envStore 未初始化时只打日志，不阻断后续告警同步
	if reg.EnvStore == nil {
		log.Errorf(ctx, "track env add app: env store is not initialized")
	} else {
		deploy.TrackEnvAddApp(ctx, reg.EnvStore, args.WorkspaceID, args.EnvName, args.AppID)
	}

	syncAlertStrategies(ctx, reg, args, record.Creator)
}

// syncAlertStrategies 查 workspace / env 后异步同步该环境告警策略
// 由 handleDeploySucceeded 调用；缺 store、缺数据或查询失败只打日志并返回，不影响已完成的部署结果
// FIXME (alert strategy): 用 go 裸起 goroutine 无法保证跨 Pod 串行，
// 后续迁移到 asynq 任务队列以解决多 Pod 并发风险
func syncAlertStrategies(ctx context.Context, reg *storereg.Registry, args Args, operator string) {
	warnLogPrefix := fmt.Sprintf(
		"skip alert strategy sync: workspace=%s app=%s envName=%s, ",
		args.WorkspaceID, args.AppID, args.EnvName,
	)
	// 同步依赖的 store 任一缺失都做不下去，提前跳过，避免后续解引用 panic
	if reg.WorkspaceStore == nil || reg.EnvStore == nil ||
		reg.AlertStrategyStore == nil || reg.AppStore == nil || reg.ResourceSnapshotStore == nil {
		log.Error(ctx, warnLogPrefix+"required store is not initialized")
		return
	}

	ws, err := reg.WorkspaceStore.Get(ctx, args.WorkspaceID)
	if err != nil {
		log.Errorf(ctx, "get workspace %s for alert sync failed: %v", args.WorkspaceID, err)
	}
	if ws == nil {
		log.Warn(ctx, warnLogPrefix+"workspace is nil")
		return
	}
	env, err := reg.EnvStore.GetByName(ctx, args.WorkspaceID, args.AppID, args.EnvName)
	if err != nil {
		log.Errorf(ctx, "get env %s for alert sync failed: %v", args.EnvName, err)
	}
	if env == nil {
		log.Warn(ctx, warnLogPrefix+"env is nil")
		return
	}
	// 告警同步走独立 goroutine，避免拖住本 tick；失败由 SyncStrategiesForAppInEnv 内部记日志
	log.Infof(
		ctx, "dispatch alert strategy sync, workspace=%s app=%s env=%s envID=%s lane=%s operator=%s",
		args.WorkspaceID, args.AppID, env.Name, env.ID.Hex(), args.TrafficLaneName, operator,
	)
	go alertstrategy.NewService(
		reg.AlertStrategyStore, reg.EnvStore, reg.AppStore, reg.ResourceSnapshotStore,
	).SyncStrategiesForAppInEnv(
		context.WithoutCancel(ctx), ws, args.AppID, env.ID, args.TrafficLaneName, operator,
	)
}
