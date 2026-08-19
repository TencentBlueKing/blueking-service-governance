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

// Package handler 包含部署相关 Gin API 的 handler。
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	deploypkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/helmdeploypoll"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// ListHelmDeployRecords 获取 Helm 应用部署记录列表。
//
//	@ID			ListHelmDeployRecords
//	@Summary	获取 Helm 应用部署记录列表
//	@Tags		deploy
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID				path		string	true	"应用 ID"
//	@Param		envName				path		string	true	"部署环境名称"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Param		keyword				query		string	false	"搜索关键字"
//	@Param		page				query		int		true	"分页页码（从 1 开始）"
//	@Param		pageSize			query		int		true	"分页大小"
//	@Success	200					{object}	serializer.ListHelmDeployRecordsOutput
//	@Failure	400					{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/helm-deploys [get]
func (h *Handler) ListHelmDeployRecords(c *gin.Context) {
	var uriInput serializer.HelmDeployURIInput
	var queryInput serializer.ListHelmDeployRecordsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	records, total, err := h.registry.HelmDeployRecordStore.List(
		ctx,
		app.ID,
		uriInput.EnvName,
		queryInput.TrafficLaneName,
		queryInput.Keyword,
		queryInput.Page,
		queryInput.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err,
			bkerrs.ErrCodeInternalServerError,
			"list deploy records for %s",
			genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, queryInput.TrafficLaneName),
		))
		return
	}

	outputRecords := make([]*serializer.HelmDeployRecordOutputObj, 0, len(records))
	for _, record := range records {
		outputRecords = append(outputRecords, new(serializer.HelmDeployRecordOutputObj).FromModel(record))
	}
	if err = h.attachHelmDeployRecordsValues(
		ctx, app, uriInput.EnvName, queryInput.TrafficLaneName, outputRecords,
	); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "attach deploy record values"))
		return
	}

	ginutils.OK(c, serializer.ListHelmDeployRecordsOutput{
		Data: &serializer.PaginatedHelmDeployRecordOutputObjs{
			Count:   total,
			Results: outputRecords,
		},
	})
}

// PreviewHelmDeploy 预览 Helm 应用部署。
//
//	@ID			PreviewHelmDeploy
//	@Summary	部署 Helm 应用预览
//	@Tags		deploy
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID				path		string	true	"应用 ID"
//	@Param		envName				path		string	true	"部署环境名称"
//	@Param		imageTag			query		string	true	"目标镜像 TAG"
//	@Param		chartVersion		query		string	true	"指定的部署的 Chart 版本"
//	@Param		valuesFileID		query		string	true	"部署使用的 ValuesFile ID"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200					{object}	serializer.PreviewHelmDeployOutput
//	@Failure	400					{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/helm-deploys/preview [get]
func (h *Handler) PreviewHelmDeploy(c *gin.Context) {
	var uriInput serializer.HelmDeployURIInput
	var queryInput serializer.PreviewHelmDeployQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := h.validateHelmDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 预览待部署的 Helm Chart 的 Manifests 差异
	envVarsReader := envvars.NewUnifiedEnvVarsReader(
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	)
	result, err := helmdeploy.PreviewHelmRelease(
		ctx,
		h.registry.BkCIProjectStore,
		h.registry.BkRepoProjectStore,
		h.registry.HelmRepoCredentialStore,
		envVarsReader,
		app,
		env,
		queryInput.TrafficLaneName,
		queryInput.ChartVersion,
		queryInput.ValuesFileID,
		queryInput.ImageTag,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "preview helm release manifests"))
		return
	}

	ginutils.OK(c, serializer.PreviewHelmDeployOutput{
		Current:        result.CurrentManifests,
		Target:         result.TargetManifests,
		MissingVars:    result.MissingVars,
		MissingEnvVars: result.MissingEnvVars,
	})
}

