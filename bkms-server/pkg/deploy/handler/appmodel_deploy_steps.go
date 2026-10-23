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

// Package handler 包含部署相关 Gin API 的 handler
package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	deploypkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	appmodeldeploysvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/alertstrategysync"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/appmodeldeploypoll"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// listAppModelDeployRecords 获取 AppModel 应用部署记录列表
func (h *Handler) listAppModelDeployRecords(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.ListAppModelDeployRecordsInput
	if err := ginutils.BindURIQuery(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 参数 & 权限校验
	ctx := c.Request.Context()
	app, _, err := h.validateAppModelDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeView, false)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取 AppModel 应用部署记录
	recordPage, err := store.Of(h.registry.DB, appmodeldeploy.RecordMeta).ListPage(
		ctx, input.Page, input.PageSize,
		deploytypes.ByAppEnvLane(app.ID, uriInput.EnvName, input.TrafficLaneName),
		appmodeldeploy.KeywordMatches(input.Keyword),
		deploytypes.NewestFirst(),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list deploy records for %s",
			genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, input.TrafficLaneName),
		))
		return
	}

	// 转换成输出格式；构建+部署触发的补充字段直接从部署记录 Extras 中读取
	outputRecords := make([]*serializer.AppModelDeployRecordOutputObj, 0, len(recordPage.Items))
	for _, record := range recordPage.Items {
		outputRecords = append(outputRecords, new(serializer.AppModelDeployRecordOutputObj).FromModel(record))
	}
	ginutils.OK(c, serializer.ListAppModelDeployRecordsOutput{
		Data: serializer.PaginatedAppModelDeployRecordsOutputObjs{Count: recordPage.Total, Results: outputRecords},
	})
}

// preCheckDeploy runs tRPC / TAF deployment pre-checks (undefined env vars and required cluster addons).
func (h *Handler) preCheckDeploy(c *gin.Context, expectedAppType string) {
	var uriInput serializer.AppEnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, environment, err := h.validateAppModelDeployAppEnv(
		ctx, uriInput.AppID, uriInput.EnvName, perm.TypeEdit, true,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if app.Type != expectedAppType {
		bkerrs.AbortWithErr(c, bkerrs.New(
			bkerrs.ErrCodeInvalidRequest,
			fmt.Sprintf("invalid app type: expected %s, got %s", expectedAppType, app.Type),
		))
		return
	}
	// 可见环境校验排在存在性、权限、应用类型校验之后，避免掩盖更具体的错误
	if err = app.CheckVisibleEnv(environment); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	checker := h.newDeployPreChecker()
	result, err := checker.Check(ctx, app, environment)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "pre-check deployment"))
		return
	}
	ginutils.OK(c, new(serializer.DeployPreCheckOutput).FromModel(result))
}

func (h *Handler) newDeployPreChecker() *deploypkg.DeployPreChecker {
	builderService := workload.NewBuilderService(
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
		h.registry.DB,
		h.newMountableFileProvider(),
	)
	return deploypkg.NewDeployPreChecker(
		builderService,
		h.registry.DB,
	)
}

// createAppModelDeploy 创建 AppModel 应用部署
func (h *Handler) createAppModelDeploy(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.CreateAppModelDeployInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 参数 & 权限校验
	ctx := c.Request.Context()
	app, environment, err := h.validateAppModelDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeEdit, true)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err = app.CheckVisibleEnv(environment); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	var strategy *releasestrategy.ReleaseStrategy
	if input.ReleaseStrategyID != "" {
		strategy, err = releasestrategy.GetStrategy(
			ctx, store.Of(h.registry.DB, releasestrategy.Meta), app.WorkspaceID, input.ReleaseStrategyID,
		)
		if err != nil {
			bkerrs.AbortWithErr(c, batchrelease.APIError(err, "get release strategy"))
			return
		}
	}

	deployService, err := h.newAppModelDeployService()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init appmodel deploy service"))
		return
	}
	deploypkg.TrackEnvAddApp(ctx, h.registry.DB, app.WorkspaceID, uriInput.EnvName, app.ID)
	deployID, err := deployService.Deploy(ctx, app, appmodeldeploysvc.DeployParams{
		EnvName:         uriInput.EnvName,
		TrafficLaneName: input.TrafficLaneName,
		ImageTag:        input.ImageTag,
		Replicas:        input.Replicas,
		ReleaseStrategy: strategy,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, toAppModelDeployAPIError(err, app, environment, input.TrafficLaneName))
		return
	}
	if strategy != nil {
		recordBatchDeployAudit(ctx, app, uriInput.EnvName, input.TrafficLaneName, deployID, strategy)
	}
	// 轮询部署状态 & 更新部署记录
	// 注：部署成功/失败的操作记录在异步任务中记录
	err = taskq.Enqueue(
		ctx, appmodeldeploypoll.Task.NewTask(appmodeldeploypoll.Args{
			WorkspaceID:     app.WorkspaceID,
			AppID:           app.ID,
			EnvName:         uriInput.EnvName,
			TrafficLaneName: input.TrafficLaneName,
			DeployID:        deployID,
		}), asynq.ProcessIn(appmodeldeploypoll.PollingInterval()),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			err, bkerrs.ErrCodeInternalServerError, "enqueue polling deploy status task",
		))
		return
	}

	ginutils.OK(c, serializer.EmptyOutput{})
}

