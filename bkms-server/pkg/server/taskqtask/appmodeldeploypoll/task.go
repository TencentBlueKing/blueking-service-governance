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
	"time"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

const (
	// name asynq 任务类型名
	name = "taskq.pollingAppModelDeployStatus"
	// tickMaxRetry 单次 tick 意外失败的 asynq 重试上限，不含轮询续跑
	tickMaxRetry = 10
	// totalFailureRetryCount 查部署状态连续失败次数上限，耗尽后标 StatusPollingBroken
	totalFailureRetryCount = 10
	// saveStatusTimeout 状态落库的独立超时，避免 handler ctx 取消导致写不进去
	saveStatusTimeout = 10 * time.Second
)

// Args AppModel 部署状态轮询的业务参数，不含用户身份
type Args struct {
	WorkspaceID        string `json:"workspaceID"`
	AppID              string `json:"appID"`
	EnvName            string `json:"envName"`
	TrafficLaneName    string `json:"trafficLaneName"`
	DeployID           string `json:"deployID"`
	FailureRetryRemain int    `json:"failureRetryRemain,omitempty"`
}

// String 输出轮询身份与剩余失败次数，便于日志对齐同一部署的连续 tick
func (args Args) String() string {
	trafficLaneName := lo.Ternary(args.TrafficLaneName == "", "default", args.TrafficLaneName)
	return fmt.Sprintf(
		"<workspace: %s, appID: %s, envName: %s, trafficLaneName: %s, id: %s, remain: %d>",
		args.WorkspaceID, args.AppID, args.EnvName, trafficLaneName, args.DeployID, args.FailureRetryRemain,
	)
}

// Task 部署状态轮询任务；init 赋值避免与 enqueueNext 引用形成包初始化环
var Task *taskq.TaskType[Args]

func init() {
	Task = taskq.NewTaskType[Args](name, handle, asynq.MaxRetry(tickMaxRetry))
}

// PollingInterval 读 TaskPoller.DeployStatus.Interval（秒），作为下一 tick 的 ProcessIn 延迟
func PollingInterval() time.Duration {
	return time.Duration(config.G.TaskPoller.DeployStatus.Interval) * time.Second
}

// pollingTimeout 读 TaskPoller.DeployStatus.Timeout（秒），从 record.StartedAt 起算轮询窗口
func pollingTimeout() time.Duration {
	return time.Duration(config.G.TaskPoller.DeployStatus.Timeout) * time.Second
}

