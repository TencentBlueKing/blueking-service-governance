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

// Package handler 包含 Helm Chart Gin API 的 handler。
package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	helmchartbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/semver"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/worker"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/task"
	helmrepo "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/source"
)

// Handler 处理 Helm Chart Gin API 请求。
type Handler struct {
	registry *storereg.Registry
}

// New 创建 Handler。
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// CreateHelmChartBuild 触发 Helm Chart 构建（从 Git 源码构建，落库 + 异步轮询）。
//
//	@ID			CreateHelmChartBuild
//	@Summary	触发 Helm Chart 构建
//	@Tags		helm-charts
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path	string								true	"应用 ID"
//	@Param		body	body	serializer.CreateHelmChartBuildInput	true	"触发 Helm Chart 构建请求"
//	@Success	200		{object}	serializer.CreateHelmChartBuildOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts/builds [post]
func (h *Handler) CreateHelmChartBuild(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.CreateHelmChartBuildInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateHelmApp(ctx, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	buildSvc := helmchartbuild.NewChartBuildService(
		h.registry.HelmChartSemverCounterStore,
		h.registry.HelmRepoCredentialStore,
		h.registry.HelmChartBuildRecordStore,
		h.registry.BkCIProjectStore,
		h.registry.BkRepoProjectStore,
		h.registry.BkCIPipelineStore,
	)
	result, err := buildSvc.ExecuteChartBuild(ctx, app, input.BumpType, input.Branch)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "trigger helm chart build"))
		return
	}

	if _, err = worker.ApplyTask(
		ctx,
		config.G.RabbitMQ.GetURI(),
		config.G.RabbitMQ.Queue,
		task.PollingHelmChartBuildStatus,
		task.PollingHelmChartBuildStatusArgs{
			WorkspaceID: app.WorkspaceID,
			AppID:       app.ID,
			BuildID:     result.BuildID,
		},
	); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			err, bkerrs.ErrCodeInternalServerError, "apply polling helm chart build status task",
		))
		return
	}

	ginutils.OK(c, serializer.CreateHelmChartBuildOutput{
		Data: &serializer.CreateHelmChartBuildOutputObj{
			ChartVersion: result.ChartVersion,
			BuildID:      result.BuildID,
		},
	})
}

// GetHelmChartSemver 查询 Helm Chart semver counter 当前值。
//
//	@ID			GetHelmChartSemver
//	@Summary	查询 Helm Chart semver counter 当前值
//	@Tags		helm-charts
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path	string	true	"应用 ID"
//	@Param		bumpType	query	string	false	"semver 递增段类型，可选值：patch/minor/major"
//	@Success	200			{object}	serializer.GetHelmChartSemverOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts/semver [get]
func (h *Handler) GetHelmChartSemver(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var queryInput serializer.GetHelmChartSemverQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateHelmApp(ctx, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	counter, err := h.registry.HelmChartSemverCounterStore.Get(ctx, app.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get helm chart semver"))
		return
	}

	data := &serializer.GetHelmChartSemverOutputObj{
		Latest: new(serializer.SemverOutputObj).FromCounter(counter),
	}
	if queryInput.BumpType != "" {
		next := counter.PreviewNext(semver.BumpType(queryInput.BumpType))
		data.Next = new(serializer.SemverOutputObj).FromCounter(next)
	}

	ginutils.OK(c, serializer.GetHelmChartSemverOutput{Data: data})
}

// ListAppHelmCharts 获取 Helm Chart 制品列表。
//
//	@ID			ListAppHelmCharts
//	@Summary	获取 Helm Chart 制品列表
//	@Tags		helm-charts
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path	string	true	"应用 ID"
//	@Param		keyword		query	string	false	"搜索关键字，按版本号模糊匹配"
//	@Param		page		query	int		true	"分页页码（从 1 开始）"
//	@Param		pageSize	query	int		true	"分页大小，可选值：5/10/20/50/100"
//	@Success	200			{object}	serializer.ListAppHelmChartsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts [get]
//
// 数据源说明：直接拉取应用对应 Helm Repo 的 index.yaml 并解析，
// 以真实反映“当前仓库里真正可用的 Chart 版本”（构建记录仅表示构建过、
// 不代表制品已成功推送到 Helm Repo）。
//
// 为什么对数据做内存过滤/分页而非 DB 分页（与镜像列表的设计差异）：
//   - 镜像列表（ListAppImages）：单仓库 tag 数量可能达成百上千，且每个 tag 的
//     digest/size/builtAt 需额外调用 Registry API 获取，存在 N+1 与高延迟
//     问题 → 采用本地 DB 快照 + 异步刷新 + DB 分页。
//   - Chart 列表：index.yaml 一次 HTTP 请求即可返回全部版本及其
//     digest/created 元数据，数据量小、无 N+1 问题 → 采用实时拉取 + 内存过滤/分页，
//     无需落库缓存。
func (h *Handler) ListAppHelmCharts(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var queryInput serializer.ListQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateHelmApp(ctx, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	repoConfig, err := h.resolveRepoConfig(ctx, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "resolve helm repo config"))
		return
	}

	repoIndex, err := helmrepo.FetchIndex(repoConfig)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "fetch helm repo index for app %s", app.ID,
		))
		return
	}
	paged := repoIndex.ListPaginatedChartEntries(
		repoConfig.ChartName,
		queryInput.Keyword,
		queryInput.Page,
		queryInput.PageSize,
	)

	envTypeMap, err := h.registry.EnvStore.GetEnvTypeMap(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list environments"))
		return
	}
	deployedEnvsMap, err := h.buildHelmChartDeployedEnvsMap(ctx, app.ID, envTypeMap)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ginutils.OK(c, serializer.ListAppHelmChartsOutput{
		Data: &serializer.PaginatedAppHelmChartsOutputObjs{
			Count: paged.TotalCount,
			Results: lo.Map(
				paged.Entries,
				func(entry helmrepo.ChartEntry, _ int) *serializer.AppHelmChartOutputObj {
					return new(serializer.AppHelmChartOutputObj).FromChartEntry(entry, deployedEnvsMap[entry.Version])
				},
			),
		},
	})
}