// deleteAppModelDeploy 删除 AppModel 应用部署
func (h *Handler) deleteAppModelDeploy(c *gin.Context) {
	startedAt := time.Now()
	metricStatus := metrics.StatusOK
	var uriInput serializer.AppEnvURIInput
	var input serializer.DeleteAppModelDeployInput
	if err := ginutils.BindURIQuery(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 参数 & 权限校验
	ctx := c.Request.Context()
	app, env, err := h.validateAppModelDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeEdit, true)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	defer func() {
		metrics.DeployUninstallFinished(metrics.DeployKindAppModel, metricStatus, startedAt)
	}()

	// 执行部署操作
	deployer := h.newDeployer(app)
	if err = deployer.Uninstall(ctx, env, input.TrafficLaneName); err != nil {
		metricStatus = metrics.StatusErr
		deployInfo := genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, input.TrafficLaneName)
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "uninstall app model app: %s", deployInfo,
		))
		return
	}

	h.handleAppModelDeployUninstalled(
		ctx, app, env, uriInput.EnvName, input.TrafficLaneName, auth.MustGetUser(ctx).ID,
	)
	ginutils.OK(c, serializer.EmptyOutput{})
}

func (h *Handler) handleAppModelDeployUninstalled(
	ctx context.Context,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	envName, trafficLaneName, operator string,
) {
	// 记录应用从环境卸载
	deploypkg.TrackEnvRemoveApp(ctx, h.registry.DB, app.WorkspaceID, envName, trafficLaneName, app.ID)

	// 应用已从环境卸载，需同步清理该应用在该环境下关联的告警策略，避免残留的无效告警。
	// 先查询 workspace 信息（清理时需要其标识），查询失败仅记录日志，不影响卸载主流程。
	ws, wsErr := store.Of(h.registry.DB, workspace.Meta).Get(ctx, workspace.ByWorkspaceID(app.WorkspaceID))
	if wsErr != nil {
		log.Errorf(ctx, "get workspace %s for alert cleanup failed: %v", app.WorkspaceID, wsErr)
	} else {
		// 异步执行清理，避免阻塞卸载接口的响应
		log.Infof(
			ctx,
			"schedule alert strategy cleanup after uninstall, workspace=%s app=%s env=%s envID=%s lane=%s operator=%s",
			ws.ID,
			app.ID,
			env.Name,
			env.ID.Hex(),
			trafficLaneName,
			operator,
		)
		if err := taskq.Enqueue(ctx, alertstrategysync.CleanupTask.NewTask(alertstrategysync.Args{
			WorkspaceID:     app.WorkspaceID,
			AppID:           app.ID,
			EnvName:         env.Name,
			TrafficLaneName: trafficLaneName,
		})); err != nil {
			log.Errorf(ctx, "enqueue alert strategy cleanup task failed: %v", err)
		}
	}

	// 清理资源拓扑快照（失败不阻塞主流程）
	snapshots := store.Of(h.registry.DB, topology.ResourceSnapshotMeta)
	if err := snapshots.DeleteIfExists(ctx, topology.ByScope(app.ID, envName, trafficLaneName)); err != nil {
		log.Errorf(
			ctx, "delete resource snapshot for app %s env %s lane %s failed: %v",
			app.ID, envName, trafficLaneName, err,
		)
	}

	// 扩缩容操作记录
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUninstall,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(envName),
		audit.WithExtras(map[string]string{"trafficLaneName": trafficLaneName}),
	)
}

