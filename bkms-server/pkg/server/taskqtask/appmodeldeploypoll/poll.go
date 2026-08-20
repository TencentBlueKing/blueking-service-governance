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
	"time"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// handle asynq 入口：store 未初始化则 ErrStopRetry
// auto deploy store 允许为 nil，普通部署 save 时跳过同步
func handle(ctx context.Context, args Args) error {
	reg := storereg.G()
	if reg == nil || reg.AppModelDeployRecordStore == nil {
		log.Errorf(ctx, "appmodel deploy poll stores not initialized, stop task: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "appmodel deploy poll stores not initialized")
	}
	return NewPoller(reg.AppModelDeployRecordStore, reg.BuildAutoDeployRecordStore).Handle(ctx, args)
}

// onExhausted tick 重试耗尽后兜底标 StatusPollingBroken，避免记录永久停在 deploying
// asynq 超时会先 cancel ctx，这里用 WithoutCancel 保证能读库落终态
func onExhausted(ctx context.Context, args Args, lastErr error) {
	ctx = context.WithoutCancel(ctx)
	log.Errorf(ctx, "appmodel deploy poll %s exhausted, try mark pollingBroken: %v", args, lastErr)

	reg := storereg.G()
	if reg == nil || reg.AppModelDeployRecordStore == nil {
		log.Errorf(ctx, "appmodel deploy poll stores not initialized, skip mark pollingBroken: %s", args)
		return
	}
	p := NewPoller(reg.AppModelDeployRecordStore, reg.BuildAutoDeployRecordStore)
	record, err := p.recordStore.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		log.Errorf(ctx, "get deploy record %s failed, skip mark pollingBroken: %v", args, err)
		return
	}
	// 已稳定 / 已卸载：终态可能已落库但副作用未跑完，补跑 onStable
	if record.Status.IsStable() || record.Status.IsUninstall() {
		if err = p.finishIfStable(ctx, record, args); err != nil {
			log.Errorf(ctx, "appmodel deploy poll %s exhausted but finish stable failed: %v", args, err)
		}
		return
	}
	message := "deploy status polling interrupted: task retries exhausted"
	if lastErr != nil {
		message = lastErr.Error()
	}
	if err = p.terminate(ctx, record, args, appmodeldeploy.StatusPollingBroken, message); err != nil {
		log.Errorf(ctx, "mark deploy %s pollingBroken failed: %v", args, err)
	}
}

// Poller 执行一次 AppModel 部署状态轮询 tick
type Poller struct {
	recordStore     appmodeldeploy.RecordStore
	autoDeployStore autodeploy.RecordStore
}

// NewPoller 注入部署轮询所需 store，供 asynq handler 与单测共用
func NewPoller(recordStore appmodeldeploy.RecordStore, autoDeployStore autodeploy.RecordStore) *Poller {
	return &Poller{recordStore: recordStore, autoDeployStore: autoDeployStore}
}

// Handle 执行一次部署状态轮询 tick
// 已稳定 / 已卸载则 finishIfStable 后返回；仍在跑则 ProcessIn 投下一 tick
// asynq MaxRetry 只约束本 tick 意外失败；轮询窗口与连续查状态失败分别由 pollingTimeout、FailureRetryRemain 截断
// 记录不存在 wrap ErrStopRetry，瞬时错误交回 asynq 重试
func (p *Poller) Handle(ctx context.Context, args Args) error {
	// store / 记录缺失无法自行恢复，停掉 asynq 重试
	if p.recordStore == nil {
		return errors.Wrap(taskq.ErrStopRetry, "appmodel deploy poll stores not initialized")
	}

	record, err := p.recordStore.Get(ctx, args.AppID, args.DeployID)
	if err != nil {
		if errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
			return errors.Wrapf(taskq.ErrStopRetry, "get deploy record: %v", err)
		}
		return errors.Wrap(err, "get deploy record")
	}
	// 已稳定 / 已卸载：不再查状态，仍走 finishIfStable 补齐副作用
	if record.Status.IsStable() || record.Status.IsUninstall() {
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
	state, err := appmodeldeploy.NewDeployStateGetter(record).Get(ctx)
	if err != nil {
		remain--
		switch {
		case windowExceeded:
			log.Warnf(ctx, "deploy %s polling window exceeded and fetch failed, mark pollingTimeout", args)
			return p.terminate(ctx, record, args, appmodeldeploy.StatusPollingTimeout, err.Error())
		case remain <= 0:
			log.Errorf(
				ctx, "stop polling deploy state %s: consecutive fetch failures exhausted budget=%d",
				args, totalFailureRetryCount,
			)
			return p.terminate(ctx, record, args, appmodeldeploy.StatusPollingBroken, err.Error())
		}
		log.Errorf(ctx, "fetch deploy %s status failed, remain=%d: %v", args, remain, err)
		return p.enqueueNext(ctx, args, remain)
	}
	// 查到状态即复位失败额度，额度约束的是连续失败
	remain = totalFailureRetryCount

	if state.Status != record.Status {
		record.Status = state.Status
		record.Message = state.Message
		if state.Status.IsStable() {
			record.EndedAt = time.Now()
		}
	}

	p.markCanceledIfSuperseded(ctx, record, args)

	// 窗口已过且仍未达终态：直接落超时终态，中间态不必再写一次库
	// 被更新部署取代的记录已在上一步标 canceled，属终态，不受此分支影响
	if windowExceeded && !record.Status.IsStable() && !record.Status.IsUninstall() {
		log.Warnf(ctx, "deploy %s polling window exceeded at status %s, mark pollingTimeout", args, record.Status)
		return p.terminate(ctx, record, args, appmodeldeploy.StatusPollingTimeout, "")
	}

	if record.Status != curStatus {
		if err = p.save(ctx, record, args); err != nil {
			// 终态落库失败交回 asynq 重试；中间态失败只打日志，下一 tick 会重写
			if record.Status.IsStable() {
				return errors.Wrap(err, "update appmodel deploy record to stable status")
			}
			log.Errorf(ctx, "failed to update appmodel deploy record: %v", err)
		}
	}
	if record.Status.IsStable() || record.Status.IsUninstall() {
		return p.finishIfStable(ctx, record, args)
	}
	return p.enqueueNext(ctx, args, remain)
}