// ListHelmChartBuildRecords 获取 Helm Chart 构建记录列表。
//
//	@ID			ListHelmChartBuildRecords
//	@Summary	获取 Helm Chart 构建记录列表
//	@Tags		helm-charts
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path	string	true	"应用 ID"
//	@Param		keyword		query	string	false	"搜索关键字，按版本号 / 构建号 / 操作人模糊匹配"
//	@Param		page		query	int		true	"分页页码（从 1 开始）"
//	@Param		pageSize	query	int		true	"分页大小，可选值：5/10/20/50/100"
//	@Success	200			{object}	serializer.ListHelmChartBuildRecordsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts/builds [get]
func (h *Handler) ListHelmChartBuildRecords(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var queryInput serializer.ListQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateHelmApp(ctx, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	records, total, err := h.registry.HelmChartBuildRecordStore.List(
		ctx, app.ID, queryInput.Keyword, queryInput.Page, queryInput.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list helm chart build records for app %s", app.ID,
		))
		return
	}

	ginutils.OK(c, serializer.ListHelmChartBuildRecordsOutput{
		Data: &serializer.PaginatedHelmChartBuildRecordOutputObjs{
			Count: total,
			Results: lo.Map(
				records,
				func(record helmchartbuild.Record, _ int) *serializer.HelmChartBuildRecordOutputObj {
					return new(serializer.HelmChartBuildRecordOutputObj).FromBuildRecord(record)
				},
			),
		},
	})
}

// GetHelmChartFiles 获取指定 Helm Chart 版本的文件树（含文本文件内容）。
//
//	@ID			GetHelmChartFiles
//	@Summary	获取指定 Helm Chart 版本的文件树
//	@Tags		helm-charts
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path	string	true	"应用 ID"
//	@Param		chartVersion	path	string	true	"Chart 版本号"
//	@Success	200				{object}	serializer.GetHelmChartFilesOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts/{chartVersion}/files [get]
func (h *Handler) GetHelmChartFiles(c *gin.Context) {
	var uriInput serializer.ChartVersionURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateHelmApp(ctx, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	repoConfig, err := h.resolveRepoConfig(ctx, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "resolve helm repo config"))
		return
	}

	reader := helmrepo.NewReader(repoConfig)
	root, err := reader.ReadChartTree(helmrepo.Version{Name: uriInput.ChartVersion})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError,
			"read chart tree for app %s version %s", app.ID, uriInput.ChartVersion,
		))
		return
	}

	ginutils.OK(c, serializer.GetHelmChartFilesOutput{
		Data: &serializer.GetHelmChartFilesOutputObj{
			ChartName:    repoConfig.ChartName,
			ChartVersion: uriInput.ChartVersion,
			Root:         new(serializer.HelmChartFileNode).FromRepoFileNode(root),
		},
	})
}

