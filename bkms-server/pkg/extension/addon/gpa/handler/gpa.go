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

// Package handler contains Gin handlers for GPA config APIs.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler handles Gin GPA config API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// gpaService 构造 GPA 下发服务（仅持有 Store，Service 在 handler 内构造）。
func (h *Handler) gpaService() *gpa.GPAService {
	return gpa.NewGPAService(h.registry.AppModelStore)
}

// abortWithGPAApplyError 统一处理 gpaService().Apply 返回的错误并中止请求。
func abortWithGPAApplyError(c *gin.Context, clusterID string, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, gpa.ErrFederationNotSupported):
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "apply gpa CR"))
	case errors.Is(err, gpa.ErrComponentNotInstalled):
		bkerrs.AbortWithErr(c, bkerrs.WrapComponentNotInstalled(err, gpa.ClusterAddonName, clusterID))
	default:
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "apply gpa CR"))
	}
}

// GetAppEnvGPAConfig 查询应用在指定环境的 GPA 配置（含 K8s 运行状态）。
//
//	@ID			GetAppEnvGPAConfig
//	@Summary	查询应用在指定环境的 GPA 配置
//	@Tags		gpa
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.GetGPAConfigOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/autoscaler [get]
func (h *Handler) GetAppEnvGPAConfig(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	_, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	config, err := h.registry.GPAConfigStore.Get(ctx, uriInput.AppID, uriInput.EnvName)
	if err != nil {
		if errors.Is(err, gpa.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"gpa config not found for app(%s) env(%s)",
				uriInput.AppID, uriInput.EnvName,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get gpa config"))
		return
	}

	// 回查 K8s 运行状态，CR 不存在时 status 为 nil（仍正常返回 DB 配置）
	var status *gpa.GPAStatus
	if s, sErr := h.gpaService().Get(ctx, env, config.Name); sErr != nil {
		if !errors.Is(sErr, gpa.ErrCRNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(sErr, bkerrs.ErrCodeInternalServerError, "get gpa status"))
			return
		}
	} else {
		status = s
	}

	output := new(serializer.GPAConfigOutputObj).FromModel(config, status)
	ginutils.OK(c, serializer.GetGPAConfigOutput{Data: output})
}

// UpsertAppEnvGPAConfig 创建或更新应用在指定环境的 GPA 配置并下发 CRD。
//
//	@ID			UpsertAppEnvGPAConfig
//	@Summary	创建或更新应用在指定环境的 GPA 配置
//	@Tags		gpa
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string						true	"应用 ID"
//	@Param		envName	path		string						true	"环境名称"
//	@Param		body	body		serializer.UpsertGPAConfigInput	true	"请求体"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/autoscaler [put]
func (h *Handler) UpsertAppEnvGPAConfig(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var jsonInput serializer.UpsertGPAConfigInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	_, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if env.Cluster.IsFederation {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(
			gpa.ErrFederationNotSupported,
			bkerrs.ErrCodeInvalidRequest,
			"gpa is not supported in federation environment",
		))
		return
	}

	// 记录变更前配置用于审计（不存在视为新建，before 为 nil）
	var before *gpa.GPAConfig
	if existing, gErr := h.registry.GPAConfigStore.Get(ctx, uriInput.AppID, uriInput.EnvName); gErr == nil {
		before = existing
	} else if !errors.Is(gErr, gpa.ErrConfigNotFound) {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(gErr, bkerrs.ErrCodeInternalServerError, "get gpa config"))
		return
	}

	config := jsonInput.ToModel(uriInput.AppID, uriInput.EnvName)
	if before == nil {
		// 新建：生成新的配置名
		config.Name = config.GenerateName()
	} else {
		// 更新：复用原有配置名，保持 Name 稳定（CR 名、审计对象不漂移）
		config.Name = before.Name
	}

	// 先下发 CRD，成功后再落库，避免 DB 与集群不一致。
	// 关闭状态下（before.Enabled == false）跳过下发，仅更新 DB，避免被意外重新启用。
	shouldApply := before == nil || before.Enabled
	if shouldApply {
		if err = h.gpaService().Apply(ctx, env, config); err != nil {
			abortWithGPAApplyError(c, env.Cluster.ClusterID, err)
			return
		}
	}

	if before == nil {
		err = h.registry.GPAConfigStore.Create(ctx, config)
	} else {
		err = h.registry.GPAConfigStore.Update(ctx, uriInput.AppID, uriInput.EnvName, jsonInput.ToUpdateData())
	}
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "save gpa config"))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		config.Name,
		audit.WithAttribute(audit.AttributeGPA),
		audit.WithDataBefore(before),
		audit.WithDataAfter(config),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithAppID(uriInput.AppID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// DeleteAppEnvGPAConfig 删除应用在指定环境的 GPA 配置并清理 CRD。
//
//	@ID			DeleteAppEnvGPAConfig
//	@Summary	删除应用在指定环境的 GPA 配置
//	@Tags		gpa
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Param		envName	path		string	true	"环境名称"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/autoscaler [delete]
func (h *Handler) DeleteAppEnvGPAConfig(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	_, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	before, err := h.registry.GPAConfigStore.Get(ctx, uriInput.AppID, uriInput.EnvName)
	if err != nil {
		if errors.Is(err, gpa.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"gpa config not found for app(%s) env(%s)",
				uriInput.AppID, uriInput.EnvName,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get gpa config"))
		return
	}

	// 清理集群中的 CR（幂等，CR 不存在视为已清理）
	if dErr := h.gpaService().Delete(ctx, env, before.Name); dErr != nil && !errors.Is(dErr, gpa.ErrCRNotFound) {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(dErr, bkerrs.ErrCodeInternalServerError, "delete gpa CR"))
		return
	}

	if err = h.registry.GPAConfigStore.Delete(ctx, uriInput.AppID, uriInput.EnvName); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "delete gpa config"))
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeDelete,
		audit.ResourceTypeApp,
		before.Name,
		audit.WithAttribute(audit.AttributeGPA),
		audit.WithDataBefore(before),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithAppID(uriInput.AppID),
		audit.WithEnvName(env.Name),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}

