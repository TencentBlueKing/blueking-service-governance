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

// handle asynq 入口：store 未初始化则 ErrStopRetry
func handle(ctx context.Context, args Args) error {
	reg := storereg.G()
	if reg == nil || reg.HelmDeployRecordStore == nil {
		log.Errorf(ctx, "helm deploy poll stores not initialized, stop task: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "helm deploy poll stores not initialized")
	}
	return NewPoller(reg.HelmDeployRecordStore).Handle(ctx, args)
}

// onExhausted tick 重试耗尽后兜底标 StatusPollingBroken 并放锁，避免记录永久停在 pending
// asynq 超时会先 cancel ctx，这里用 WithoutCancel 保证能读库落终态
func onExhausted(ctx context.Context, args Args, lastErr error) {
	ctx = context.WithoutCancel(ctx)
	log.Errorf(ctx, "helm deploy poll %s exhausted, try mark pollingBroken: %v", args, lastErr)

	reg := storereg.G()
	if reg == nil || reg.HelmDeployRecordStore == nil {
		log.Errorf(ctx, "helm deploy poll stores not initialized, skip mark pollingBroken: %s", args)
		return
	}
	p := NewPoller(reg.HelmDeployRecordStore)
	record, err := p.recordStore.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		log.Errorf(ctx, "get deploy record %s failed, skip mark pollingBroken: %v", args, err)
		return
	}
	// 已稳定：终态可能已落库但副作用未跑完，补跑 onStable / 放锁
	if helm.IsStable(record.Status) {
		if err = p.finishIfStable(ctx, record, args); err != nil {
			log.Errorf(ctx, "helm deploy poll %s exhausted but finish stable failed: %v", args, err)
		}
		return
	}
	message := "helm deploy status polling interrupted: task retries exhausted"
	if lastErr != nil {
		message = lastErr.Error()
	}
	if err = p.terminate(ctx, record, args, helm.StatusPollingBroken, message); err != nil {
		log.Errorf(ctx, "mark deploy %s pollingBroken failed: %v", args, err)
	}
}

// Poller 执行一次 Helm 部署状态轮询 tick
type Poller struct {
	recordStore helmdeploy.RecordStore
}

// NewPoller 注入部署轮询所需 store，供 asynq handler 与单测共用
func NewPoller(recordStore helmdeploy.RecordStore) *Poller {
	return &Poller{recordStore: recordStore}
}

// Handle 执行一次部署状态轮询 tick
// 已稳定则 finishIfStable 后返回；仍在跑则 ProcessIn 投下一 tick
// asynq MaxRetry 只约束本 tick 意外失败；轮询窗口与连续查状态失败分别由 pollingTimeout、FailureRetryRemain 截断
// 记录不存在 wrap ErrStopRetry，瞬时错误交回 asynq 重试
func (p *Poller) Handle(ctx context.Context, args Args) error {
	if p.recordStore == nil {
		return errors.Wrap(taskq.ErrStopRetry, "helm deploy poll stores not initialized")
	}

	record, err := p.recordStore.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		if errors.Is(err, helmdeploy.ErrRecordNotFound) {
			return errors.Wrapf(taskq.ErrStopRetry, "get deploy record: %v", err)
		}
		return errors.Wrap(err, "get deploy record")
	}
	// 已稳定：不再查 Release，仍走 finishIfStable 补齐副作用 / 放锁
	if helm.IsStable(record.Status) {
		log.Infof(ctx, "deploy %s already %s, skip polling", args, record.Status)
		return p.finishIfStable(ctx, record, args)
	}

	// 拓扑刷新整轮只做一次，标记随 enqueueNext 透传
	if !args.TopologyRefreshed {
		go triggerTopologyRefresh(context.WithoutCancel(ctx), args, record)
		args.TopologyRefreshed = true
	}

	// 查状态失败走业务计数；首 tick 未带 remain 时补满
	remain := lo.Ternary(args.FailureRetryRemain > 0, args.FailureRetryRemain, totalFailureRetryCount)
	// 窗口按 StartedAt 计算，worker 积压时本 tick 可能一上来就在窗口外，
	// 故先查一次真实状态再据此收尾，避免把已经成功的部署误判为超时
	windowExceeded := time.Since(record.StartedAt) >= pollingTimeout()

	curStatus := record.Status
	release, err := fetchReleaseStatus(ctx, record)
	if err != nil {
		remain--
		return p.onFetchFailed(ctx, record, args, remain, windowExceeded, err)
	}
	// 查到状态即复位失败额度，额度约束的是连续失败
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

	// 窗口已过且仍未达终态：直接落超时终态，中间态不必再写一次库
	// message 沿用刚观测到的 Release description，便于回溯超时前停在哪一步
	if windowExceeded && !helm.IsStable(record.Status) {
		log.Warnf(ctx, "deploy %s polling window exceeded at status %s, mark pollingTimeout", args, record.Status)
		return p.terminate(ctx, record, args, helm.StatusPollingTimeout, "")
	}

	if record.Status != curStatus {
		if err = p.save(ctx, record, args); err != nil {
			// 终态落库失败交回 asynq 重试；中间态失败只打日志，下一 tick 会重写
			if helm.IsStable(record.Status) {
				return errors.Wrap(err, "update helm deploy record to stable status")
			}
			log.Errorf(ctx, "failed to update helm deploy record: %v", err)
		}
	}
	if helm.IsStable(record.Status) {
		return p.finishIfStable(ctx, record, args)
	}
	return p.enqueueNext(ctx, args, remain)
}

