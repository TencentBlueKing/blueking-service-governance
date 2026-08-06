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
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/cmdb"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// ListBKCCAuthorizedBusinesses 获取用户有权限的 bkcc 业务信息
//
//	@ID			ListBKCCAuthorizedBusinesses
//	@Summary	获取用户有权限的 BKCC 业务列表
//	@Tags		bkintegrations-bkcc
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.ListBKCCAuthorizedBusinessesOutput
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/bkcc/businesses/authorized [get]
func (h *Handler) ListBKCCAuthorizedBusinesses(c *gin.Context) {
	ctx := c.Request.Context()
	cmdbSvc, err := cmdb.NewService(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial cmdb service"))
		return
	}

	details, err := cmdbSvc.ListBusinessesWithLevel2Details(ctx)
	if err != nil {
		bkerrs.AbortWithErr(
			c,
			bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list user businesses with level2 details"),
		)
		return
	}

	ginutils.OK(
		c,
		&slz.ListBKCCAuthorizedBusinessesOutput{
			Data: lo.Map(details, func(item cmdb.BusinessDetail, _ int) *slz.BusinessInfoOutput {
				return new(slz.BusinessInfoOutput).FromModel(item)
			}),
		},
	)
}
