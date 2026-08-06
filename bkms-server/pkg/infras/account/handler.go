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

package account

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/usertoken"
)

const tokenRequestTimeout = 10 * time.Second

// Config contains account API runtime configuration.
type Config struct {
	// AuthEnvName 是用户 token 签发时使用的环境
	AuthEnvName string
	// LoginURL 是 bk-login 服务的地址，用于拼接重定向时的登录链接
	LoginURL string
}

// APIHandler implements the account Gin APIs.
type APIHandler struct {
	cfg         Config
	tokenClient usertoken.TokenClient
}

// NewHandler creates an account API handler.
func NewHandler(cfg Config, tokenClient usertoken.TokenClient) *APIHandler {
	return &APIHandler{cfg: cfg, tokenClient: tokenClient}
}

// CreateToken gets an access token for the currently logged-in user.
//
//	@ID			CreateToken
//	@Summary	Get current user access token
//	@Tags		account
//	@Produce	json
//	@Router		/user_token/token [get]
func (h *APIHandler) CreateToken(c *gin.Context) {
	username, _ := c.Cookie("bk_uid")
	ticket, _ := c.Cookie("bk_ticket")
	if username == "" || ticket == "" {
		if c.Query("redirect_login") != "true" {
			writeLegacyError(c, "no user credentials found in cookie, please login first.")
			return
		}
		if h.cfg.LoginURL == "" {
			writeLegacyError(c, `plugin config "bk_login_url" is required when redirect_login is true.`)
			return
		}
		c.Redirect(http.StatusFound, h.cfg.LoginURL+"/plain/?c_url="+url.QueryEscape(c.Request.URL.String()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), tokenRequestTimeout)
	defer cancel()
	token, err := h.tokenClient.GetToken(
		ctx,
		username,
		map[string]string{"bk_ticket": ticket},
		h.cfg.AuthEnvName,
		parseNeedNewToken(c.Query("need_new_token")),
	)
	if err != nil {
		writeLegacyError(c, "get token error: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, token)
}

// RefreshToken refreshes an access token.
//
//	@ID			RefreshToken
//	@Summary	Refresh user access token
//	@Tags		account
//	@Produce	json
//	@Router		/user_token/refresh [get]
func (h *APIHandler) RefreshToken(c *gin.Context) {
	refreshToken := c.Query("refresh_token")
	if refreshToken == "" {
		writeLegacyError(c, "refresh_token is required.")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), tokenRequestTimeout)
	defer cancel()
	token, err := h.tokenClient.RefreshToken(
		ctx, refreshToken, h.cfg.AuthEnvName, parseNeedNewToken(c.Query("need_new_token")),
	)
	if err != nil {
		writeLegacyError(c, "refresh token error: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, token)
}

// ValidateToken validates an access token.
//
//	@ID			ValidateToken
//	@Summary	Validate user access token
//	@Tags		account
//	@Produce	json
//	@Router		/user_token/validate [get]
func (h *APIHandler) ValidateToken(c *gin.Context) {
	accessToken := c.Query("access_token")
	if accessToken == "" {
		writeLegacyError(c, "access_token is required.")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), tokenRequestTimeout)
	defer cancel()
	username, err := h.tokenClient.GetUserInfo(ctx, accessToken)
	if err != nil {
		writeLegacyError(c, "validate token error: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"username": username})
}

// GetCurrentUser returns the user authenticated by the auth middleware.
//
//	@ID			GetCurrentUser
//	@Summary	Get current user
//	@Tags		account
//	@Produce	json
//	@Router		/simple_account/info [get]
func (h *APIHandler) GetCurrentUser(c *gin.Context) {
	// 直接从上下文中获取用户信息，信息由中间件注入
	result, ok := auth.GetResult(c.Request.Context())
	if !ok {
		writeLegacyError(c, "failed to get the authenticated user info, check the middleware config")
		return
	}
	if !result.RequestUser.IsAuthenticated() {
		// This response intentionally preserves the historical response contract during migration.
		c.JSON(http.StatusUnauthorized, gin.H{"login_url": result.LoginURL, "code": "1000"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": result.RequestUser.GetID()})
}

func parseNeedNewToken(raw string) bool {
	value, err := strconv.ParseBool(raw)
	return err == nil && value
}

func writeLegacyError(c *gin.Context, message string) {
	// Account endpoints retain the legacy error shape for compatibility and may adopt bkms errors later.
	c.JSON(http.StatusBadRequest, gin.H{"message": message, "code": "1000"})
}
