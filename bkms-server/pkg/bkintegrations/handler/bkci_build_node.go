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

	bkciproject "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// ListBkCIBuildNodes 获取工作空间蓝盾项目下可关联的第三方构建机
//
//	@ID			ListBkCIBuildNodes
//	@Summary	获取工作空间蓝盾项目下可关联的第三方构建机
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.ListBuildNodeCandidatesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkci-build-nodes [get]
func (h *Handler) ListBkCIBuildNodes(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	candidates, err := bkciproject.NewBuildNodeManager(h.registry.DB, uriInput.WorkspaceID).ListCandidates(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, toBuildNodeAPIError(err, "list bkci build nodes"))
		return
	}

	ginutils.OK(c, slz.ListBuildNodeCandidatesOutput{
		Data: lo.Map(candidates, func(cand bkciproject.BuildNodeCandidate, _ int) *slz.BuildNodeCandidateOutput {
			return slz.NewBuildNodeCandidateOutput(cand)
		}),
	})
}

// GetBuildNodeBinding 查询工作空间关联的构建机
//
//	@ID			GetBuildNodeBinding
//	@Summary	查询工作空间关联的构建机
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.GetBuildNodeBindingOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/build-node-binding [get]
func (h *Handler) GetBuildNodeBinding(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	state, err := bkciproject.NewBuildNodeManager(h.registry.DB, uriInput.WorkspaceID).Get(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get build node binding"))
		return
	}

	ginutils.OK(c, slz.GetBuildNodeBindingOutput{
		Data: slz.NewBuildNodeBindingStateOutput(state, h.canEditWorkspace(ctx, uriInput.WorkspaceID)),
	})
}

// BindBuildNode 关联或更换工作空间的构建机
//
//	@ID			BindBuildNode
//	@Summary	关联或更换工作空间的构建机
//	@Tags		bkintegrations-bkci
//	@Accept		json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		body		body		serializer.BindBuildNodeInput	true	"要关联的构建机"
//	@Success	204			{object}	nil
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/build-node-binding [put]
func (h *Handler) BindBuildNode(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	var input slz.BindBuildNodeInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	res, err := bkciproject.NewBuildNodeManager(h.registry.DB, uriInput.WorkspaceID).Bind(ctx, input.NodeHashID)
	if err != nil {
		bkerrs.AbortWithErr(c, toBuildNodeAPIError(err, "bind build node"))
		return
	}

	// 重复提交同一台时没有变更，不写操作记录
	if res.Changed {
		opts := []audit.Option{
			audit.WithDataAfter(slz.NewBuildNodeAuditSnapshot(res.After)),
			audit.WithWorkspaceID(uriInput.WorkspaceID),
		}
		if res.Before != nil {
			opts = append(opts, audit.WithDataBefore(slz.NewBuildNodeAuditSnapshot(res.Before)))
		}
		go audit.AddOperationRecordAsync(
			ctx, audit.OperationTypeUpdate, audit.ResourceTypeBuildNode, res.After.NodeHashID, opts...,
		)
	}

	ginutils.NoContent(c)
}

// UnbindBuildNode 解除工作空间的构建机关联，之后内置流水线回到公共构建机
//
//	@ID			UnbindBuildNode
//	@Summary	解除工作空间的构建机关联
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	slz.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/build-node-binding [delete]
func (h *Handler) UnbindBuildNode(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	removed, err := bkciproject.NewBuildNodeManager(h.registry.DB, uriInput.WorkspaceID).Unbind(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "unbind build node"))
		return
	}

	// 本来就没有关联时没有变更，不写操作记录
	if removed != nil {
		go audit.AddOperationRecordAsync(
			ctx, audit.OperationTypeDelete, audit.ResourceTypeBuildNode, removed.NodeHashID,
			audit.WithDataBefore(slz.NewBuildNodeAuditSnapshot(removed)),
			audit.WithWorkspaceID(uriInput.WorkspaceID),
		)
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// canEditWorkspace 当前用户能否修改构建机关联（与写接口一样以 edit_workspace 为准，只授予空间管理员）。
// 只用于前端决定是否展示操作按钮，鉴权失败按无权限处理，写接口仍会再次校验
func (h *Handler) canEditWorkspace(ctx context.Context, workspaceID string) bool {
	return perm.NewManager().HasEditWorkspacePerm(ctx, workspaceID) == nil
}

// toBuildNodeAPIError 把构建机关联的业务错误映射为接口错误。文案使用英文，由前端拦截器直接展示
func toBuildNodeAPIError(err error, internalMessage string) error {
	switch {
	case errors.Is(err, bkciproject.ErrProjectNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "bkci project of the workspace is not initialized")
	case errors.Is(err, bkciproject.ErrBuildNodeNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "build node not found in the bkci project")
	case errors.Is(err, bkciproject.ErrBuildNodeOffline):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "build node agent is offline")
	case errors.Is(err, bkciproject.ErrBuildNodeOSUnsupported):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "only linux build nodes are supported")
	case errors.Is(err, bkciproject.ErrBuildNodeNoUsePerm):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "no permission to use the build node in bkci")
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, internalMessage)
	}
}