// onFetchFailed 查状态失败后决定去向：窗口已过落超时终态，失败额度耗尽落中断终态，
// 否则按剩余额度投下一 tick。两种终态都带上本次错误，便于回溯中断原因
func (p *Poller) onFetchFailed(
	ctx context.Context,
	record *helmdeploy.Record,
	args Args,
	remain int,
	windowExceeded bool,
	fetchErr error,
) error {
	switch {
	case windowExceeded:
		log.Warnf(ctx, "deploy %s polling window exceeded and fetch failed, mark pollingTimeout", args)
		return p.terminate(ctx, record, args, helm.StatusPollingTimeout, fetchErr.Error())
	case remain <= 0:
		log.Errorf(
			ctx, "stop polling release %s: consecutive fetch failures exhausted budget=%d",
			args, totalFailureRetryCount,
		)
		return p.terminate(ctx, record, args, helm.StatusPollingBroken, fetchErr.Error())
	}
	log.Errorf(ctx, "fetch deploy %s status failed, remain=%d: %v", args, remain, fetchErr)
	return p.enqueueNext(ctx, args, remain)
}

// enqueueNext 按配置间隔 ProcessIn 投递下一 tick
func (p *Poller) enqueueNext(ctx context.Context, args Args, remain int) error {
	args.FailureRetryRemain = remain
	interval := PollingInterval()
	log.Infof(ctx, "schedule next poll for deploy %s in %s remain=%d", args, interval, remain)
	if err := taskq.Enqueue(ctx, Task.NewTask(args), asynq.ProcessIn(interval)); err != nil {
		return errors.Wrap(err, "enqueue next helm deploy poll tick")
	}
	return nil
}

// terminate 把记录标为指定终态并落库；卸载只放锁，其余走 onStable
// 落库失败返回 error，避免副作用已触发而 DB 仍是中间态
func (p *Poller) terminate(
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
	if err := p.save(ctx, record, args); err != nil {
		return errors.Wrapf(err, "update helm deploy record to %s", status)
	}
	return p.finishIfStable(ctx, record, args)
}

// finishIfStable 稳定态收尾：卸载只放锁，其余走 onStable
func (p *Poller) finishIfStable(ctx context.Context, record *helmdeploy.Record, args Args) error {
	if record.Status == helm.StatusUninstalled {
		p.releaseLockIfLatest(ctx, args)
		return nil
	}
	return p.onStable(ctx, record, args)
}

// save 落部署记录
// 库中已卸载则改回内存 status 并返回，不写库；随后 finishIfStable 按卸载收尾
func (p *Poller) save(ctx context.Context, record *helmdeploy.Record, args Args) error {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveStatusTimeout)
	defer cancel()

	fromStatus := helm.StatusUnknown
	dbRecord, err := p.recordStore.Get(saveCtx, args.AppID, args.DeployID)
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
	return p.recordStore.Update(saveCtx, record)
}

// onStable 终态副作用失败只打日志，不改部署结果
func (p *Poller) onStable(ctx context.Context, record *helmdeploy.Record, args Args) error {
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
	p.releaseLockIfLatest(ctx, args)
	return nil
}

// releaseLockIfLatest 仅当 latest 仍是本 deployID 时放锁，避免误放更新部署的锁
func (p *Poller) releaseLockIfLatest(ctx context.Context, args Args) {
	latest, err := p.recordStore.GetLatest(ctx, args.AppID, args.EnvName, args.TrafficLaneName)
	if err != nil {
		log.Errorf(ctx, "skip release helm deploy lock for %s: get latest: %v", args, err)
		return
	}
	if latest != nil && latest.ID.Hex() == args.DeployID {
		releaseDeployLock(ctx, args)
		return
	}
	log.Infof(ctx, "skip release helm deploy lock for %s: latest is no longer this record", args)
}

// releaseDeployLock 释放同应用+环境+泳道的部署锁
func releaseDeployLock(ctx context.Context, args Args) {
	helmdeploy.NewDeployLock(args.AppID, args.EnvName, args.TrafficLaneName).Release(ctx)
}

// fetchReleaseStatus 初始化 Helm action 并查询一次 Release 状态
func fetchReleaseStatus(ctx context.Context, record *helmdeploy.Record) (*helm.Release, error) {
	debugLog := helm.NewHelmDebugLogger(ctx, record.ReleaseName, "polling-status")
	cfg, err := helm.NewActionConfiguration(record.ClusterID, record.Namespace, debugLog)
	if err != nil {
		return nil, errors.Wrapf(err, "init action configuration for polling %s", record.ReleaseName)
	}
	return helm.GetReleaseStatus(cfg, record.ReleaseName)
}