// CreateHelmDeploy 部署 Helm 应用。
//
//	@ID			CreateHelmDeploy
//	@Summary	部署 Helm 应用
//	@Tags		deploy
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		envName	path		string							true	"部署环境名称"
//	@Param		body	body		serializer.CreateHelmDeployInput	true	"部署 Helm 应用请求"
//	@Success	201		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/helm-deploys [post]
func (h *Handler) CreateHelmDeploy(c *gin.Context) {
	var uriInput serializer.HelmDeployURIInput
	var input serializer.CreateHelmDeployInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := h.validateHelmDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 进行锁检查防止并发部署，如果出错则需要及时释放
	deployLock := helmdeploy.NewDeployLock(app.ID, env.Name, input.TrafficLaneName)
	if ok := deployLock.Acquire(ctx); !ok {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeAborted, "concurrency deploy conflict occurred"))
		return
	}

	deploypkg.TrackEnvAddApp(ctx, h.registry.EnvStore, app.WorkspaceID, uriInput.EnvName, app.ID)
	// 执行部署的若干步骤
	deployID, err := h.execHelmAppDeploySteps(
		ctx, app, env, input.TrafficLaneName, input.ChartVersion, input.ValuesFileID, input.ImageTag,
	)
	if err != nil {
		deployLock.Release(ctx)
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "execute deploy steps"))
		return
	}

	// 轮询部署状态 & 更新部署记录
	err = taskq.Enqueue(
		ctx,
		helmdeploypoll.Task.NewTask(helmdeploypoll.Args{
			WorkspaceID:     app.WorkspaceID,
			AppID:           app.ID,
			EnvName:         uriInput.EnvName,
			TrafficLaneName: input.TrafficLaneName,
			DeployID:        deployID,
		}),
		asynq.ProcessIn(helmdeploypoll.PollingInterval()),
	)
	if err != nil {
		deployLock.Release(ctx)
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			err, bkerrs.ErrCodeInternalServerError, "enqueue polling deploy status task",
		))
		return
	}

	ginutils.JSON(c, http.StatusCreated, serializer.EmptyOutput{})
}

// PreviewRollbackHelmDeploy 预览 Helm 部署版本回滚。
//
//	@ID			PreviewRollbackHelmDeploy
//	@Summary	Helm 部署版本回滚预览
//	@Tags		deploy
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID				path		string	true	"应用 ID"
//	@Param		envName				path		string	true	"部署环境名称"
//	@Param		deployID			path		string	true	"准备回滚到的记录 ID"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200					{object}	serializer.PreviewHelmDeployOutput
//	@Failure	400					{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/helm-deploys/{deployID}/preview [get]
func (h *Handler) PreviewRollbackHelmDeploy(c *gin.Context) {
	var uriInput serializer.HelmDeployRecordURIInput
	var queryInput serializer.HelmDeployTrafficLaneQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取回滚目标部署记录
	record, err := h.getDeployRecordForRollback(ctx, app.ID, uriInput.DeployID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "get deploy record"))
		return
	}

	// 获取版本之间的 Manifests 差异
	result, err := helmdeploy.PreviewRollbackHelmRelease(ctx, record)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "diff helm release manifests"))
		return
	}

	ginutils.OK(c, serializer.PreviewHelmDeployOutput{
		Current: result.CurrentManifests,
		Target:  result.TargetManifests,
	})
}

