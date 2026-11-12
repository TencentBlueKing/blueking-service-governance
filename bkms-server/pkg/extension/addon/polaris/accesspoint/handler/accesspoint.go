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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
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
//	@Description	返回空间自建接入点；平台配置了北极星时追加平台托管接入点（id 为 platform）。不返回 Token 明文
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
//	@Description	依次检查网络、鉴权开关、Token，返回失败的阶段与北极星上的命名空间，整体 10 秒超时。
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

// GetNetworkInfo 获取平台访问北极星的网络信息。
//
//	@ID				GetPolarisAccessPointNetworkInfo
//	@Summary		获取平台访问北极星的网络信息
//	@Description	返回平台访问北极星的出口 IP，用户需要把它们加入自建北极星的白名单
//	@Tags			polaris-access-points
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	slz.NetworkInfoResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points/network-info [get]
func (h *Handler) GetNetworkInfo(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	egressIPs := config.G.Network.EgressIPs
	if egressIPs == nil {
		egressIPs = []string{}
	}
	ginutils.OK(c, slz.NetworkInfoResponse{Data: &slz.NetworkInfoOutput{EgressIPs: egressIPs}})
}

// ListNamespaces 获取接入点上可选的北极星命名空间。
//
//	@ID				ListPolarisAccessPointNamespaces
//	@Summary		获取北极星接入点的命名空间
//	@Description	空间自建接入点实时读取北极星并过滤系统命名空间 Polaris；平台托管接入点返回固定列表
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
//	@Description	仅在平台允许空间自建接入点时提供
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

// ListOperators 获取空间候选集群的 bcs-polaris-operator 组件状态。
//
//	@ID				ListPolarisOperators
//	@Summary		获取北极星组件状态视图
//	@Description	返回候选集群的组件状态、与接入点不一致的参数、最近的安装任务，以及可选的 chart 版本。
//	@Description	仅在平台允许空间自建接入点时提供
//	@Tags			polaris-access-points
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	slz.PolarisOperatorListResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-operators [get]
func (h *Handler) ListOperators(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "list polaris operators")
}

// InstallOperators 在接入点绑定的集群上安装或更新 bcs-polaris-operator。
//
//	@ID				InstallPolarisOperators
//	@Summary		安装或更新北极星组件
//	@Description	集群必须已绑定该接入点；每个集群创建一个异步任务，同一集群同时只允许一个运行中的任务。
//	@Description	仅在平台允许空间自建接入点时提供
//	@Tags			polaris-access-points
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID		path		string					true	"工作空间 ID"
//	@Param			accessPointID	path		string					true	"接入点 ID"
//	@Param			body			body		slz.OperatorInstallInput	true	"目标集群与 chart 版本"
//	@Success		200				{object}	slz.OperatorTaskListResponse
//	@Failure		400				{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points/{accessPointID}/operator-installs [post]
func (h *Handler) InstallOperators(c *gin.Context) {
	var uri slz.AccessPointURIInput
	var input slz.OperatorInstallInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateWorkspaceByID(c.Request.Context(), h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "install polaris operators")
}

// GetEnvOperator 获取环境所在集群的 bcs-polaris-operator 组件状态。
//
//	@ID				GetEnvPolarisOperator
//	@Summary		获取环境的北极星组件状态
//	@Description	仅在平台允许空间自建接入点时提供
//	@Tags			polaris-access-points
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			envID	path		string	true	"环境 ID"
//	@Success		200		{object}	slz.PolarisOperatorResponse
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/envs/{envID}/polaris-operator [get]
func (h *Handler) GetEnvOperator(c *gin.Context) {
	var uri slz.EnvURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if _, err := perm.ValidateEnvByID(c.Request.Context(), h.registry, uri.EnvID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	abortNotImplemented(c, "get env polaris operator")
}

// abortNotImplemented 契约已冻结、业务逻辑尚未实现的接口统一返回 NOT_IMPLEMENTED
func abortNotImplemented(c *gin.Context, api string) {
	bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotImplemented, "%s is not implemented yet", api))
}
