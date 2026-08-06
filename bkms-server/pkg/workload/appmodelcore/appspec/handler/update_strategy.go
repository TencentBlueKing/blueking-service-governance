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

// GetAppDefaultAppSpecUpdateStrategy 获取应用默认 updateStrategy section 配置。
//
//	@ID			GetAppDefaultAppSpecUpdateStrategy
//	@Summary	获取应用默认 updateStrategy section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.AppSpecUpdateStrategySectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-update-strategy [get]
func (h *Handler) GetAppDefaultAppSpecUpdateStrategy(c *gin.Context) {
	getDefaultSection(h, c, appspec.UpdateStrategySection, "update strategy", updateStrategyOutput)
}

// SetAppDefaultAppSpecUpdateStrategy 设置应用默认 updateStrategy section 配置。
//
//	@ID			SetAppDefaultAppSpecUpdateStrategy
//	@Summary	设置应用默认 updateStrategy section 配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string												true	"应用 ID"
//	@Param		body	body		serializer.SetAppDefaultAppSpecUpdateStrategyInput	true	"updateStrategy 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-update-strategy [put]
func (h *Handler) SetAppDefaultAppSpecUpdateStrategy(c *gin.Context) {
	setDefaultSection(h, c, appspec.UpdateStrategySection, "update strategy", defaultUpdateStrategyInput)
}

// GetEnvAppSpecUpdateStrategy 获取应用在某环境下的 updateStrategy section 配置。
//
//	@ID			GetEnvAppSpecUpdateStrategy
//	@Summary	获取应用环境 updateStrategy section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecUpdateStrategySectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/update-strategy [get]
func (h *Handler) GetEnvAppSpecUpdateStrategy(c *gin.Context) {
	getEnvSection(h, c, appspec.UpdateStrategySection, "update strategy", updateStrategyOutput)
}

// GetEnvEffectiveAppSpecUpdateStrategy 获取应用在某环境下实际生效的 updateStrategy section 配置。
//
//	@ID			GetEnvEffectiveAppSpecUpdateStrategy
//	@Summary	获取应用环境最终生效的 updateStrategy section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecUpdateStrategySectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/update-strategy/effective [get]
func (h *Handler) GetEnvEffectiveAppSpecUpdateStrategy(c *gin.Context) {
	getEnvEffectiveSection(h, c, appspec.UpdateStrategySection, "update strategy", updateStrategyOutput)
}

// SetEnvAppSpecUpdateStrategy 设置应用在某环境下的 updateStrategy section 配置。
//
//	@ID			SetEnvAppSpecUpdateStrategy
//	@Summary	设置应用环境 updateStrategy section 覆盖配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string										true	"应用 ID"
//	@Param		envName	path		string										true	"环境名称"
//	@Param		body	body		serializer.SetEnvAppSpecUpdateStrategyInput	true	"updateStrategy 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/update-strategy [put]
func (h *Handler) SetEnvAppSpecUpdateStrategy(c *gin.Context) {
	setEnvSection(h, c, appspec.UpdateStrategySection, "update strategy", envUpdateStrategyInput)
}

// DeleteEnvAppSpecUpdateStrategy 删除应用在某环境下的 updateStrategy section 配置。
//
//	@ID			DeleteEnvAppSpecUpdateStrategy
//	@Summary	删除应用环境 updateStrategy section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/update-strategy [delete]
func (h *Handler) DeleteEnvAppSpecUpdateStrategy(c *gin.Context) {
	deleteEnvSection(h, c, appspec.UpdateStrategySection, "update strategy")
}

func updateStrategyOutput(
	spec *appspec.UpdateStrategySpec,
	_ *bkmsapp.Application,
) *serializer.AppSpecUpdateStrategyOutput {
	return new(serializer.AppSpecUpdateStrategyOutput).FromModel(spec)
}

func defaultUpdateStrategyInput(
	input serializer.SetAppDefaultAppSpecUpdateStrategyInput,
	_ *bkmsapp.Application,
) *appspec.UpdateStrategySpec {
	return input.AppSpecUpdateStrategy.ToModel()
}

func envUpdateStrategyInput(
	input serializer.SetEnvAppSpecUpdateStrategyInput,
	_ *bkmsapp.Application,
) *appspec.UpdateStrategySpec {
	return input.AppSpecUpdateStrategy.ToModel()
}
