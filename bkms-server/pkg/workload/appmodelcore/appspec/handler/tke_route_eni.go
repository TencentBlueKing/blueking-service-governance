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

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// GetAppDefaultAppSpecTkeRouteEni 获取应用默认 tkeRouteEni section 配置。
//
//	@ID			GetAppDefaultAppSpecTkeRouteEni
//	@Summary	获取应用默认 tkeRouteEni section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.AppSpecTkeRouteEniSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-tke-route-eni [get]
func (h *Handler) GetAppDefaultAppSpecTkeRouteEni(c *gin.Context) {
	getDefaultSection(h, c, appspec.TkeRouteEniSection, "tkeRouteEni", tkeRouteEniOutput)
}

// SetAppDefaultAppSpecTkeRouteEni 设置应用默认 tkeRouteEni section 配置。
//
//	@ID			SetAppDefaultAppSpecTkeRouteEni
//	@Summary	设置应用默认 tkeRouteEni section 配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string									true	"应用 ID"
//	@Param		body	body		serializer.AppSpecTkeRouteEniInput		true	"tkeRouteEni 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-tke-route-eni [put]
func (h *Handler) SetAppDefaultAppSpecTkeRouteEni(c *gin.Context) {
	setDefaultSection(h, c, appspec.TkeRouteEniSection, "tkeRouteEni", defaultTkeRouteEniInput)
}

// GetEnvAppSpecTkeRouteEni 获取应用在某环境下的 tkeRouteEni section 覆盖配置。
//
//	@ID			GetEnvAppSpecTkeRouteEni
//	@Summary	获取应用环境 tkeRouteEni section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecTkeRouteEniSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/tke-route-eni [get]
func (h *Handler) GetEnvAppSpecTkeRouteEni(c *gin.Context) {
	getEnvSection(h, c, appspec.TkeRouteEniSection, "tkeRouteEni", tkeRouteEniOutput)
}

// GetEnvEffectiveAppSpecTkeRouteEni 获取应用在某环境下实际生效的 tkeRouteEni section 配置。
//
//	@ID			GetEnvEffectiveAppSpecTkeRouteEni
//	@Summary	获取应用环境最终生效的 tkeRouteEni section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecTkeRouteEniSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/tke-route-eni/effective [get]
func (h *Handler) GetEnvEffectiveAppSpecTkeRouteEni(c *gin.Context) {
	getEnvEffectiveSection(h, c, appspec.TkeRouteEniSection, "tkeRouteEni", tkeRouteEniOutput)
}

// SetEnvAppSpecTkeRouteEni 设置应用在某环境下的 tkeRouteEni section 覆盖配置。
//
//	@ID			SetEnvAppSpecTkeRouteEni
//	@Summary	设置应用环境 tkeRouteEni section 覆盖配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"环境名称"
//	@Param		body	body		serializer.AppSpecTkeRouteEniInput	true	"tkeRouteEni 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/tke-route-eni [put]
func (h *Handler) SetEnvAppSpecTkeRouteEni(c *gin.Context) {
	setEnvSection(h, c, appspec.TkeRouteEniSection, "tkeRouteEni", envTkeRouteEniInput)
}

// DeleteEnvAppSpecTkeRouteEni 删除应用在某环境下的 tkeRouteEni section 覆盖配置。
//
//	@ID			DeleteEnvAppSpecTkeRouteEni
//	@Summary	删除应用环境 tkeRouteEni section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/tke-route-eni [delete]
func (h *Handler) DeleteEnvAppSpecTkeRouteEni(c *gin.Context) {
	deleteEnvSection(h, c, appspec.TkeRouteEniSection, "tkeRouteEni")
}

func tkeRouteEniOutput(
	spec *appspec.TkeRouteEniSpec,
	_ *bkmsapp.Application,
) *serializer.AppSpecTkeRouteEniOutput {
	return new(serializer.AppSpecTkeRouteEniOutput).FromModel(spec)
}

func defaultTkeRouteEniInput(
	input serializer.AppSpecTkeRouteEniInput,
	_ *bkmsapp.Application,
) *appspec.TkeRouteEniSpec {
	return input.ToModel()
}

func envTkeRouteEniInput(
	input serializer.AppSpecTkeRouteEniInput,
	_ *bkmsapp.Application,
) *appspec.TkeRouteEniSpec {
	return input.ToModel()
}
