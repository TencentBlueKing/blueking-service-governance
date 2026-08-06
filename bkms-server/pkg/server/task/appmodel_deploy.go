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
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// PollingTrpcDeployStatusArgs 轮询 Trpc 部署状态的参数
type PollingTrpcDeployStatusArgs = PollingDeployStatusArgs

// pollingTrpcDeployStatus 轮询 Trpc 应用部署状态
func pollingTrpcDeployStatus(ctx context.Context, args PollingTrpcDeployStatusArgs) (*EmptyResult, error) {
	log.Infof(ctx, "start polling trpc deploy %s status, timeout: %ds", args, config.G.TaskPoller.DeployStatus.Timeout)

	// 设置轮询上下文和定时器
	ctx, cancel, ticker := setPollingContext(ctx, config.G.TaskPoller.DeployStatus)
	defer cancel()
	defer ticker.Stop()

	reg := storereg.G()
	store := reg.AppModelDeployRecordStore
	record, err := store.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		return nil, errors.Wrapf(err, "get deploy record")
	}
	var buildAutoDeployOperator *autodeploy.Operator
	if reg.BuildAutoDeployRecordStore != nil {
		buildAutoDeployOperator, err = autodeploy.NewOperator(reg.BuildAutoDeployRecordStore)
		if err != nil {
			return nil, errors.Wrap(err, "init build auto deploy operator")
		}
	}

	// 初始化状态获取器
	var state *appmodeldeploy.DeployState
	stateGetter := appmodeldeploy.NewDeployStateGetter(record)
	// 获取状态失败重试次数（防止网络波动等原因）
	failureRetryCount := TotalFailureRetryCount
	for {
		// 记录当前状态以便后续比对
		curStatus := record.Status

		select {
		case <-ctx.Done():
			log.Warnf(ctx, "context timeout, stop update helm deploy %s status", args)
			record.Status = appmodeldeploy.StatusPollingTimeout
		case <-ticker.C:
			state, err = stateGetter.Get(ctx)
			if err != nil {
				failureRetryCount--
				if failureRetryCount <= 0 {
					log.Errorf(ctx, "stop polling deploy state %s after %d retries", args, TotalFailureRetryCount)
					record.Status = appmodeldeploy.StatusPollingBroken
					record.Message = err.Error()
				}
				continue
			}

			if state.Status != record.Status {
				record.Status = state.Status
				record.Message = state.Message
				if state.Status.IsStable() {
					record.EndedAt = time.Now()
				}
			}
		}

		// 如果有更新的部署，则当前部署标记为用户取消
		if latestRecord, gErr := store.GetLatest(ctx, args.AppID, args.EnvName, args.TrafficLaneName); gErr != nil {
			// 获取最新部署记录失败，不影响继续执行轮询任务
			log.Errorf(ctx, "failed to get latest deploy record: %v", gErr)
		} else if latestRecord != nil && latestRecord.ID != record.ID {
			log.Warnf(ctx, "deploy %s status polling stoped by newer deployment %s", args, latestRecord.ID.Hex())
			// 当前部署标记为用户取消
			record.Status = appmodeldeploy.StatusCanceled
			record.Message = "deployment canceled: superseded by newer deployment"
			record.EndedAt = time.Now()
		}

		// 若 record 状态变更，则需要保存入库
		if record.Status != curStatus {
			// 使用闭包函数确保 defer 能在每次迭代时正确执行，避免资源泄露
			// 闭包返回 true 表示需要提前退出轮询
			shouldReturn := func() bool {
				// 为最终状态更新创建独立的 context，确保即使原 ctx 超时也能完成写入
				saveCtx, saveCancel := context.WithTimeout(context.Background(), saveStatusTimeout)
				defer saveCancel()
				// 更新前检查数据库状态，若已被修改为卸载状态则停止轮询，避免覆盖
				if dbRecord, gErr := store.Get(saveCtx, args.AppID, args.DeployID); gErr != nil {
					log.Errorf(saveCtx, "failed to get trpc deploy record: %v", gErr)
				} else if dbRecord.Status.IsUninstall() {
					log.Warnf(
						saveCtx, "trpc deploy %s status is %s (from db), stop deploy status polling",
						args, dbRecord.Status,
					)
					return true
				}

				log.Infof(
					saveCtx, "trpc deploy %s status changed from %s to %s, message: %s",
					args, curStatus, record.Status, record.Message,
				)
				// 更新部署记录
				if err = store.Update(saveCtx, record); err != nil {
					log.Errorf(saveCtx, "failed to update trpc deploy record: %v", err)
				}
				// 更新构建自动部署记录
				if buildAutoDeployOperator != nil {
					patch := autodeploy.StatusPatch{
						Stage:   autodeploy.StageDeploy,
						Status:  string(record.Status),
						Message: record.Message,
					}
					if record.Status.IsStable() {
						endedAt := record.EndedAt
						patch.EndedAt = &endedAt
					}
					uErr := buildAutoDeployOperator.UpdateStatus(saveCtx, autodeploy.Locator{
						AppID:    args.AppID,
						DeployID: args.DeployID,
					}, patch)
					if uErr != nil && !errors.Is(uErr, autodeploy.ErrRecordNotFound) {
						log.Errorf(saveCtx, "failed to update build auto deploy record by deployID: %v", uErr)
					}
				}
				return false
			}()
			if shouldReturn {
				return &emptyResult, nil
			}
		}

		// 如果最新状态已经是最终态，退出轮询
		if record.Status.IsStable() {
			log.Infof(ctx, "trpc deploy %s status is %s (Stable)，stop polling and release lock", args, record.Status)

			// 部署成功时，记录应用到环境的关联
			if record.Status == appmodeldeploy.StatusDeployed {
				log.Infof(
					ctx,
					"deploy succeeded, start post-deploy hooks for workspace=%s app=%s env=%s lane=%s operator=%s",
					args.WorkspaceID,
					args.AppID,
					args.EnvName,
					args.TrafficLaneName,
					record.Creator,
				)
				// 1. 记录应用到环境的部署关联（envStore 未初始化时仅告警，不阻断主流程）。
				if reg.EnvStore == nil {
					log.Errorf(ctx, "track env add app: env store is not initialized")
				} else {
					deploy.TrackEnvAddApp(ctx, reg.EnvStore, args.WorkspaceID, args.EnvName, args.AppID)
				}

				// 2. 异步将应用关联的告警策略同步到当前环境（失败仅记录日志，不影响部署结果）。
				ws, wsErr := reg.WorkspaceStore.Get(ctx, args.WorkspaceID)
				if wsErr != nil {
					log.Errorf(ctx, "get workspace %s for alert sync failed: %v", args.WorkspaceID, wsErr)
				}
				var env *envmodel.Environment
				if reg.EnvStore == nil {
					log.Errorf(ctx, "env store is not initialized for alert sync")
				} else {
					env, err = reg.EnvStore.GetByName(ctx, args.WorkspaceID, args.AppID, args.EnvName)
					if err != nil {
						log.Errorf(ctx, "get env %s for alert sync failed: %v", args.EnvName, err)
					}
				}
				if ws != nil && env != nil {
					log.Infof(
						ctx,
						"dispatch alert strategy sync, workspace=%s app=%s env=%s envID=%s lane=%s operator=%s",
						args.WorkspaceID, args.AppID, env.Name, env.ID.Hex(), args.TrafficLaneName, record.Creator,
					)
					// TODO(alertstrategy): 用 go 裸起 goroutine 无法保证跨 Pod 串行，
					// 后续迁移到 asynq 任务队列以解决多 Pod 并发风险。
					go alertstrategy.NewService(
						reg.AlertStrategyStore, reg.EnvStore, reg.AppStore, reg.ResourceSnapshotStore,
					).SyncStrategiesForAppInEnv(
						context.WithoutCancel(ctx), ws, args.AppID, env.ID, args.TrafficLaneName, record.Creator,
					)
				} else {
					log.Warnf(
						ctx,
						"skip alert strategy sync: ws or env is nil, workspace=%s app=%s envName=%s wsNil=%v envNil=%v",
						args.WorkspaceID, args.AppID, args.EnvName, ws == nil, env == nil,
					)
				}
				// 3. 部署成功后异步触发资源范围刷新
				var resourceKeys []topology.ResourceKeyEntry
				for _, rk := range record.ResourceKeys {
					resourceKeys = append(resourceKeys, topology.ResourceKeyEntry{Kind: rk.Kind, Name: rk.Name})
				}
				go triggerTopologyRefreshAfterAppModelDeploy(
					context.WithoutCancel(ctx),
					args,
					record.ClusterID,
					record.Namespace,
					resourceKeys,
					record.LabelSelector,
				)
			}

			// 转换为操作结果 & 记录操作审计
			opResult := lo.Ternary(
				record.Status == appmodeldeploy.StatusDeployed,
				audit.ResultSuccess,
				audit.ResultFailed,
			)
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
