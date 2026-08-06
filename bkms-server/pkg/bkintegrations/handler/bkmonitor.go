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

package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	bkmmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking/serializer"
)

// GetApmServiceName 获取 Apm 服务名称
//
//	@ID			GetApmServiceName
//	@Summary	获取应用环境的 APM 服务名称
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		envName		path		string	true	"环境名称"
//	@Success	200			{object}	serializer.GetApmServiceNameResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/bkmonitor/apm-service-name [get]
func (h *Handler) GetApmServiceName(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := ginperm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	appModel, err := h.registry.AppModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "getting app model"))
		return
	}

	_, content, err := appcfg.GetEnvContent(ctx, h.registry.AppConfigFileStore, app.ID, env.Name)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "getting env content"))
		return
	}

	appEnvVars, err := envvars.BuildAppEnvVars(
		ctx, app, appModel, env,
		envvars.NewUnifiedEnvVarsReader(
			h.registry.ScopedEnvVarStore,
			h.registry.AppDepsVarReader,
			h.registry.PolarisVarReader,
		),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "building app env vars"))
		return
	}

	serviceName, svcErr := bkmmodel.GetApmServiceName(app.Type, content, appEnvVars.ToMap())
	if svcErr != nil {
		log.Errorf(ctx, "get apm service name error for app %s env %s: %v", app.ID, env.Name, svcErr)
		if errors.Is(svcErr, bkmmodel.ErrAPMConfigMissing) {
			bkerrs.AbortWithErr(c, bkerrs.WrapAPMConfigMissing(svcErr, app.ID, env.Name))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(svcErr, bkerrs.ErrCodeInternalServerError,
			"resolve apm service name for app %s env %s", app.ID, env.Name))
		return
	}

	ginutils.OK(
		c,
		&serializer.GetApmServiceNameResp{Data: &serializer.GetApmServiceNameOutput{ServiceName: serviceName}},
	)
}

// ListApms 从蓝鲸监控 API 获取该工作空间的所有 Apm 详情
//
//	@ID			ListApms
//	@Summary	获取工作空间下的 APM 列表
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListApmsResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkmonitor/apms [get]
func (h *Handler) ListApms(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	client, err := bkmapi.New(auth.MustGetUser(ctx).ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "new bkmonitor client"))
		return
	}

	resp, err := client.ListApmApp(ctx, cast.ToInt64(ws.BkSystems.BkMonitorProjectID))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list apm app"))
		return
	}

	envList, err := h.registry.EnvStore.ListStdEnvs(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	envMap := lo.SliceToMap(envList, func(item envmodel.Environment) (string, envmodel.Environment) {
		return item.Name, item
	})

	apmMap, err := h.registry.ApmInstConfigStore.GetApmIDMap(ctx, ws.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "list apm from store"))
		return
	}

	results := lo.Map(resp, func(item *bkmapi.ApmApp, _ int) *serializer.ApmOutput {
		data := &serializer.ApmOutput{
			ApmID:       item.ID,
			Type:        bkmmodel.ApmTypeCustom,
			BkBizID:     item.BkBizID,
			Token:       item.Token,
			Name:        item.AppName,
			Description: item.Description,
			Creator:     item.Creator,
		}
		createdAt := cast.ToTime(item.CreatedAt)
		if !createdAt.IsZero() {
			data.CreatedAt = &createdAt
		}
		if envData, ok := envMap[data.Name]; ok {
			data.Type = envData.Type
		}
		if storedApm, ok := apmMap[item.ID]; ok {
			data.AssociatedEnvs = lo.Map(
				storedApm.AssociatedEnvs,
				func(info bkmmodel.EnvInfo, _ int) *serializer.ApmEnvInfoOutput {
					return new(serializer.ApmEnvInfoOutput).FromModel(info)
				},
			)
		}
		if item.MetricConfig != nil && item.MetricConfig.BkDataID != 0 {
			data.MetricReady = true
		}
		if item.TraceConfig != nil && item.TraceConfig.BkDataID != 0 {
			data.TraceReady = true
		}
		if item.LogConfig != nil && item.LogConfig.BkDataID != 0 {
			data.LogReady = true
		}
		if item.ProfilingConfig != nil && item.ProfilingConfig.BkDataID != 0 {
			data.ProfilingReady = true
		}
		return data
	})

	ginutils.OK(c, &serializer.ListApmsResp{Data: &serializer.ListApmOutput{
		Count:   int64(len(results)),
		Results: results,
	}})
}

