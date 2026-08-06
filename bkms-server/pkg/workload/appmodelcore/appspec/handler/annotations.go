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

// GetAppDefaultAppSpecAnnotations 获取应用默认 annotations section 配置。
//
//	@ID			GetAppDefaultAppSpecAnnotations
//	@Summary	获取应用默认 annotations section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.AppSpecAnnotationsSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-annotations [get]
func (h *Handler) GetAppDefaultAppSpecAnnotations(c *gin.Context) {
	getDefaultSection(h, c, appspec.AnnotationsSection, "annotations", annotationsOutput)
}

// SetAppDefaultAppSpecAnnotations 设置应用默认 annotations section 配置。
//
//	@ID			SetAppDefaultAppSpecAnnotations
//	@Summary	设置应用默认 annotations section 配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string										true	"应用 ID"
//	@Param		body	body		serializer.AppSpecAnnotationsInput	true	"annotations 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-annotations [put]
func (h *Handler) SetAppDefaultAppSpecAnnotations(c *gin.Context) {
	setDefaultSection(h, c, appspec.AnnotationsSection, "annotations", defaultAnnotationsInput)
}

// GetEnvAppSpecAnnotations 获取应用在某环境下的 annotations section 覆盖配置。
//
//	@ID			GetEnvAppSpecAnnotations
//	@Summary	获取应用环境 annotations section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecAnnotationsSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/annotations [get]
func (h *Handler) GetEnvAppSpecAnnotations(c *gin.Context) {
	getEnvSection(h, c, appspec.AnnotationsSection, "annotations", annotationsOutput)
}

// GetEnvEffectiveAppSpecAnnotations 获取应用在某环境下实际生效的 annotations section 配置。
//
//	@ID			GetEnvEffectiveAppSpecAnnotations
//	@Summary	获取应用环境最终生效的 annotations section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecAnnotationsSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/annotations/effective [get]
func (h *Handler) GetEnvEffectiveAppSpecAnnotations(c *gin.Context) {
	getEnvEffectiveSection(h, c, appspec.AnnotationsSection, "annotations", annotationsOutput)
}

// SetEnvAppSpecAnnotations 设置应用在某环境下的 annotations section 覆盖配置。
//
//	@ID			SetEnvAppSpecAnnotations
//	@Summary	设置应用环境 annotations section 覆盖配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string									true	"应用 ID"
//	@Param		envName	path		string									true	"环境名称"
//	@Param		body	body		serializer.AppSpecAnnotationsInput	true	"annotations 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/annotations [put]
func (h *Handler) SetEnvAppSpecAnnotations(c *gin.Context) {
	setEnvSection(h, c, appspec.AnnotationsSection, "annotations", envAnnotationsInput)
}

// DeleteEnvAppSpecAnnotations 删除应用在某环境下的 annotations section 覆盖配置。
//
//	@ID			DeleteEnvAppSpecAnnotations
//	@Summary	删除应用环境 annotations section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/annotations [delete]
func (h *Handler) DeleteEnvAppSpecAnnotations(c *gin.Context) {
	deleteEnvSection(h, c, appspec.AnnotationsSection, "annotations")
}

func annotationsOutput(
	spec *appspec.AnnotationsSpec,
	_ *bkmsapp.Application,
) *serializer.AppSpecAnnotationsOutput {
	return new(serializer.AppSpecAnnotationsOutput).FromModel(spec)
}

func defaultAnnotationsInput(
	input serializer.AppSpecAnnotationsInput,
	_ *bkmsapp.Application,
) *appspec.AnnotationsSpec {
	return input.ToModel()
}

func envAnnotationsInput(
	input serializer.AppSpecAnnotationsInput,
	_ *bkmsapp.Application,
) *appspec.AnnotationsSpec {
	return input.ToModel()
}
