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

package tenant

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// Required resolves the tenant header for every authenticated business request.
//
// 单租模式下仅接受空 tenant 或 default，并统一写回 default 到上下文；
// 多租模式下要求显式传入 tenant header，并在写入上下文前校验用户是否属于该租户。
func Required(enableMultiTenantMode bool, verifier Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先按模式规范化 header：单租只接受空/default，多租必须显式传入。
		tenantID, err := ValidateTenantMode(c.Request.Header.Get(HeaderTenantID), enableMultiTenantMode)
		if err != nil {
			abortWithStatus(c, tenantStatusCode(err), err.Error())
			return
		}

		// 多租模式下还要确认当前登录用户属于该租户，避免只靠 header 越权。
		if enableMultiTenantMode {
			user, userErr := auth.GetUser(c.Request.Context())
			if userErr != nil || user.ID == "" {
				abortWithStatus(c, http.StatusUnauthorized, "authenticated user is required")
				return
			}
			if err := verifyTenantAccess(c.Request.Context(), user, tenantID, verifier); err != nil {
				abortWithStatus(c, tenantStatusCode(err), err.Error())
				return
			}
		}

		// 校验通过后再写入上下文，后续业务只消费 context 中的 tenantID。
		ctx := WithTenantID(c.Request.Context(), tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func verifyTenantAccess(ctx context.Context, user auth.User, tenantID string, verifier Verifier) error {
	// TODO: 认证中间件应一次设置好当前用户（id/租户/名字/状态），并做登录态级缓存，
	// 避免每个请求重复打 bk-login/bk-user。tenant 校验应只读取已有 request.user，
	// 不再回源外部接口。后续用户认证链路需要整体优化
	if user.GetTenantID() != "" {
		if user.GetTenantID() != tenantID {
			return ErrTenantAccessDenied
		}
		return nil
	}
	if verifier == nil {
		return errors.New("tenant verifier is not configured")
	}
	// 登录态未带租户时，回源 bk-user 校验用户归属和账号状态。
	return verifier.Verify(ctx, user.ID, tenantID)
}

func tenantStatusCode(err error) int {
	switch {
	case errors.Is(err, ErrTenantIDRequired), errors.Is(err, ErrTenantIDInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrTenantAccessDenied), errors.Is(err, ErrTenantUserDisabled):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func abortWithStatus(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"message": message,
		},
	})
}
