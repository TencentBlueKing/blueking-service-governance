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

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
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
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/appmodeldeploypoll"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

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
