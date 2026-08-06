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

// GetAppDefaultAppSpecLifecycle 获取应用默认 lifecycle section 配置。
//
//	@ID			GetAppDefaultAppSpecLifecycle
//	@Summary	获取应用默认 lifecycle section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.AppSpecLifecycleSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-lifecycle [get]
func (h *Handler) GetAppDefaultAppSpecLifecycle(c *gin.Context) {
	getDefaultSection(h, c, appspec.LifecycleSection, "lifecycle", lifecycleOutput)
}

// SetAppDefaultAppSpecLifecycle 设置应用默认 lifecycle section 配置。
//
//	@ID			SetAppDefaultAppSpecLifecycle
//	@Summary	设置应用默认 lifecycle section 配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string										true	"应用 ID"
//	@Param		body	body		serializer.SetAppDefaultAppSpecLifecycleInput	true	"lifecycle 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-lifecycle [put]
func (h *Handler) SetAppDefaultAppSpecLifecycle(c *gin.Context) {
	setDefaultSection(h, c, appspec.LifecycleSection, "lifecycle", defaultLifecycleInput)
}

// GetEnvAppSpecLifecycle 获取应用在某环境下的 lifecycle section 配置。
//
//	@ID			GetEnvAppSpecLifecycle
//	@Summary	获取应用环境 lifecycle section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecLifecycleSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/lifecycle [get]
func (h *Handler) GetEnvAppSpecLifecycle(c *gin.Context) {
	getEnvSection(h, c, appspec.LifecycleSection, "lifecycle", lifecycleOutput)
}

// GetEnvEffectiveAppSpecLifecycle 获取应用在某环境下实际生效的 lifecycle section 配置。
//
//	@ID			GetEnvEffectiveAppSpecLifecycle
//	@Summary	获取应用环境最终生效的 lifecycle section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecLifecycleSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/lifecycle/effective [get]
func (h *Handler) GetEnvEffectiveAppSpecLifecycle(c *gin.Context) {
	getEnvEffectiveSection(h, c, appspec.LifecycleSection, "lifecycle", lifecycleOutput)
}

// SetEnvAppSpecLifecycle 设置应用在某环境下的 lifecycle section 配置。
//
//	@ID			SetEnvAppSpecLifecycle
//	@Summary	设置应用环境 lifecycle section 覆盖配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"环境名称"
//	@Param		body	body		serializer.SetEnvAppSpecLifecycleInput	true	"lifecycle 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/lifecycle [put]
func (h *Handler) SetEnvAppSpecLifecycle(c *gin.Context) {
	setEnvSection(h, c, appspec.LifecycleSection, "lifecycle", envLifecycleInput)
}

// DeleteEnvAppSpecLifecycle 删除应用在某环境下的 lifecycle section 配置。
//
//	@ID			DeleteEnvAppSpecLifecycle
//	@Summary	删除应用环境 lifecycle section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/lifecycle [delete]
func (h *Handler) DeleteEnvAppSpecLifecycle(c *gin.Context) {
	deleteEnvSection(h, c, appspec.LifecycleSection, "lifecycle")
}

func lifecycleOutput(
	spec *appspec.LifecycleSpec,
	_ *bkmsapp.Application,
) *serializer.AppSpecLifecycleOutput {
	return new(serializer.AppSpecLifecycleOutput).FromModel(spec)
}

func defaultLifecycleInput(
	input serializer.SetAppDefaultAppSpecLifecycleInput,
	_ *bkmsapp.Application,
) *appspec.LifecycleSpec {
	return input.AppSpecLifecycle.ToModel()
}

func envLifecycleInput(
	input serializer.SetEnvAppSpecLifecycleInput,
	_ *bkmsapp.Application,
) *appspec.LifecycleSpec {
	return input.AppSpecLifecycle.ToModel()
}
