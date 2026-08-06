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

// GetAppDefaultAppSpecLabels 获取应用默认 labels section 配置。
//
//	@ID			GetAppDefaultAppSpecLabels
//	@Summary	获取应用默认 labels section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.AppSpecLabelsSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-labels [get]
func (h *Handler) GetAppDefaultAppSpecLabels(c *gin.Context) {
	getDefaultSection(h, c, appspec.LabelsSection, "labels", labelsOutput)
}

// SetAppDefaultAppSpecLabels 设置应用默认 labels section 配置。
//
//	@ID			SetAppDefaultAppSpecLabels
//	@Summary	设置应用默认 labels section 配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string									true	"应用 ID"
//	@Param		body	body		serializer.AppSpecLabelsInput	true	"labels 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-labels [put]
func (h *Handler) SetAppDefaultAppSpecLabels(c *gin.Context) {
	setDefaultSection(h, c, appspec.LabelsSection, "labels", defaultLabelsInput)
}

// GetEnvAppSpecLabels 获取应用在某环境下的 labels section 覆盖配置。
//
//	@ID			GetEnvAppSpecLabels
//	@Summary	获取应用环境 labels section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecLabelsSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/labels [get]
func (h *Handler) GetEnvAppSpecLabels(c *gin.Context) {
	getEnvSection(h, c, appspec.LabelsSection, "labels", labelsOutput)
}

// GetEnvEffectiveAppSpecLabels 获取应用在某环境下实际生效的 labels section 配置。
//
//	@ID			GetEnvEffectiveAppSpecLabels
//	@Summary	获取应用环境最终生效的 labels section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecLabelsSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/labels/effective [get]
func (h *Handler) GetEnvEffectiveAppSpecLabels(c *gin.Context) {
	getEnvEffectiveSection(h, c, appspec.LabelsSection, "labels", labelsOutput)
}

// SetEnvAppSpecLabels 设置应用在某环境下的 labels section 覆盖配置。
//
//	@ID			SetEnvAppSpecLabels
//	@Summary	设置应用环境 labels section 覆盖配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"环境名称"
//	@Param		body	body		serializer.AppSpecLabelsInput	true	"labels 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/labels [put]
func (h *Handler) SetEnvAppSpecLabels(c *gin.Context) {
	setEnvSection(h, c, appspec.LabelsSection, "labels", envLabelsInput)
}

// DeleteEnvAppSpecLabels 删除应用在某环境下的 labels section 覆盖配置。
//
//	@ID			DeleteEnvAppSpecLabels
//	@Summary	删除应用环境 labels section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/labels [delete]
func (h *Handler) DeleteEnvAppSpecLabels(c *gin.Context) {
	deleteEnvSection(h, c, appspec.LabelsSection, "labels")
}

func labelsOutput(
	spec *appspec.LabelsSpec,
	_ *bkmsapp.Application,
) *serializer.AppSpecLabelsOutput {
	return new(serializer.AppSpecLabelsOutput).FromModel(spec)
}

func defaultLabelsInput(
	input serializer.AppSpecLabelsInput,
	_ *bkmsapp.Application,
) *appspec.LabelsSpec {
	return input.ToModel()
}

func envLabelsInput(
	input serializer.AppSpecLabelsInput,
	_ *bkmsapp.Application,
) *appspec.LabelsSpec {
	return input.ToModel()
}