// listAppModelResourceSnapshots 列出 AppModel 应用某次部署下发的资源清单快照元数据
func (h *Handler) listAppModelResourceSnapshots(c *gin.Context) {
	var uriInput serializer.DeployURIInput
	var input serializer.ListAppModelResourceSnapshotsInput
	if err := ginutils.BindURIQuery(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 参数 & 权限校验
	ctx := c.Request.Context()
	app, _, err := h.validateAppModelDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeView, false)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取资源清单快照元数据（分页）
	snapshots := store.Of(h.registry.DB, appmodeldeploy.ResourceSnapshotMeta)
	page, err := appmodeldeploy.ListSnapshotMeta(
		ctx, snapshots, app.ID, uriInput.DeployID, input.Page, input.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			err, bkerrs.ErrCodeInternalServerError, "list app model resource snapshots",
		))
		return
	}
	outputs := make([]*serializer.AppModelResourceSnapshot, 0, len(page.Items))
	for _, row := range page.Items {
		outputs = append(outputs, new(serializer.AppModelResourceSnapshot).FromModel(row, false))
	}
	ginutils.OK(c, serializer.ListAppModelResourceSnapshotsOutput{
		Data: serializer.PaginatedAppModelResourceSnapshotsOutputObjs{Count: page.Total, Results: outputs},
	})
}

// getAppModelResourceSnapshot 获取 AppModel 类型应用部署记录对应的资源快照
func (h *Handler) getAppModelResourceSnapshot(c *gin.Context) {
	var uriInput serializer.ResourceSnapshotURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 参数 & 权限校验
	ctx := c.Request.Context()
	app, _, err := h.validateAppModelDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeView, false)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取资源清单快照
	snapshots := store.Of(h.registry.DB, appmodeldeploy.ResourceSnapshotMeta)
	row, err := appmodeldeploy.GetSnapshot(ctx, snapshots, app.ID, uriInput.DeployID, uriInput.SnapshotID)
	if err != nil {
		if errors.Is(err, appmodeldeploy.ErrResourceSnapshotRowNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeNotFound, "resource snapshot not found"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			err, bkerrs.ErrCodeInternalServerError, "get app model resource snapshot",
		))
		return
	}
	ginutils.OK(c, serializer.GetAppModelResourceSnapshotOutput{
		Snapshot: new(serializer.AppModelResourceSnapshot).FromModel(*row, true),
	})
}

// getLatestAppModelDeployStatus 应用最新一次部署的状态（含应用类型检查、自动区分手动部署/构建后自动部署）
func (h *Handler) getLatestAppModelDeployStatus(c *gin.Context, expectedAppType string) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.GetLatestAppModelDeployStatusInput
	if err := ginutils.BindURIQuery(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if app.Type != expectedAppType {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "invalid app type: %s", app.Type))
		return
	}

	status, err := h.latestAppModelDeployStatus(ctx, app.ID, uriInput.EnvName, input.TrafficLaneName)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ginutils.OK(c, serializer.GetLatestAppModelDeployStatusOutput{Data: status})
}

// latestAppModelDeployStatus 合并手动部署与构建自动部署两种来源，返回指定环境（及用到）的最新部署记录
func (h *Handler) latestAppModelDeployStatus(
	ctx context.Context, appID, envName, trafficLaneName string,
) (*serializer.LatestDeployStatus, error) {
	// 获取最新一次构建自动部署结果
	latestBuildAutoDeploy, err := autodeploy.GetLatest(ctx, h.registry.DB, appID, envName, trafficLaneName)
	if err != nil && !errors.Is(err, autodeploy.ErrRecordNotFound) {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get latest build auto deploy record")
	}
	if errors.Is(err, autodeploy.ErrRecordNotFound) {
		latestBuildAutoDeploy = nil
	}

	// 获取最新一次应用部署结果
	latestDeploy, err := appmodeldeploy.GetLatest(ctx, h.registry.DB, appID, envName, trafficLaneName)
	if err != nil && !errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get latest app model deploy record")
	}
	if errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
		latestDeploy = nil
	}

	// 自动部署以及直接部署均为空，为未部署场景，返回空 status
	if latestBuildAutoDeploy == nil && latestDeploy == nil {
		return nil, nil
	}

	// 根据部署记录判断使用哪个 status
	var status *serializer.LatestDeployStatus
	if shouldUseBuildAutoDeployRecord(latestBuildAutoDeploy, latestDeploy) {
		status = new(serializer.LatestDeployStatus).FromBuildAutoDeployRecord(latestBuildAutoDeploy)
	} else {
		status = new(serializer.LatestDeployStatus).FromDeployRecord(latestDeploy)
	}
	// 实时判断是否存在部署记录，前端判断是否轮询 instance
	status.HasDeployRecord = latestDeploy != nil
	return status, nil
}

