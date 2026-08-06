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

// Package handler contains Gin handlers for AppSpec APIs.
package handler

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

// Handler handles Gin AppSpec API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

type sectionOutput[T any] struct {
	Data T `json:"data"`
}

// The generic section helpers keep the standard AppSpec API flow in one place:
// bind URI/body, validate permission, read or replace a section, map validation
// errors, write audit records, and return the Gin response. Section handlers only
// pass the section handle plus input/output conversion functions, which avoids
// copying that flow for every AppSpec section.

// getDefaultSection handles the shared flow for querying an app-level default section.
func getDefaultSection[T, O any](
	h *Handler,
	c *gin.Context,
	section appspec.SectionHandle[T],
	sectionName string,
	toOutput func(*T, *bkmsapp.Application) O,
) {
	var uriInput serializer.AppURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	sectionValue, err := appspec.GetDefaultSection(
		ctx,
		h.registry.AppSpecStore,
		h.registry.AppModelStore,
		app.ID,
		section,
	)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "getting default %s", sectionName),
		)
		return
	}
	ginutils.OK(c, sectionOutput[O]{Data: toOutput(sectionValue, app)})
}

// setDefaultSection handles the shared flow for replacing an app-level default section.
func setDefaultSection[T, I any](
	h *Handler,
	c *gin.Context,
	section appspec.SectionHandle[T],
	sectionName string,
	toModel func(I, *bkmsapp.Application) *T,
) {
	var uriInput serializer.AppURIInput
	var input I
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	oldSection, err := appspec.GetDefaultSection(
		ctx,
		h.registry.AppSpecStore,
		h.registry.AppModelStore,
		app.ID,
		section,
	)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "getting default %s before update", sectionName),
		)
		return
	}
	newSection := toModel(input, app)

	if err = appspec.SetDefaultSection(
		ctx,
		h.registry.AppSpecStore,
		h.registry.AppModelStore,
		app.ID,
		section,
		newSection,
		appspec.SectionWriteModeReplace,
	); err != nil {
		if errors.Is(err, appspec.ErrAppSpecValidation) {
			bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidRequest, "validate %s", sectionName))
			return
		}
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "setting default %s", sectionName),
		)
		return
	}
	addAppSpecAudit(ctx, app, "", audit.OperationTypeUpdate, oldSection, newSection)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// getEnvSection handles the shared flow for querying a raw env-level section override.
func getEnvSection[T, O any](
	h *Handler,
	c *gin.Context,
	section appspec.SectionHandle[T],
	sectionName string,
	toOutput func(*T, *bkmsapp.Application) O,
) {
	var uriInput serializer.AppEnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	sectionValue, err := appspec.GetEnvSection(ctx, h.registry.AppSpecStore, app.ID, env.Name, section)
	if err != nil {
		if errors.Is(err, appspec.ErrAppSpecNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeNotFound, "env %s not found", sectionName))
			return
		}
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "getting env %s", sectionName),
		)
		return
	}
	if sectionValue == nil {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "env %s not found", sectionName))
		return
	}
	ginutils.OK(c, sectionOutput[O]{Data: toOutput(sectionValue, app)})
}

// getEnvEffectiveSection handles the shared flow for querying the merged env-effective section.
func getEnvEffectiveSection[T, O any](
	h *Handler,
	c *gin.Context,
	section appspec.SectionHandle[T],
	sectionName string,
	toOutput func(*T, *bkmsapp.Application) O,
) {
	var uriInput serializer.AppEnvURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	sectionValue, err := appspec.GetEnvEffectiveSection(
		ctx,
		h.registry.AppSpecStore,
		h.registry.AppModelStore,
		app.ID,
		env.Name,
		section,
	)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "getting env effective %s", sectionName),
		)
		return
	}
	ginutils.OK(c, sectionOutput[O]{Data: toOutput(sectionValue, app)})
}

// setEnvSection handles the shared flow for replacing a raw env-level section override.
func setEnvSection[T, I any](
	h *Handler,
	c *gin.Context,
	section appspec.SectionHandle[T],
	sectionName string,
	toModel func(I, *bkmsapp.Application) *T,
) {
	var uriInput serializer.AppEnvURIInput
	var input I
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	oldSection, err := appspec.GetEnvSection(ctx, h.registry.AppSpecStore, app.ID, env.Name, section)
	if err != nil && !errors.Is(err, appspec.ErrAppSpecNotFound) {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "getting env %s before update", sectionName),
		)
		return
	}
	newSection := toModel(input, app)

	if err = appspec.SetEnvSection(
		ctx,
		h.registry.AppSpecStore,
		app.ID,
		env.Name,
		section,
		newSection,
		appspec.SectionWriteModeReplace,
	); err != nil {
		if errors.Is(err, appspec.ErrAppSpecValidation) {
			bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidRequest, "validate env %s", sectionName))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "setting env %s", sectionName))
		return
	}
	addAppSpecAudit(ctx, app, env.Name, audit.OperationTypeUpdate, oldSection, newSection)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// deleteEnvSection handles the shared flow for deleting a raw env-level section override.
func deleteEnvSection[T any](
	h *Handler,
	c *gin.Context,
	section appspec.SectionHandle[T],
	sectionName string,
) {
	var uriInput serializer.AppEnvURIInput
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

	oldSection, err := appspec.GetEnvSection(ctx, h.registry.AppSpecStore, app.ID, env.Name, section)
	if err != nil && !errors.Is(err, appspec.ErrAppSpecNotFound) {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "getting env %s before deletion", sectionName),
		)
		return
	}

	if err = appspec.SetEnvSection(
		ctx,
		h.registry.AppSpecStore,
		app.ID,
		env.Name,
		section,
		nil,
		appspec.SectionWriteModeReplace,
	); err != nil {
		if errors.Is(err, appspec.ErrAppSpecValidation) {
			bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInvalidRequest, "validate env %s", sectionName))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "deleting env %s", sectionName))
		return
	}
	if oldSection != nil {
		addAppSpecAudit(ctx, app, env.Name, audit.OperationTypeDelete, oldSection, nil)
	}
	ginutils.OK(c, serializer.EmptyOutput{})
}

func addAppSpecAudit(
	ctx context.Context,
	app *bkmsapp.Application,
	envName string,
	opType audit.OperationType,
	before any,
	after any,
) {
	opts := []audit.Option{
		audit.WithAttribute(audit.AttributeAppModel),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	}
	if envName != "" {
		opts = append(opts, audit.WithEnvName(envName))
	}
	if before != nil {
		opts = append(opts, audit.WithDataBefore(before))
	}
	if after != nil {
		opts = append(opts, audit.WithDataAfter(after))
	}

	go audit.AddOperationRecordAsync(context.WithoutCancel(ctx), opType, audit.ResourceTypeApp, app.ID, opts...)
}
