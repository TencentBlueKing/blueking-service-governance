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
	buildserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build/serializer"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/log"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// buildLogDeps 构建日志请求的执行依赖
type buildLogDeps struct {
	// Service 构建日志服务，封装日志查询与聚合逻辑。
	Service *log.Service
	// Query 构建日志查询对象，包含项目代码、流水线 ID、构建 ID 等信息
	Query *log.BuildLogQuery
	// BKCIClient 蓝盾 BKCI API 客户端，用于拉取构建日志
	BKCIClient bkciapi.Client
}

// StreamBuildLogs 流式推送应用构建日志（SSE）
//
//	@ID				StreamBuildLogs
//	@Summary		流式推送应用构建日志
//	@Tags			builds
//	@Produce		text/event-stream
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path	string	true	"应用 ID"
//	@Param			buildID		path	string	true	"蓝盾构建 ID"
//	@Success		200			{string}	string	"SSE event stream"
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/builds/{buildID}/logs/stream [get]
func (h *Handler) StreamBuildLogs(c *gin.Context) {
	deps, err := h.prepareBuildLogDeps(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	log.WriteSSEBuildLogs(c, deps.Service, deps.BKCIClient, deps.Query, buildLogSSEErrorOutput)
}

// DownloadBuildLogs 下载应用构建日志
//
//	@ID				DownloadBuildLogs
//	@Summary		下载应用构建日志
//	@Tags			builds
//	@Produce		octet-stream
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID		path	string	true	"应用 ID"
//	@Param			buildID		path	string	true	"蓝盾构建 ID"
//	@Success		200			{string}	string	"binary log stream"
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/builds/{buildID}/logs/download [get]
func (h *Handler) DownloadBuildLogs(c *gin.Context) {
	deps, err := h.prepareBuildLogDeps(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := log.WriteDownloadBuildLogs(c, deps.Service, deps.BKCIClient, deps.Query); err != nil {
		bkerrs.AbortWithErr(c, wrapBuildLogHTTPError(err, deps.Query))
		return
	}
}

// prepareBuildLogDeps 为应用构建日志请求准备执行依赖。
func (h *Handler) prepareBuildLogDeps(c *gin.Context) (*buildLogDeps, error) {
	var uriInput buildserializer.BuildLogURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		return nil, err
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		return nil, err
	}

	bkciClient, err := bkciapi.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init bkci client")
	}

	record, err := h.registry.BuildRecordStore.Get(ctx, app.ID, uriInput.BuildID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get build record")
	}
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, record.WorkspaceID)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get bkci project")
	}

	svc := log.NewService()
	query := buildLogQueryFromRecord(app.ID, uriInput.BuildID, record, project)

	return &buildLogDeps{
		Service:    svc,
		Query:      query,
		BKCIClient: bkciClient,
	}, nil
}

func buildLogQueryFromRecord(
	appID, buildID string,
	record *imagebuild.Record,
	project *bkciproject.Project,
) *log.BuildLogQuery {
	return &log.BuildLogQuery{
		ProjectCode: project.Code,
		PipelineID:  record.PipelineID,
		BuildID:     buildID,
		AppID:       appID,
	}
}

func wrapBuildLogHTTPError(err error, query *log.BuildLogQuery) *bkerrs.Error {
	switch {
	// 过期或已清理日志在蓝盾无法拉取，特殊处理页面提示
	case errors.Is(err, bkciapi.BuildLogExpired), errors.Is(err, bkciapi.BuildLogCleaned):
		return bkerrs.WrapBuildLogUnavailable(err, query.AppID, query.BuildID)
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "aggregate build logs")
	}
}

func buildLogSSEErrorOutput(err error, query *log.BuildLogQuery) any {
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
