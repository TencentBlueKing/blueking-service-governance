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

// FIXME: 镜像构建状态轮询已迁至 asynq（pkg/server/taskqtask/buildpoll）
// 本文件仅保留 RabbitMQ 存量消费路径，存量队列耗尽后移除此文件

package task

import (
	"context"
	"fmt"
	"time"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelinevar"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	appmodeldeploysvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/worker"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// AutoDeployArgs 构建成功后自动部署的参数
type AutoDeployArgs struct {
	EnvName         string `json:"envName"`
	TrafficLaneName string `json:"trafficLaneName"`
	Replicas        int32  `json:"replicas"`
}

// PollingBuildStatusArgs 轮询构建状态的参数
type PollingBuildStatusArgs struct {
	WorkspaceID  string          `json:"workspaceID"`
	PipelineType string          `json:"pipelineType"`
	AppID        string          `json:"appID"`
	BuildID      string          `json:"buildID"`
	AutoDeploy   *AutoDeployArgs `json:"autoDeploy,omitempty"`
}

// String 参数内容字符串化
func (args PollingBuildStatusArgs) String() string {
	return fmt.Sprintf(
		"<workspace: %s, pipelineType: %s, appID: %s, buildID: %s>",
		args.WorkspaceID, args.PipelineType, args.AppID, args.BuildID,
	)
}

// fetchAndUpdateBuildRecord 获取并更新构建记录对象数据
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
		// 额外的一些构建信息，如代码库信息，构建号等（非启动参数）
		extras := map[string]string{}
		for _, varName := range pipelinevar.RequiredVariables {
			if value, ok := buildState.Variables[varName]; ok {
				extras[varName] = value
			} else {
				// 确保字段都存在，即使值是空的
				extras[varName] = ""
			}
		}

		record.Status = nextStatus
		record.Extras = extras

		if buildStatus.IsFinished() {
			// 如果能从蓝盾获取到构建结束时间，则使用，否则使用 time.Now()
			endTimestamp := cast.ToInt64(extras[pipelinevar.BuildEndTime])
			record.EndedAt = lo.Ternary(endTimestamp != 0, time.UnixMilli(endTimestamp), time.Now())
		}
	}

	return nil
}

func startDeployAfterBuild(ctx context.Context, record *build.Record, args PollingBuildStatusArgs) (string, error) {
	// 构建依赖service
	if args.AutoDeploy == nil {
		return "", nil
	}
	reg := storereg.G()
	deployService, err := newAppModelDeployServiceFromRegistry(reg)
	if err != nil {
		return "", errors.Wrap(err, "init appmodel deploy service")
	}
	// 启动部署
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

	// 投递轮询部署状态任务
	_, err = worker.ApplyTask(
		ctx,
		config.G.RabbitMQ.GetURI(),
		config.G.RabbitMQ.Queue,
		PollingTrpcDeployStatus,
		PollingTrpcDeployStatusArgs{
			WorkspaceID:     args.WorkspaceID,
			AppID:           args.AppID,
			EnvName:         args.AutoDeploy.EnvName,
			TrafficLaneName: args.AutoDeploy.TrafficLaneName,
			DeployID:        deployID,
		},
	)
	if err != nil {
		return deployID, errors.Wrap(err, "apply polling deploy status task")
	}
	return deployID, nil
}

// syncBuildStatus 同步构建状态
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
	// 重复投递的构建轮询可能晚于部署状态推进到达；一旦记录已经关联部署链路，
	// 就不再允许 build 维度状态回写覆盖 deploy 维度状态。
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
	return operator.UpdateStatus(ctx, autodeploy.Locator{
		AppID:   appID,
		BuildID: buildID,
	}, patch)
}

func triggerDeployAfterBuild(
	ctx context.Context,
	operator *autodeploy.Operator,
	record *build.Record,
	args PollingBuildStatusArgs,
) error {
	buildDeployRecord, err := operator.GetByBuildID(ctx, args.AppID, args.BuildID)
	if err != nil {
		return err
	}
	// 轮询任务可能重复投递；若已经进入部署阶段，则不再重复触发部署
	if buildDeployRecord.Stage == autodeploy.StageDeploy || buildDeployRecord.DeployID != "" {
		return nil
	}

	deployID, triggerErr := startDeployAfterBuild(ctx, record, args)
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
		return operator.UpdateStatus(ctx, autodeploy.Locator{
			AppID:   args.AppID,
			BuildID: args.BuildID,
		}, patch)
	}
	return operator.UpdateStatus(ctx, autodeploy.Locator{
		AppID:   args.AppID,
		BuildID: args.BuildID,
	}, autodeploy.StatusPatch{
		Stage:    autodeploy.StageDeploy,
		Status:   string(appmodeldeploy.StatusDeploying),
		Message:  "",
		DeployID: &deployID,
	})
}