// RollbackHelmDeploy 回滚到指定 Helm 部署版本。
//
//	@ID			RollbackHelmDeploy
//	@Summary	Helm 回滚到指定的部署版本
//	@Tags		deploy
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		envName	path		string							true	"部署环境名称"
//	@Param		deployID	path		string							true	"部署记录 ID"
//	@Param		body	body		serializer.RollbackHelmDeployInput	false	"Helm 回滚请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/helm-deploys/{deployID} [put]
func (h *Handler) RollbackHelmDeploy(c *gin.Context) {
	var uriInput serializer.HelmDeployRecordURIInput
	var input serializer.RollbackHelmDeployInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := h.bindOptionalJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取回滚目标部署记录
	record, err := h.getDeployRecordForRollback(ctx, app.ID, uriInput.DeployID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "get deploy record"))
		return
	}

	// 进行锁检查防止并发部署，如果出错则需要及时释放
	deployLock := helmdeploy.NewDeployLock(app.ID, uriInput.EnvName, input.TrafficLaneName)
	if ok := deployLock.Acquire(ctx); !ok {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeAborted, "concurrency deploy conflict occurred"))
		return
	}

	// 执行回滚相关操作
	deployID, err := h.execHelmAppRollbackSteps(ctx, record)
	if err != nil {
		deployLock.Release(ctx)
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "execute rollback steps"))
		return
	}

	// 轮询部署状态 & 更新部署记录
	err = taskq.Enqueue(
		ctx,
		helmdeploypoll.Task.NewTask(helmdeploypoll.Args{
			WorkspaceID:     app.WorkspaceID,
			AppID:           app.ID,
			EnvName:         uriInput.EnvName,
			TrafficLaneName: input.TrafficLaneName,
			DeployID:        deployID,
		}),
		asynq.ProcessIn(helmdeploypoll.PollingInterval()),
	)
	if err != nil {
		deployLock.Release(ctx)
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			err, bkerrs.ErrCodeInternalServerError, "enqueue polling deploy status task",
		))
		return
	}

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeleteHelmDeploy 删除 Helm 应用部署。
//
//	@ID			DeleteHelmDeploy
//	@Summary	删除 Helm 应用部署
//	@Tags		deploy
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID				path		string	true	"应用 ID"
//	@Param		envName				path		string	true	"部署环境名称"
//	@Param		deployID			path		string	true	"部署记录 ID"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200					{object}	serializer.EmptyOutput
//	@Failure	400					{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/helm-deploys/{deployID} [delete]
func (h *Handler) DeleteHelmDeploy(c *gin.Context) {
	startedAt := time.Now()
	metricStatus := metrics.StatusOK
	var uriInput serializer.HelmDeployRecordURIInput
	var queryInput serializer.HelmDeployTrafficLaneQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	defer func() {
		metrics.DeployUninstallFinished(metrics.DeployKindHelm, metricStatus, startedAt)
	}()

	// 找到最新部署记录
	record, err := h.registry.HelmDeployRecordStore.GetLatest(
		ctx, app.ID, uriInput.EnvName, queryInput.TrafficLaneName,
	)
	if err != nil {
		metricStatus = metrics.StatusErr
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get latest deploy record"))
		return
	}
	// 检查提供的部署记录 ID 是否与最新一次的一致
	if record.ID.Hex() != uriInput.DeployID {
		metricStatus = metrics.StatusErr
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidArgument, "must provide latest deploy record id"))
		return
	}

	// 对部署进行删除操作
	deployInfo := genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, queryInput.TrafficLaneName)
	if err = helmdeploy.UninstallHelmRelease(ctx, record); err != nil {
		metricStatus = metrics.StatusErr
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "uninstall helm release for %s", deployInfo,
		))
		return
	}
	// 修改状态为已卸载
	record.Status = helm.StatusUninstalled
	if err = h.registry.HelmDeployRecordStore.Update(ctx, record); err != nil {
		metricStatus = metrics.StatusErr
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "update deploy record for %s", deployInfo,
		))
		return
	}

	// 记录应用从环境卸载
	deploypkg.TrackEnvRemoveApp(
		ctx, h.registry.EnvStore, app.WorkspaceID, uriInput.EnvName, queryInput.TrafficLaneName, app.ID,
	)
	// 应用已从环境卸载，需同步清理该应用在该环境下关联的告警策略，避免残留无效告警。
	// 先查询 workspace 与 env 信息（清理时需要其标识）；任一查询失败仅记录日志，不中断卸载主流程。
	ws, wsErr := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if wsErr != nil {
		log.Errorf(ctx, "get workspace %s for alert cleanup failed: %v", app.WorkspaceID, wsErr)
	} else {
		env, envErr := h.registry.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, uriInput.EnvName)
		if envErr != nil {
			log.Errorf(ctx, "get env %s for alert cleanup failed: %v", uriInput.EnvName, envErr)
		} else {
			// 异步执行清理，避免阻塞卸载接口响应
			log.Infof(
				ctx,
				"schedule alert strategy cleanup after uninstall, workspace=%s app=%s env=%s envID=%s lane=%s operator=%s",
				ws.ID,
				app.ID,
				env.Name,
				env.ID.Hex(),
				queryInput.TrafficLaneName,
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
				queryInput.TrafficLaneName,
				auth.MustGetUser(ctx).ID,
			)
		}
	}

	// 清理资源拓扑快照（失败不阻塞主流程）
	if err = h.registry.ResourceSnapshotStore.Delete(
		ctx, app.ID, uriInput.EnvName, queryInput.TrafficLaneName,
	); err != nil {
		log.Errorf(
			ctx, "delete resource snapshot for app %s env %s lane %s failed: %v",
			app.ID, uriInput.EnvName, queryInput.TrafficLaneName, err,
		)
	}

	ginutils.OK(c, serializer.EmptyOutput{})
}
