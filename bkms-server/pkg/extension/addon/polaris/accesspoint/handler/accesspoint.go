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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint/binding"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
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

func (h *Handler) manager() *accesspoint.Manager {
	return accesspoint.NewManager(h.registry.DB)
}

func (h *Handler) bindings() *binding.Service {
	return binding.NewService(h.registry.DB)
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
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	result, err := h.manager().List(ctx, uri.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "list polaris access points"))
		return
	}
	outputs := make([]*slz.AccessPointOutput, 0, len(result.AccessPoints))
	for i := range result.AccessPoints {
		p := &result.AccessPoints[i]
		outputs = append(outputs, slz.NewAccessPointOutput(p, result.BoundClusterCounts[p.ID.Hex()]))
	}
	ginutils.OK(c, slz.AccessPointListResponse{Data: outputs})
}

// CreateAccessPoint 创建空间自建北极星接入点。
//
//	@ID				CreatePolarisAccessPoint
//	@Summary		创建北极星接入点
//	@Description	保存前服务端重新执行连通测试，失败时拒绝；Token 选填，加密存储。
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
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	p, err := h.manager().Create(ctx, uri.WorkspaceID, accesspoint.CreateParams{
		Name:        input.Name,
		ConsoleURL:  input.ConsoleURL,
		ClientHosts: input.ClientHosts,
		Token:       input.Token,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "create polaris access point"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeCreate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributePolarisAccessPoint),
		audit.WithDataAfter(auditData(p)),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	ginutils.OK(c, slz.AccessPointResponse{Data: slz.NewAccessPointOutput(p, 0)})
}

// UpdateAccessPoint 更新空间自建北极星接入点。
//
//	@ID				UpdatePolarisAccessPoint
//	@Summary		更新北极星接入点
//	@Description	字段缺省表示不修改；客户端地址或 Token 变化时重新执行连通测试，失败时拒绝保存。
//	@Description	接入点被北极星配置使用时仍允许修改地址，集群组件会显示为需要更新。
//	@Description	不修改绑定的集群，绑定集群使用 PUT .../polaris-access-points/{accessPointID}/clusters
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
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr := h.manager()
	before, after, err := mgr.Update(ctx, uri.WorkspaceID, uri.AccessPointID, accesspoint.UpdateParams{
		Name:        input.Name,
		ConsoleURL:  input.ConsoleURL,
		ClientHosts: input.ClientHosts,
		Token:       input.Token,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "update polaris access point"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeUpdate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributePolarisAccessPoint),
		audit.WithDataBefore(auditData(before)),
		audit.WithDataAfter(auditData(after)),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	boundClusterCount, err := mgr.CountBoundClusters(ctx, uri.WorkspaceID, after.ID.Hex())
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "count clusters bound to polaris access point"))
		return
	}
	ginutils.OK(c, slz.AccessPointResponse{Data: slz.NewAccessPointOutput(after, boundClusterCount)})
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
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	p, err := h.manager().Delete(ctx, uri.WorkspaceID, uri.AccessPointID, polaris.ListConfigUsagesByAccessPoint)
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "delete polaris access point"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeDelete, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributePolarisAccessPoint),
		audit.WithDataBefore(auditData(p)),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	ginutils.OK(c, slz.EmptyResponse{})
}

// TestConnectivity 测试北极星接入点的连通性。
//
//	@ID				TestPolarisAccessPointConnectivity
//	@Summary		测试北极星接入点连通性
//	@Description	依次检查网络、鉴权开关、Token，返回失败的原因与北极星上的命名空间，整体 10 秒超时。
//	@Description	传入已有接入点 ID 时记录本次结果；此时 token 缺省时使用已保存的 Token，显式传空字符串表示不带 Token
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
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	report, err := h.manager().TestConnectivity(ctx, uri.WorkspaceID, accesspoint.ConnectivityParams{
		ID:          input.ID,
		ClientHosts: input.ClientHosts,
		Token:       input.Token,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "test polaris access point connectivity"))
		return
	}
	ginutils.OK(c, slz.ConnectivityTestResponse{Data: slz.NewConnectivityTestOutput(report)})
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
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	namespaces, err := h.manager().ListNamespaces(ctx, uri.WorkspaceID, uri.AccessPointID)
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "list polaris access point namespaces"))
		return
	}
	ginutils.OK(c, slz.NamespacesResponse{Data: namespaces})
}

