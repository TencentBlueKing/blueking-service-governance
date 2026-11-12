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

// Package handler contains Gin handlers for platform config APIs.
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/platconfig"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/platconfig/serializer"
)

var _ platconfig.Handler = (*GinHandler)(nil)

// GinHandler handles Gin platform config API requests.
type GinHandler struct {
	cfg *config.Config
}

// New creates a Handler.
func New(cfg *config.Config) *GinHandler {
	return &GinHandler{cfg: cfg}
}

// GetFrontendFeatures 查询前端特性开关
//
//	@ID			GetFrontendFeatures
//	@Summary	查询前端特性开关
//	@Tags		platconfig
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.FrontendFeaturesResponse
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/platform/frontend-features [get]
func (h *GinHandler) GetFrontendFeatures(c *gin.Context) {
	ginutils.OK(c, serializer.FrontendFeaturesResponse{Data: platconfig.GetFrontendFeatures(h.cfg)})
}