func markBuildAsTerminated(record *build.Record, status build.Status) {
	record.Status = status
	if record.EndedAt.IsZero() {
		record.EndedAt = time.Now()
	}
}

// pollingBuildStatus 轮询蓝盾流水线构建状态
// TODO: 后续切换到 asyncq 后重构此函数，届时轮询循环、状态保存、构建后触发（自动部署 & 镜像快照刷新）
// 等职责将拆分为独立子任务/回调，届时应移除下方的圈复杂度忽略注释
//
//nolint:gocyclo,cyclop,gocognit // 当前基于 machinery 长轮询实现，逻辑高度耦合于单函数内，暂忽略圈复杂度检查
func pollingBuildStatus(ctx context.Context, args PollingBuildStatusArgs) (*EmptyResult, error) {
	log.Infof(ctx, "start polling build %s status, timeout: %s", args, buildPollingTimeout)

	// 使用 24 小时超时创建上下文
	ctx, cancel := context.WithTimeout(ctx, buildPollingTimeout)
	defer cancel()

	reg := storereg.G()

	// 获取构建记录
	store := reg.BuildRecordStore
	if store == nil {
		return nil, errors.New("build record store is not initialized")
	}
	record, err := store.Get(ctx, args.AppID, args.BuildID)
	if err != nil {
		return nil, errors.Wrap(err, "get build record")
	}
	autoDeployEnabled := args.AutoDeploy != nil
	var buildAutoDeployOperator *autodeploy.Operator
	if autoDeployEnabled {
		if reg.BuildAutoDeployRecordStore == nil {
			return nil, errors.New("build auto deploy record store is not initialized")
		}
		buildAutoDeployOperator, err = autodeploy.NewOperator(reg.BuildAutoDeployRecordStore)
		if err != nil {
			return nil, errors.Wrap(err, "init build auto deploy updater")
		}
	}

	// 记录构建开始时间，用于计算动态间隔（由于构建轮询是长时间任务，可能存在重复投递的情况，因此需要以构建开始时间为准）
	startTime := record.StartedAt
	curInterval := calcBuildPollingInterval(time.Since(startTime))

	// 获取流水线信息
	pipelineStore := reg.BkCIPipelineStore
	if pipelineStore == nil {
		return nil, errors.New("bkci pipeline store is not initialized")
	}
	pipeline, err := pipelineStore.GetByWorkspaceAndType(ctx, args.WorkspaceID, args.PipelineType)
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s type %s pipeline", args.WorkspaceID, args.PipelineType)
	}

	user, err := auth.GetUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get authed user")
	}

	apiClient, err := bkciapi.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "create bkci api client")
	}
	// 获取状态失败重试次数（防止网络波动等原因）
	failureRetryCount := TotalFailureRetryCount
	for {
		// 记录当前状态以便后续比对
		curStatus := record.Status

		select {
		case <-ctx.Done():
			log.Warnf(ctx, "context timeout, stop update %s status", args)
			markBuildAsTerminated(record, build.StatusPollingTimeout)
		case <-time.After(curInterval):
			// 根据已运行时长动态调整轮询间隔
			newInterval := calcBuildPollingInterval(time.Since(startTime))
			if newInterval != curInterval {
				log.Infof(ctx, "build %s polling interval changed from %s to %s", args, curInterval, newInterval)
				curInterval = newInterval
			}
			// 获取并更新流水线构建状态
			if err = fetchAndUpdateBuildRecord(ctx, apiClient, pipeline, record, args.BuildID); err != nil {
				// 轮询流程中失败超过指定次数，停止轮询
				failureRetryCount--
				if failureRetryCount <= 0 {
					log.Errorf(ctx, "stop polling pipeline build %s after %d retries", args, TotalFailureRetryCount)
					markBuildAsTerminated(record, build.StatusPollingBroken)
				}
			}
		}

		// 若 record 状态变更，则需要保存入库
		if record.Status != curStatus {
			log.Infof(ctx, "build %s status changed from %s to %s", args, curStatus, record.Status)
			// 使用闭包函数确保 defer 能在每次迭代时正确执行，避免资源泄露
			func() {
				// 为最终状态更新创建独立的 context，确保即使原 ctx 超时也能完成写入
				saveCtx, saveCancel := context.WithTimeout(context.Background(), saveStatusTimeout)
				defer saveCancel()
				// 更新构建记录
				if err = store.Update(saveCtx, record); err != nil {
					log.Errorf(saveCtx, "failed to update build record: %v", err)
				}
				if autoDeployEnabled {
					uErr := syncBuildStatus(saveCtx, buildAutoDeployOperator, args.AppID, args.BuildID,
						record)
					if uErr != nil {
						log.Errorf(saveCtx, "failed to update build auto deploy record by buildID: %v", uErr)
					}
				}
			}()
		}

		// 如果最新状态已经是结束态，退出轮询
		if record.Status.IsTerminated() {
			metrics.BuildFinished(string(record.Status), record.StartedAt, time.Now())
			if record.Status == build.StatusSuccess && autoDeployEnabled {
				if err = triggerDeployAfterBuild(ctx, buildAutoDeployOperator, record, args); err != nil {
					log.Errorf(ctx, "handle build auto deploy after build success failed: %v", err)
				}
			}

			log.Infof(ctx, "build %s status is %s (Terminated)，stop polling and release lock", args, record.Status)

			// 转换为操作结果 & 记录操作审计
			opResult := lo.Ternary(record.Status == build.StatusSuccess, audit.ResultSuccess, audit.ResultFailed)
			go audit.AddOperationRecordAsync(
				context.WithoutCancel(ctx), audit.OperationTypeBuild, audit.ResourceTypeApp, args.AppID,
				audit.WithResult(opResult), audit.WithWorkspaceID(args.WorkspaceID), audit.WithAppID(args.AppID),
			)

			// 构建成功后异步触发镜像快照刷新（包含从远程拉取新 tag 和详情补全）
			if record.Status == build.StatusSuccess {
				imageTag := record.Params[pipelineparam.ImageTag]
				go func() {
					tCtx := context.WithoutCancel(ctx)
					if refreshErr := triggerSnapshotRefreshAfterBuild(
						tCtx, args.AppID, imageTag,
					); refreshErr != nil {
						log.Errorf(
							tCtx, "trigger snapshot refresh after build for app %s failed: %v",
							args.AppID, refreshErr,
						)
					}
				}()
			}

			return &emptyResult, nil
		}
	}
}

