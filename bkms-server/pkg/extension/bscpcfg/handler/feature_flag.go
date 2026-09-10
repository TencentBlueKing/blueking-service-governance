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
	cfgmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// GetFeatureFlag 查询指定应用的 bscpcfg 能力开关状态
//
//	@ID			GetBscpCfgFeatureFlag
//	@Summary	获取应用的 bscpcfg FeatureFlag
//	@Tags		bscpcfg
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string	true	"应用 ID"
//	@Success	200		{object}	featureFlagOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/bscpcfg/feature-flag [get]
func (h *Handler) GetFeatureFlag(c *gin.Context) {
	var uri slz.AppIDURI
	if err := ginutils.BindURI(c, &uri); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateAppByID(ctx, h.registry, uri.AppID, perm.TypeEdit); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	flag, err := h.registry.BscpCfgStore.GetFeatureFlag(ctx, uri.AppID)
	if err != nil {
		if errors.Is(err, cfgmodel.ErrFeatureFlagNotFound) {
			ginutils.OK(c, &slz.FeatureFlagResponse{Data: &slz.FeatureFlagOutput{Enabled: false}})
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get feature flag"))
		return
	}

	ginutils.OK(c, &slz.FeatureFlagResponse{Data: &slz.FeatureFlagOutput{Enabled: flag.Enabled}})
}