// ReplaceClusters 全量替换北极星接入点绑定的集群。
//
//	@ID				ReplacePolarisAccessPointClusters
//	@Summary		替换北极星接入点绑定的集群
//	@Description	全量替换接入点绑定的集群，传空数组表示解除全部绑定；集群须为空间 BCS 项目下的非共享集群，已绑定其他接入点的集群会被换绑。
//	@Description	待解绑或换绑的集群上有环境正被北极星配置使用时拒绝修改（详情码 POLARIS_CLUSTER_IN_USE），全部校验通过后才写入
//	@Tags			polaris-access-points
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID		path		string							true	"工作空间 ID"
//	@Param			accessPointID	path		string							true	"接入点 ID"
//	@Param			body			body		slz.AccessPointClustersInput	true	"绑定的集群"
//	@Success		200				{object}	slz.AccessPointClustersResponse
//	@Failure		400				{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-access-points/{accessPointID}/clusters [put]
func (h *Handler) ReplaceClusters(c *gin.Context) {
	var uri slz.AccessPointURIInput
	var input slz.AccessPointClustersInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	before, after, err := h.bindings().ReplaceClusters(ctx, ws, uri.AccessPointID, input.ClusterIDs)
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "bind clusters to polaris access point"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeUpdate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributePolarisAccessPoint),
		audit.WithDataBefore(map[string]any{"accessPointID": uri.AccessPointID, "clusterIDs": before}),
		audit.WithDataAfter(map[string]any{"accessPointID": uri.AccessPointID, "clusterIDs": after}),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	ginutils.OK(c, slz.AccessPointClustersResponse{Data: &slz.AccessPointClustersOutput{ClusterIDs: after}})
}

// ListClusters 获取空间候选集群的北极星接入点绑定情况。
//
//	@ID				ListPolarisClusters
//	@Summary		获取北极星集群绑定视图
//	@Description	候选集群为空间 BCS 项目下的全部非共享集群，含没有环境的集群；有环境的集群排在前面，每组内按集群 ID 排序。
//	@Description	envs 含特性环境；inUse 为 true 表示集群上有环境被北极星配置使用，不能解绑或换绑
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
	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	views, err := h.bindings().ListClusters(ctx, ws)
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "list polaris clusters"))
		return
	}
	outputs := make([]*slz.PolarisClusterOutput, 0, len(views))
	for i := range views {
		outputs = append(outputs, slz.NewPolarisClusterOutput(&views[i]))
	}
	ginutils.OK(c, slz.PolarisClusterListResponse{Data: outputs})
}

// BindCluster 将单个集群绑定到北极星接入点。
//
//	@ID				BindPolarisCluster
//	@Summary		绑定集群的北极星接入点
//	@Description	集群须为空间 BCS 项目下的非共享集群；已绑定其他接入点时换绑，accessPointID 传空字符串表示解绑。
//	@Description	换绑或解绑时，集群上有环境正被北极星配置使用则拒绝（详情码 POLARIS_CLUSTER_IN_USE）；返回变更后集群绑定的接入点 ID
//	@Tags			polaris-access-points
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string						true	"工作空间 ID"
//	@Param			clusterID	path		string						true	"集群 ID"
//	@Param			body		body		slz.PolarisClusterBindInput	true	"绑定的接入点"
//	@Success		200			{object}	slz.PolarisClusterBindResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/polaris-clusters/{clusterID} [put]
func (h *Handler) BindCluster(c *gin.Context) {
	var uri slz.PolarisClusterURIInput
	var input slz.PolarisClusterBindInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	before, after, err := h.bindings().BindCluster(ctx, ws, uri.ClusterID, *input.AccessPointID)
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "bind cluster to polaris access point"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeUpdate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributePolarisAccessPoint),
		audit.WithDataBefore(map[string]any{"clusterID": uri.ClusterID, "accessPointID": before}),
		audit.WithDataAfter(map[string]any{"clusterID": uri.ClusterID, "accessPointID": after}),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	ginutils.OK(c, slz.PolarisClusterBindResponse{
		Data: &slz.PolarisClusterBindOutput{ClusterID: uri.ClusterID, AccessPointID: after},
	})
}

// ListOperatorValues 获取已绑定接入点的集群上 bcs-polaris-operator 应使用的北极星 values。
//
// TODO: 已安装组件的集群暂不展示期望 values 与 release 当前 values 的差异，交互形态待与产品确认。
// 倾向的方案是前端对比本接口返回的期望 values 与集群插件列表返回的 currentValues，展示差异，由用户选择是否覆盖。
//
//	@ID				ListPolarisOperatorValues
//	@Summary		获取北极星组件的期望 values
//	@Description	返回已绑定接入点的集群上由接入点推导出的北极星相关 values，按集群 ID 排序。
//	@Description	不读取集群内的 release，也不返回安装状态。
//	@Description	values 只含 args[name=polaris_api_type]、env 中的北极星地址、鉴权方式与集群、BCS 项目标识，
//	@Description	以及 authProxy.bcsPolarisPlatformID，不含 Token。
//	@Description	安装或更新时以集群插件接口返回的 currentValues 为基础覆盖这些键（args 按 name 合并），
//	@Description	再调用集群插件安装接口
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
	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	views, err := h.bindings().ListOperatorValues(ctx, ws, query.AccessPointID)
	if err != nil {
		bkerrs.AbortWithErr(c, accesspoint.APIError(err, "list polaris operator values"))
		return
	}
	ginutils.OK(c, slz.PolarisOperatorValuesListResponse{Data: slz.NewPolarisOperatorValuesListOutput(views)})
}

// auditData 审计记录接入点的地址等信息，不写入 Token
func auditData(p *accesspoint.PolarisAccessPoint) map[string]any {
	return map[string]any{
		"id":          p.ID.Hex(),
		"name":        p.Name,
		"consoleURL":  p.ConsoleURL,
		"clientHosts": p.ClientHosts,
		"hasToken":    p.Token != "",
	}
}
