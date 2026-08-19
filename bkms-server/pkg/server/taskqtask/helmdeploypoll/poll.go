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

package helmdeploypoll

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	helmrelease "helm.sh/helm/v3/pkg/release"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// handle asynq 入口：registry / 部署记录 store 缺失则打日志并 ErrStopRetry，否则交给 Manager
func handle(ctx context.Context, args Args) error {
	reg := storereg.G()
	if reg == nil || reg.HelmDeployRecordStore == nil {
		log.Errorf(ctx, "helm deploy poll stores not initialized, stop task: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "helm deploy poll stores not initialized")
	}
	return NewManager(reg.HelmDeployRecordStore).Handle(ctx, args)
}

// onExhausted tick 重试耗尽（如 Redis 抖动导致连续投递失败）意味着轮询链已断，后续不会再有 tick
// 来推进状态，兜底把记录标为 StatusPollingBroken 并释放部署锁，避免记录永久停在 pending 且锁泄漏
// asynq 在 lease 过期 / 任务超时时会先 cancel ctx 再调 ErrorHandler，故这里用 WithoutCancel 保证能读库落终态
func onExhausted(ctx context.Context, args Args, lastErr error) {
	ctx = context.WithoutCancel(ctx)
	log.Errorf(ctx, "helm deploy poll %s exhausted, try mark pollingBroken: %v", args, lastErr)

	reg := storereg.G()
	if reg == nil || reg.HelmDeployRecordStore == nil {
		log.Errorf(ctx, "helm deploy poll stores not initialized, skip mark pollingBroken: %s", args)
		return
	}
	m := NewManager(reg.HelmDeployRecordStore)
	record, err := m.recordStore.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		log.Errorf(ctx, "get deploy record %s failed, skip mark pollingBroken: %v", args, err)
		return
	}
	// 已是稳定态：终态可能已由本 tick 写入但副作用未跑完（save 成功、客户端报错后重试），补跑 onStable / 放锁
	if helm.IsStable(record.Status) {
		if err = m.finishIfStable(ctx, record, args); err != nil {
			log.Errorf(ctx, "helm deploy poll %s exhausted but finish stable failed: %v", args, err)
		}
		return
	}
	message := "helm deploy status polling interrupted: task retries exhausted"
	if lastErr != nil {
		message = lastErr.Error()
	}
	if err = m.terminate(ctx, record, args, helm.StatusPollingBroken, message); err != nil {
		log.Errorf(ctx, "mark deploy %s pollingBroken failed: %v", args, err)
	}
}

// Manager 执行一次 Helm 部署状态轮询 tick
type Manager struct {
	recordStore helmdeploy.RecordStore
}

// NewManager 注入部署轮询所需 store，供 asynq handler 与单测共用
func NewManager(recordStore helmdeploy.RecordStore) *Manager {
	return &Manager{recordStore: recordStore}
}