// triggerSnapshotRefreshAfterBuild 构建成功后异步触发镜像快照刷新
// 调用完整的 RefreshSnapshots 流程，确保新构建产物的 tag 先从远程拉取写入快照表，再异步补全详情。
// imageTag 为本次构建产出的标签，作为强制同步标签下发，使重复使用同一标签构建时详情也能刷新；
// 解析不到标签时退化为默认刷新行为。
func triggerSnapshotRefreshAfterBuild(ctx context.Context, appID, imageTag string) error {
	reg := storereg.G()
	svc := newSnapshotServiceFromRegistry(reg)
	var forceDetailSyncTags []string
	if imageTag != "" {
		forceDetailSyncTags = []string{imageTag}
	} else {
		log.Warnf(ctx, "build succeeded for app %s but imageTag is empty, fallback to default snapshot refresh", appID)
	}
	if _, err := svc.RefreshAppSnapshots(ctx, appID, forceDetailSyncTags...); err != nil {
		return errors.Wrapf(err, "refresh snapshots for app %s after build", appID)
	}
	return nil
}

func newSnapshotServiceFromRegistry(reg *storereg.Registry) *snapshot.Service {
	return snapshot.NewService(reg.SnapshotStore, reg.BuildConfigStore, reg.AppStore)
}

func newAppModelDeployServiceFromRegistry(reg *storereg.Registry) (*appmodeldeploysvc.Service, error) {
	return appmodeldeploysvc.NewService(appmodeldeploysvc.ServiceDeps{
		AppStore:                            reg.AppStore,
		EnvStore:                            reg.EnvStore,
		PromotionStore:                      reg.PromotionStore,
		SnapshotService:                     newSnapshotServiceFromRegistry(reg),
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
