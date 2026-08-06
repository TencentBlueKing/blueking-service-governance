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

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	deploypkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	appmodeldeploysvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/worker"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/task"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
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
	records, total, err := h.registry.AppModelDeployRecordStore.List(
		ctx, app.ID, uriInput.EnvName, input.TrafficLaneName, input.Keyword, input.Page, input.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list deploy records for %s",
			genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, input.TrafficLaneName),
		))
		return
	}

	// 转换成输出格式；构建+部署触发的补充字段直接从部署记录 Extras 中读取
	outputRecords := make([]*serializer.AppModelDeployRecordOutputObj, 0, len(records))
	for _, record := range records {
		outputRecords = append(outputRecords, new(serializer.AppModelDeployRecordOutputObj).FromModel(record))
	}
	ginutils.OK(c, serializer.ListAppModelDeployRecordsOutput{
		Data: serializer.PaginatedAppModelDeployRecordsOutputObjs{Count: total, Results: outputRecords},
	})
}

// preCheckDeployEnvVars 检查 AppModel 部署中引用但未定义的环境变量
func (h *Handler) preCheckDeployEnvVars(c *gin.Context, expectedAppType string) {
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

	checker := h.newEnvVarPreChecker()
	result, err := checker.Check(ctx, app, environment)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "pre-check deployment env vars"))
		return
	}
	ginutils.OK(c, new(serializer.EnvVarPreCheckOutput).FromModel(result))
}

func (h *Handler) newEnvVarPreChecker() *deploypkg.EnvVarPreChecker {
	builderService := workload.NewBuilderService(
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
		h.registry.WorkspaceCompsStore,
		h.registry.PolarisConfigStore,
		h.registry.BscpCfgStore,
		h.registry.AppModelStore,
		h.registry.AppSpecStore,
		h.registry.BuildConfigStore,
	)
	return deploypkg.NewEnvVarPreChecker(
		h.registry.AppModelStore,
		builderService,
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
	app, _, err := h.validateAppModelDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeEdit, true)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	deployService, err := h.newAppModelDeployService()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init appmodel deploy service"))
		return
	}
	deploypkg.TrackEnvAddApp(ctx, h.registry.EnvStore, app.WorkspaceID, uriInput.EnvName, app.ID)
	deployID, err := deployService.Deploy(ctx, app, appmodeldeploysvc.DeployParams{
		EnvName:         uriInput.EnvName,
		TrafficLaneName: input.TrafficLaneName,
		ImageTag:        input.ImageTag,
		Replicas:        input.Replicas,
	})
	if err != nil {
		deployInfo := genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, input.TrafficLaneName)
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "deploy app model app: %s", deployInfo,
		))
		return
	}
	// 轮询部署状态 & 更新部署记录
	// 注：部署成功/失败的操作记录在异步任务中记录
	_, err = worker.ApplyTask(
		ctx,
		config.G.RabbitMQ.GetURI(),
		config.G.RabbitMQ.Queue,
		task.PollingTrpcDeployStatus,
		task.PollingTrpcDeployStatusArgs{
			WorkspaceID:     app.WorkspaceID,
			AppID:           app.ID,
			EnvName:         uriInput.EnvName,
			TrafficLaneName: input.TrafficLaneName,
			DeployID:        deployID,
		},
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "apply polling deploy status task"))
		return
	}

	ginutils.OK(c, serializer.EmptyOutput{})
}