// CreateEnvApm 为环境创建 APM 并绑定
//
//	@ID			CreateEnvApm
//	@Summary	为环境创建 APM 并绑定
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string	true	"环境 ID"
//	@Success	200		{object}	serializer.CreateEnvApmResp
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/bkmonitor/apms [post]
func (h *Handler) CreateEnvApm(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := ginperm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ws, err := h.registry.WorkspaceStore.Get(ctx, env.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}
	if ws.BkSystems.BkMonitorProjectID == "" {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "bk monitor project is not ready for workspace %s", ws.ID),
		)
		return
	}

	apmSvc := bkmmodel.NewApmService(storereg.G().ApmInstConfigStore, storereg.G().ScopedEnvVarStore)
	apm, err := apmSvc.CreateAndBindToEnv(
		ctx,
		env.ID,
		env.Name,
		ws.BkSystems.BkBCSProjectCode,
		bkmmodel.CreateApmInstParams{
			WorkspaceID:  ws.ID,
			Username:     auth.MustGetUser(ctx).ID,
			BkmProjectID: cast.ToInt64(ws.BkSystems.BkMonitorProjectID),
		},
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create and bind apm to env"))
		return
	}

	// APM 创建成功后，异步将 workspace 下 admin/sre 人员同步到新告警组
	go bkmmodel.NewUserGroupService(perm.NewManager(), storereg.G().EnvStore).SyncMembersForEnvWithRetry(
		context.WithoutCancel(ctx), ws, env.Name, auth.MustGetUser(ctx).ID,
	)

	ginutils.OK(c, &serializer.CreateEnvApmResp{Data: &serializer.ApmOutput{
		ApmID: apm.ApmID,
		Token: apm.Token,
		Name:  apm.Name,
		AssociatedEnvs: lo.Map(apm.AssociatedEnvs, func(info bkmmodel.EnvInfo, _ int) *serializer.ApmEnvInfoOutput {
			return new(serializer.ApmEnvInfoOutput).FromModel(info)
		}),
	}})
}

// BindApmToEnv 将环境绑定到指定 APM 上
//
//	@ID			BindApmToEnv
//	@Summary	将环境绑定到指定 APM
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string	true	"环境 ID"
//	@Param		apmID	path		string	true	"APM ID"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/bkmonitor/apms/{apmID} [put]
func (h *Handler) BindApmToEnv(c *gin.Context) {
	var uriInput serializer.EnvApmURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := ginperm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, ginperm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ws, err := h.registry.WorkspaceStore.Get(ctx, env.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}

	apmSvc := bkmmodel.NewApmService(storereg.G().ApmInstConfigStore, storereg.G().ScopedEnvVarStore)
	apm, err := apmSvc.Get(
		ctx,
		cast.ToInt64(uriInput.ApmID),
		bkmmodel.CreateApmInstParams{
			WorkspaceID:  env.WorkspaceID,
			Username:     auth.MustGetUser(ctx).ID,
			BkmProjectID: cast.ToInt64(ws.BkSystems.BkMonitorProjectID),
		},
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get or create apm"))
		return
	}

	if err = apmSvc.BindToEnv(ctx, apm, env.ID, env.Name); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "bind apm to env"))
		return
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// GetEnvApm 查询环境绑定的 APM
//
//	@ID			GetEnvApm
//	@Summary	查询环境绑定的 APM
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		envID	path		string	true	"环境 ID"
//	@Success	200		{object}	serializer.GetEnvApmResp
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/envs/{envID}/bkmonitor/apms [get]
func (h *Handler) GetEnvApm(c *gin.Context) {
	var uriInput serializer.EnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	env, err := ginperm.ValidateEnvByID(ctx, h.registry, uriInput.EnvID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	apm, err := h.registry.ApmInstConfigStore.GetByEnvID(ctx, env.ID)
	if err != nil {
		if errors.Is(err, bkmmodel.ErrApmInstConfigNotFound) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "apm not found for environment %s", uriInput.EnvID),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get apm by env id"))
		return
	}

	ginutils.OK(c, &serializer.GetEnvApmResp{Data: new(serializer.GetEnvApmOutput).FromModel(*apm)})
}

