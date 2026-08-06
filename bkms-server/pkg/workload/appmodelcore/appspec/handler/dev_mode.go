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

// GetEnvAppSpecDevMode 获取应用在某环境下的 devMode section 配置。
//
//	@ID			GetEnvAppSpecDevMode
//	@Summary	获取应用环境 devMode section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecDevModeSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/dev-mode [get]
func (h *Handler) GetEnvAppSpecDevMode(c *gin.Context) {
	getEnvSection(h, c, appspec.DevModeSection, "dev mode", devModeOutput)
}

// GetEnvEffectiveAppSpecDevMode 获取应用在某环境下实际生效的 devMode section 配置。
//
//	@ID			GetEnvEffectiveAppSpecDevMode
//	@Summary	获取应用环境最终生效的 devMode section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecDevModeSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/dev-mode/effective [get]
func (h *Handler) GetEnvEffectiveAppSpecDevMode(c *gin.Context) {
	getEnvEffectiveSection(h, c, appspec.DevModeSection, "dev mode", devModeOutput)
}

// SetEnvAppSpecDevMode 设置应用在某环境下的 devMode section 配置。
//
//	@ID			SetEnvAppSpecDevMode
//	@Summary	设置应用环境 devMode section 覆盖配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"环境名称"
//	@Param		body	body		serializer.SetEnvAppSpecDevModeInput	true	"devMode 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/dev-mode [put]
func (h *Handler) SetEnvAppSpecDevMode(c *gin.Context) {
	setEnvSection(h, c, appspec.DevModeSection, "dev mode", envDevModeInput)
}

// DeleteEnvAppSpecDevMode 删除应用在某环境下的 devMode section 配置。
//
//	@ID			DeleteEnvAppSpecDevMode
//	@Summary	删除应用环境 devMode section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/dev-mode [delete]
func (h *Handler) DeleteEnvAppSpecDevMode(c *gin.Context) {
	deleteEnvSection(h, c, appspec.DevModeSection, "dev mode")
}

func devModeOutput(
	spec *appspec.DevModeSpec,
	app *bkmsapp.Application,
) *serializer.AppSpecDevModeOutput {
	return new(serializer.AppSpecDevModeOutput).FromModel(spec, app.Type)
}

func envDevModeInput(
	input serializer.SetEnvAppSpecDevModeInput,
	app *bkmsapp.Application,
) *appspec.DevModeSpec {
	return input.AppSpecDevMode.ToModel(app.Type)
}