// Handle 执行一次部署状态轮询 tick：读本地记录，必要时查 Release 并落库
// 已稳定则走 finishIfStable（补副作用 / 放锁）后返回；仍在跑则 ProcessIn 投递下一 tick（新任务，retry 从 0 计）
// asynq MaxRetry(tickMaxRetry) 只约束本 tick 的意外失败（如 enqueue 失败），不约束轮询次数
// 轮询窗口由 pollingTimeout 截断，查状态失败次数由 FailureRetryRemain 截断
// 仅记录不存在等不可恢复错误 wrap taskq.ErrStopRetry；瞬时错误交回 asynq 退避重试，
// 否则一次 DB 抖动就会断掉轮询链，让部署记录永久停在 pending 且锁泄漏
func (m *Manager) Handle(ctx context.Context, args Args) error {
	if m.recordStore == nil {
		return errors.Wrap(taskq.ErrStopRetry, "helm deploy poll stores not initialized")
	}

	record, err := m.recordStore.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		if errors.Is(err, helmdeploy.ErrRecordNotFound) {
			return errors.Wrapf(taskq.ErrStopRetry, "get deploy record: %v", err)
		}
		return errors.Wrap(err, "get deploy record")
	}
	// 迟到或重复 tick：已稳定则不再查 Release；仍走 finishIfStable，
	// 覆盖「终态已落库但客户端报错 / 进程崩溃，onStable 尚未执行」的重试路径
	if helm.IsStable(record.Status) {
		log.Infof(ctx, "deploy %s already %s, skip polling", args, record.Status)
		return m.finishIfStable(ctx, record, args)
	}
	if time.Since(record.StartedAt) >= pollingTimeout() {
		log.Warnf(ctx, "deploy %s polling window exceeded, mark pollingTimeout", args)
		return m.terminate(ctx, record, args, helm.StatusPollingTimeout, "")
	}

	// 拓扑刷新是重操作，整轮部署只在首个 tick 触发；标记随 enqueueNext 透传给后续 tick
	if !args.TopologyRefreshed {
		go triggerTopologyRefresh(context.WithoutCancel(ctx), args, record)
		args.TopologyRefreshed = true
	}

	// 查状态失败次数走业务计数，不走 asynq MaxRetry；首 tick 未带 remain 时补满
	remain := lo.Ternary(args.FailureRetryRemain > 0, args.FailureRetryRemain, totalFailureRetryCount)

	curStatus := record.Status
	release, err := fetchReleaseStatus(ctx, record)
	if err != nil {
		remain--
		if remain <= 0 {
			log.Errorf(ctx, "stop polling release %s after %d retries", args, totalFailureRetryCount)
			return m.terminate(ctx, record, args, helm.StatusPollingBroken, err.Error())
		}
		log.Errorf(ctx, "fetch deploy %s status failed, remain=%d: %v", args, remain, err)
		return m.enqueueNext(ctx, args, remain)
	}
	// 查到状态即视为集群侧可达，失败额度复位，保证额度约束的是连续失败而非整轮累计
	remain = totalFailureRetryCount

	deployStatus := release.DeployResult.Status
	if deployStatus != record.Status {
		record.Status = deployStatus
		record.Revision = release.Version
		record.Message = release.DeployResult.Description
		if helm.IsStable(deployStatus) {
			record.EndedAt = time.Now()
		}
	}

	if record.Status != curStatus {
		if err = m.save(ctx, record, args); err != nil {
			// 终态未落库就发指标 / 审计 / 放锁会造成 DB 与副作用不一致，交回 asynq 重试本 tick；
			// 中间态落库失败不阻断轮询，下一 tick 会重新写
			if helm.IsStable(record.Status) {
				return errors.Wrap(err, "update helm deploy record to stable status")
			}
			log.Errorf(ctx, "failed to update helm deploy record: %v", err)
		}
	}
	if helm.IsStable(record.Status) {
		return m.finishIfStable(ctx, record, args)
	}
	return m.enqueueNext(ctx, args, remain)
}

// enqueueNext 按配置间隔 ProcessIn 投递下一 tick；新任务 retry 从 0 计
func (m *Manager) enqueueNext(ctx context.Context, args Args, remain int) error {
	args.FailureRetryRemain = remain
	interval := PollingInterval()
	log.Infof(ctx, "schedule next poll for deploy %s in %s remain=%d", args, interval, remain)
	if err := taskq.Enqueue(ctx, Task.NewTask(args), asynq.ProcessIn(interval)); err != nil {
		return errors.Wrap(err, "enqueue next helm deploy poll tick")
	}
	return nil
}

// terminate 把记录标为指定终态并落库；卸载态只放锁，其余走 onStable
// 落库失败返回 error 交回 asynq 重试，避免指标 / 审计 / 放锁已触发而 DB 仍是中间态
func (m *Manager) terminate(
	ctx context.Context,
	record *helmdeploy.Record,
	args Args,
	status helmrelease.Status,
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
		return errors.Wrapf(err, "update helm deploy record to %s", status)
	}
	return m.finishIfStable(ctx, record, args)
}

