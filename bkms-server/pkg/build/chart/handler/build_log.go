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
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	bkciproject "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	helmchartbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/serializer"
	buildlog "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/log"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

type buildLogDeps struct {
	Service    *buildlog.Service
	Query      *buildlog.BuildLogQuery
	BKCIClient bkciapi.Client
}

// StreamHelmChartBuildLogs 流式推送 Helm Chart 构建日志（SSE）。
//
//	@ID			StreamHelmChartBuildLogs
//	@Summary	流式推送 Helm Chart 构建日志
//	@Tags		helm-charts
//	@Produce	text/event-stream
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path	string	true	"应用 ID"
//	@Param		buildID		path	string	true	"蓝盾构建 ID"
//	@Success	200			{string}	string	"SSE event stream"
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Failure	500			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts/builds/{buildID}/logs/stream [get]
func (h *Handler) StreamHelmChartBuildLogs(c *gin.Context) {
	deps, err := h.prepareBuildLogDeps(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	buildlog.WriteSSEBuildLogs(c, deps.Service, deps.BKCIClient, deps.Query, buildLogSSEErrorOutput)
}

// DownloadHelmChartBuildLogs 下载 Helm Chart 构建日志。
//
//	@ID			DownloadHelmChartBuildLogs
//	@Summary	下载 Helm Chart 构建日志
//	@Tags		helm-charts
//	@Produce	octet-stream
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path	string	true	"应用 ID"
//	@Param		buildID		path	string	true	"蓝盾构建 ID"
//	@Success	200			{string}	string	"binary log stream"
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Failure	500			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/charts/builds/{buildID}/logs/download [get]
func (h *Handler) DownloadHelmChartBuildLogs(c *gin.Context) {
	deps, err := h.prepareBuildLogDeps(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := buildlog.WriteDownloadBuildLogs(c, deps.Service, deps.BKCIClient, deps.Query); err != nil {
		bkerrs.AbortWithErr(c, wrapBuildLogHTTPError(err, deps.Query))
		return
	}
}

// prepareBuildLogDeps 为 Helm Chart 构建日志请求准备执行依赖。
func (h *Handler) prepareBuildLogDeps(
	c *gin.Context,
) (*buildLogDeps, error) {
	var uriInput serializer.BuildLogURIInput
	if err := c.ShouldBindUri(&uriInput); err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "bind uri")
	}

	ctx := c.Request.Context()
	app, err := h.validateHelmApp(ctx, uriInput.AppID, perm.TypeView)
	if err != nil {
		return nil, err
	}

	bkciClient, err := bkciapi.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init bkci client")
	}

	record, err := h.registry.HelmChartBuildRecordStore.Get(ctx, app.ID, uriInput.BuildID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get chart build record")
	}
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, record.WorkspaceID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get bkci project")
	}

	svc := buildlog.NewService()
	query := buildLogQueryFromRecord(app.ID, uriInput.BuildID, record, project)

	return &buildLogDeps{
		Service:    svc,
		Query:      query,
		BKCIClient: bkciClient,
	}, nil
}

func buildLogQueryFromRecord(
	appID, buildID string,
	record *helmchartbuild.Record,
	project *bkciproject.Project,
) *buildlog.BuildLogQuery {
	return &buildlog.BuildLogQuery{
		ProjectCode: project.Code,
		PipelineID:  record.PipelineID,
		BuildID:     buildID,
		AppID:       appID,
	}
}

func wrapBuildLogHTTPError(err error, query *buildlog.BuildLogQuery) *bkerrs.Error {
	switch {
	// 过期或已清理日志在蓝盾无法拉取，特殊处理页面提示
	case errors.Is(err, bkciapi.BuildLogExpired), errors.Is(err, bkciapi.BuildLogCleaned):
		return bkerrs.WrapBuildLogUnavailable(err, query.AppID, query.BuildID)
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "aggregate build logs")
	}
}

func buildLogSSEErrorOutput(err error, query *buildlog.BuildLogQuery) any {
	wrappedErr := wrapBuildLogHTTPError(err, query)
	details := wrappedErr.Details()
	return bkerrs.GinErrorOutput{
		Error: bkerrs.GinError{
			Code:    wrappedErr.Code(),
			Message: wrappedErr.Error(),
			System:  bkerrs.SystemName,
			Module:  bkerrs.ModuleName,
			Details: details.AsMaps(),
		},
	}
}