// ToggleAppEnvGPAConfig 开关应用在指定环境的 GPA。
//
//	@ID			ToggleAppEnvGPAConfig
//	@Summary	开关应用在指定环境的 GPA
//	@Tags		gpa
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"环境名称"
//	@Param		body	body		serializer.ToggleGPAConfigInput	true	"请求体"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/autoscaler/toggle [patch]
func (h *Handler) ToggleAppEnvGPAConfig(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var jsonInput serializer.ToggleGPAConfigInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	_, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	before, err := h.registry.GPAConfigStore.Get(ctx, uriInput.AppID, uriInput.EnvName)
	if err != nil {
		if errors.Is(err, gpa.ErrConfigNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"gpa config not found for app(%s) env(%s)",
				uriInput.AppID, uriInput.EnvName,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get gpa config"))
		return
	}

	// 幂等：状态未变化时直接返回
	if before.Enabled == jsonInput.Enabled {
		ginutils.OK(c, serializer.EmptyOutput{})
		return
	}

	if jsonInput.Enabled {
		// 开启：按 DB 中最新配置重新下发 CR
		if err = h.gpaService().Apply(ctx, env, before); err != nil {
			abortWithGPAApplyError(c, env.Cluster.ClusterID, err)
			return
		}
	} else {
		// 关闭：删除集群中的 CR（不存在视为已清理）
		if dErr := h.gpaService().Delete(ctx, env, before.Name); dErr != nil && !errors.Is(dErr, gpa.ErrCRNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(dErr, bkerrs.ErrCodeInternalServerError, "delete gpa CR"))
			return
		}
	}

	enabled := jsonInput.Enabled
	if err = h.registry.GPAConfigStore.Update(ctx, uriInput.AppID, uriInput.EnvName, &gpa.ConfigUpdateData{
		Enabled: &enabled,
	}); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update gpa config"))
		return
	}

	after := *before
	after.Enabled = enabled

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		before.Name,
		audit.WithAttribute(audit.AttributeGPA),
		audit.WithDataBefore(before),
		audit.WithDataAfter(&after),
		audit.WithWorkspaceID(env.WorkspaceID),
		audit.WithAppID(uriInput.AppID),
		audit.WithEnvName(env.Name),
		audit.WithExtras(map[string]any{
			"enabled": enabled,
		}),
	)

	ginutils.OK(c, serializer.EmptyOutput{})
}
