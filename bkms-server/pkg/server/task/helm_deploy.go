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
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// PollingDeployStatusArgs 轮询部署的参数
type PollingDeployStatusArgs struct {
	WorkspaceID     string `json:"workspaceID"`
	AppID           string `json:"appID"`
	EnvName         string `json:"envName"`
	TrafficLaneName string `json:"trafficLaneName"`
	DeployID        string `json:"deployID"`
}

// String 参数内容字符串化
func (args PollingDeployStatusArgs) String() string {
	// 如果 trafficLaneName 为空，即使用默认泳道，与页面中展示值保持一致为 default
	trafficLaneName := lo.Ternary(args.TrafficLaneName == "", "default", args.TrafficLaneName)
	return fmt.Sprintf(
		"<workspace: %s, appID: %s, envName: %s, trafficLaneName: %s, id: %s>",
		args.WorkspaceID, args.AppID, args.EnvName, trafficLaneName, args.DeployID,
	)
}

// PollingHelmDeployStatusArgs 轮询 Helm 部署状态的参数
type PollingHelmDeployStatusArgs = PollingDeployStatusArgs

// pollingHelmDeployStatus 轮询 Helm 应用部署状态
func pollingHelmDeployStatus(ctx context.Context, args PollingHelmDeployStatusArgs) (*EmptyResult, error) {
	log.Infof(ctx, "start polling deploy %s status, timeout: %ds", args, config.G.TaskPoller.DeployStatus.Timeout)

	// 轮询退出（不论成功 / 失败）都要释放锁
	defer helmdeploy.NewDeployLock(args.AppID, args.EnvName, args.TrafficLaneName).Release(ctx)

	// 设置轮询上下文和定时器
	ctx, cancel, ticker := setPollingContext(ctx, config.G.TaskPoller.DeployStatus)
	defer cancel()
	defer ticker.Stop()

	// 获取部署记录
	store, err := helmdeploy.NewRecordStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create deploy record store")
	}
	record, err := store.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		return nil, errors.Wrapf(err, "get deploy record")
	}

	// 初始化 Helm SDK action.Configuration，用于后续轮询查询 Release 状态
	debugLog := helm.NewHelmDebugLogger(ctx, record.ReleaseName, "polling-status")
	cfg, err := helm.NewActionConfiguration(record.ClusterID, record.Namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init action configuration for polling %s", record.ReleaseName)
	}

	// 获取状态失败重试次数（防止网络波动等原因）
	failureRetryCount := TotalFailureRetryCount
	for {
		// 记录当前状态以便后续比对
		curStatus := record.Status

		select {
		case <-ctx.Done():
			log.Warnf(ctx, "context timeout, stop update helm deploy %s status", args)
			record.Status = helm.StatusPollingTimeout
		case <-ticker.C:
			// 使用 Helm SDK 查询 Release 状态
			release, gErr := helm.GetReleaseStatus(cfg, record.ReleaseName)
			if gErr != nil {
				log.Errorf(ctx, "failed to get release %s: %v", args, gErr)
				// 轮询流程中失败超过指定次数，停止轮询
				failureRetryCount--
				if failureRetryCount <= 0 {
					log.Errorf(ctx, "stop polling release %s after %d retries", args, TotalFailureRetryCount)
					record.Status = helm.StatusPollingBroken
					record.Message = gErr.Error()
				}
				break
			}
			deployStatus := release.DeployResult.Status
			if deployStatus != record.Status {
				record.Status = deployStatus
				record.Revision = release.Version
				record.Message = release.DeployResult.Description
				if helm.IsStable(deployStatus) {
					record.EndedAt = time.Now()
				}
			}
		}

		// 若 record 状态变更，则需要保存入库
		if record.Status != curStatus {
			log.Infof(
				ctx, "helm deploy %s status changed from %s to %s, message: %s",
				args, curStatus, record.Status, record.Message,
			)
			// 使用闭包函数确保 defer 能在每次迭代时正确执行，避免资源泄露
			func() {
				// 为最终状态更新创建独立的 context，确保即使原 ctx 超时也能完成写入
				saveCtx, saveCancel := context.WithTimeout(context.Background(), saveStatusTimeout)
				defer saveCancel()
				// 更新部署记录
				if err = store.Update(saveCtx, record); err != nil {
					log.Errorf(saveCtx, "failed to update helm deploy record: %v", err)
				}
			}()
		}

		// 如果最新状态已经是最终态，退出轮询
		if helm.IsStable(record.Status) {
			// 泳道标签已通过 PostRenderer 在部署时前置注入，无需异步 Patch
			log.Infof(ctx, "helm deploy %s status is %s (Stable), stop polling and release lock", args, record.Status)

			// 部署成功时，记录应用到环境的关联
			if record.Status == helm.StatusDeployed {
				reg := storereg.G()
				// 1. 记录应用到环境的部署关联（envStore 初始化失败时仅告警，不阻断主流程）。
				envStore, sErr := envmodel.NewEnvironmentStoreMongo(database.Client(), database.Name())
				if sErr != nil {
					log.Errorf(ctx, "track env add app: create env store: %v", sErr)
				} else {
					deploy.TrackEnvAddApp(ctx, envStore, args.WorkspaceID, args.EnvName, args.AppID)
				}

				// 2. 异步将应用关联的告警策略同步到当前环境（失败仅记录日志，不影响部署结果）。
				ws, wsErr := reg.WorkspaceStore.Get(ctx, args.WorkspaceID)
				if wsErr != nil {
					log.Errorf(ctx, "get workspace %s for alert sync failed: %v", args.WorkspaceID, wsErr)
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
				if ws != nil && env != nil {
					log.Infof(
						ctx,
						"dispatch alert strategy sync, workspace=%s app=%s env=%s envID=%s lane=%s operator=%s",
						args.WorkspaceID, args.AppID, env.Name, env.ID.Hex(), args.TrafficLaneName, record.Operator,
					)
					// TODO(alertstrategy): 用 go 裸起 goroutine 无法保证跨 Pod 串行，
					// 后续迁移到 asynq 任务队列以解决多 Pod 并发风险。
					go alertstrategy.NewService(
						reg.AlertStrategyStore,
						reg.EnvStore,
						reg.AppStore,
						reg.ResourceSnapshotStore,
					).SyncStrategiesForAppInEnv(
						context.WithoutCancel(ctx), ws, args.AppID, env.ID, args.TrafficLaneName, record.Operator,
					)
				} else {
					log.Warnf(
						ctx,
						"skip alert strategy sync: ws or env is nil, workspace=%s app=%s envName=%s wsNil=%v envNil=%v",
						args.WorkspaceID, args.AppID, args.EnvName, ws == nil, env == nil,
					)
				}

				// 3. 部署成功后异步触发资源范围刷新
				go triggerTopologyRefreshAfterHelmDeploy(
					context.WithoutCancel(ctx), args, record.ClusterID, record.Namespace, record.ReleaseName,
				)
			}

			// 转换为操作结果 & 记录操作审计
			opResult := lo.Ternary(record.Status == helm.StatusDeployed, audit.ResultSuccess, audit.ResultFailed)
			go audit.AddOperationRecordAsync(
				context.WithoutCancel(ctx),
				audit.OperationTypeDeploy, audit.ResourceTypeApp, args.AppID,
				audit.WithResult(opResult), audit.WithWorkspaceID(args.WorkspaceID),
				audit.WithAppID(args.AppID), audit.WithEnvName(args.EnvName),
			)
			return &emptyResult, nil
		}
	}
}
