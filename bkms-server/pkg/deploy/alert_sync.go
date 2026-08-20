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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/alertstrategysync"
)

// SyncAlertStrategiesAfterDeploy 在部署成功后投递告警策略同步任务到 asynq 队列
// 投递失败时只打日志并返回，不影响已经完成的部署结果
func SyncAlertStrategiesAfterDeploy(
	ctx context.Context,
	workspaceID, appID, envName, trafficLaneName, operator string,
) {
	log.Infof(
		ctx, "enqueue alert strategy sync task, workspace=%s app=%s env=%s lane=%s operator=%s",
		workspaceID, appID, envName, trafficLaneName, operator,
	)
	if err := taskq.Enqueue(ctx, alertstrategysync.SyncTask.NewTask(alertstrategysync.Args{
		WorkspaceID:     workspaceID,
		AppID:           appID,
		EnvName:         envName,
		TrafficLaneName: trafficLaneName,
	})); err != nil {
		log.Errorf(ctx, "enqueue alert strategy sync task failed: %v", err)
	}
}
