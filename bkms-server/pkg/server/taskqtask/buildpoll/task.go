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
	"fmt"
	"time"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelinevar"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	appmodeldeploysvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/appmodeldeploypoll"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

const (
	// name asynq 任务类型名
	name = "taskq.pollingBuildStatus"
	// tickMaxRetry 单次 tick 意外失败的 asynq 重试上限，不含轮询续跑
	tickMaxRetry = 10
	// totalFailureRetryCount 查蓝盾连续失败次数上限，耗尽后标 StatusPollingBroken
	totalFailureRetryCount = 10
	// saveStatusTimeout 状态落库的独立超时，避免 handler ctx 取消导致写不进去
	saveStatusTimeout = 10 * time.Second
	// pollingTimeout 从 record.StartedAt 起算的轮询窗口，超时标 StatusPollingTimeout
	pollingTimeout = 24 * time.Hour
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

// AutoDeployArgs 构建成功后自动部署参数
type AutoDeployArgs struct {
	EnvName         string `json:"envName"`
	TrafficLaneName string `json:"trafficLaneName"`
	Replicas        int32  `json:"replicas"`
}

// Args 镜像构建状态轮询的业务参数，不含用户身份
type Args struct {
	WorkspaceID        string          `json:"workspaceID"`
	PipelineType       string          `json:"pipelineType"`
	AppID              string          `json:"appID"`
	BuildID            string          `json:"buildID"`
	AutoDeploy         *AutoDeployArgs `json:"autoDeploy,omitempty"`
	FailureRetryRemain int             `json:"failureRetryRemain,omitempty"`
}

// String 输出轮询身份、自动部署目标与剩余失败次数，便于日志对齐同一构建的连续 tick
func (args Args) String() string {
	autoDeploy := "off"
	if args.AutoDeploy != nil {
		autoDeploy = fmt.Sprintf("%s/%s", args.AutoDeploy.EnvName, args.AutoDeploy.TrafficLaneName)
	}
	return fmt.Sprintf(
		"<workspace: %s, pipelineType: %s, appID: %s, buildID: %s, autoDeploy: %s, remain: %d>",
		args.WorkspaceID, args.PipelineType, args.AppID, args.BuildID, autoDeploy, args.FailureRetryRemain,
	)
}

// Task 构建状态轮询任务；init 赋值避免与 enqueueNext 引用形成包初始化环
var Task *taskq.TaskType[Args]

func init() {
	Task = taskq.NewTaskType[Args](name, handle, asynq.MaxRetry(tickMaxRetry))
}

// handle asynq 入口：registry / 必要 store 缺失则打日志并 ErrStopRetry，否则交给 Manager
func handle(ctx context.Context, args Args) error {
	reg := storereg.G()
	if reg == nil ||
		reg.BuildRecordStore == nil ||
		reg.BkCIPipelineStore == nil ||
		reg.BuildAutoDeployRecordStore == nil {
		log.Errorf(ctx, "build poll stores not initialized, stop task: %s", args)
		return errors.Wrap(taskq.ErrStopRetry, "build poll stores not initialized")
	}
	return NewManager(
		reg.BuildRecordStore,
		reg.BkCIPipelineStore,
		reg.BuildAutoDeployRecordStore,
	).Handle(ctx, args)
}

// Manager 执行一次构建状态轮询 tick
type Manager struct {
	recordStore     build.RecordStore
	pipelineStore   bkci.PipelineStore
	autoDeployStore autodeploy.RecordStore
}

// NewManager 注入构建轮询所需 store，供 asynq handler 与单测共用
func NewManager(
	recordStore build.RecordStore,
	pipelineStore bkci.PipelineStore,
	autoDeployStore autodeploy.RecordStore,
) *Manager {
	return &Manager{
		recordStore:     recordStore,
		pipelineStore:   pipelineStore,
		autoDeployStore: autoDeployStore,
	}
}

// Handle 执行一次构建状态轮询 tick：读本地记录，必要时查蓝盾并落库。
// 记录已终态则直接返回；仍在跑则 ProcessIn 投递下一 tick（新任务，retry 从 0 计）。
// asynq MaxRetry(tickMaxRetry) 只约束本 tick 的意外失败（如 enqueue 失败），不约束轮询次数；
// 轮询窗口由 pollingTimeout 截断，查蓝盾失败次数由 FailureRetryRemain 截断。
// 不可恢复错误 wrap taskq.ErrStopRetry，避免 asynq 空转重试。
func (m *Manager) Handle(ctx context.Context, args Args) error {
	// store / 记录缺失无法自行恢复，停掉 asynq 重试
	if m.recordStore == nil || m.pipelineStore == nil {
		return errors.Wrap(taskq.ErrStopRetry, "build poll stores not initialized")
	}

	record, err := m.recordStore.Get(ctx, args.AppID, args.BuildID)
	if err != nil {
		return errors.Wrapf(taskq.ErrStopRetry, "get build record: %v", err)
	}
	// 迟到或重复 tick：记录已终态则不再查蓝盾
	if record.Status.IsTerminated() {
		log.Infof(ctx, "build %s already terminated as %s, skip tick", args, record.Status)
		return nil
	}
	// 轮询窗口到点，与查蓝盾失败无关，直接标超时停掉
	if time.Since(record.StartedAt) >= pollingTimeout {
		log.Warnf(ctx, "build %s polling window exceeded, mark pollingTimeout", args)
		return m.terminate(ctx, record, args, build.StatusPollingTimeout)
	}

	user, err := auth.GetUser(ctx)
	if err != nil {
		return errors.Wrapf(taskq.ErrStopRetry, "get authed user: %v", err)
	}
	apiClient, err := bkciapi.New(user)
	if err != nil {
		return errors.Wrapf(taskq.ErrStopRetry, "create bkci api client: %v", err)
	}
	pipeline, err := m.pipelineStore.GetByWorkspaceAndType(ctx, args.WorkspaceID, args.PipelineType)
	if err != nil {
		return errors.Wrapf(
			taskq.ErrStopRetry, "get workspace %s type %s pipeline: %v",
			args.WorkspaceID, args.PipelineType, err,
		)
	}

	// 查蓝盾失败次数走业务计数，不走 asynq MaxRetry；首 tick 未带 remain 时补满
	remain := args.FailureRetryRemain
	if remain <= 0 {
		remain = totalFailureRetryCount
	}

	curStatus := record.Status
	if err = fetchAndUpdateBuildRecord(ctx, apiClient, pipeline, record, args.BuildID); err != nil {
		remain--
		if remain <= 0 {
			log.Errorf(ctx, "stop polling pipeline build %s after %d retries", args, totalFailureRetryCount)
			return m.terminate(ctx, record, args, build.StatusPollingBroken)
		}
		// 查询失败但额度未用尽：投下一 tick，本 tick 返回 nil，不扣 asynq 重试
		log.Errorf(ctx, "fetch build %s status failed, remain=%d: %v", args, remain, err)
		return m.enqueueNext(ctx, args, remain, record.StartedAt)
	}

	// 落库失败只打日志，不让本 tick 失败，避免 asynq 重试把状态写乱
	if record.Status != curStatus {
		log.Infof(ctx, "build %s status changed from %s to %s", args, curStatus, record.Status)
		if err = m.save(ctx, record, args); err != nil {
			log.Errorf(ctx, "failed to update build record: %v", err)
		}
	}
	if record.Status.IsTerminated() {
		return m.onTerminated(ctx, record, args)
	}
	// 仍在跑：ProcessIn 投新任务续跑，retry 从 0 计
	return m.enqueueNext(ctx, args, remain, record.StartedAt)
}

// enqueueNext 按已运行时长计算间隔，ProcessIn 投递下一 tick；新任务 retry 从 0 计
func (m *Manager) enqueueNext(ctx context.Context, args Args, remain int, startedAt time.Time) error {
	args.FailureRetryRemain = remain
	interval := PollingInterval(time.Since(startedAt))
	log.Infof(ctx, "schedule next poll for build %s in %s remain=%d", args, interval, remain)
	if err := taskq.Enqueue(ctx, Task.NewTask(args), asynq.ProcessIn(interval)); err != nil {
		return errors.Wrap(err, "enqueue next build poll tick")
	}
	return nil
}

// terminate 把记录标为指定终态并落库，再走 onTerminated 副作用
func (m *Manager) terminate(ctx context.Context, record *build.Record, args Args, status build.Status) error {
	record.Status = status
	if record.EndedAt.IsZero() {
		record.EndedAt = time.Now()
	}
	if err := m.save(ctx, record, args); err != nil {
		log.Errorf(ctx, "failed to update build record: %v", err)
	}
	return m.onTerminated(ctx, record, args)
}

// save 落构建记录；启用自动部署时同步 auto deploy 记录
func (m *Manager) save(ctx context.Context, record *build.Record, args Args) error {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveStatusTimeout)
	defer cancel()

	if err := m.recordStore.Update(saveCtx, record); err != nil {
		return err
	}
	if args.AutoDeploy == nil {
		return nil
	}
	// 生产路径 store 必注入；缺失只打日志，不让本 tick 失败
	if m.autoDeployStore == nil {
		log.Errorf(saveCtx, "build %s auto deploy enabled but store is nil, skip sync", args)
		return nil
	}
	operator, err := autodeploy.NewOperator(m.autoDeployStore)
	if err != nil {
		return err
	}
	return syncBuildStatus(saveCtx, operator, args.AppID, args.BuildID, record)
}

