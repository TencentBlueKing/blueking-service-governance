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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// PreviewComponentDef 试运行预览组件定义：
// 用内置变量、属性默认值、请求提供的输出模板渲染附加资源和 patch 预览，并返回。
//
//	@ID				PreviewComponentDef
//	@Summary		预览组件定义（试运行）
//	@Tags			component-defs
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			body	body		serializer.PreviewComponentDefInput	true	"预览组件定义请求"
//	@Success		200		{object}	serializer.PreviewOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/component-defs/preview [post]
func (h *Handler) PreviewComponentDef(c *gin.Context) {
	var input serializer.PreviewComponentDefInput
	if err := ginutils.BindJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	builder := component.NewPreviewBuilder(
		input.CompDefName,
		input.PropertiesToModel(),
		input.Patchers,
		input.Specs,
	)
	result, err := builder.Build()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "preview component def"))
		return
	}

	ginutils.OK(c, new(serializer.PreviewOutput).FromModel(result))
}

// PreviewComponentInst 使用指定的组件定义试运行预览将要创建的组件实例：
// 用内置变量、请求中提供的属性值、指定的组件定义的输出模板渲染附加资源和 patch 预览，并返回。
//
//	@ID				PreviewComponentInst
//	@Summary		预览组件实例（试运行）
//	@Tags			component-insts
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			body	body		serializer.PreviewComponentInstInput	true	"预览组件实例请求"
//	@Success		200		{object}	serializer.PreviewOutput
//	@Failure		400		{object}	bkerrs.GinErrorOutput
//	@Router			/component-insts/preview [post]
func (h *Handler) PreviewComponentInst(c *gin.Context) {
	var input serializer.PreviewComponentInstInput
	if err := ginutils.BindJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	compDef, err := h.registry.ComponentDefStore.Get(ctx, input.Type, component.DefaultComponentDefVersion)
	if err != nil {
		if errors.Is(err, component.ErrComponentDefNotFound) {
			bkerrs.AbortWithErr(c, bkerrs.Errorf(
				bkerrs.ErrCodeNotFound,
				"component def(%s:%s) not found",
				input.Type,
				component.DefaultComponentDefVersion,
			))
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get component def"))
		return
	}

	builder := component.
		NewPreviewBuilder(input.Type, compDef.Properties, compDef.Patchers, compDef.Specs).
		WithPropertyValues(input.Properties)
	result, err := builder.Build()
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "preview component inst"))
		return
	}

	ginutils.OK(c, new(serializer.PreviewOutput).FromModel(result))
}
