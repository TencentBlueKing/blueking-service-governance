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
	"github.com/samber/lo"

	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// ListBkCIOAuthGitProjects 获取蓝盾授权 Git 项目列表
//
//	@ID			ListBkCIOAuthGitProjects
//	@Summary	获取蓝盾 OAuth 授权 Git 项目列表
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		keyword		query		string	false	"搜索关键词"
//	@Success	200			{object}	serializer.ListBkCIOAuthGitProjectsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkci-git-projects [get]
func (h *Handler) ListBkCIOAuthGitProjects(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	var queryInput slz.BkCIGitProjectsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace project"))
		return
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return
	}

	projects, err := client.ListOAuthGitProjects(ctx, project.Code, queryInput.Keyword)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkci oauth git projects"))
		return
	}

	var objs []*slz.BkCIOAuthGitProjectOutput
	for _, p := range projects {
		objs = append(objs, new(slz.BkCIOAuthGitProjectOutput).FromModel(p))
	}

	ginutils.OK(c, &slz.ListBkCIOAuthGitProjectsOutput{Data: objs})
}

// GetBkCIOAuthUrl 获取 OAuth 授权 Git 项目给蓝盾的 URL
//
//	@ID			GetBkCIOAuthUrl
//	@Summary	获取蓝盾 OAuth 授权 URL
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	serializer.GetBkCIOAuthUrlOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkci-oauth-url [get]
func (h *Handler) GetBkCIOAuthUrl(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace project"))
		return
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return
	}

	url, err := client.GetOAuthUrl(ctx, project.Code)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get bkci oauth url"))
		return
	}

	ginutils.OK(c, &slz.GetBkCIOAuthUrlOutput{Data: url})
}

// ListBkCIPipelines 获取蓝盾流水线列表
//
//	@ID			ListBkCIPipelines
//	@Summary	获取蓝盾流水线列表
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		keyword		query		string	false	"搜索关键词"
//	@Param		page		query		int		true	"页码，最小为 1"
//	@Param		pageSize	query		int		true	"每页数量，可选值: 5, 10, 20, 50, 100"
//	@Success	200			{object}	serializer.ListBkCIPipelinesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkci-pipelines [get]
func (h *Handler) ListBkCIPipelines(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	var queryInput slz.BkCIPipelinesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace project"))
		return
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return
	}

	total, pipelines, err := client.ListPipelines(
		ctx,
		project.Code,
		queryInput.Keyword,
		queryInput.Page,
		queryInput.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkci pipelines"))
		return
	}

	ginutils.OK(c, &slz.ListBkCIPipelinesOutput{Data: &slz.PaginatedBkCIPipelineOutput{
		Count: total, Results: lo.Map(pipelines, func(p bkci.Pipeline, _ int) *slz.BkCIPipelineOutput {
			return new(slz.BkCIPipelineOutput).FromModel(p)
		}),
	}})
}

// GetBkCIPipeline 获取蓝盾流水线详情
//
//	@ID			GetBkCIPipeline
//	@Summary	获取蓝盾流水线详情
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		pipelineID	path		string	true	"流水线 ID"
//	@Success	200			{object}	serializer.GetBkCIPipelineOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkci-pipelines/{pipelineID} [get]
func (h *Handler) GetBkCIPipeline(c *gin.Context) {
	var uriInput slz.BkCIPipelineURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get bkci project"))
		return
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return
	}

	pipeline, err := client.GetPipeline(ctx, project.Code, uriInput.PipelineID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get bkci pipeline"))
		return
	}

	ginutils.OK(c, &slz.GetBkCIPipelineOutput{Data: new(slz.BkCIPipelineDetailOutput).FromModel(*pipeline)})
}