// onTerminated 终态副作用失败只打日志，不改构建结果，也不让本 tick 失败
func (m *Manager) onTerminated(ctx context.Context, record *build.Record, args Args) error {
	metrics.BuildFinished(string(record.Status), record.StartedAt, time.Now())

	if record.Status == build.StatusSuccess && args.AutoDeploy != nil {
		if err := m.triggerDeploy(ctx, record, args); err != nil {
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
func (m *Manager) triggerDeploy(ctx context.Context, record *build.Record, args Args) error {
	if m.autoDeployStore == nil {
		return errors.New("build auto deploy record store is not initialized")
	}
	operator, err := autodeploy.NewOperator(m.autoDeployStore)
	if err != nil {
		return err
	}
	return triggerDeployAfterBuild(ctx, operator, record, args)
}

// fetchAndUpdateBuildRecord 查一次蓝盾并把状态写回 record。
// 仅状态变化时更新 Extras；结束时间优先用蓝盾 BuildEndTime，缺失则用 now。
// 变量缺失时 extras 填空串，保证 RequiredVariables 字段都在。
func fetchAndUpdateBuildRecord(
	ctx context.Context,
	client bkciapi.Client,
	pipeline *bkci.Pipeline,
	record *build.Record,
	buildID string,
) error {
	buildState, err := client.GetPipelineBuildState(ctx, pipeline.ProjectCode, pipeline.ID, buildID)
	if err != nil {
		log.Errorf(
			ctx, "failed to get project %s pipeline %s build %s: %v",
			pipeline.ProjectCode, pipeline.ID, buildID, err,
		)
		return err
	}

	var nextStatus build.Status
	buildStatus := bkciapi.PipelineBuildStatus(buildState.Status)
	switch {
	case buildStatus.IsSuccess():
		nextStatus = build.StatusSuccess
	case buildStatus.IsFailure():
		nextStatus = build.StatusFailed
	case buildStatus.IsRunning():
		nextStatus = build.StatusRunning
	case buildStatus.IsCancel():
		nextStatus = build.StatusCanceled
	default:
		nextStatus = build.StatusUnknown
	}

	if nextStatus != record.Status {
		extras := map[string]string{}
		for _, varName := range pipelinevar.RequiredVariables {
			extras[varName] = buildState.Variables[varName]
		}
		record.Status = nextStatus
		record.Extras = extras
		if buildStatus.IsFinished() {
			endTimestamp := cast.ToInt64(extras[pipelinevar.BuildEndTime])
			record.EndedAt = lo.Ternary(endTimestamp != 0, time.UnixMilli(endTimestamp), time.Now())
		}
	}
	return nil
}

// syncBuildStatus 把构建状态同步到 auto deploy 记录。
// 已进入部署阶段或已有 DeployID 则跳过，避免覆盖后续部署结果。
func syncBuildStatus(
	ctx context.Context,
	operator *autodeploy.Operator,
	appID, buildID string,
	record *build.Record,
) error {
	currentRecord, err := operator.GetByBuildID(ctx, appID, buildID)
	if err != nil {
		return err
	}
	if currentRecord.Stage == autodeploy.StageDeploy || currentRecord.DeployID != "" {
		return nil
	}
	patch := autodeploy.StatusPatch{
		Stage:   autodeploy.StageBuild,
		Status:  string(record.Status),
		Message: string(record.Status),
	}
	if record.Status.IsTerminated() {
		endedAt := record.EndedAt
		patch.EndedAt = &endedAt
	}
	return operator.UpdateStatus(ctx, autodeploy.Locator{AppID: appID, BuildID: buildID}, patch)
}

// triggerDeployAfterBuild 构建成功后启动自动部署，并回写 auto deploy 记录。
// 已进入部署阶段则跳过；触发失败也会把记录标为部署失败。
func triggerDeployAfterBuild(
	ctx context.Context,
	operator *autodeploy.Operator,
	record *build.Record,
	args Args,
) error {
	current, err := operator.GetByBuildID(ctx, args.AppID, args.BuildID)
	if err != nil {
		return err
	}
	if current.Stage == autodeploy.StageDeploy || current.DeployID != "" {
		return nil
	}

	deployID, triggerErr := startDeployAfterBuild(ctx, record, args)
	locator := autodeploy.Locator{AppID: args.AppID, BuildID: args.BuildID}
	if triggerErr != nil {
		patch := autodeploy.StatusPatch{
			Stage:   autodeploy.StageDeploy,
			Status:  string(appmodeldeploy.StatusFailed),
			Message: triggerErr.Error(),
		}
		if deployID != "" {
			patch.DeployID = &deployID
		}
		endedAt := time.Now()
		patch.EndedAt = &endedAt
		return operator.UpdateStatus(ctx, locator, patch)
	}
	return operator.UpdateStatus(ctx, locator, autodeploy.StatusPatch{
		Stage:    autodeploy.StageDeploy,
		Status:   string(appmodeldeploy.StatusDeploying),
		Message:  "",
		DeployID: &deployID,
	})
}

// startDeployAfterBuild 调 AppModel 部署，并投递部署状态轮询（asynq）。
// 部署已创建但轮询投递失败时仍返回 deployID，由调用方把记录标失败。
func startDeployAfterBuild(ctx context.Context, record *build.Record, args Args) (string, error) {
	if args.AutoDeploy == nil {
		return "", nil
	}
	deployService, err := newAppModelDeployService(storereg.G())
	if err != nil {
		return "", errors.Wrap(err, "init appmodel deploy service")
	}
	deployID, err := deployService.DeployByAppID(ctx, args.AppID, appmodeldeploysvc.DeployParams{
		EnvName:         args.AutoDeploy.EnvName,
		TrafficLaneName: args.AutoDeploy.TrafficLaneName,
		ImageTag:        record.Params[pipelineparam.ImageTag],
		UpdateStrategy:  string(tkex.RollingGameDeploymentUpdateStrategyType),
		Replicas:        args.AutoDeploy.Replicas,
		BuildAutoDeployInfo: &appmodeldeploy.BuildAutoDeployInfo{
			Branch:   record.Params[pipelineparam.RepoRevision],
			CommitID: record.Extras[pipelinevar.GitRepoHeadCommitID],
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "start appmodel deploy after build")
	}

	err = taskq.Enqueue(ctx, appmodeldeploypoll.Task.NewTask(appmodeldeploypoll.Args{
		WorkspaceID:     args.WorkspaceID,
		AppID:           args.AppID,
		EnvName:         args.AutoDeploy.EnvName,
		TrafficLaneName: args.AutoDeploy.TrafficLaneName,
		DeployID:        deployID,
	}))
	if err != nil {
		return deployID, errors.Wrap(err, "enqueue polling deploy status task")
	}
	return deployID, nil
}

// refreshSnapshots 构建成功后刷新镜像快照。
// imageTag 非空则强制同步该 tag，避免同 tag 重建后详情不更新；空则走默认刷新。
func refreshSnapshots(ctx context.Context, appID, imageTag string) error {
	reg := storereg.G()
	svc := snapshot.NewService(reg.SnapshotStore, reg.BuildConfigStore, reg.AppStore)
	var forceTags []string
	if imageTag != "" {
		forceTags = []string{imageTag}
	} else {
		log.Warnf(ctx, "build succeeded for app %s but imageTag is empty, fallback to default snapshot refresh", appID)
	}
	if _, err := svc.RefreshAppSnapshots(ctx, appID, forceTags...); err != nil {
		return errors.Wrapf(err, "refresh snapshots for app %s after build", appID)
	}
	return nil
}

// newAppModelDeployService 从 registry 组装 AppModel 部署服务，供构建成功后自动部署使用
func newAppModelDeployService(reg *storereg.Registry) (*appmodeldeploysvc.Service, error) {
	return appmodeldeploysvc.NewService(appmodeldeploysvc.ServiceDeps{
		AppStore:       reg.AppStore,
		EnvStore:       reg.EnvStore,
		PromotionStore: reg.PromotionStore,
		SnapshotService: snapshot.NewService(
			reg.SnapshotStore,
			reg.BuildConfigStore,
			reg.AppStore,
		),
		AppModelStore:                       reg.AppModelStore,
		WorkspaceStore:                      reg.WorkspaceStore,
		ImageRegistryStore:                  reg.ImageRegistryStore,
		ScopedEnvVarStore:                   reg.ScopedEnvVarStore,
		AppDepsVarReader:                    reg.AppDepsVarReader,
		PolarisVarReader:                    reg.PolarisVarReader,
		WorkspaceCompsStore:                 reg.WorkspaceCompsStore,
		PolarisConfigStore:                  reg.PolarisConfigStore,
		BscpCfgStore:                        reg.BscpCfgStore,
		AppSpecStore:                        reg.AppSpecStore,
		BuildConfigStore:                    reg.BuildConfigStore,
		BuildAutoDeployRecordStore:          reg.BuildAutoDeployRecordStore,
		AppModelDeployRecordStore:           reg.AppModelDeployRecordStore,
		AppModelDeployResourceSnapshotStore: reg.AppModelDeployResourceSnapshotStore,
		AppConfigFileStore:                  reg.AppConfigFileStore,
	})
}
