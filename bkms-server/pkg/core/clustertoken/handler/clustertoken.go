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

// Package handler 包含集群 Token 与集群接入 API 的 Handler 实现
package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/clustertoken"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/clustertoken/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ clustertoken.Handler = (*Handler)(nil)

// Handler 处理集群 Token 与集群接入 API 请求
type Handler struct {
	registry *storereg.Registry
}

// New 创建 Handler
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

func (h *Handler) manager() *clustertoken.Manager {
	return clustertoken.NewManager(h.registry.DB)
}

// ListTokens 获取空间的集群 Token。
//
//	@ID				ListClusterTokens
//	@Summary		获取空间的集群 Token 列表
//	@Description	按创建时间排序，只返回脱敏后的 Token 与使用它的集群数量
//	@Tags			cluster-tokens
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	slz.ClusterTokenListResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/cluster-tokens [get]
func (h *Handler) ListTokens(c *gin.Context) {
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

	result, err := h.manager().ListTokens(ctx, uri.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "list cluster tokens"))
		return
	}
	outputs := make([]*slz.ClusterTokenOutput, 0, len(result.Tokens))
	for i := range result.Tokens {
		t := &result.Tokens[i]
		outputs = append(outputs, slz.NewClusterTokenOutput(t, result.ClusterCounts[t.ID.Hex()]))
	}
	ginutils.OK(c, slz.ClusterTokenListResponse{Data: outputs})
}

// CreateToken 新建集群 Token。
//
//	@ID				CreateClusterToken
//	@Summary		新建集群 Token
//	@Description	Token 加密存储；空间内已有相同 Token 时返回 ALREADY_EXISTS；空间内没有默认 Token 时设为默认
//	@Tags			cluster-tokens
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string			true	"工作空间 ID"
//	@Param			body		body		slz.TokenInput	true	"Token"
//	@Success		200			{object}	slz.ClusterTokenResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/cluster-tokens [post]
func (h *Handler) CreateToken(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	var input slz.TokenInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	t, err := h.manager().CreateToken(ctx, uri.WorkspaceID, input.Token)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "create cluster token"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeCreate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributeClusterToken),
		audit.WithDataAfter(tokenAuditData(t)),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	ginutils.OK(c, slz.ClusterTokenResponse{Data: slz.NewClusterTokenOutput(t, 0)})
}

// ReplaceToken 更换集群 Token 的值。
//
//	@ID				ReplaceClusterToken
//	@Summary		更换集群 Token
//	@Description	使用该 Token 的集群一并生效；新值与空间内其他 Token 相同时返回 ALREADY_EXISTS
//	@Tags			cluster-tokens
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string			true	"工作空间 ID"
//	@Param			tokenID		path		string			true	"Token ID"
//	@Param			body		body		slz.TokenInput	true	"新的 Token"
//	@Success		200			{object}	slz.ClusterTokenResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/cluster-tokens/{tokenID} [put]
func (h *Handler) ReplaceToken(c *gin.Context) {
	var uri slz.TokenURIInput
	var input slz.TokenInput
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
	before, after, err := mgr.ReplaceToken(ctx, uri.WorkspaceID, uri.TokenID, input.Token)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "replace cluster token"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeUpdate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributeClusterToken),
		audit.WithDataBefore(tokenAuditData(before)),
		audit.WithDataAfter(tokenAuditData(after)),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	h.respondToken(c, mgr, uri.WorkspaceID, after)
}

// SetDefaultToken 把集群 Token 设为空间的默认 Token。
//
//	@ID				SetDefaultClusterToken
//	@Summary		设为默认集群 Token
//	@Description	空间内至多一个默认 Token，原默认 Token 自动取消；并发设置冲突时返回 ABORTED，重试即可
//	@Tags			cluster-tokens
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			tokenID		path		string	true	"Token ID"
//	@Success		200			{object}	slz.ClusterTokenResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/cluster-tokens/{tokenID}/default [put]
func (h *Handler) SetDefaultToken(c *gin.Context) {
	var uri slz.TokenURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr := h.manager()
	t, err := mgr.SetDefaultToken(ctx, uri.WorkspaceID, uri.TokenID)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "set default cluster token"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeUpdate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributeClusterToken),
		audit.WithDataAfter(tokenAuditData(t)),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	h.respondToken(c, mgr, uri.WorkspaceID, t)
}

