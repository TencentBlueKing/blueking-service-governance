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

// Package handler 包含北极星接入点 API 的 Handler 实现
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ accesspoint.Handler = (*Handler)(nil)

// Handler 处理北极星接入点 API 请求
type Handler struct {
	registry *storereg.Registry
}

// New 创建 Handler
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListAccessPoints 获取空间可用的北极星接入点。
//
//	@ID				ListPolarisAccessPoints
//	@Summary		获取空间的北极星接入点列表
//	@Description	允许空间自建接入点时返回空间自建接入点
//	@Tags			polaris-access-points
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	slz.AccessPointListResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points [get]
func (h *Handler) ListAccessPoints(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "list polaris access points")
}

// CreateAccessPoint 创建空间自建北极星接入点。
//
//	@ID				CreatePolarisAccessPoint
//	@Summary		创建北极星接入点
//	@Description	保存前服务端重新执行连通测试，失败或北极星开启了客户端鉴权时拒绝；Token 选填，加密存储。
//	@Description	传入 clusterIDs 时同时把这些集群绑定到该接入点
//	@Tags			polaris-access-points
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string						true	"工作空间 ID"
//	@Param			body		body		slz.AccessPointCreateInput	true	"接入点"
//	@Success		200			{object}	slz.AccessPointResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points [post]
func (h *Handler) CreateAccessPoint(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	var input slz.AccessPointCreateInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "create polaris access point")
}

// UpdateAccessPoint 更新空间自建北极星接入点。
//
//	@ID				UpdatePolarisAccessPoint
//	@Summary		更新北极星接入点
//	@Description	字段缺省表示不修改；客户端地址或 Token 变化时重新执行连通测试，失败时拒绝保存。
//	@Description	接入点被北极星配置使用时仍允许修改地址，集群组件会显示为需要更新。
//	@Description	传入 clusterIDs 时全量替换该接入点绑定的集群
//	@Tags			polaris-access-points
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID		path		string						true	"工作空间 ID"
//	@Param			accessPointID	path		string						true	"接入点 ID"
//	@Param			body			body		slz.AccessPointUpdateInput	true	"待修改的字段"
//	@Success		200				{object}	slz.AccessPointResponse
//	@Failure		400				{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points/{accessPointID} [patch]
func (h *Handler) UpdateAccessPoint(c *gin.Context) {
	var uri slz.AccessPointURIInput
	var input slz.AccessPointUpdateInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "update polaris access point")
}

// DeleteAccessPoint 删除空间自建北极星接入点。
//
//	@ID				DeletePolarisAccessPoint
//	@Summary		删除北极星接入点
//	@Description	接入点仍绑定集群，或仍有北极星配置固定使用它时拒绝删除
//	@Tags			polaris-access-points
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID		path		string	true	"工作空间 ID"
//	@Param			accessPointID	path		string	true	"接入点 ID"
//	@Success		200				{object}	slz.EmptyResponse
//	@Failure		400				{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points/{accessPointID} [delete]
func (h *Handler) DeleteAccessPoint(c *gin.Context) {
	var uri slz.AccessPointURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "delete polaris access point")
}

// TestConnectivity 测试北极星接入点的连通性。
//
//	@ID				TestPolarisAccessPointConnectivity
//	@Summary		测试北极星接入点连通性
//	@Description	依次检查网络、鉴权开关、Token，返回失败的原因与北极星上的命名空间，整体 10 秒超时。
//	@Description	传入已有接入点 ID 且 token 为空时使用已保存的 Token，并记录本次结果
//	@Tags			polaris-access-points
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string						true	"工作空间 ID"
//	@Param			body		body		slz.ConnectivityTestInput	true	"接入地址"
//	@Success		200			{object}	slz.ConnectivityTestResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points/connectivity-test [post]
func (h *Handler) TestConnectivity(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	var input slz.ConnectivityTestInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "test polaris access point connectivity")
}

// ListNamespaces 获取接入点上可选的北极星命名空间。
//
//	@ID				ListPolarisAccessPointNamespaces
//	@Summary		获取北极星接入点的命名空间
//	@Description	实时读取空间自建接入点上的北极星命名空间，并过滤系统命名空间 Polaris
//	@Tags			polaris-access-points
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID		path		string	true	"工作空间 ID"
//	@Param			accessPointID	path		string	true	"接入点 ID"
//	@Success		200				{object}	slz.NamespacesResponse
//	@Failure		400				{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points/{accessPointID}/namespaces [get]
func (h *Handler) ListNamespaces(c *gin.Context) {
	var uri slz.AccessPointURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "list polaris access point namespaces")
}

// ListClusters 获取空间候选集群的北极星接入点绑定情况。
//
//	@ID				ListPolarisClusters
//	@Summary		获取北极星集群绑定视图
//	@Description	候选集群为空间 BCS 项目下的全部非共享集群，含没有环境的集群；先列出有环境的集群，同类按集群 ID 排序。
//	@Tags			polaris-access-points
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	slz.PolarisClusterListResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-clusters [get]
func (h *Handler) ListClusters(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "list polaris clusters")
}

// ListOperatorValues 获取已绑定接入点的集群上 bcs-polaris-operator 应使用的北极星 values。
//
// TODO(北极星组件安装子需求): 已安装组件的集群暂不展示期望 values 与 release 当前 values 的差异，
// 交互形态待与产品确认。倾向的方案是前端对比本接口返回的期望 values 与集群插件列表返回的 currentValues，
// 展示差异，由用户选择是否覆盖。
//
//	@ID				ListPolarisOperatorValues
//	@Summary		获取北极星组件的期望 values
//	@Description	返回由接入点推导出的北极星相关 values
//	@Tags			polaris-access-points
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID		path		string	true	"工作空间 ID"
//	@Param			accessPointID	query		string	false	"只返回绑定到该接入点的集群"
//	@Success		200				{object}	slz.PolarisOperatorValuesListResponse
//	@Failure		400				{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-operator-values [get]
func (h *Handler) ListOperatorValues(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	var query slz.OperatorValuesQueryInput
	if err := ginutils.BindURIQuery(c, &uri, &query); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "list polaris operator values")
}

// abortNotImplemented 契约已冻结、业务逻辑尚未实现的接口统一返回 NOT_IMPLEMENTED
func abortNotImplemented(c *gin.Context, api string) {
	bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotImplemented, "%s is not implemented yet", api))
}
