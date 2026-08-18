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

package buildpoll

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// PollingInterval 按已运行时长返回下一 tick 延迟：10s → 15s → 30s → 1min → 5min
func PollingInterval(elapsed time.Duration) time.Duration {
	switch {
	case elapsed >= 3*time.Hour:
		return 300 * time.Second
	case elapsed >= 1*time.Hour:
		return 60 * time.Second
	case elapsed >= 30*time.Minute:
		return 30 * time.Second
	case elapsed >= 15*time.Minute:
		return 15 * time.Second
	default:
		return 10 * time.Second
	}
}

// handle asynq 入口：registry / 必要 store 缺失则打日志并 ErrStopRetry，否则交给 poller
func handle(ctx context.Context, args Args) error {
	reg := storereg.G()
	if reg == nil ||
		reg.BuildRecordStore == nil ||
		reg.BkCIPipelineStore == nil ||
		reg.BuildAutoDeployRecordStore == nil {
		log.Errorf(ctx, "build poll stores not initialized, stop task: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "build poll stores not initialized")
	}
	return newPoller(
		reg.BuildRecordStore,
		reg.BkCIPipelineStore,
		reg.BuildAutoDeployRecordStore,
	).runTick(ctx, args)
}

// onExhausted tick 重试耗尽（如 Redis 抖动导致连续投递失败）意味着轮询链已断，后续不会再有 tick
// 来推进状态，兜底把记录标为 StatusPollingBroken，避免构建记录永久停在 running
func onExhausted(ctx context.Context, args Args, lastErr error) {
	log.Errorf(ctx, "build poll %s exhausted, try mark pollingBroken: %v", args, lastErr)

	reg := storereg.G()
	if reg == nil || reg.BuildRecordStore == nil {
		log.Errorf(ctx, "build poll stores not initialized, skip mark pollingBroken: %s", args)
		return
	}
	p := newPoller(reg.BuildRecordStore, reg.BkCIPipelineStore, reg.BuildAutoDeployRecordStore)
	record, err := p.recordStore.Get(ctx, args.AppID, args.BuildID)
	if err != nil {
		log.Errorf(ctx, "get build record %s failed, skip mark pollingBroken: %v", args, err)
		return
	}
	// 已终态说明状态已由其他路径推进到位，无需兜底
	if record.Status.IsTerminated() {
		return
	}
	if err = p.terminate(ctx, record, args, build.StatusPollingBroken); err != nil {
		log.Errorf(ctx, "mark build %s pollingBroken failed: %v", args, err)
	}
}

// poller 执行一次构建状态轮询 tick
type poller struct {
	recordStore     build.RecordStore
	pipelineStore   bkci.PipelineStore
	autoDeployStore autodeploy.RecordStore
}

// newPoller 注入构建轮询所需 store，供 asynq handler 与单测共用
func newPoller(
	recordStore build.RecordStore,
	pipelineStore bkci.PipelineStore,
	autoDeployStore autodeploy.RecordStore,
) *poller {
	return &poller{
		recordStore:     recordStore,
		pipelineStore:   pipelineStore,
		autoDeployStore: autoDeployStore,
	}
}

// runTick 执行一次构建状态轮询 tick：读本地记录，必要时查蓝盾并落库。
// 记录已终态则直接返回；仍在跑则 ProcessIn 投递下一 tick（新任务，retry 从 0 计）。
// asynq MaxRetry(tickMaxRetry) 只约束本 tick 的意外失败（如 enqueue 失败），不约束轮询次数；
// 轮询窗口由 pollingTimeout 截断，查蓝盾失败次数由 FailureRetryRemain 截断。
// 仅记录 / 流水线不存在等不可恢复错误 wrap taskq.ErrStopRetry；瞬时错误交回 asynq 退避重试，
// 否则一次 DB 抖动就会断掉轮询链，让构建记录永久停在 running。
func (p *poller) runTick(ctx context.Context, args Args) error {
	// store / 记录缺失无法自行恢复，停掉 asynq 重试
	if p.recordStore == nil || p.pipelineStore == nil {
		return errors.Wrap(taskq.ErrStopRetry, "build poll stores not initialized")
	}

	record, err := p.recordStore.Get(ctx, args.AppID, args.BuildID)
	if err != nil {
		if errors.Is(err, build.ErrRecordNotFound) {
			return errors.Wrapf(taskq.ErrStopRetry, "get build record: %v", err)
		}
		return errors.Wrap(err, "get build record")
	}
	// 迟到或重复 tick：记录已终态则不再查蓝盾
	if record.Status.IsTerminated() {
		log.Infof(ctx, "build %s already terminated as %s, skip tick", args, record.Status)
		return nil
	}
	// 轮询窗口到点，与查蓝盾失败无关，直接标超时停掉
	if time.Since(record.StartedAt) >= pollingTimeout {
		log.Warnf(ctx, "build %s polling window exceeded, mark pollingTimeout", args)
		return p.terminate(ctx, record, args, build.StatusPollingTimeout)
	}

	user, err := auth.GetUser(ctx)
	if err != nil {
		return errors.Wrapf(taskq.ErrStopRetry, "get authed user: %v", err)
	}
	apiClient, err := bkciapi.New(user)
	if err != nil {
		return errors.Wrapf(taskq.ErrStopRetry, "create bkci api client: %v", err)
	}
	pipeline, err := p.pipelineStore.GetByWorkspaceAndType(ctx, args.WorkspaceID, args.PipelineType)
	if err != nil {
		if errors.Is(err, bkci.ErrPipelineNotFound) {
			return errors.Wrapf(
				taskq.ErrStopRetry, "get workspace %s type %s pipeline: %v",
				args.WorkspaceID, args.PipelineType, err,
			)
		}
		return errors.Wrapf(err, "get workspace %s type %s pipeline", args.WorkspaceID, args.PipelineType)
	}

	// 查蓝盾失败次数走业务计数，不走 asynq MaxRetry；首 tick 未带 remain 时补满
	remain := lo.Ternary(args.FailureRetryRemain > 0, args.FailureRetryRemain, totalFailureRetryCount)

	curStatus := record.Status
	if err = fetchAndUpdateBuildRecord(ctx, apiClient, pipeline, record, args.BuildID); err != nil {
		remain--
		if remain <= 0 {
			log.Errorf(ctx, "stop polling pipeline build %s after %d retries", args, totalFailureRetryCount)
			return p.terminate(ctx, record, args, build.StatusPollingBroken)
		}
		// 查询失败但额度未用尽：投下一 tick，本 tick 返回 nil，不扣 asynq 重试
		log.Errorf(ctx, "fetch build %s status failed, remain=%d: %v", args, remain, err)
		return p.enqueueNext(ctx, args, remain, record.StartedAt)
	}
	// 查到状态即视为蓝盾侧可达，失败额度复位，保证额度约束的是连续失败而非整轮累计
	remain = totalFailureRetryCount

	if record.Status != curStatus {
		log.Infof(ctx, "build %s status changed from %s to %s", args, curStatus, record.Status)
		if err = p.save(ctx, record, args); err != nil {
			// 终态未落库就触发自动部署 / 发指标会造成 DB 与副作用不一致，交回 asynq 重试本 tick；
			// 中间态落库失败不阻断轮询，下一 tick 会重新写
			if record.Status.IsTerminated() {
				return errors.Wrap(err, "update build record to terminated status")
			}
			log.Errorf(ctx, "failed to update build record: %v", err)
		}
	}
	if record.Status.IsTerminated() {
		return p.onTerminated(ctx, record, args)
	}
	// 仍在跑：ProcessIn 投新任务续跑，retry 从 0 计
	return p.enqueueNext(ctx, args, remain, record.StartedAt)
}

// enqueueNext 按已运行时长计算间隔，ProcessIn 投递下一 tick；新任务 retry 从 0 计
func (p *poller) enqueueNext(ctx context.Context, args Args, remain int, startedAt time.Time) error {
	args.FailureRetryRemain = remain
	interval := PollingInterval(time.Since(startedAt))
	log.Infof(ctx, "schedule next poll for build %s in %s remain=%d", args, interval, remain)
	if err := taskq.Enqueue(ctx, Task.NewTask(args), asynq.ProcessIn(interval)); err != nil {
		return errors.Wrap(err, "enqueue next build poll tick")
	}
	return nil
}

// terminate 把记录标为指定终态并落库，再走 onTerminated 副作用
// 落库失败返回 error 交回 asynq 重试，避免自动部署 / 指标已触发而 DB 仍是中间态
func (p *poller) terminate(ctx context.Context, record *build.Record, args Args, status build.Status) error {
	record.Status = status
	if record.EndedAt.IsZero() {
		record.EndedAt = time.Now()
	}
	if err := p.save(ctx, record, args); err != nil {
		return errors.Wrapf(err, "update build record to %s", status)
	}
	return p.onTerminated(ctx, record, args)
}

// save 落构建记录；启用自动部署时同步 auto deploy 记录
// 只有构建记录本身写失败才返回 error（调用方据此决定是否重试本 tick），
// auto deploy 记录属于附带同步，失败只打日志，避免重试时被终态早退分支跳过副作用
func (p *poller) save(ctx context.Context, record *build.Record, args Args) error {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveStatusTimeout)
	defer cancel()

	if err := p.recordStore.Update(saveCtx, record); err != nil {
		return err
	}
	if args.AutoDeploy == nil {
		return nil
	}
	// 生产路径 store 必注入；缺失只打日志，不让本 tick 失败
	if p.autoDeployStore == nil {
		log.Errorf(saveCtx, "build %s auto deploy enabled but store is nil, skip sync", args)
		return nil
	}
	operator, err := autodeploy.NewOperator(p.autoDeployStore)
	if err != nil {
		log.Errorf(saveCtx, "failed to init build auto deploy operator: %v", err)
		return nil
	}
	if err = syncBuildStatus(saveCtx, operator, args.AppID, args.BuildID, record); err != nil {
		log.Errorf(saveCtx, "failed to sync build %s status to auto deploy record: %v", args, err)
	}
	return nil
}