// ListChartVersions 获取 Helm Chart 版本列表。
//
//	@ID			ListChartVersions
//	@Summary	获取 Helm Chart 版本列表
//	@Tags		helm-charts
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path	string	true	"应用 ID"
//	@Success	200		{object}	serializer.ListChartVersionsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts/versions [get]
func (h *Handler) ListChartVersions(c *gin.Context) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateHelmApp(ctx, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	repoConfig, err := h.resolveRepoConfig(ctx, app)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "resolve helm repo config"))
		return
	}

	reader := helmrepo.NewReader(repoConfig)
	versions, err := reader.ListVersions()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "listing versions"))
		return
	}

	ginutils.OK(c, serializer.ListChartVersionsOutput{
		Data: lo.Map(versions, func(version helmrepo.Version, _ int) *serializer.ChartVersionOutputObj {
			return new(serializer.ChartVersionOutputObj).FromRepoVersion(version)
		}),
	})
}

// GetValuesFile 获取 values 文件。
//
//	@ID			GetValuesFile
//	@Summary	获取 values 文件
//	@Tags		helm-charts
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path	string	true	"应用 ID"
//	@Param		chartVersion	path	string	true	"Chart 版本号"
//	@Success	200				{object}	serializer.GetValuesFileOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts/{chartVersion}/valuesfile [get]
func (h *Handler) GetValuesFile(c *gin.Context) {
	var uriInput serializer.ChartVersionURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateHelmApp(ctx, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	repoConfig, err := h.resolveRepoConfig(ctx, app)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	reader := helmrepo.NewReader(repoConfig)
	content, err := reader.ReadFile(helmrepo.Version{Name: uriInput.ChartVersion}, "values.yaml")
	if err != nil {
		log.Error(ctx, errors.Wrap(err, "reading app config file").Error())
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "reading app config file"))
		return
	}

	ginutils.OK(c, serializer.GetValuesFileOutput{Data: string(content)})
}

// validateHelmApp 根据 appID 校验应用、检查指定权限，并确保应用类型基于 Helm。
func (h *Handler) validateHelmApp(
	ctx context.Context,
	appID string,
	permType perm.Type,
) (*bkmsapp.Application, error) {
	app, err := perm.ValidateAppByID(ctx, h.registry, appID, permType)
	if err != nil {
		return nil, err
	}
	if !bkmsapp.IsHelmBasedType(app.Type) {
		return nil, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "app type is not helm based, but %s", app.Type)
	}
	return app, nil
}

// resolveRepoConfig 将应用 Helm source 解析为可访问的 Helm repo 配置。
func (h *Handler) resolveRepoConfig(
	ctx context.Context,
	app *bkmsapp.Application,
) (*bkmsapp.HelmRepoConfig, error) {
	return helmrepo.ResolveConfig(
		ctx,
		h.registry.BkCIProjectStore,
		h.registry.BkRepoProjectStore,
		h.registry.HelmRepoCredentialStore,
		app,
	)
}

// buildHelmChartDeployedEnvsMap 构建 chartVersion 到已部署环境列表的映射。
func (h *Handler) buildHelmChartDeployedEnvsMap(
	ctx context.Context,
	appID string,
	envTypeMap map[string]string,
) (map[string][]*serializer.DeployedEnvInfo, error) {
	pairs, err := h.registry.HelmDeployRecordStore.ListChartVersionDeployedEnvs(ctx, appID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list chart version deploy envs")
	}

	m := make(map[string][]*serializer.DeployedEnvInfo, len(pairs))
	for _, pair := range pairs {
		envType, exists := envTypeMap[pair.EnvName]
		if !exists {
			continue
		}
		m[pair.ChartVersion] = append(
			m[pair.ChartVersion],
			&serializer.DeployedEnvInfo{EnvName: pair.EnvName, EnvType: envType},
		)
	}
	return m, nil
}