// DeleteToken 删除集群 Token。
//
//	@ID				DeleteClusterToken
//	@Summary		删除集群 Token
//	@Description	仍有集群使用该 Token 时拒绝删除（详情码 CLUSTER_TOKEN_IN_USE，extras.clusters 列出集群）
//	@Tags			cluster-tokens
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			tokenID		path		string	true	"Token ID"
//	@Success		200			{object}	slz.ClusterTokenEmptyResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/cluster-tokens/{tokenID} [delete]
func (h *Handler) DeleteToken(c *gin.Context) {
	var uri slz.TokenURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	t, err := h.manager().DeleteToken(ctx, uri.WorkspaceID, uri.TokenID)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "delete cluster token"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeDelete, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributeClusterToken),
		audit.WithDataBefore(tokenAuditData(t)),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	ginutils.OK(c, slz.ClusterTokenEmptyResponse{})
}

// ListCandidates 获取空间可接入的集群。
//
//	@ID				ListClusterCandidates
//	@Summary		获取可接入的集群
//	@Description	返回空间关联的 BCS 项目下的全部 k8s 集群（含共享集群），并标记是否已接入；
//	@Description	空间未关联 BCS 项目时返回空列表
//	@Tags			workspace-clusters
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Success		200			{object}	slz.ClusterCandidateListResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/cluster-candidates [get]
func (h *Handler) ListCandidates(c *gin.Context) {
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

	candidates, err := h.manager().ListCandidates(ctx, uri.WorkspaceID, ws.BkSystems.BkBCSProjectID)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "list cluster candidates"))
		return
	}
	outputs := make([]*slz.ClusterCandidateOutput, 0, len(candidates))
	for i := range candidates {
		outputs = append(outputs, slz.NewClusterCandidateOutput(&candidates[i]))
	}
	ginutils.OK(c, slz.ClusterCandidateListResponse{Data: outputs})
}

// ListClusters 获取空间已接入的集群。
//
//	@ID				ListWorkspaceClusters
//	@Summary		获取已接入的集群
//	@Description	按接入时间排序，包含使用的 Token、使用该集群的环境数量与权限检测状态。
//	@Description	tokenID 可选，只返回使用该 Token 的集群
//	@Tags			workspace-clusters
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			tokenID		query		string	false	"只返回使用该 Token 的集群"
//	@Success		200			{object}	slz.WorkspaceClusterListResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/clusters [get]
func (h *Handler) ListClusters(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	var query slz.ListClustersQueryInput
	if err := ginutils.BindURIQuery(c, &uri, &query); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	result, err := h.manager().ListClusters(ctx, uri.WorkspaceID, query.TokenID)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "list workspace clusters"))
		return
	}
	outputs := make([]*slz.WorkspaceClusterOutput, 0, len(result.Clusters))
	for i := range result.Clusters {
		a := &result.Clusters[i]
		var token *clustertoken.ClusterToken
		if t, ok := result.Tokens[a.TokenID]; ok {
			token = &t
		}
		outputs = append(outputs, slz.NewWorkspaceClusterOutput(a, token, result.EnvCounts[a.ClusterID]))
	}
	ginutils.OK(c, slz.WorkspaceClusterListResponse{Data: outputs})
}

// AccessClusters 把集群接入空间。
//
//	@ID				AccessClusters
//	@Summary		接入集群
//	@Description	集群须属于空间关联的 BCS 项目，可以是共享集群；已接入的集群返回 ALREADY_EXISTS。
//	@Description	tokenID 与 token 必须且只能填一个：tokenID 为已有 Token，token 为新 Token（与已有 Token 相同时复用）
//	@Tags			workspace-clusters
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string							true	"工作空间 ID"
//	@Param			body		body		slz.WorkspaceClusterCreateInput	true	"待接入的集群与 Token"
//	@Success		200			{object}	slz.WorkspaceClusterListResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/clusters [post]
func (h *Handler) AccessClusters(c *gin.Context) {
	var uri slz.WorkspaceURIInput
	var input slz.WorkspaceClusterCreateInput
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

	result, err := h.manager().Access(
		ctx, uri.WorkspaceID, ws.BkSystems.BkBCSProjectID, input.ClusterIDs, input.ToRef(),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "access clusters"))
		return
	}

	outputs := make([]*slz.WorkspaceClusterOutput, 0, len(result.Clusters))
	for i := range result.Clusters {
		a := &result.Clusters[i]
		go audit.AddOperationRecordAsync(
			ctx, audit.OperationTypeCreate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
			audit.WithAttribute(audit.AttributeWorkspaceCluster),
			audit.WithDataAfter(clusterAuditData(a, result.Token)),
			audit.WithWorkspaceID(uri.WorkspaceID),
		)
		outputs = append(outputs, slz.NewWorkspaceClusterOutput(a, result.Token, 0))
	}
	ginutils.OK(c, slz.WorkspaceClusterListResponse{Data: outputs})
}

