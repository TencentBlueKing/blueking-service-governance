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
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	cfgmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/serializer"
	svc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// InitMetadata 初始化配置管理（幂等）。
//
//	@ID			InitBscpCfgMetadata
//	@Summary	初始化配置管理
//	@Tags		bscpcfg
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	slz.MetadataResponse
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/metadata [post]
func (h *Handler) InitMetadata(c *gin.Context) {
	var uri slz.AppIDURI
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uri.AppID, perm.TypeEdit)
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

	var workloadName string
	var workloadKind string
	if app.Type == bkmsapp.AppTypeTRPC {
		workloadName = app.Name
		workloadKind = k8skind.GameDeploy
	}

	appConfig, err := mgr.InitMetadata(ctx, &svc.InitMetadataParams{
		AppID:        app.ID,
		WorkloadName: workloadName,
		WorkloadKind: workloadKind,
		BscpBizID:    ws.BkSystems.BkCCBizID,
		Operator:     auth.MustGetUser(ctx).ID,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "init metadata"))
		return
	}

	ginutils.OK(c, &slz.MetadataResponse{Data: new(slz.MetadataOutput).FromModel(appConfig)})
}

// PatchMetadata 修改配置管理元信息（mountPath 等）。
//
//	@ID			PatchBscpCfgMetadata
//	@Summary	修改配置管理元信息（mountPath）
//	@Tags		bscpcfg
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string					true	"应用 ID"
//	@Param		body	body		slz.PatchMetadataInput	true	"更新配置请求体"
//	@Success	200		{object}	slz.MetadataResponse
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/metadata [patch]
func (h *Handler) PatchMetadata(c *gin.Context) {
	var uri slz.AppIDURI
	var input slz.PatchMetadataInput
	if err := ginutils.BindURIJSON(c, &uri, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	updateData, err := input.ToUpdateModel()
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uri.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr, err := h.newManager(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	appConfig, err := h.registry.BscpCfgStore.GetMetadata(ctx, app.ID)
	if err != nil {
		if errors.Is(err, cfgmodel.ErrMetadataNotFound) {
			bkerrs.AbortWithErr(
				c, bkerrs.New(bkerrs.ErrCodeNotFound, "bscpcfg metadata not found, please init first"),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get bscpcfg metadata"))
		return
	}

	if err = mgr.PatchMetadata(ctx, app.ID, updateData); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update metadata"))
		return
	}

	// 将更新后的值反映到返回结果
	updateData.ApplyTo(appConfig)
	ginutils.OK(c, &slz.MetadataResponse{Data: new(slz.MetadataOutput).FromModel(appConfig)})
}

// DeleteMetadata 删除配置管理元信息（级联删除该应用下所有环境绑定）。
//
//	@ID			DeleteBscpCfgMetadata
//	@Summary	删除配置管理元信息（级联删除所有环境绑定）
//	@Tags		bscpcfg
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/metadata [delete]
func (h *Handler) DeleteMetadata(c *gin.Context) {
	var uri slz.AppIDURI
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uri.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	mgr, err := h.newManager(c)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = mgr.DeleteByApp(ctx, app.ID); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete bscpcfg metadata"))
		return
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// GetMetadata 获取配置管理元信息。
//
//	@ID			GetBscpCfgMetadata
//	@Summary	获取配置管理元信息
//	@Tags		bscpcfg
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	slz.MetadataResponse
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/metadata [get]
func (h *Handler) GetMetadata(c *gin.Context) {
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

	appConfig, err := h.registry.BscpCfgStore.GetMetadata(ctx, app.ID)
	if err != nil {
		if errors.Is(err, cfgmodel.ErrMetadataNotFound) {
			bkerrs.AbortWithErr(
				c, bkerrs.New(bkerrs.ErrCodeNotFound, "bscpcfg metadata not found, please init first"),
			)
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get bscpcfg metadata"))
		return
	}

	ginutils.OK(c, &slz.MetadataResponse{Data: new(slz.MetadataOutput).FromModel(appConfig)})
}