// newAppModelDeployService 创建 AppModel 部署编排服务
func (h *Handler) newAppModelDeployService() (*appmodeldeploysvc.Service, error) {
	reg := h.registry
	snapshotService := snapshot.NewService(reg.DB)

	return appmodeldeploysvc.NewService(appmodeldeploysvc.ServiceDeps{
		DB:               reg.DB,
		SnapshotService:  snapshotService,
		AppDepsVarReader: reg.AppDepsVarReader,
		PolarisVarReader: reg.PolarisVarReader,
	})
}

// newDeployer 创建 AppModel 部署器
func (h *Handler) newDeployer(app *bkmsapp.Application) *appmodeldeploy.Deployer {
	return appmodeldeploy.NewDeployer(
		workload.NewBuilderService(
			h.registry.AppDepsVarReader,
			h.registry.PolarisVarReader,
			h.registry.DB,
			h.newMountableFileProvider(),
		),
		h.registry.DB,
		polaris.NewPolarisEnvStateManager(h.registry.DB),
		hostport.NewEnvStateManager(h.registry.DB),
		app,
	)
}

// shouldUseBuildAutoDeployRecord 判断是否应该使用构建自动部署记录作为最新部署状态
func shouldUseBuildAutoDeployRecord(
	buildAutoDeployRecord *autodeploy.Record,
	deployRecord *appmodeldeploy.Record,
) bool {
	if buildAutoDeployRecord == nil {
		return false
	}
	if deployRecord == nil {
		return true
	}
	if buildAutoDeployRecord.DeployID != "" && buildAutoDeployRecord.DeployID == deployRecord.ID.Hex() {
		return true
	}
	return !buildAutoDeployRecord.CreatedAt.Before(deployRecord.CreatedAt)
}

// toAppModelDeployAPIError 将 AppModel 部署下发失败的错误转换为对应错误码的 API 错误
func toAppModelDeployAPIError(
	err error,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
	trafficLaneName string,
) error {
	var checkErr *clusteraddon.RequiredAddonsNotInstalledError
	if errors.As(err, &checkErr) {
		components := lo.Map(checkErr.Missing, func(addon clusteraddon.AddonReference, _ int) string {
			return addon.Name
		})
		return bkerrs.WrapComponentsNotInstalled(err, components, env.Cluster.ClusterID)
	}
	deployInfo := genDeployInfo(app.WorkspaceID, app.ID, env.Name, trafficLaneName)
	if errors.Is(err, batchrelease.ErrWorkloadNotSupported) ||
		errors.Is(err, batchrelease.ErrRolloutNotInstalled) {
		return batchrelease.APIError(err, "batch release app model app: "+deployInfo)
	}
	return bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "deploy app model app: %s", deployInfo)
}

// recordBatchDeployAudit 记录按策略分批部署的操作记录（含策略名称）。
// 分批发布可能长期暂停等待人工确认，因此在下发时记录，不等待轮询任务进入终态。
func recordBatchDeployAudit(
	ctx context.Context,
	app *bkmsapp.Application,
	envName, trafficLaneName, deployID string,
	strategy *releasestrategy.ReleaseStrategy,
) {
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeDeploy,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(envName),
		audit.WithExtras(map[string]string{
			"trafficLaneName":     trafficLaneName,
			"deployID":            deployID,
			"releaseMode":         string(appmodeldeploy.ReleaseModeBatch),
			"releaseStrategyID":   strategy.ID.Hex(),
			"releaseStrategyName": strategy.Name,
		}),
	)
}
