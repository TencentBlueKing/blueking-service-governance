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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger"
	triggerserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// HandleBuildTriggerCallback 接收蓝盾触发专用流水线的回调并发起镜像构建。
//
// 该路由不走用户票据鉴权，而是校验应用独享的回调凭证。三态处理结果均以 HTTP 200 返回，
// 由响应体的 result 字段区分，便于流水线侧脚本留痕。
//
// 空实现只校验凭证请求头是否存在，凭证内容比对、限流、去重与发起构建由
// 「构建回调凭证鉴权与限流」「构建回调处理与版本号生成」两个子需求落地
//
//	@ID				HandleBuildTriggerCallback
//	@Summary		接收构建触发回调
//	@Tags			build-triggers
//	@Accept			json
//	@Produce		json
//	@Param			appID							path		string									true	"应用 ID"
//	@Param			X-Bkms-Build-Trigger-Token		header		string									true	"应用独享的回调凭证"
//	@Param			body							body		triggerserializer.CallbackEventInput	true	"回调事件"
//	@Success		200								{object}	triggerserializer.CallbackOutput
//	@Failure		400								{object}	bkerrs.GinErrorOutput
//	@Failure		401								{object}	bkerrs.GinErrorOutput
//	@Failure		404								{object}	bkerrs.GinErrorOutput
//	@Failure		429								{object}	bkerrs.GinErrorOutput
//	@Router			/apps/{appID}/build-triggers/callback [post]
func (h *Handler) HandleBuildTriggerCallback(c *gin.Context) {
	if c.GetHeader(triggerserializer.CallbackCredentialHeader) == "" {
		bkerrs.AbortWithErr(c, bkerrs.New(
			bkerrs.ErrCodeUnauthenticated, "build trigger callback credential is required",
		))
		return
	}

	var uriInput triggerserializer.AppURIInput
	var input triggerserializer.CallbackEventInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ginutils.OK(c, triggerserializer.CallbackOutput{
		Data: &triggerserializer.CallbackResultOutputObj{Result: string(trigger.ResultBuilt)},
	})
}
