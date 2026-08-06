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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	cfgmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/serializer"
	svc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// CreateEnvBinding 创建环境绑定（前置条件：已调用 InitMetadata）。
//
//	@ID			CreateBscpCfgEnvBinding
//	@Summary	创建环境绑定
//	@Tags		bscpcfg
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		envName		path		string	true	"环境名称"
//	@Success	200			{object}	slz.EnvBindingResponse
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/envs/{envName}/binding [post]
func (h *Handler) CreateEnvBinding(c *gin.Context) {
	var uri slz.AppEnvURI
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uri.AppID, uri.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ws, err := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}
	if ws.BkSystems.BkCCBizID == "" {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeNotFound, "workspace missing bizID"))
		return
	}

	mgr, err := h.newManager(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	config, err := mgr.CreateEnvBinding(ctx, &svc.CreateEnvBindingParams{
		AppID:     app.ID,
		AppName:   app.Name,
		EnvName:   strings.TrimSpace(uri.EnvName),
		Workspace: ws,
		BscpBizID: ws.BkSystems.BkCCBizID,
		Operator:  auth.MustGetUser(ctx).ID,
	})
	if err != nil {
		if errors.Is(err, cfgmodel.ErrEnvBindingAlreadyExists) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeAlreadyExists,
				"bscpcfg env binding already exists for app %s env %s", app.ID, uri.EnvName))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create bscpcfg env binding"))
		return
	}

	ginutils.OK(c, &slz.EnvBindingResponse{Data: new(slz.EnvBindingOutput).FromModel(config)})
}

// DeleteEnvBinding 删除环境绑定。
//
//	@ID			DeleteBscpCfgEnvBinding
//	@Summary	删除环境绑定
//	@Tags		bscpcfg
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		envName		path		string	true	"环境名称"
//	@Success	200
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/envs/{envName}/binding [delete]
func (h *Handler) DeleteEnvBinding(c *gin.Context) {
	var uri slz.AppEnvURI
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uri.AppID, uri.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr, err := h.newManager(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = mgr.DeleteEnvBinding(ctx, app.ID, uri.EnvName); err != nil {
		if errors.Is(err, cfgmodel.ErrEnvBindingNotFound) {
			bkerrs.AbortWithErr(
				c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "bscpcfg env binding not found for env %s", uri.EnvName),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete bscpcfg env binding"))
		return
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// PatchEnvBinding 更新环境绑定的下发服务列表。
//
//	@ID			PatchBscpCfgEnvBinding
//	@Summary	更新环境绑定（更新绑定的服务列表）
//	@Tags		bscpcfg
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string						true	"应用 ID"
//	@Param		envName		path		string						true	"环境名称"
//	@Param		body		body		slz.PatchEnvBindingInput	true	"更新配置请求体"
//	@Success	200
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/envs/{envName}/binding [patch]
func (h *Handler) PatchEnvBinding(c *gin.Context) {
	var uri slz.AppEnvURI
	var input slz.PatchEnvBindingInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if input.Services == nil {
		ginutils.OK(c, slz.EmptyOutput{})
		return
	}

	newApps := lo.UniqBy(
		lo.FilterMap(input.Services, func(obj slz.ServiceRefInput, _ int) (cfgmodel.ServiceRef, bool) {
			id := strings.TrimSpace(obj.ID)
			name := strings.TrimSpace(obj.Name)
			if id == "" || name == "" {
				return cfgmodel.ServiceRef{}, false
			}
			return cfgmodel.ServiceRef{ID: id, Name: name}, true
		}),
		func(app cfgmodel.ServiceRef) string { return app.ID },
	)
	if len(newApps) == 0 {
		ginutils.OK(c, slz.EmptyOutput{})
		return
	}

	ctx := c.Request.Context()
	app, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uri.AppID, uri.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ws, err := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get workspace"))
		return
	}
	if ws.BkSystems.BkCCBizID == "" {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeNotFound, "workspace missing bizID"))
		return
	}

	mgr, err := h.newManager(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = mgr.BindServices(ctx, app.ID, uri.EnvName, ws.BkSystems.BkCCBizID, newApps); err != nil {
		if errors.Is(err, cfgmodel.ErrEnvBindingNotFound) {
			bkerrs.AbortWithErr(
				c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "bscpcfg env binding not found for env %s", uri.EnvName),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update bscpcfg env binding"))
		return
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// GetEnvBinding 获取指定环境的绑定详情。
//
//	@ID			GetBscpCfgEnvBinding
//	@Summary	获取指定环境的绑定详情
//	@Tags		bscpcfg
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		envName		path		string	true	"环境名称"
//	@Success	200			{object}	slz.EnvBindingResponse
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/envs/{envName}/binding [get]
func (h *Handler) GetEnvBinding(c *gin.Context) {
	var uri slz.AppEnvURI
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uri.AppID, uri.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr, err := h.newManager(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	config, err := mgr.GetSnapshot(ctx, app.ID, uri.EnvName)
	if err != nil {
		switch {
		case errors.Is(err, cfgmodel.ErrMetadataNotFound):
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "bscpcfg metadata not found for app %s", uri.AppID),
			)
		case errors.Is(err, cfgmodel.ErrEnvBindingNotFound):
			bkerrs.AbortWithErr(
				c,
				bkerrs.Errorf(bkerrs.ErrCodeNotFound, "bscpcfg env binding not found for env %s", uri.EnvName),
			)
		default:
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get bscpcfg env binding"))
		}
		return
	}
	ginutils.OK(c, &slz.EnvBindingResponse{Data: new(slz.EnvBindingOutput).FromModel(config)})
}

// ListEnvBindings 获取所有环境的绑定列表。
//
//	@ID			ListBscpCfgEnvBindings
//	@Summary	获取所有环境的绑定列表
//	@Tags		bscpcfg
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	slz.EnvBindingListResponse
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/envs [get]
func (h *Handler) ListEnvBindings(c *gin.Context) {
	var uri slz.AppIDURI
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uri.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr, err := h.newManager(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	configs, err := mgr.ListSnapshots(ctx, app.ID)
	if err != nil {
		if errors.Is(err, cfgmodel.ErrEnvBindingNotFound) {
			ginutils.OK(c, &slz.EnvBindingListResponse{Data: []*slz.EnvBindingOutput{}})
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list env bindings"))
		return
	}

	ginutils.OK(
		c,
		&slz.EnvBindingListResponse{
			Data: lo.Map(configs, func(config *cfgmodel.Snapshot, _ int) *slz.EnvBindingOutput {
				return new(slz.EnvBindingOutput).FromModel(config)
			}),
		},
	)
}