// GetBkCIPipelineVariables 获取蓝盾流水线变量列表
//
//	@ID			GetBkCIPipelineVariables
//	@Summary	获取蓝盾流水线变量列表
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		pipelineID	path		string	true	"流水线 ID"
//	@Success	200			{object}	serializer.GetBkCIPipelineVariablesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkci-pipelines/{pipelineID}/variables [get]
func (h *Handler) GetBkCIPipelineVariables(c *gin.Context) {
	var uriInput slz.BkCIPipelineURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace project"))
		return
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return
	}

	pipeline, err := client.GetPipeline(ctx, project.Code, uriInput.PipelineID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get bkci pipeline"))
		return
	}

	ginutils.OK(
		c,
		&slz.GetBkCIPipelineVariablesOutput{
			Data: lo.Map(pipeline.Variables, func(v bkci.PipelineVariable, _ int) *slz.BkCIPipelineVariableOutput {
				return new(slz.BkCIPipelineVariableOutput).FromModel(v)
			}),
		},
	)
}

// ListBkCIRepositoryBranches 获取代码仓库分支列表
//
//	@ID			ListBkCIRepositoryBranches
//	@Summary	获取代码仓库分支列表
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		repositoryID	query		string	true	"代码仓库 ID 或名称"
//	@Param		repositoryType	query		string	true	"代码仓库标识类型，可选值: ID, NAME"
//	@Param		search		query		string	false	"搜索关键词"
//	@Param		page		query		int		true	"页码，最小为 1"
//	@Param		pageSize	query		int		true	"每页数量，可选值: 5, 10, 20, 50, 100"
//	@Success	200			{object}	serializer.ListBkCIRepositoryBranchesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkci-repositories/branches [get]
func (h *Handler) ListBkCIRepositoryBranches(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	var queryInput slz.BkCIRepositoryRefsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace project"))
		return
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return
	}

	refs, err := client.ListRepositoryBranches(
		ctx,
		project.Code,
		queryInput.RepositoryID,
		queryInput.RepositoryType,
		queryInput.Search,
		queryInput.Page,
		queryInput.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkci repository branches"))
		return
	}

	ginutils.OK(c, &slz.ListBkCIRepositoryBranchesOutput{Data: newBkCIRepositoryRefsOutput(refs)})
}

// ListBkCIRepositoryTags 获取代码仓库标签列表
//
//	@ID			ListBkCIRepositoryTags
//	@Summary	获取代码仓库标签列表
//	@Tags		bkintegrations-bkci
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		repositoryID	query		string	true	"代码仓库 ID 或名称"
//	@Param		repositoryType	query		string	true	"代码仓库标识类型，可选值: ID, NAME"
//	@Param		search		query		string	false	"搜索关键词"
//	@Param		page		query		int		true	"页码，最小为 1"
//	@Param		pageSize	query		int		true	"每页数量，可选值: 5, 10, 20, 50, 100"
//	@Success	200			{object}	serializer.ListBkCIRepositoryTagsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/bkci-repositories/tags [get]
func (h *Handler) ListBkCIRepositoryTags(c *gin.Context) {
	var uriInput slz.BkCIWorkspaceURIInput
	var queryInput slz.BkCIRepositoryRefsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	project, err := h.registry.BkCIProjectStore.GetByWorkspace(ctx, uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace project"))
		return
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return
	}

	refs, err := client.ListRepositoryTags(
		ctx,
		project.Code,
		queryInput.RepositoryID,
		queryInput.RepositoryType,
		queryInput.Search,
		queryInput.Page,
		queryInput.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkci repository tags"))
		return
	}

	ginutils.OK(c, &slz.ListBkCIRepositoryTagsOutput{Data: newBkCIRepositoryRefsOutput(refs)})
}

func newBkCIRepositoryRefsOutput(refs []bkci.RepositoryRef) []*slz.BkCIRepositoryRefOutput {
	return lo.Map(refs, func(ref bkci.RepositoryRef, _ int) *slz.BkCIRepositoryRefOutput {
		return new(slz.BkCIRepositoryRefOutput).FromModel(ref)
	})
}
