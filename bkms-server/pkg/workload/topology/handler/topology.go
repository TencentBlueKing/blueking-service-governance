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

// Package handler contains Gin handlers for topology APIs.
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology/serializer"
)

var _ topology.Handler = (*Handler)(nil)

// Handler handles Gin topology API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// GetResourceTopology 获取应用资源拓扑
//
//	@ID			GetResourceTopology
//	@Summary	获取应用资源拓扑
//	@Tags		topology
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200				{object}	serializer.GetResourceTopologyOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/resource-topology [get]
func (h *Handler) GetResourceTopology(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.TrafficLaneQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	_, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	graph, err := topology.NewService(
		h.registry.ResourceSnapshotStore,
		topology.NewBuilder(),
		h.registry.AppModelStore,
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	).GetTopology(ctx, uriInput.AppID, uriInput.EnvName, queryInput.TrafficLaneName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get resource topology"))
		return
	}

	ginutils.OK(c, serializer.GetResourceTopologyOutput{
		Data: new(serializer.ResourceTopologyDataOutputObj).FromModel(graph),
	})
}

// GetTopologyNodeDetail 获取节点详情
//
//	@ID			GetTopologyNodeDetail
//	@Summary	获取节点详情
//	@Tags		topology
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		nodeID			path		string	true	"拓扑节点 ID（base64url 无填充编码）"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200				{object}	serializer.GetTopologyNodeDetailOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID} [get]
func (h *Handler) GetTopologyNodeDetail(c *gin.Context) {
	var uriInput serializer.TopologyNodeURIInput
	var queryInput serializer.TrafficLaneQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	_, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	detail, err := topology.NewService(
		h.registry.ResourceSnapshotStore,
		topology.NewBuilder(),
		h.registry.AppModelStore,
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	).GetNodeDetail(ctx, uriInput.AppID, uriInput.EnvName, queryInput.TrafficLaneName, uriInput.NodeID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get node detail"))
		return
	}

	ginutils.OK(c, serializer.GetTopologyNodeDetailOutput{
		Data: new(serializer.TopologyNodeDetailOutputObj).FromModel(detail),
	})
}

// ListTopologyNodeEvents 获取节点事件列表
//
//	@ID			ListTopologyNodeEvents
//	@Summary	获取节点事件列表
//	@Tags		topology
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		nodeID			path		string	true	"拓扑节点 ID（base64url 无填充编码）"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Param		level			query		string	false	"事件级别（可选过滤参数，可选值：Normal, Warning）"
//	@Param		startedAt		query		int		false	"起始时间戳（可选过滤参数，如：1772223278）"
//	@Param		endedAt			query		int		false	"结束时间戳（可选过滤参数，如：1772223278）"
//	@Param		page			query		int		true	"分页页码（从 1 开始）"
//	@Param		pageSize		query		int		true	"每页数量"
//	@Success	200				{object}	serializer.ListTopologyNodeEventsOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID}/events [get]
func (h *Handler) ListTopologyNodeEvents(c *gin.Context) {
	var uriInput serializer.TopologyNodeURIInput
	var queryInput serializer.ListTopologyNodeEventsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	_, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	paginatedEvents, err := topology.NewService(
		h.registry.ResourceSnapshotStore,
		topology.NewBuilder(),
		h.registry.AppModelStore,
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	).ListNodeEvents(
		ctx,
		uriInput.AppID,
		uriInput.EnvName,
		queryInput.TrafficLaneName,
		env.Cluster.ProjectCode,
		uriInput.NodeID,
		queryInput.Level,
		queryInput.StartedAt,
		queryInput.EndedAt,
		queryInput.Page,
		queryInput.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list node events"))
		return
	}

	ginutils.OK(c, serializer.ListTopologyNodeEventsOutput{
		Data: new(serializer.PaginatedTopologyNodeEventsOutputObj).FromModel(paginatedEvents),
	})
}

// GetTopologyNodeManifest 获取节点 Manifest
//
//	@ID			GetTopologyNodeManifest
//	@Summary	获取节点 Manifest
//	@Tags		topology
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		nodeID			path		string	true	"拓扑节点 ID（base64url 无填充编码）"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200				{object}	serializer.GetTopologyNodeManifestOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/resource-topology/nodes/{nodeID}/manifest [get]
func (h *Handler) GetTopologyNodeManifest(c *gin.Context) {
	var uriInput serializer.TopologyNodeURIInput
	var queryInput serializer.TrafficLaneQueryInput
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

	manifest, err := topology.NewService(
		h.registry.ResourceSnapshotStore,
		topology.NewBuilder(),
		h.registry.AppModelStore,
		h.registry.ScopedEnvVarStore,
		h.registry.AppDepsVarReader,
		h.registry.PolarisVarReader,
	).GetNodeManifest(ctx, app, env, queryInput.TrafficLaneName, uriInput.NodeID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get node manifest"))
		return
	}

	ginutils.OK(c, serializer.GetTopologyNodeManifestOutput{
		Data: new(serializer.TopologyNodeManifestOutputObj).FromModel(manifest),
	})
}