// GetInstanceTimeSeries 查询实例监控指标时序数据
//
//	@ID			GetInstanceTimeSeries
//	@Summary	查询实例监控指标时序数据
//	@Tags		bkintegrations-bkmonitor
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string		true	"应用 ID"
//	@Param		envName		path		string		true	"环境名称"
//	@Param		instances	query		[]string	true	"实例名称列表"
//	@Param		metricKey	query		string		true	"指标标识"
//	@Param		startTime	query		int			true	"开始时间（Unix 时间戳）"
//	@Param		endTime	    query		int			true	"结束时间（Unix 时间戳）"
//	@Param		interval	query		int			false	"汇聚周期（秒），默认 60"
//	@Success	200			{object}	serializer.InstanceTimeSeriesResp
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/bkmonitor/instance-time-series [get]
func (h *Handler) GetInstanceTimeSeries(c *gin.Context) {
	ctx := c.Request.Context()
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.GetInstanceTimeSeriesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	queryInput.Normalize()
	app, env, err := ginperm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ws, err := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}
	if ws.BkSystems.BkMonitorProjectID == "" {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "bk monitor project is not ready for workspace %s", ws.ID),
		)
		return
	}
	if env.Cluster.ClusterID == "" || env.Cluster.Namespace == "" {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "cluster info is not configured for environment %s", env.Name),
		)
		return
	}
	metricSvc := bkmmodel.NewMetricTimeSeriesService()

	result, err := metricSvc.QueryTimeSeries(ctx, &bkmmodel.MetricsQuery{
		BkBizID:    cast.ToInt64(ws.BkSystems.BkMonitorProjectID),
		ClusterID:  env.Cluster.ClusterID,
		Namespace:  env.Cluster.Namespace,
		Instances:  queryInput.Instances,
		MetricKeys: []string{queryInput.MetricKey},
		StartTime:  queryInput.StartTime,
		EndTime:    queryInput.EndTime,
		Interval:   queryInput.Interval,
		Username:   auth.MustGetUser(ctx).ID,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "query instance metrics"))
		return
	}

	// 转换为响应结构
	respData := make(map[string]*serializer.MetricTimeSeries, len(result.Metrics))
	for key, metric := range result.Metrics {
		series := make([]*serializer.TimeSeriesItem, 0, len(metric.Series))
		for _, s := range metric.Series {
			item := &serializer.TimeSeriesItem{
				Instance:   s.Instance,
				DataPoints: s.DataPoints,
				Stat: &serializer.TimeSeriesItemStat{
					Count: s.Stat.Count,
					Sum:   s.Stat.Sum,
					Min:   s.Stat.Min,
					Max:   s.Stat.Max,
					Avg:   s.Stat.Avg,
					Last:  s.Stat.Last,
				},
			}
			series = append(series, item)
		}
		respData[key] = &serializer.MetricTimeSeries{
			DisplayName: metric.DisplayName,
			Unit:        metric.Unit,
			Series:      series,
		}
	}

	ginutils.OK(c, &serializer.InstanceTimeSeriesResp{Data: respData})
}