// markCanceledIfSuperseded 同环境出现更新部署时把当前记录标取消
// 查最新记录失败只打日志，不影响本 tick 状态推进
func (p *Poller) markCanceledIfSuperseded(ctx context.Context, record *appmodeldeploy.Record, args Args) {
	latest, err := p.recordStore.GetLatest(ctx, args.AppID, args.EnvName, args.TrafficLaneName)
	if err != nil {
		log.Errorf(ctx, "failed to get latest deploy record: %v", err)
		return
	}
	if latest == nil || latest.ID.Hex() == args.DeployID {
		return
	}
	log.Warnf(ctx, "deploy %s status polling stopped by newer deployment %s", args, latest.ID.Hex())
	record.Status = appmodeldeploy.StatusCanceled
	record.Message = "deployment canceled: superseded by newer deployment"
	record.EndedAt = time.Now()
}

// enqueueNext 按配置间隔 ProcessIn 投递下一 tick
func (p *Poller) enqueueNext(ctx context.Context, args Args, remain int) error {
	args.FailureRetryRemain = remain
	interval := PollingInterval()
	log.Infof(ctx, "schedule next poll for deploy %s in %s remain=%d", args, interval, remain)
	if err := taskq.Enqueue(ctx, Task.NewTask(args), asynq.ProcessIn(interval)); err != nil {
		return errors.Wrap(err, "enqueue next appmodel deploy poll tick")
	}
	return nil
}

// terminate 把记录标为指定终态并落库；卸载直接返回，其余走 onStable
// 落库失败返回 error，避免副作用已触发而 DB 仍是中间态
func (p *Poller) terminate(
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
	if err := p.save(ctx, record, args); err != nil {
		return errors.Wrapf(err, "update appmodel deploy record to %s", status)
	}
	return p.finishIfStable(ctx, record, args)
}

// finishIfStable 稳定态收尾：卸载不发审计 / 指标，其余走 onStable
func (p *Poller) finishIfStable(ctx context.Context, record *appmodeldeploy.Record, args Args) error {
	if record.Status.IsUninstall() {
		return nil
	}
	return p.onStable(ctx, record, args)
}

// save 落部署记录，有 auto deploy store 时同步其状态
// 库中已卸载则改回内存 status 并返回，不写库；随后 finishIfStable 按卸载收尾
// 部署记录写失败才返回 error；auto deploy 同步失败只打日志
func (p *Poller) save(ctx context.Context, record *appmodeldeploy.Record, args Args) error {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveStatusTimeout)
	defer cancel()

	fromStatus := appmodeldeploy.Status("unknown")
	dbRecord, err := p.recordStore.Get(saveCtx, args.AppID, args.DeployID)
	if err != nil {
		log.Errorf(saveCtx, "failed to get appmodel deploy record: %v", err)
	} else if dbRecord.Status.IsUninstall() {
		log.Warnf(
			saveCtx, "appmodel deploy %s status is %s (from db), stop deploy status polling",
			args, dbRecord.Status,
		)
		record.Status = dbRecord.Status
		return nil
	} else {
		fromStatus = dbRecord.Status
	}

	log.Infof(
		saveCtx, "appmodel deploy %s status changed from %s to %s, message: %s",
		args, fromStatus, record.Status, record.Message,
	)
	if err = p.recordStore.Update(saveCtx, record); err != nil {
		return err
	}
	if p.autoDeployStore == nil {
		log.Warnf(saveCtx, "deploy %s auto deploy store is nil, skip sync", args)
		return nil
	}
	operator, err := autodeploy.NewOperator(p.autoDeployStore)
	if err != nil {
		log.Errorf(saveCtx, "failed to init build auto deploy operator: %v", err)
		return nil
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

// onStable 终态副作用失败只打日志，不改部署结果
func (p *Poller) onStable(ctx context.Context, record *appmodeldeploy.Record, args Args) error {
	metrics.DeployFinished(metrics.DeployKindAppModel, string(record.Status), record.StartedAt, time.Now())
	log.Infof(ctx, "appmodel deploy %s status is %s (Stable), stop polling", args, record.Status)

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