// SwitchClusterToken 更换集群使用的 Token。
//
//	@ID				SwitchClusterToken
//	@Summary		更换集群使用的 Token
//	@Description	tokenID 与 token 必须且只能填一个：tokenID 为已有 Token，token 为新 Token（与已有 Token 相同时复用）
//	@Tags			workspace-clusters
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string				true	"工作空间 ID"
//	@Param			clusterID	path		string				true	"集群 ID"
//	@Param			body		body		slz.TokenRefInput	true	"新的 Token"
//	@Success		200			{object}	slz.WorkspaceClusterResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/clusters/{clusterID}/token [put]
func (h *Handler) SwitchClusterToken(c *gin.Context) {
	var uri slz.ClusterURIInput
	var input slz.TokenRefInput
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
	before, after, token, err := mgr.SwitchToken(ctx, uri.WorkspaceID, uri.ClusterID, input.ToRef())
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "switch cluster token"))
		return
	}

	if before.TokenID != after.TokenID {
		go audit.AddOperationRecordAsync(
			ctx, audit.OperationTypeUpdate, audit.ResourceTypeWorkspace, uri.WorkspaceID,
			audit.WithAttribute(audit.AttributeWorkspaceCluster),
			audit.WithDataBefore(clusterAuditData(before, nil)),
			audit.WithDataAfter(clusterAuditData(after, token)),
			audit.WithWorkspaceID(uri.WorkspaceID),
		)
	}
	envCount, err := mgr.CountEnvsOnCluster(ctx, uri.WorkspaceID, uri.ClusterID)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "count envs on cluster"))
		return
	}
	ginutils.OK(c, slz.WorkspaceClusterResponse{Data: slz.NewWorkspaceClusterOutput(after, token, envCount)})
}

// RemoveCluster 移除空间接入的集群。
//
//	@ID				RemoveWorkspaceCluster
//	@Summary		移除集群
//	@Description	移除后新建环境时不能再选择这个集群。集群仍被环境（含特性环境）使用或绑定了北极星接入点时拒绝移除
//	@Description	（详情码 WORKSPACE_CLUSTER_IN_USE，extras.envs 列出环境，extras.polarisBound 表示是否绑定了北极星接入点）
//	@Tags			workspace-clusters
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			clusterID	path		string	true	"集群 ID"
//	@Success		200			{object}	slz.ClusterTokenEmptyResponse
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/clusters/{clusterID} [delete]
func (h *Handler) RemoveCluster(c *gin.Context) {
	var uri slz.ClusterURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	wc, err := h.manager().Remove(ctx, uri.WorkspaceID, uri.ClusterID, h.polarisBound)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "remove workspace cluster"))
		return
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeDelete, audit.ResourceTypeWorkspace, uri.WorkspaceID,
		audit.WithAttribute(audit.AttributeWorkspaceCluster),
		audit.WithDataBefore(clusterAuditData(wc, nil)),
		audit.WithWorkspaceID(uri.WorkspaceID),
	)
	ginutils.OK(c, slz.ClusterTokenEmptyResponse{})
}

// respondToken 返回单个 Token 及使用它的集群数量
func (h *Handler) respondToken(
	c *gin.Context,
	mgr *clustertoken.Manager,
	workspaceID string,
	t *clustertoken.ClusterToken,
) {
	result, err := mgr.ListTokens(c.Request.Context(), workspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, clustertoken.APIError(err, "count clusters using cluster token"))
		return
	}
	ginutils.OK(c, slz.ClusterTokenResponse{Data: slz.NewClusterTokenOutput(t, result.ClusterCounts[t.ID.Hex()])})
}

// polarisBound 集群在空间内是否绑定了北极星接入点
func (h *Handler) polarisBound(ctx context.Context, workspaceID, clusterID string) (bool, error) {
	bound, err := store.Of(h.registry.DB, accesspoint.BindingMeta).
		Exists(ctx, accesspoint.ByCluster(workspaceID, clusterID))
	return bound, errors.Wrapf(err, "check polaris binding of cluster %s", clusterID)
}

// tokenAuditData 审计记录中的 Token，不包含明文
func tokenAuditData(t *clustertoken.ClusterToken) map[string]any {
	return map[string]any{
		"id":        t.ID.Hex(),
		"masked":    t.Masked,
		"isDefault": t.IsDefault,
	}
}

// clusterAuditData 审计记录中的集群接入，token 为 nil 时只记录 Token ID
func clusterAuditData(a *clustertoken.WorkspaceCluster, token *clustertoken.ClusterToken) map[string]any {
	data := map[string]any{
		"clusterID":   a.ClusterID,
		"clusterName": a.ClusterName,
		"isShared":    a.IsShared,
		"tokenID":     a.TokenID,
	}
	if token != nil {
		data["tokenMasked"] = token.Masked
	}
	return data
}
