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

// FIXME: Helm 部署状态轮询已迁至 asynq（pkg/server/taskqtask/helmdeploypoll）
// 本文件仅保留 RabbitMQ 存量消费路径，存量队列耗尽后移除此实现

package task

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
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

	// 进入部署轮询后异步触发一次资源范围刷新
	go triggerTopologyRefreshForHelmDeploy(
		context.WithoutCancel(ctx),
		args,
		record.ClusterID,
		record.Namespace,
		record.ReleaseName,
	)

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
			metrics.DeployFinished(metrics.DeployKindHelm, string(record.Status), record.StartedAt, time.Now())
			// 泳道标签已通过 PostRenderer 在部署时前置注入，无需异步 Patch
			log.Infof(ctx, "helm deploy %s status is %s (Stable), stop polling and release lock", args, record.Status)

			// 部署成功时，记录应用到环境的关联
			if record.Status == helm.StatusDeployed {
				handleHelmDeploySucceeded(ctx, args, record)
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

// handleHelmDeploySucceeded 处理 Helm 部署成功后的后置动作
// 包括记录应用与环境的部署关联，以及异步同步当前环境下的告警策略；这些动作失败时只记录日志，不阻断部署轮询结果
func handleHelmDeploySucceeded(ctx context.Context, args PollingDeployStatusArgs, record *helmdeploy.Record) {
	reg := storereg.G()

	// 1. 记录应用到环境的部署关联（envStore 未初始化时仅告警，不阻断主流程）
	if reg.EnvStore == nil {
		log.Errorf(ctx, "track env add app: env store is not initialized")
	} else {
		deploy.TrackEnvAddApp(ctx, reg.EnvStore, args.WorkspaceID, args.EnvName, args.AppID)
	}

	// 2. 异步将应用关联的告警策略同步到当前环境（失败仅记录日志，不影响部署结果）
	deploy.SyncAlertStrategiesAfterDeploy(
		ctx, args.WorkspaceID, args.AppID, args.EnvName, args.TrafficLaneName, record.Operator,
	)
}
