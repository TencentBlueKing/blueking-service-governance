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
	"context"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	tafapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/taf"
)

// UpdateAppTafSpec 更新应用 Taf 配置。
//
//	@ID			UpdateAppTafSpec
//	@Summary	更新应用 Taf 配置
//	@Tags		app
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		body	body		serializer.UpdateAppModelSpecInput	true	"更新 Taf 配置请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/taf-spec [put]
func (h *Handler) UpdateAppTafSpec(c *gin.Context) {
	var uriInput serializer.AppURIInput
	var input serializer.UpdateAppModelSpecInput
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

	// 检查应用类型
	if app.Type != bkmsapp.AppTypeTAF {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "only TAF app supports taf spec"))
		return
	}

	if input.AppModelSpec == nil || input.AppModelSpec.TafSpec == nil {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "appModelSpec.tafSpec is required"))
		return
	}

	// 更新 AppModel 应用配置（复用现有逻辑）
	if err = h.updateTafApp(ctx, app, input.AppModelSpec); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "update taf app"))
		return
	}

	ginutils.OK(c, serializer.EmptyOutput{})
}

// createTafApp 创建 TAF 应用（handler 层，负责参数转换）
func (h *Handler) createTafApp(
	ctx context.Context,
	app *bkmsapp.Application,
	input *serializer.AppModelSpecInput,
) error {
	// 参数转换：serializer 输入 -> 内部类型
	params := input.ToTafCreateParams()

	// 调用 taf 服务
	svc := tafapp.NewService(
		h.registry.AppModelStore,
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
		h.registry.AppStore,
	)
	return svc.Create(ctx, app, params)
}

// updateTafApp 更新 TAF 应用（handler 层，负责参数转换和审计）
func (h *Handler) updateTafApp(
	ctx context.Context,
	app *bkmsapp.Application,
	input *serializer.AppModelSpecInput,
) error {
	// 参数转换
	params := input.ToTafUpdateParams()

	// 调用 taf 服务
	svc := tafapp.NewService(
		h.registry.AppModelStore,
		h.registry.AppConfigFileStore,
		h.registry.AppConfigFileVersionStore,
		h.registry.AppStore,
	)
	oldModel, newModel, err := svc.Update(ctx, app, params)
	if err != nil {
		return err
	}

	// 审计（在 handler 层处理）
	go audit.AddOperationRecordAsync(
		ctx,
		audit.OperationTypeUpdate,
		audit.ResourceTypeApp,
		app.ID,
		audit.WithAttribute(audit.AttributeAppModel),
		audit.WithDataBefore(oldModel),
		audit.WithDataAfter(newModel),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
	)
	return nil
}
