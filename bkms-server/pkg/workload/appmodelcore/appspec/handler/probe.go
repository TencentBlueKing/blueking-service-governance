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
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	probesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/probe"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// GetAppDefaultAppSpecProbe 获取应用默认 probe section 配置。
//
//	@ID			GetAppDefaultAppSpecProbe
//	@Summary	获取应用默认 probe section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	serializer.AppSpecProbeSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-probe [get]
func (h *Handler) GetAppDefaultAppSpecProbe(c *gin.Context) {
	getDefaultSection(h, c, appspec.ProbeSection, "probe", probeOutput)
}

// SetAppDefaultAppSpecProbe 设置应用默认 probe section 配置。
//
//	@ID			SetAppDefaultAppSpecProbe
//	@Summary	设置应用默认 probe section 配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string									true	"应用 ID"
//	@Param		body	body		serializer.SetAppDefaultAppSpecProbeInput	true	"probe 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/app-spec/default-probe [put]
func (h *Handler) SetAppDefaultAppSpecProbe(c *gin.Context) {
	setDefaultSection(h, c, appspec.ProbeSection, "probe", defaultProbeInput)
}

// GetEnvAppSpecProbe 获取应用在某环境下的 probe section 配置。
//
//	@ID			GetEnvAppSpecProbe
//	@Summary	获取应用环境 probe section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecProbeSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/probe [get]
func (h *Handler) GetEnvAppSpecProbe(c *gin.Context) {
	getEnvSection(h, c, appspec.ProbeSection, "probe", probeOutput)
}

// GetEnvEffectiveAppSpecProbe 获取应用在某环境下实际生效的 probe section 配置。
//
//	@ID			GetEnvEffectiveAppSpecProbe
//	@Summary	获取应用环境最终生效的 probe section 配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.AppSpecProbeSectionOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/probe/effective [get]
func (h *Handler) GetEnvEffectiveAppSpecProbe(c *gin.Context) {
	getEnvEffectiveSection(h, c, appspec.ProbeSection, "probe", probeOutput)
}

// SetEnvAppSpecProbe 设置应用在某环境下的 probe section 配置。
//
//	@ID			SetEnvAppSpecProbe
//	@Summary	设置应用环境 probe section 覆盖配置
//	@Tags		app-spec
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		envName	path		string							true	"环境名称"
//	@Param		body	body		serializer.SetEnvAppSpecProbeInput	true	"probe 配置"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/probe [put]
func (h *Handler) SetEnvAppSpecProbe(c *gin.Context) {
	setEnvSection(h, c, appspec.ProbeSection, "probe", envProbeInput)
}

// DeleteEnvAppSpecProbe 删除应用在某环境下的 probe section 配置。
//
//	@ID			DeleteEnvAppSpecProbe
//	@Summary	删除应用环境 probe section 覆盖配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/probe [delete]
func (h *Handler) DeleteEnvAppSpecProbe(c *gin.Context) {
	deleteEnvSection(h, c, appspec.ProbeSection, "probe")
}

// DeleteEnvAppSpecProbeByType 删除应用在某环境下指定类型的探针配置。
//
//	@ID			DeleteEnvAppSpecProbeByType
//	@Summary	删除应用环境下指定类型的探针配置
//	@Tags		app-spec
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		envName		path		string	true	"环境名称"
//	@Param		probeType	path		string	true	"探针类型，可选 liveness、readiness、startup"
//	@Success	200			{object}	serializer.EmptyOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/app-spec/probe/{probeType} [delete]
func (h *Handler) DeleteEnvAppSpecProbeByType(c *gin.Context) {
	var uriInput serializer.AppEnvProbeTypeURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	probes, err := appspec.GetEnvSection(ctx, h.registry.AppSpecStore, app.ID, env.Name, appspec.ProbeSection)
	if err != nil {
		if errors.Is(err, appspec.ErrAppSpecNotFound) {
			ginutils.OK(c, serializer.EmptyOutput{})
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "getting env probe for deletion"))
		return
	}
	if probes == nil {
		ginutils.OK(c, serializer.EmptyOutput{})
		return
	}

	oldProbes := probesection.Clone(probes)
	switch uriInput.ProbeType {
	case "liveness":
		if probes.Liveness == nil {
			ginutils.OK(c, serializer.EmptyOutput{})
			return
		}
		probes.Liveness = nil
	case "readiness":
		if probes.Readiness == nil {
			ginutils.OK(c, serializer.EmptyOutput{})
			return
		}
		probes.Readiness = nil
	case "startup":
		if probes.Startup == nil {
			ginutils.OK(c, serializer.EmptyOutput{})
			return
		}
		probes.Startup = nil
	}

	var updated *appspec.ProbeSpec
	if probes.Liveness != nil || probes.Readiness != nil || probes.Startup != nil {
		updated = probes
	}

	if err = appspec.SetEnvSection(
		ctx,
		h.registry.AppSpecStore,
		app.ID,
		env.Name,
		appspec.ProbeSection,
		updated,
		appspec.SectionWriteModeReplace,
	); err != nil {
		if errors.Is(err, appspec.ErrAppSpecValidation) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "validate env probe"))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "deleting env probe by type"))
		return
	}
	addAppSpecAudit(ctx, app, env.Name, audit.OperationTypeUpdate, oldProbes, updated)
	ginutils.OK(c, serializer.EmptyOutput{})
}

func probeOutput(
	spec *appspec.ProbeSpec,
	_ *bkmsapp.Application,
) *serializer.AppSpecProbeOutput {
	return new(serializer.AppSpecProbeOutput).FromModel(spec)
}

func defaultProbeInput(
	input serializer.SetAppDefaultAppSpecProbeInput,
	_ *bkmsapp.Application,
) *appspec.ProbeSpec {
	return input.AppSpecProbe.ToModel()
}

func envProbeInput(
	input serializer.SetEnvAppSpecProbeInput,
	_ *bkmsapp.Application,
) *appspec.ProbeSpec {
	return input.AppSpecProbe.ToModel()
}
