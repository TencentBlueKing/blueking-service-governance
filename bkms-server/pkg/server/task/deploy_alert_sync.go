package task

import (
	"context"
	"fmt"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// syncAlertStrategiesAfterDeploy 在部署成功后异步同步应用关联的告警策略
// Workspace 或 Environment 获取失败时仅记录日志并提前返回，不影响已经完成的部署结果
func syncAlertStrategiesAfterDeploy(
	ctx context.Context,
	reg *storereg.Registry,
	envStore envmodel.EnvironmentStore,
	args PollingDeployStatusArgs,
	operator string,
) {
	warnLogPrefix := fmt.Sprintf(
		"skip alert strategy sync: workspace=%s app=%s envName=%s, ",
		args.WorkspaceID, args.AppID, args.EnvName,
	)
	ws, err := reg.WorkspaceStore.Get(ctx, args.WorkspaceID)
	if err != nil {
		log.Errorf(ctx, "get workspace %s for alert sync failed: %v", args.WorkspaceID, err)
	}
	if ws == nil {
		log.Warn(ctx, warnLogPrefix+"workspace is nil")
		return
	}

	var env *envmodel.Environment
	if envStore == nil {
		log.Errorf(ctx, "env store is not initialized for alert sync")
	} else {
		env, err = envStore.GetByName(ctx, args.WorkspaceID, args.AppID, args.EnvName)
		if err != nil {
			log.Errorf(ctx, "get env %s for alert sync failed: %v", args.EnvName, err)
		}
	}
	if env == nil {
		log.Warn(ctx, warnLogPrefix+"env is nil")
		return
	}

	log.Infof(
		ctx, "dispatch alert strategy sync, workspace=%s app=%s env=%s envID=%s lane=%s operator=%s",
		args.WorkspaceID, args.AppID, env.Name, env.ID.Hex(), args.TrafficLaneName, operator,
	)
	// FIXME (alert strategy): 用 go 裸起 goroutine 无法保证跨 Pod 串行，
	// 后续迁移到 asynq 任务队列以解决多 Pod 并发风险
	go alertstrategy.NewService(
		reg.AlertStrategyStore, envStore, reg.AppStore, reg.ResourceSnapshotStore,
	).SyncStrategiesForAppInEnv(
		context.WithoutCancel(ctx), ws, args.AppID, env.ID, args.TrafficLaneName, operator,
	)
}
