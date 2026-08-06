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

package platmgt

import (
	"slices"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	platmgtadmin "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/admin"
)

// RequirePlatformRole returns a middleware that ensures current user has one of the allowed platform roles.
func RequirePlatformRole(store platmgtadmin.Store, allowedRoles ...platmgtadmin.RoleCode) gin.HandlerFunc {
	platAdminService := platmgtadmin.NewService(store)

	return func(c *gin.Context) {
		username := auth.MustGetUser(c.Request.Context()).ID
		roleCode, ok, err := platAdminService.GetRole(c.Request.Context(), username)
		if err != nil {
			bkerrs.AbortWithErr(
				c,
				bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "check platform administrator permission"),
			)
			return
		}
		if !ok || !slices.Contains(allowedRoles, roleCode) {
			bkerrs.AbortWithErr(
				c,
				bkerrs.New(bkerrs.ErrCodeNoPermission, "platform administrator permission required"),
			)
			return
		}
		c.Next()
	}
}
