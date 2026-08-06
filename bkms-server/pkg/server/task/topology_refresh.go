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

package task

import (
	"context"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// triggerTopologyRefreshAfterHelmDeploy Helm 应用部署成功后触发拓扑资源范围刷新
// 该函数应在 goroutine 中调用
func triggerTopologyRefreshAfterHelmDeploy(
	ctx context.Context,
	args PollingDeployStatusArgs,
	clusterID, namespace, releaseName string,
) {
	store, err := topology.NewResourceSnapshotStoreMongo(database.Client(), database.Name())
	if err != nil {
		log.Errorf(ctx, "topology refresh (helm): create store: %v", err)
		return
	}

	refresher := topology.NewRefresher(store)
	refresher.TriggerRefresh(
		ctx,
		topology.RefreshArgs{
			AppID:           args.AppID,
			EnvName:         args.EnvName,
			TrafficLaneName: args.TrafficLaneName,
			ClusterID:       clusterID,
			Namespace:       namespace,
			ReleaseName:     releaseName,
		},
	)
}

// triggerTopologyRefreshAfterAppModelDeploy AppModel 部署成功后触发拓扑资源范围刷新
// 基于部署记录中的 ResourceKeys 和 LabelSelector 进行刷新
// 该函数应在 goroutine 中调用
func triggerTopologyRefreshAfterAppModelDeploy(
	ctx context.Context,
	args PollingDeployStatusArgs,
	clusterID, namespace string,
	resourceKeys []topology.ResourceKeyEntry,
	labelSelector map[string]string,
) {
	store, err := topology.NewResourceSnapshotStoreMongo(database.Client(), database.Name())
	if err != nil {
		log.Errorf(ctx, "topology refresh (appmodel): create store: %v", err)
		return
	}

	refresher := topology.NewRefresher(store)
	refresher.TriggerRefresh(
		ctx,
		topology.RefreshArgs{
			AppID:           args.AppID,
			EnvName:         args.EnvName,
			TrafficLaneName: args.TrafficLaneName,
			ClusterID:       clusterID,
			Namespace:       namespace,
			ResourceKeys:    resourceKeys,
			LabelSelector:   labelSelector,
		},
	)
}