func handle(ctx context.Context, args Args) error {
	reg := storereg.G()
	if reg == nil ||
		reg.AppModelDeployRecordStore == nil ||
		reg.BuildAutoDeployRecordStore == nil {
		log.Errorf(ctx, "appmodel deploy poll stores not initialized, stop task: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "appmodel deploy poll stores not initialized")
	}
	return NewManager(reg.AppModelDeployRecordStore, reg.BuildAutoDeployRecordStore).Handle(ctx, args)
}

// Manager 执行一次 AppModel 部署状态轮询 tick
type Manager struct {
	recordStore     appmodeldeploy.RecordStore
	autoDeployStore autodeploy.RecordStore
}

// NewManager 注入部署轮询所需 store，供 asynq handler 与单测共用
func NewManager(recordStore appmodeldeploy.RecordStore, autoDeployStore autodeploy.RecordStore) *Manager {
	return &Manager{recordStore: recordStore, autoDeployStore: autoDeployStore}
}

// Handle 执行一次部署状态轮询 tick：读本地记录，必要时查集群状态并落库。
// 已稳定或已卸载则直接返回；仍在跑则 ProcessIn 投递下一 tick（新任务，retry 从 0 计）。
// asynq MaxRetry(tickMaxRetry) 只约束本 tick 的意外失败（如 enqueue 失败），不约束轮询次数；
// 轮询窗口由 pollingTimeout 截断，查状态失败次数由 FailureRetryRemain 截断。
// 不可恢复错误 wrap taskq.ErrStopRetry，避免 asynq 空转重试。
func (m *Manager) Handle(ctx context.Context, args Args) error {
	// store / 记录缺失无法自行恢复，停掉 asynq 重试
	if m.recordStore == nil {
		return errors.Wrap(taskq.ErrStopRetry, "appmodel deploy poll stores not initialized")
	}

	record, err := m.recordStore.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		return errors.Wrapf(taskq.ErrStopRetry, "get deploy record: %v", err)
	}
	// 迟到或重复 tick：已稳定 / 已卸载则不再查状态
	if record.Status.IsStable() || record.Status.IsUninstall() {
		log.Infof(ctx, "deploy %s already %s, skip tick", args, record.Status)
		return nil
	}
	// 轮询窗口到点，与查状态失败无关，直接标超时停掉
	if time.Since(record.StartedAt) >= pollingTimeout() {
		log.Warnf(ctx, "deploy %s polling window exceeded, mark pollingTimeout", args)
		return m.terminate(ctx, record, args, appmodeldeploy.StatusPollingTimeout, "")
	}

	go triggerTopologyRefresh(context.WithoutCancel(ctx), args, record)

	// 查状态失败次数走业务计数，不走 asynq MaxRetry；首 tick 未带 remain 时补满
	remain := args.FailureRetryRemain
	if remain <= 0 {
		remain = totalFailureRetryCount
	}

	curStatus := record.Status
	state, err := appmodeldeploy.NewDeployStateGetter(record).Get(ctx)
	if err != nil {
		remain--
		if remain <= 0 {
			log.Errorf(ctx, "stop polling deploy state %s after %d retries", args, totalFailureRetryCount)
			return m.terminate(ctx, record, args, appmodeldeploy.StatusPollingBroken, err.Error())
		}
		// 查询失败但额度未用尽：投下一 tick，本 tick 返回 nil，不扣 asynq 重试
		log.Errorf(ctx, "fetch deploy %s status failed, remain=%d: %v", args, remain, err)
		return m.enqueueNext(ctx, args, remain)
	}

	if state.Status != record.Status {
		record.Status = state.Status
		record.Message = state.Message
		if state.Status.IsStable() {
			record.EndedAt = time.Now()
		}
	}

	// 同环境出现更新的部署，当前记录标取消，避免两路轮询互相覆盖
	if latest, gErr := m.recordStore.GetLatest(ctx, args.AppID, args.EnvName, args.TrafficLaneName); gErr != nil {
		log.Errorf(ctx, "failed to get latest deploy record: %v", gErr)
	} else if latest != nil && latest.ID != record.ID {
		log.Warnf(ctx, "deploy %s status polling stoped by newer deployment %s", args, latest.ID.Hex())
		record.Status = appmodeldeploy.StatusCanceled
		record.Message = "deployment canceled: superseded by newer deployment"
		record.EndedAt = time.Now()
	}

	// 落库失败只打日志，不让本 tick 失败，避免 asynq 重试把状态写乱
	if record.Status != curStatus {
		if err = m.save(ctx, record, args); err != nil {
			log.Errorf(ctx, "failed to update trpc deploy record: %v", err)
		}
	}
	if record.Status.IsUninstall() {
		return nil
	}
	if record.Status.IsStable() {
		return m.onStable(ctx, record, args)
	}
	// 仍在跑：ProcessIn 投新任务续跑，retry 从 0 计
	return m.enqueueNext(ctx, args, remain)
}

// enqueueNext 按配置间隔 ProcessIn 投递下一 tick；新任务 retry 从 0 计
func (m *Manager) enqueueNext(ctx context.Context, args Args, remain int) error {
	args.FailureRetryRemain = remain
	interval := PollingInterval()
	log.Infof(ctx, "schedule next poll for deploy %s in %s remain=%d", args, interval, remain)
	if err := taskq.Enqueue(ctx, Task.NewTask(args), asynq.ProcessIn(interval)); err != nil {
		return errors.Wrap(err, "enqueue next appmodel deploy poll tick")
	}
	return nil
}

