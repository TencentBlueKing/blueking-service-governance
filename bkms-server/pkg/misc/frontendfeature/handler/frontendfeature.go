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

// Package handler 提供前端功能开关的 Gin 视图逻辑。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/frontendfeature/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// Handler 是前端功能开关的 Gin handler。
type Handler struct{}

// New 创建前端功能开关 Handler。
func New() *Handler {
	return &Handler{}
}

// GetFrontendFeatures 获取前端功能开关。
//
//	@ID				GetFrontendFeatures
//	@Summary		获取前端功能开关
//	@Description	返回由部署配置决定的功能开关，前端据此决定页面入口是否展示
//	@Tags			frontend-features
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Success		200	{object}	slz.FrontendFeaturesResponse
//	@Failure		400	{object}	bkerrs.GinErrorOutput
//	@Router			/frontend-features [get]
func (h *Handler) GetFrontendFeatures(c *gin.Context) {
	ginutils.OK(c, slz.FrontendFeaturesResponse{Data: slz.NewFrontendFeaturesOutput(config.G)})
}
