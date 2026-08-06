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

// Package handler contains Gin handlers for app networking APIs.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking/serializer"
)

// Handler handles Gin networking API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// CreateAppService 创建应用下的 Service
//
//	@ID			CreateAppService
//	@Summary	创建应用下的 Service
//	@Tags		app-networking
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string						true	"应用 ID"
//	@Param		body	body		slz.CreateAppServiceInput	true	"创建 Service 请求体"
//	@Success	200		{object}	slz.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	500		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/services [post]
func (h *Handler) CreateAppService(c *gin.Context) {
	var uriInput slz.AppURIInput
	var input slz.CreateAppServiceInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	appObj, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ports := lo.Map(input.Ports, func(p slz.ServicePortInput, _ int) networking.ServicePort {
		return networking.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort,
			Protocol:   networking.Protocol(p.Protocol),
		}
	})
	svc := &networking.Service{
		AppID:    appObj.ID,
		Name:     input.Name,
		Selector: input.Selector,
		Ports:    ports,
	}
	if input.TrafficLaneEnabled != nil {
		svc.TrafficLaneEnabled = *input.TrafficLaneEnabled
	}

	if err = h.registry.AppServiceStore.Create(ctx, svc); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "create app service"))
		return
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// ListAppServices 获取应用下的 Services
//
//	@ID			ListAppServices
//	@Summary	获取应用下的 Service 列表
//	@Tags		app-networking
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	slz.ListAppServicesOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	500		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/services [get]
func (h *Handler) ListAppServices(c *gin.Context) {
	var uriInput slz.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	appObj, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	services, err := h.registry.AppServiceStore.ListByApp(ctx, appObj.ID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list app services"))
		return
	}

	ginutils.OK(
		c,
		&slz.ListAppServicesOutput{Data: lo.Map(services, func(svc networking.Service, _ int) *slz.AppServiceOutput {
			return new(slz.AppServiceOutput).FromModel(svc)
		})},
	)
}

// UpdateAppService 更新应用下的 Service
//
//	@ID			UpdateAppService
//	@Summary	更新应用下的 Service
//	@Tags		app-networking
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string						true	"应用 ID"
//	@Param		name	path		string						true	"Service 名称"
//	@Param		body	body		slz.UpdateAppServiceInput	true	"更新 Service 请求体"
//	@Success	200		{object}	slz.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	500		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/services/{name} [put]
func (h *Handler) UpdateAppService(c *gin.Context) {
	var uriInput slz.AppServiceURIInput
	var input slz.UpdateAppServiceInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	appObj, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ports := lo.Map(input.Ports, func(p slz.ServicePortInput, _ int) networking.ServicePort {
		return networking.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort,
			Protocol:   networking.Protocol(p.Protocol),
		}
	})

	updateData := &networking.SvcUpdateData{
		Selector:           input.Selector,
		Ports:              ports,
		TrafficLaneEnabled: input.TrafficLaneEnabled,
	}

	if err = h.registry.AppServiceStore.Update(ctx, appObj.ID, uriInput.Name, updateData); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "update app service"))
		return
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// DeleteAppService 删除应用下的 Service
//
//	@ID			DeleteAppService
//	@Summary	删除应用下的 Service
//	@Tags		app-networking
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		name	path		string	true	"Service 名称"
//	@Success	200		{object}	slz.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	500		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/services/{name} [delete]
func (h *Handler) DeleteAppService(c *gin.Context) {
	var uriInput slz.AppServiceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	appObj, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.registry.AppServiceStore.Delete(ctx, appObj.ID, uriInput.Name); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "delete app service"))
		return
	}

	ginutils.OK(c, slz.EmptyOutput{})
}

// ListTrafficLaneCandidateApps 查询空间下的候选应用列表(用于泳道关联)
//
//	@ID			ListTrafficLaneCandidateApps
//	@Summary	查询空间下的泳道候选应用列表
//	@Tags		app-networking
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		workspaceID	path		string	true	"工作空间 ID"
//	@Success	200			{object}	slz.ListTrafficLaneCandidateAppsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	500			{object}	bkerrs.GinErrorOutput
//	@Router		/workspaces/{workspaceID}/traffic-lanes/candidate-apps [get]
func (h *Handler) ListTrafficLaneCandidateApps(c *gin.Context) {
	var uriInput slz.WorkspaceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	apps, err := h.registry.AppStore.ListApps(ctx, &app.ListOpts{
		WorkspaceID: ws.ID,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list apps"))
		return
	}

	appNames := lo.SliceToMap(apps, func(a *app.Application) (string, string) {
		return a.ID, a.Name
	})

	grouped, err := h.registry.AppServiceStore.GroupByAppID(ctx, lo.Keys(appNames))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "group services by app id"))
		return
	}

	ginutils.OK(
		c,
		&slz.ListTrafficLaneCandidateAppsOutput{
			Data: lo.MapToSlice(appNames, func(appID, appName string) slz.TrafficLaneCandidateAppOutput {
				return slz.TrafficLaneCandidateAppOutput{
					AppName: appName,
					Services: lo.Map(
						grouped[appID],
						func(svc networking.Service, _ int) slz.CandidateAppServiceOutput {
							return slz.CandidateAppServiceOutput{
								Name:               svc.Name,
								TrafficLaneEnabled: svc.TrafficLaneEnabled,
							}
						},
					),
				}
			}),
		},
	)
}
