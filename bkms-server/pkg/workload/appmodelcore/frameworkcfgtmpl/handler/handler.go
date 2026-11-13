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

// Package handler contains Gin handlers for workspace framework config templates.
package handler

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/frameworkcfgtmpl"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/frameworkcfgtmpl/serializer"
)

// Handler handles workspace framework config template APIs.
type Handler struct {
	registry *storereg.Registry
}

// New creates a framework config template API handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListTemplates lists framework config templates of a workspace.
//
//	@ID			ListWorkspaceFrameworkConfigTemplates
//	@Summary	查询工作空间框架配置模板列表
//	@Description	返回平台内置模板与空间自定义模板，每个框架、语言先列内置模板；不返回模板内容
//	@Tags		framework-config-templates
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		framework	query		string	false	"框架：trpc、taf"
//	@Param		language	query		string	false	"语言：go、cpp"
//	@Success	200			{object}	serializer.ListTemplatesOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/framework-config-templates [get]
func (h *Handler) ListTemplates(c *gin.Context) {
	var uri serializer.WorkspaceURIInput
	var query serializer.ListTemplatesQueryInput
	if err := ginutils.BindURIQuery(c, &uri, &query); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	views, err := frameworkcfgtmpl.List(ctx, h.registry.DB, workspace.ID, frameworkcfgtmpl.Group{
		Framework: query.Framework,
		Language:  query.Language,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "list framework templates"))
		return
	}
	data := lo.Map(views, func(view *frameworkcfgtmpl.View, _ int) *serializer.TemplateOutputObj {
		return new(serializer.TemplateOutputObj).FromView(view)
	})
	ginutils.OK(c, serializer.ListTemplatesOutput{Data: data})
}

// GetTemplate returns a framework config template with its content.
//
//	@ID			GetWorkspaceFrameworkConfigTemplate
//	@Summary	查询工作空间框架配置模板详情
//	@Tags		framework-config-templates
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		templateID	path		string	true	"模板 ID，内置模板为 builtin-{framework}-{language}"
//	@Success	200			{object}	serializer.TemplateOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/framework-config-templates/{templateID} [get]
func (h *Handler) GetTemplate(c *gin.Context) {
	var uri serializer.TemplateURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	view, err := frameworkcfgtmpl.Get(ctx, h.registry.DB, workspace.ID, uri.TemplateID)
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "get framework template"))
		return
	}
	ginutils.OK(c, serializer.TemplateOutput{Data: new(serializer.TemplateOutputObj).FromView(view)})
}

// CreateTemplate creates a workspace framework config template.
//
//	@ID			CreateWorkspaceFrameworkConfigTemplate
//	@Summary	新建工作空间框架配置模板
//	@Tags		framework-config-templates
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		body		body		serializer.CreateTemplateInput	true	"框架配置模板"
//	@Success	200			{object}	serializer.TemplateOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	409			{object}	bkerrs.GinErrorOutput	"同一框架、语言下模板名称已存在，或并发设置了其它默认模板"
//	@Router		/workspaces/{workspaceID}/framework-config-templates [post]
func (h *Handler) CreateTemplate(c *gin.Context) {
	var uri serializer.WorkspaceURIInput
	var input serializer.CreateTemplateInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	created, err := frameworkcfgtmpl.Create(ctx, h.registry.DB, workspace.ID, input.ToParams(auth.MustGetUser(ctx).ID))
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "create framework template"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeCreate, created.ID, nil, created)
	ginutils.OK(c, serializer.TemplateOutput{Data: new(serializer.TemplateOutputObj).FromView(created)})
}

// UpdateTemplate updates a workspace framework config template.
//
//	@ID			UpdateWorkspaceFrameworkConfigTemplate
//	@Summary	编辑工作空间框架配置模板
//	@Description	框架与语言不可修改；平台内置模板只读
//	@Tags		framework-config-templates
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string							true	"工作空间 ID"
//	@Param		templateID	path		string							true	"模板 ID"
//	@Param		body		body		serializer.UpdateTemplateInput	true	"框架配置模板"
//	@Success	200			{object}	serializer.TemplateOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	409			{object}	bkerrs.GinErrorOutput	"同一框架、语言下模板名称已存在，或并发设置了其它默认模板"
//	@Router		/workspaces/{workspaceID}/framework-config-templates/{templateID} [put]
func (h *Handler) UpdateTemplate(c *gin.Context) {
	var uri serializer.TemplateURIInput
	var input serializer.UpdateTemplateInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	before, after, err := frameworkcfgtmpl.Update(
		ctx, h.registry.DB, workspace.ID, uri.TemplateID, input.ToParams(auth.MustGetUser(ctx).ID),
	)
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "update framework template"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeUpdate, after.ID, before, after)
	ginutils.OK(c, serializer.TemplateOutput{Data: new(serializer.TemplateOutputObj).FromView(after)})
}