// terminate 把记录标为指定终态并落库；卸载态直接返回，其余走 onStable 副作用
func (m *Manager) terminate(
	ctx context.Context,
	record *appmodeldeploy.Record,
	args Args,
	status appmodeldeploy.Status,
	message string,
) error {
	record.Status = status
	if message != "" {
		record.Message = message
	}
	if record.EndedAt.IsZero() {
		record.EndedAt = time.Now()
	}
	if err := m.save(ctx, record, args); err != nil {
		log.Errorf(ctx, "failed to update trpc deploy record: %v", err)
	}
	if record.Status.IsUninstall() {
		return nil
	}
	return m.onStable(ctx, record, args)
}

// save 落部署记录；库中已是卸载态则不再覆盖。有 auto deploy store 时同步其状态
func (m *Manager) save(ctx context.Context, record *appmodeldeploy.Record, args Args) error {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveStatusTimeout)
	defer cancel()

	dbRecord, err := m.recordStore.Get(saveCtx, args.AppID, args.DeployID)
	if err != nil {
		log.Errorf(saveCtx, "failed to get trpc deploy record: %v", err)
	} else if dbRecord.Status.IsUninstall() {
		log.Warnf(
			saveCtx, "trpc deploy %s status is %s (from db), stop deploy status polling",
			args, dbRecord.Status,
		)
		record.Status = dbRecord.Status
		return nil
	}

	log.Infof(
		saveCtx, "trpc deploy %s status changed to %s, message: %s",
		args, record.Status, record.Message,
	)
	if err = m.recordStore.Update(saveCtx, record); err != nil {
		return err
	}
	// 普通部署没有 auto deploy 记录；store 未注入时只打日志，不让本 tick 失败
	if m.autoDeployStore == nil {
		log.Errorf(saveCtx, "deploy %s auto deploy store is nil, skip sync", args)
		return nil
	}
	operator, err := autodeploy.NewOperator(m.autoDeployStore)
	if err != nil {
		return err
	}
	patch := autodeploy.StatusPatch{
		Stage:   autodeploy.StageDeploy,
		Status:  string(record.Status),
		Message: record.Message,
	}
	if record.Status.IsStable() {
		endedAt := record.EndedAt
		patch.EndedAt = &endedAt
	}
	uErr := operator.UpdateStatus(saveCtx, autodeploy.Locator{
		AppID:    args.AppID,
		DeployID: args.DeployID,
	}, patch)
	if uErr != nil && !errors.Is(uErr, autodeploy.ErrRecordNotFound) {
		log.Errorf(saveCtx, "failed to update build auto deploy record by deployID: %v", uErr)
	}
	return nil
}

// onStable 终态副作用失败只打日志，不改部署结果，也不让本 tick 失败
func (m *Manager) onStable(ctx context.Context, record *appmodeldeploy.Record, args Args) error {
	metrics.DeployFinished(metrics.DeployKindAppModel, string(record.Status), record.StartedAt, time.Now())
	log.Infof(ctx, "trpc deploy %s status is %s (Stable)，stop polling", args, record.Status)

	if record.Status == appmodeldeploy.StatusDeployed {
		handleDeploySucceeded(ctx, args, record)
	}

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
	return nil
}

// triggerTopologyRefresh 按部署记录的 ResourceKeys / LabelSelector 刷拓扑资源范围
// 由 Handle 在 goroutine 中调用，resourceKeys 为空则跳过；失败只打日志，不影响本 tick
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
		log.Errorf(ctx, "track env add app: registry is not initialized")
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

	// 查 workspace / env 后异步同步该环境告警策略；缺数据或查询失败只打日志并返回
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
	if reg.EnvStore == nil {
		log.Errorf(ctx, "env store is not initialized for alert sync")
		log.Warn(ctx, warnLogPrefix+"env is nil")
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
		args.WorkspaceID, args.AppID, env.Name, env.ID.Hex(), args.TrafficLaneName, record.Creator,
	)
	go alertstrategy.NewService(
		reg.AlertStrategyStore, reg.EnvStore, reg.AppStore, reg.ResourceSnapshotStore,
	).SyncStrategiesForAppInEnv(
		context.WithoutCancel(ctx), ws, args.AppID, env.ID, args.TrafficLaneName, record.Creator,
	)
}
