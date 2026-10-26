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
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelinevar"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/revision"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	appmodeldeploysvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel/service"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/appmodeldeploypoll"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
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
	// 以流水线实际检出方式为准；空修订在 CheckoutBy 已回落成 BRANCH
	rev := revision.CodeRevision{
		Type:  revision.TypeFromCheckoutBy(record.Params[pipelineparam.RepoCheckoutBy]),
		Value: record.Params[pipelineparam.RepoRevision],
	}

	deployID, err := deployService.DeployByAppID(ctx, args.AppID, appmodeldeploysvc.DeployParams{
		EnvName:         args.AutoDeploy.EnvName,
		TrafficLaneName: args.AutoDeploy.TrafficLaneName,
		ImageTag:        record.Params[pipelineparam.ImageTag],
		UpdateStrategy:  string(tkex.RollingGameDeploymentUpdateStrategyType),
		Replicas:        args.AutoDeploy.Replicas,
		BuildAutoDeployInfo: &appmodeldeploy.BuildAutoDeployInfo{
			Branch:       rev.LegacyBranch(),
			RevisionType: string(rev.Type),
			Revision:     rev.Value,
			CommitID:     record.Extras[pipelinevar.GitRepoHeadCommitID],
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "start appmodel deploy after build")
	}

	err = taskq.Enqueue(
		ctx,
		appmodeldeploypoll.Task.NewTask(appmodeldeploypoll.Args{
			WorkspaceID:     args.WorkspaceID,
			AppID:           args.AppID,
			EnvName:         args.AutoDeploy.EnvName,
			TrafficLaneName: args.AutoDeploy.TrafficLaneName,
			DeployID:        deployID,
		}),
		asynq.ProcessIn(appmodeldeploypoll.PollingInterval()),
	)
	if err != nil {
		return deployID, errors.Wrap(err, "enqueue polling deploy status task")
	}
	return deployID, nil
}

// enqueueRecordImageDigest 投递记录构建产物镜像 digest 的任务，失败由任务按 imageDigestMaxRetry 重试
func enqueueRecordImageDigest(ctx context.Context, appID, buildID string) error {
	task := ImageDigestTask.NewTask(ImageDigestArgs{AppID: appID, BuildID: buildID})
	return errors.Wrap(taskq.Enqueue(ctx, task), "enqueue record image digest task")
}

// handleRecordImageDigest asynq 入口：数据库缺失则 ErrStopRetry，否则交给 recordImageDigestOfBuild
func handleRecordImageDigest(ctx context.Context, args ImageDigestArgs) error {
	reg := storereg.G()
	if reg == nil || reg.DB == nil {
		return errors.Wrapf(taskq.ErrStopRetry, "record image digest of build %s: stores not initialized", args)
	}
	return recordImageDigestOfBuild(ctx, reg.DB, args)
}

// recordImageDigestOfBuild 为一条成功构建记录 digest。已记录过则跳过，保证重复投递与重试幂等；
// 超出 imageDigestRecordWindow（如 worker 停机后积压的任务）则放弃
func recordImageDigestOfBuild(ctx context.Context, db *mongo.Database, args ImageDigestArgs) error {
	record, err := store.Of(db, build.RecordMeta).Get(ctx, build.ByBuild(args.AppID, args.BuildID))
	if err != nil {
		if errors.Is(err, build.ErrRecordNotFound) {
			return errors.Wrapf(taskq.ErrStopRetry, "get build record %s: %v", args, err)
		}
		return errors.Wrapf(err, "get build record %s", args)
	}
	if record.Status != build.StatusSuccess || record.ImageDigest != "" {
		return nil
	}
	if elapsed := time.Since(record.EndedAt); elapsed > imageDigestRecordWindow {
		return errors.Wrapf(
			taskq.ErrStopRetry, "build %s ended %s ago, beyond image digest record window %s",
			args, elapsed, imageDigestRecordWindow,
		)
	}
	return recordImageDigest(ctx, db, record)
}

// recordImageDigest 构建成功后从镜像仓库读取产物 digest 并回写构建记录，供制品列表按 digest 精确关联构建记录。
// 产物不在应用制品仓库时跳过：这类镜像不会出现在制品列表中，且仓库凭证也不一定适用
func recordImageDigest(ctx context.Context, db *mongo.Database, record *build.Record) error {
	imageTag := record.Params[pipelineparam.ImageTag]
	if imageTag == "" {
		return nil
	}
	info, err := snapshot.NewService(db).ResolveRepoKeyForApp(ctx, record.AppID)
	if err != nil {
		return errors.Wrapf(err, "resolve image repo for app %s", record.AppID)
	}
	if record.Artifact != info.RepoName+":"+imageTag {
		log.Infof(
			ctx, "build %s artifact %s is not in app image repo %s, skip recording image digest",
			record.BuildID, record.Artifact, info.RepoName,
		)
		return nil
	}

	detail, err := registry.New(info.Username, info.Password, true).GetTagDetail(ctx, info.RepoName, imageTag)
	if err != nil {
		return errors.Wrapf(err, "get image %s detail", record.Artifact)
	}
	records := store.Of(db, build.RecordMeta)
	return build.UpdateRecordImageDigest(ctx, records, record.AppID, record.BuildID, detail.Digest)
}

// refreshSnapshots 构建成功后刷新镜像快照。
// imageTag 非空则强制同步该 tag，避免同 tag 重建后详情不更新；空则走默认刷新。
func refreshSnapshots(ctx context.Context, appID, imageTag string) error {
	reg := storereg.G()
	svc := snapshot.NewService(reg.DB)
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
		DB:               reg.DB,
		SnapshotService:  snapshot.NewService(reg.DB),
		AppDepsVarReader: reg.AppDepsVarReader,
		PolarisVarReader: reg.PolarisVarReader,
	})
}
