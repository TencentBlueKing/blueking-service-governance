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

package deploy

import (
	"context"
	"fmt"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// SyncAlertStrategiesAfterDeploy 在部署成功后异步同步应用关联的告警策略
// 缺 store、Workspace / Environment 获取失败时只打日志并返回，不影响已经完成的部署结果
func SyncAlertStrategiesAfterDeploy(
	ctx context.Context,
	workspaceID, appID, envName, trafficLaneName, operator string,
) {
	reg := storereg.G()
	warnLogPrefix := fmt.Sprintf(
		"skip alert strategy sync: workspace=%s app=%s envName=%s, ",
		workspaceID, appID, envName,
	)
	if reg == nil {
		log.Errorf(ctx, "skip alert strategy sync: registry is not initialized")
		return
	}
	// 同步依赖的 store 任一缺失都做不下去，提前跳过，避免后续解引用 panic
	if reg.WorkspaceStore == nil || reg.EnvStore == nil ||
		reg.AlertStrategyStore == nil || reg.AppStore == nil || reg.ResourceSnapshotStore == nil {
		log.Error(ctx, warnLogPrefix+"required store is not initialized")
		return
	}

	ws, err := reg.WorkspaceStore.Get(ctx, workspaceID)
	if err != nil {
		log.Errorf(ctx, "get workspace %s for alert sync failed: %v", workspaceID, err)
		return
	}
	if ws == nil {
		log.Warn(ctx, warnLogPrefix+"workspace is nil")
		return
	}
	env, err := reg.EnvStore.GetByName(ctx, workspaceID, appID, envName)
	if err != nil {
		log.Errorf(ctx, "get env %s for alert sync failed: %v", envName, err)
		return
	}
	if env == nil {
		log.Warn(ctx, warnLogPrefix+"env is nil")
		return
	}

	log.Infof(
		ctx, "dispatch alert strategy sync, workspace=%s app=%s env=%s envID=%s lane=%s operator=%s",
		workspaceID, appID, env.Name, env.ID.Hex(), trafficLaneName, operator,
	)
	// FIXME (alert strategy): 用 go 裸起 goroutine 无法保证跨 Pod 串行，
	// 后续迁移到 asynq 任务队列以解决多 Pod 并发风险
	go alertstrategy.NewService(
		reg.AlertStrategyStore, reg.EnvStore, reg.AppStore, reg.ResourceSnapshotStore,
	).SyncStrategiesForAppInEnv(
		context.WithoutCancel(ctx), ws, appID, env.ID, trafficLaneName, operator,
	)
}
