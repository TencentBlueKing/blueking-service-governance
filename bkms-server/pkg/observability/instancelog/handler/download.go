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

// Package handler contains Gin handlers for instance log APIs.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	httpresp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/http"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/instancelog"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// downloadAppInstanceLogsURIInput is the path input for downloading app instance logs.
type downloadAppInstanceLogsURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
	// 实例 ID
	InstanceID string `uri:"instanceID" binding:"required,min=1"`
}

// downloadAppInstanceLogsQueryInput is the query input for downloading app instance logs.
type downloadAppInstanceLogsQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
	// 是否获取重启前日志
	Previous bool `form:"previous"`
}

var _ instancelog.Handler = (*Handler)(nil)

// Handler handles Gin instance log API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// DownloadAppInstanceLogs 下载应用运行实例（Pod）日志
//
//	@ID				DownloadAppInstanceLogs
//	@Summary		下载应用运行实例（Pod）日志
//	@Tags			instance-log
//	@Produce		octet-stream
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			appID				path		string	true	"应用 ID"
//	@Param			envName				path		string	true	"部署环境名称"
//	@Param			instanceID			path		string	true	"实例 ID"
//	@Param			trafficLaneName		query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Param			previous			query		bool	false	"是否获取重启前日志"
//	@Success		200					{string}	string	"binary log stream"
//	@Failure		400					{object}	bkerrs.GinErrorOutput
//	@Failure		404					{object}	bkerrs.GinErrorOutput
//	@Failure		500					{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/envs/{envName}/instances/{instanceID}/logs/download [get]
func (h *Handler) DownloadAppInstanceLogs(c *gin.Context) {
	var uriInput downloadAppInstanceLogsURIInput
	var queryInput downloadAppInstanceLogsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr, err := instancelog.NewLogManager(
		ctx,
		h.registry.AppModelDeployRecordStore,
		app,
		env,
		queryInput.TrafficLaneName,
		uriInput.InstanceID,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, instancelog.WrapManagerError(
			err,
			uriInput.AppID,
			uriInput.EnvName,
			uriInput.InstanceID))
		return
	}

	result, err := mgr.PrepareDownload(ctx, uriInput.InstanceID, queryInput.Previous)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "prepare download app instance logs"),
		)
		return
	}
	defer result.Reader.Close()

	c.DataFromReader(
		http.StatusOK,
		// Unknown content length: let Gin stream the response.
		-1,
		httpresp.AttachmentContentType,
		result.Reader,
		map[string]string{
			"Content-Disposition": httpresp.BuildAttachmentDisposition(result.Filename),
		},
	)
}
