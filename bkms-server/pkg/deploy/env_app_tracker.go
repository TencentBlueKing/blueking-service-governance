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

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
)

// TrackEnvAddApp 记录应用尝试部署到环境，部署成功轮询时也可幂等补写
// 由于 AddApp 使用 $addToSet 天然幂等，因此无需判断泳道
func TrackEnvAddApp(
	ctx context.Context,
	envStore envmodel.EnvironmentStore,
	workspaceID, envName, appID string,
) {
	env, err := envStore.GetByName(ctx, workspaceID, appID, envName)
	if err != nil {
		log.Errorf(ctx, "track env add app: get env by name (workspace: %s, env: %s): %v", workspaceID, envName, err)
		return
	}
	if err = envStore.AddApp(ctx, env.ID, appID); err != nil {
		log.Errorf(ctx, "track env add app: add app %s to env %s: %v", appID, envName, err)
	}
}

// TrackEnvRemoveApp 记录应用从环境卸载（卸载成功时调用）
// 仅在空泳道或基线泳道卸载时才移除记录，避免卸载特性泳道时误删记录
func TrackEnvRemoveApp(
	ctx context.Context,
	envStore envmodel.EnvironmentStore,
	workspaceID, envName, trafficLaneName, appID string,
) {
	if !isEmptyOrBaselineLane(ctx, workspaceID, envName, trafficLaneName) {
		return
	}
	env, err := envStore.GetByName(ctx, workspaceID, appID, envName)
	if err != nil {
		log.Errorf(
			ctx, "track env remove app: get env by name (workspace: %s, env: %s): %v", workspaceID, envName, err,
		)
		return
	}
	if err = envStore.RemoveApp(ctx, env.ID, appID); err != nil {
		log.Errorf(ctx, "track env remove app: remove app %s from env %s: %v", appID, envName, err)
	}
}

// isEmptyOrBaselineLane 判断当前泳道是否为空泳道或基线泳道
func isEmptyOrBaselineLane(ctx context.Context, workspaceID, envName, trafficLaneName string) bool {
	// 空泳道，说明没有启用泳道功能，直接返回 true
	if trafficLaneName == "" {
		return true
	}
	// 非空泳道，需要查询基线泳道信息进行比对
	baselineLane, err := trafficmanager.New().GetBaselineTrafficLane(ctx, workspaceID, envName)
	if err != nil {
		log.Errorf(
			ctx, "check baseline lane: get baseline lane (workspace: %s, env: %s): %v", workspaceID, envName, err,
		)
		return false
	}
	return baselineLane != nil && baselineLane.LaneName == trafficLaneName
}