// onTerminated 终态副作用失败只打日志，不改构建结果，也不让本 tick 失败
func (p *poller) onTerminated(ctx context.Context, record *build.Record, args Args) error {
	metrics.BuildFinished(string(record.Status), record.StartedAt, time.Now())

	if record.Status == build.StatusSuccess && args.AutoDeploy != nil {
		if err := p.triggerDeploy(ctx, record, args); err != nil {
			log.Errorf(ctx, "handle build auto deploy after build success failed: %v", err)
		}
	}

	opResult := audit.ResultFailed
	if record.Status == build.StatusSuccess {
		opResult = audit.ResultSuccess
	}
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx), audit.OperationTypeBuild, audit.ResourceTypeApp, args.AppID,
		audit.WithResult(opResult), audit.WithWorkspaceID(args.WorkspaceID), audit.WithAppID(args.AppID),
	)

	if record.Status == build.StatusSuccess {
		imageTag := record.Params[pipelineparam.ImageTag]
		go func() {
			tCtx := context.WithoutCancel(ctx)
			if err := refreshSnapshots(tCtx, args.AppID, imageTag); err != nil {
				log.Errorf(tCtx, "trigger snapshot refresh after build for app %s failed: %v", args.AppID, err)
			}
		}()
	}

	log.Infof(ctx, "build %s status is %s (Terminated), stop polling", args, record.Status)
	return nil
}

// triggerDeploy 构建成功后触发自动部署；store 未初始化返回 error，由 onTerminated 吞掉
func (p *poller) triggerDeploy(ctx context.Context, record *build.Record, args Args) error {
	if p.autoDeployStore == nil {
		return errors.New("build auto deploy record store is not initialized")
	}
	operator, err := autodeploy.NewOperator(p.autoDeployStore)
	if err != nil {
		return err
	}
	return triggerDeployAfterBuild(ctx, operator, record, args)
}
