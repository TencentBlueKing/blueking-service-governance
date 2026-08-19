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

// handle asynq 入口：registry / 部署记录 store 缺失则打日志并 ErrStopRetry，否则交给 Poller
// BuildAutoDeployRecordStore 允许为 nil：普通部署没有自动部署记录，save 内部会降级跳过同步
func handle(ctx context.Context, args Args) error {
	reg := storereg.G()
	if reg == nil || reg.AppModelDeployRecordStore == nil {
		log.Errorf(ctx, "appmodel deploy poll stores not initialized, stop task: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "appmodel deploy poll stores not initialized")
	}
	return NewPoller(reg.AppModelDeployRecordStore, reg.BuildAutoDeployRecordStore).Handle(ctx, args)
}

// onExhausted tick 重试耗尽（如 Redis 抖动导致连续投递失败）意味着轮询链已断，后续不会再有 tick
// 来推进状态，兜底把记录标为 StatusPollingBroken，避免部署记录永久停在 deploying
// asynq 在 lease 过期 / 任务超时时会先 cancel ctx 再调 ErrorHandler，故这里用 WithoutCancel 保证能读库落终态
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
	// 已是稳定态 / 卸载态：终态可能已由本 tick 写入但副作用未跑完（save 成功、客户端报错后重试），补跑 onStable
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

// Handle 执行一次部署状态轮询 tick：读本地记录，必要时查集群状态并落库。
// 已稳定或已卸载则走 finishIfStable（补副作用）后返回；仍在跑则 ProcessIn 投递下一 tick（新任务，retry 从 0 计）。
// asynq MaxRetry(tickMaxRetry) 只约束本 tick 的意外失败（如 enqueue 失败），不约束轮询次数；
// 轮询窗口由 pollingTimeout 截断，查状态失败次数由 FailureRetryRemain 截断。
// 仅记录不存在等不可恢复错误 wrap taskq.ErrStopRetry；瞬时错误交回 asynq 退避重试，
// 否则一次 DB 抖动就会断掉轮询链，让部署记录永久停在 deploying。
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
	// 迟到或重复 tick：已稳定 / 已卸载则不再查状态；仍走 finishIfStable
	// asynq 至少一次投递下，save 成功但客户端报错 / 进程崩溃时要补齐 onStable
	// 副作用按至少一次设计，重复 tick 可能重复审计 / 指标；漏做比重复更糟
	if record.Status.IsStable() || record.Status.IsUninstall() {
		log.Infof(ctx, "deploy %s already %s, skip polling", args, record.Status)
		return p.finishIfStable(ctx, record, args)
	}
	// 轮询窗口到点，与查状态失败无关，直接标超时停掉
	if time.Since(record.StartedAt) >= pollingTimeout() {
		log.Warnf(ctx, "deploy %s polling window exceeded, mark pollingTimeout", args)
		return p.terminate(ctx, record, args, appmodeldeploy.StatusPollingTimeout, "")
	}

	// 拓扑刷新是重操作，整轮部署只在首个 tick 触发；标记随 enqueueNext 透传给后续 tick
	if !args.TopologyRefreshed {
		go triggerTopologyRefresh(context.WithoutCancel(ctx), args, record)
		args.TopologyRefreshed = true
	}

	// 查状态失败次数走业务计数，不走 asynq MaxRetry；首 tick 未带 remain 时补满
	remain := lo.Ternary(args.FailureRetryRemain > 0, args.FailureRetryRemain, totalFailureRetryCount)

	curStatus := record.Status
	state, err := appmodeldeploy.NewDeployStateGetter(record).Get(ctx)
	if err != nil {
		remain--
		if remain <= 0 {
			log.Errorf(
				ctx, "stop polling deploy state %s: consecutive fetch failures exhausted budget=%d",
				args, totalFailureRetryCount,
			)
			return p.terminate(ctx, record, args, appmodeldeploy.StatusPollingBroken, err.Error())
		}
		// 查询失败但额度未用尽：投下一 tick，本 tick 返回 nil，不扣 asynq 重试
		log.Errorf(ctx, "fetch deploy %s status failed, remain=%d: %v", args, remain, err)
		return p.enqueueNext(ctx, args, remain)
	}
	// 查到状态即视为集群侧可达，失败额度复位，保证额度约束的是连续失败而非整轮累计
	remain = totalFailureRetryCount

	if state.Status != record.Status {
		record.Status = state.Status
		record.Message = state.Message
		if state.Status.IsStable() {
			record.EndedAt = time.Now()
		}
	}

	p.markCanceledIfSuperseded(ctx, record, args)

	if record.Status != curStatus {
		if err = p.save(ctx, record, args); err != nil {
			// 终态未落库就发指标 / 审计会造成 DB 与审计不一致，交回 asynq 重试本 tick；
			// 中间态落库失败不阻断轮询，下一 tick 会重新写
			if record.Status.IsStable() {
				return errors.Wrap(err, "update appmodel deploy record to stable status")
			}
			log.Errorf(ctx, "failed to update appmodel deploy record: %v", err)
		}
	}
	if record.Status.IsStable() || record.Status.IsUninstall() {
		return p.finishIfStable(ctx, record, args)
	}
	// 仍在跑：ProcessIn 投新任务续跑，retry 从 0 计
	return p.enqueueNext(ctx, args, remain)
}

// markCanceledIfSuperseded 同环境出现更新的部署时把当前记录标取消，避免两路轮询互相覆盖
// 查最新记录失败不影响本 tick 的状态推进，只打日志
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

// enqueueNext 按配置间隔 ProcessIn 投递下一 tick；新任务 retry 从 0 计
func (p *Poller) enqueueNext(ctx context.Context, args Args, remain int) error {
	args.FailureRetryRemain = remain
	interval := PollingInterval()
	log.Infof(ctx, "schedule next poll for deploy %s in %s remain=%d", args, interval, remain)
	if err := taskq.Enqueue(ctx, Task.NewTask(args), asynq.ProcessIn(interval)); err != nil {
		return errors.Wrap(err, "enqueue next appmodel deploy poll tick")
	}
	return nil
}

// terminate 把记录标为指定终态并落库；卸载态直接返回，其余走 onStable 副作用
// 落库失败返回 error 交回 asynq 重试，避免指标 / 审计已记终态而 DB 仍是中间态
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
// 已稳定早退与 terminate 共用，保证「终态已在 DB、副作用未跑完」的重试仍能补齐
func (p *Poller) finishIfStable(ctx context.Context, record *appmodeldeploy.Record, args Args) error {
	if record.Status.IsUninstall() {
		return nil
	}
	return p.onStable(ctx, record, args)
}

// save 落部署记录。有 auto deploy store 时同步其状态
// 库中已是卸载态视为卸载抢占：把内存里的 record.Status 改回库中卸载态并返回 nil，不写库
// 调用方随后走 finishIfStable 时按卸载态收尾（不发审计 / 指标）
// 只有部署记录本身写失败才返回 error（调用方据此决定是否重试本 tick），
// auto deploy 记录属于附带同步，失败只打日志，避免重试时被终态早退分支跳过副作用
func (p *Poller) save(ctx context.Context, record *appmodeldeploy.Record, args Args) error {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveStatusTimeout)
	defer cancel()

	// 库中状态即本次变更的起点，取来既用于卸载抢占检查，也用于补全日志的 from 字段
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
	// 普通部署没有 auto deploy 记录；store 未注入时只打日志，不让本 tick 失败
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

// onStable 终态副作用失败只打日志，不改部署结果，也不让本 tick 失败
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