// DeleteTemplate deletes a workspace framework config template.
//
//	@ID			DeleteWorkspaceFrameworkConfigTemplate
//	@Summary	删除工作空间框架配置模板
//	@Description	删除默认模板后，该框架、语言回落到平台内置模板；平台内置模板不能删除
//	@Tags		framework-config-templates
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		templateID	path		string	true	"模板 ID"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/framework-config-templates/{templateID} [delete]
func (h *Handler) DeleteTemplate(c *gin.Context) {
	var uri serializer.TemplateURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	deleted, err := frameworkcfgtmpl.Delete(ctx, h.registry.DB, workspace.ID, uri.TemplateID)
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "delete framework template"))
		return
	}
	recordAudit(ctx, workspace.ID, audit.OperationTypeDelete, deleted.ID, deleted, nil)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// SetDefaultTemplate sets a template as the default of its framework and language.
//
//	@ID			SetWorkspaceFrameworkConfigTemplateDefault
//	@Summary	设为默认框架配置模板
//	@Description	创建应用时默认使用该模板；对平台内置模板调用表示恢复使用平台默认模板
//	@Tags		framework-config-templates
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Param		templateID	path		string	true	"模板 ID，内置模板为 builtin-{framework}-{language}"
//	@Success	200			{object}	serializer.TemplateOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	409			{object}	bkerrs.GinErrorOutput	"并发设置了其它默认模板"
//	@Router		/workspaces/{workspaceID}/framework-config-templates/{templateID}/default [put]
func (h *Handler) SetDefaultTemplate(c *gin.Context) {
	var uri serializer.TemplateURIInput
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	workspace, err := perm.ValidateWorkspaceByID(ctx, h.registry, uri.WorkspaceID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	before, after, err := frameworkcfgtmpl.SetDefault(ctx, h.registry.DB, workspace.ID, uri.TemplateID)
	if err != nil {
		bkerrs.AbortWithErr(c, apiError(err, "set default framework template"))
		return
	}
	if !before.IsDefault {
		// 只有默认状态发生变化，审计中不记录文件内容
		recordAudit(
			ctx,
			workspace.ID,
			audit.OperationTypeUpdate,
			after.ID,
			withoutContent(before),
			withoutContent(after),
		)
	}
	ginutils.OK(c, serializer.TemplateOutput{Data: new(serializer.TemplateOutputObj).FromView(after)})
}

func withoutContent(view *frameworkcfgtmpl.View) *frameworkcfgtmpl.View {
	copied := *view
	copied.Content = ""
	return &copied
}

func apiError(err error, operation string) error {
	switch {
	case errors.Is(err, frameworkcfgtmpl.ErrTemplateNotFound):
		return bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, operation)
	case errors.Is(err, frameworkcfgtmpl.ErrTemplateConflict):
		return bkerrs.Wrap(err, bkerrs.ErrCodeAlreadyExists, operation)
	case errors.Is(err, frameworkcfgtmpl.ErrInvalidTemplate),
		errors.Is(err, frameworkcfgtmpl.ErrBuiltinReadOnly):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, operation)
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, operation)
	}
}

func recordAudit(
	ctx context.Context,
	workspaceID string,
	operation audit.OperationType,
	templateID string,
	before, after *frameworkcfgtmpl.View,
) {
	options := []audit.Option{audit.WithWorkspaceID(workspaceID)}
	if before != nil {
		options = append(options, audit.WithDataBefore(new(serializer.TemplateOutputObj).FromView(before)))
	}
	if after != nil {
		options = append(options, audit.WithDataAfter(new(serializer.TemplateOutputObj).FromView(after)))
	}
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		operation,
		audit.ResourceTypeFrameworkConfigTemplate,
		templateID,
		options...,
	)
}