// deleteAppModelDeploy 删除 AppModel 应用部署
func (h *Handler) deleteAppModelDeploy(c *gin.Context) {
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

	// 执行部署操作
	deployer := h.newDeployer(app)
	if err = deployer.Uninstall(ctx, uriInput.EnvName, input.TrafficLaneName); err != nil {
		deployInfo := genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, input.TrafficLaneName)
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "uninstall app model app: %s", deployInfo,
		))
		return
	}

	// 记录应用从环境卸载
	deploypkg.TrackEnvRemoveApp(
		ctx, h.registry.EnvStore, app.WorkspaceID, uriInput.EnvName, input.TrafficLaneName, app.ID,
	)
	// 应用已从环境卸载，需同步清理该应用在该环境下关联的告警策略，避免残留的无效告警。
	// 先查询 workspace 信息（清理时需要其标识），查询失败仅记录日志，不影响卸载主流程。
	if ws, wsErr := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID); wsErr != nil {
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
			input.TrafficLaneName,
			auth.MustGetUser(ctx).ID,
		)
		// TODO(alertstrategy): 用 go 裸起 goroutine 无法保证跨 Pod 串行，
		// 后续迁移到 asynq 任务队列以解决多 Pod 并发风险。
		go alertstrategy.NewService(
			h.registry.AlertStrategyStore,
			h.registry.EnvStore,
			h.registry.AppStore,
			h.registry.ResourceSnapshotStore,
		).CleanupStrategiesForAppInEnv(
			context.WithoutCancel(ctx),
			ws,
			app.ID,
			env.ID,
			input.TrafficLaneName,
			auth.MustGetUser(ctx).ID,
		)
	}

	// 清理资源拓扑快照（失败不阻塞主流程）
	if err = h.registry.ResourceSnapshotStore.Delete(
		ctx, app.ID, uriInput.EnvName, input.TrafficLaneName,
	); err != nil {
		log.Errorf(
			ctx, "delete resource snapshot for app %s env %s lane %s failed: %v",
			app.ID, uriInput.EnvName, input.TrafficLaneName, err,
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
		audit.WithEnvName(uriInput.EnvName),
		audit.WithExtras(map[string]string{"trafficLaneName": input.TrafficLaneName}),
	)
	ginutils.OK(c, serializer.EmptyOutput{})
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
	rows, total, err := h.registry.AppModelDeployResourceSnapshotStore.ListMetaByDeployRecord(
		ctx, app.ID, uriInput.DeployID, input.Page, input.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			err, bkerrs.ErrCodeInternalServerError, "list app model resource snapshots",
		))
		return
	}
	outputs := make([]*serializer.AppModelResourceSnapshot, 0, len(rows))
	for _, row := range rows {
		outputs = append(outputs, new(serializer.AppModelResourceSnapshot).FromModel(row, false))
	}
	ginutils.OK(c, serializer.ListAppModelResourceSnapshotsOutput{
		Data: serializer.PaginatedAppModelResourceSnapshotsOutputObjs{Count: total, Results: outputs},
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
	row, err := h.registry.AppModelDeployResourceSnapshotStore.GetByID(
		ctx, app.ID, uriInput.DeployID, uriInput.SnapshotID,
	)
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
	latestBuildAutoDeploy, err := h.registry.BuildAutoDeployRecordStore.GetLatest(ctx, appID, envName, trafficLaneName)
	if err != nil && !errors.Is(err, autodeploy.ErrRecordNotFound) {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get latest build auto deploy record")
	}
	if errors.Is(err, autodeploy.ErrRecordNotFound) {
		latestBuildAutoDeploy = nil
	}

	// 获取最新一次应用部署结果
	latestDeploy, err := h.registry.AppModelDeployRecordStore.GetLatest(ctx, appID, envName, trafficLaneName)
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
	snapshotService := snapshot.NewService(reg.SnapshotStore, reg.BuildConfigStore, reg.AppStore)

	return appmodeldeploysvc.NewService(appmodeldeploysvc.ServiceDeps{
		AppStore:                            reg.AppStore,
		EnvStore:                            reg.EnvStore,
		PromotionStore:                      reg.PromotionStore,
		SnapshotService:                     snapshotService,
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

// newDeployer 创建 AppModel 部署器
func (h *Handler) newDeployer(app *bkmsapp.Application) *appmodeldeploy.Deployer {
	return appmodeldeploy.NewDeployer(
		h.registry.AppModelDeployRecordStore,
		h.registry.AppModelDeployResourceSnapshotStore,
		h.registry.BuildAutoDeployRecordStore,
		h.registry.AppModelStore,
		workload.NewBuilderService(
			h.registry.ScopedEnvVarStore,
			h.registry.AppDepsVarReader,
			h.registry.PolarisVarReader,
			h.registry.WorkspaceCompsStore,
			h.registry.PolarisConfigStore,
			h.registry.BscpCfgStore,
			h.registry.AppModelStore,
			h.registry.AppSpecStore,
			h.registry.BuildConfigStore,
		),
		h.registry.AppSpecStore,
		h.registry.BuildConfigStore,
		h.registry.AppConfigFileStore,
		polaris.NewPolarisEnvStateManager(h.registry.PolarisConfigStore),
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
