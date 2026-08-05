package task

import (
	"context"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// triggerTopologyRefreshAfterHelmDeploy Helm 应用部署进入稳定终态后触发拓扑资源范围刷新
// 该函数应在 goroutine 中调用
func triggerTopologyRefreshAfterHelmDeploy(
	ctx context.Context,
	args PollingDeployStatusArgs,
	clusterID, namespace, releaseName string,
) {
	if releaseName == "" {
		log.Warnf(ctx, "skip topology refresh (helm): releaseName is empty")
		return
	}

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

// triggerTopologyRefreshAfterAppModelDeploy AppModel 部署进入稳定终态后触发拓扑资源范围刷新
// 基于部署记录中的 ResourceKeys 和 LabelSelector 进行刷新
// 该函数应在 goroutine 中调用
func triggerTopologyRefreshAfterAppModelDeploy(
	ctx context.Context,
	args PollingDeployStatusArgs,
	clusterID, namespace string,
	resourceKeys []topology.ResourceKeyEntry,
	labelSelector map[string]string,
) {
	if len(resourceKeys) == 0 {
		log.Warnf(ctx, "skip topology refresh (app model): resourceKeys is empty")
		return
	}

	store, err := topology.NewResourceSnapshotStoreMongo(database.Client(), database.Name())
	if err != nil {
		log.Errorf(ctx, "topology refresh (app model): create store: %v", err)
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
