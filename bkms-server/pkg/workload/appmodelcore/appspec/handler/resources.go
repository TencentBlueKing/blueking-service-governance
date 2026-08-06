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

// GetAppDefaultAppSpecResources 获取应用默认 resources section 配置。
//
//	@ID			GetAppDefaultAppSpecResources
//	@Summary	获取应用默认 resources section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.AppSpecResourcesSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-resources [get]
func (h *Handler) GetAppDefaultAppSpecResources(c *gin.Context) {
	getDefaultSection(h, c, appspec.ResourcesSection, "resources", resourcesOutput)
}

// SetAppDefaultAppSpecResources 设置应用默认 resources section 配置。
//
//	@ID			SetAppDefaultAppSpecResources
//	@Summary	设置应用默认 resources section 配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string										true	"应用 ID"
//	@Param		body	body		serializer.SetAppDefaultAppSpecResourcesInput	true	"resources 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-resources [put]
func (h *Handler) SetAppDefaultAppSpecResources(c *gin.Context) {
	setDefaultSection(h, c, appspec.ResourcesSection, "resources", defaultResourcesInput)
}

// GetEnvAppSpecResources 获取应用在某环境下的 resources section 配置。
//
//	@ID			GetEnvAppSpecResources
//	@Summary	获取应用环境 resources section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecResourcesSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/resources [get]
func (h *Handler) GetEnvAppSpecResources(c *gin.Context) {
	getEnvSection(h, c, appspec.ResourcesSection, "resources", resourcesOutput)
}

// GetEnvEffectiveAppSpecResources 获取应用在某环境下实际生效的 resources section 配置。
//
//	@ID			GetEnvEffectiveAppSpecResources
//	@Summary	获取应用环境最终生效的 resources section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecResourcesSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/resources/effective [get]
func (h *Handler) GetEnvEffectiveAppSpecResources(c *gin.Context) {
	getEnvEffectiveSection(h, c, appspec.ResourcesSection, "resources", resourcesOutput)
}

// SetEnvAppSpecResources 设置应用在某环境下的 resources section 配置。
//
//	@ID			SetEnvAppSpecResources
//	@Summary	设置应用环境 resources section 覆盖配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"环境名称"
//	@Param		body	body		serializer.SetEnvAppSpecResourcesInput	true	"resources 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/resources [put]
func (h *Handler) SetEnvAppSpecResources(c *gin.Context) {
	setEnvSection(h, c, appspec.ResourcesSection, "resources", envResourcesInput)
}

// DeleteEnvAppSpecResources 删除应用在某环境下的 resources section 配置。
//
//	@ID			DeleteEnvAppSpecResources
//	@Summary	删除应用环境 resources section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/resources [delete]
func (h *Handler) DeleteEnvAppSpecResources(c *gin.Context) {
	deleteEnvSection(h, c, appspec.ResourcesSection, "resources")
}

func resourcesOutput(
	spec *appspec.ResourcesSpec,
	_ *bkmsapp.Application,
) *serializer.AppSpecResourcesOutput {
	return new(serializer.AppSpecResourcesOutput).FromModel(spec)
}

func defaultResourcesInput(
	input serializer.SetAppDefaultAppSpecResourcesInput,
	_ *bkmsapp.Application,
) *appspec.ResourcesSpec {
	return input.AppSpecResources.ToModel()
}

func envResourcesInput(
	input serializer.SetEnvAppSpecResourcesInput,
	_ *bkmsapp.Application,
) *appspec.ResourcesSpec {
	return input.AppSpecResources.ToModel()
}