// finishIfStable 稳定态收尾：卸载只放锁，其余走 onStable（指标 / 审计 / 成功后置 / 放锁）
// 已稳定早退与 terminate 共用，保证「终态已在 DB、副作用未跑完」的重试仍能补齐
func (m *Manager) finishIfStable(ctx context.Context, record *helmdeploy.Record, args Args) error {
	if record.Status == helm.StatusUninstalled {
		m.releaseLockIfLatest(ctx, args)
		return nil
	}
	return m.onStable(ctx, record, args)
}

// save 落部署记录；库中已是已卸载则不再覆盖
func (m *Manager) save(ctx context.Context, record *helmdeploy.Record, args Args) error {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveStatusTimeout)
	defer cancel()

	// 库中状态即本次变更的起点，取来既用于卸载抢占检查，也用于补全日志的 from 字段
	fromStatus := helm.StatusUnknown
	dbRecord, err := m.recordStore.Get(saveCtx, args.AppID, args.DeployID)
	if err != nil {
		log.Errorf(saveCtx, "failed to get helm deploy record: %v", err)
	} else if dbRecord.Status == helm.StatusUninstalled {
		log.Warnf(
			saveCtx, "helm deploy %s status is %s (from db), stop deploy status polling",
			args, dbRecord.Status,
		)
		record.Status = dbRecord.Status
		return nil
	} else {
		fromStatus = dbRecord.Status
	}

	log.Infof(
		saveCtx, "helm deploy %s status changed from %s to %s, message: %s",
		args, fromStatus, record.Status, record.Message,
	)
	return m.recordStore.Update(saveCtx, record)
}

// onStable 终态副作用失败只打日志，不改部署结果，也不让本 tick 失败
func (m *Manager) onStable(ctx context.Context, record *helmdeploy.Record, args Args) error {
	metrics.DeployFinished(metrics.DeployKindHelm, string(record.Status), record.StartedAt, time.Now())
	log.Infof(ctx, "helm deploy %s status is %s (Stable), stop polling and release lock", args, record.Status)

	if record.Status == helm.StatusDeployed {
		handleDeploySucceeded(ctx, args, record)
	}

	opResult := lo.Ternary(
		record.Status == helm.StatusDeployed,
		audit.ResultSuccess,
		audit.ResultFailed,
	)
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeDeploy, audit.ResourceTypeApp, args.AppID,
		audit.WithResult(opResult), audit.WithWorkspaceID(args.WorkspaceID),
		audit.WithAppID(args.AppID), audit.WithEnvName(args.EnvName),
	)
	m.releaseLockIfLatest(ctx, args)
	return nil
}

// releaseLockIfLatest 仅当 latest 仍是本 deployID 时释放锁，避免 Del 误放更新部署的锁
func (m *Manager) releaseLockIfLatest(ctx context.Context, args Args) {
	latest, err := m.recordStore.GetLatest(ctx, args.AppID, args.EnvName, args.TrafficLaneName)
	if err != nil {
		log.Errorf(ctx, "skip release helm deploy lock for %s: get latest: %v", args, err)
		return
	}
	if latest == nil || latest.ID.Hex() != args.DeployID {
		log.Infof(ctx, "skip release helm deploy lock for %s: latest is no longer this record", args)
		return
	}
	releaseDeployLock(ctx, args)
}

// releaseDeployLock 释放同应用+环境+泳道的部署锁；抽成函数便于单测 mock
func releaseDeployLock(ctx context.Context, args Args) {
	helmdeploy.NewDeployLock(args.AppID, args.EnvName, args.TrafficLaneName).Release(ctx)
}

// fetchReleaseStatus 初始化 Helm action 并查询一次 Release 状态；抽成函数便于单测 mock
func fetchReleaseStatus(ctx context.Context, record *helmdeploy.Record) (*helm.Release, error) {
	debugLog := helm.NewHelmDebugLogger(ctx, record.ReleaseName, "polling-status")
	cfg, err := helm.NewActionConfiguration(record.ClusterID, record.Namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init action configuration for polling %s", record.ReleaseName)
	}
	return helm.GetReleaseStatus(cfg, record.ReleaseName)
}
