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

// Package handler 代码库接口的 Gin 视图逻辑
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler 代码库接口的 Gin handler
type Handler struct {
	registry *storereg.Registry
}

// New 创建代码库 Handler
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// TestConnectivity 测试代码库连通性
//
//	@ID				TestCodeRepositoryConnectivity
//	@Summary		测试代码库连通性
//	@Description	校验仓库地址与凭证，不保存代码库
//	@Tags			code-repositories
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string						true	"工作空间 ID"
//	@Param			body		body		slz.ConnectivityTestInput	true	"代码库配置"
//	@Success		200			{object}	slz.EmptyOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/code-repositories/connectivity-test [post]
func (h *Handler) TestConnectivity(c *gin.Context) {
	var uriInput slz.WorkspaceURIInput
	var input slz.ConnectivityTestInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	if _, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	controller, err := coderepo.Probe(ctx, h.registry.DB, input.ToSpec(uriInput.WorkspaceID))
	if err == nil {
		err = controller.Touch(ctx)
	}
	if err != nil {
		bkerrs.AbortWithErr(c, coderepo.ToBkErr(err, "test repository connectivity"))
		return
	}
	ginutils.OK(c, slz.EmptyOutput{})
}

// ListRefs 查询未保存代码库的分支或 Tag
//
//	@ID				ListCodeRepositoryRefs
//	@Summary		查询代码库分支或 Tag
//	@Description	查询未保存代码库的分支或 Tag，已保存的使用 bkci-repositories/branches、tags
//	@Tags			code-repositories
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string				true	"工作空间 ID"
//	@Param			body		body		slz.ListRefsInput	true	"代码库配置与查询条件"
//	@Success		200			{object}	slz.ListRefsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/code-repositories/refs [post]
func (h *Handler) ListRefs(c *gin.Context) {
	var uriInput slz.WorkspaceURIInput
	var input slz.ListRefsInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	if _, err := ginperm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, ginperm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	controller, err := coderepo.Probe(ctx, h.registry.DB, input.ToSpec(uriInput.WorkspaceID))
	if err != nil {
		bkerrs.AbortWithErr(c, coderepo.ToBkErr(err, "list repository refs"))
		return
	}
	refs, err := controller.ListRefs(ctx, input.ToQuery())
	if err != nil {
		bkerrs.AbortWithErr(c, coderepo.ToBkErr(err, "list repository refs"))
		return
	}
	ginutils.OK(c, slz.NewListRefsOutput(refs))
}
