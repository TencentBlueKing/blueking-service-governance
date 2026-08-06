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

// Package auth authenticates BlueKing users in Gin requests.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/ctxkey"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth/backends"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/usertoken"
)

const (
	// BackendBkTicket 使用 bk_ticket 完成用户认证。
	BackendBkTicket = "bk_ticket"
	// BackendBkToken 使用 bk_token 完成用户认证。
	BackendBkToken = "bk_token"

	// authTimeout 调用认证后端服务的超时时间。
	authTimeout = 10 * time.Second
)

// Config 是用户认证中间件配置。
type Config struct {
	// BackendType 是用于认证的后端类型，例如 bk_ticket 或 bk_token。
	BackendType string
	// LoginURL 是用于认证的 bk_login 服务的访问地址。
	LoginURL string
	// AllowSetUserInHeader 允许通过请求头直接设置已认证用户。
	// 该选项仅可用于本地开发和测试。
	AllowSetUserInHeader bool
}

// Result 表示完整的用户认证结果，包括认证失败的请求。
type Result struct {
	// RequestUser 是认证结果中的用户信息，可能是未认证的匿名用户。
	RequestUser RequestUser

	// LoginURL 是提供给客户端的有效登录地址。
	LoginURL string
	// ErrorMsg 是认证过程中发生的错误信息，认证成功时为空。
	ErrorMsg string
}

type resultContextKey struct{}

// GetResult 获取中间件写入请求上下文的认证结果。
func GetResult(ctx context.Context) (Result, bool) {
	result, ok := ctx.Value(resultContextKey{}).(Result)
	return result, ok
}

// Optional 尝试认证当前用户，无论认证是否成功都会继续处理请求。
// 在一些特殊的不强制要求用户认证的 API 中使用。
func Optional(cfg Config, tokenClient usertoken.TokenClient) gin.HandlerFunc {
	authBackend, backendType := getBackend(cfg)
	return middleware(authBackend, backendType, tokenClient, cfg.AllowSetUserInHeader, false)
}

// Required 认证当前用户，认证失败时中止请求。
// 应该作为项目最主要的默认认证中间件使用。
func Required(cfg Config, tokenClient usertoken.TokenClient) gin.HandlerFunc {
	authBackend, backendType := getBackend(cfg)
	return middleware(authBackend, backendType, tokenClient, cfg.AllowSetUserInHeader, true)
}

// 制造中间件函数的工厂函数。
// Args:
//   - authBackend: 用于认证的后端服务对象。
//   - backendType: 认证后端类型，例如 bk_ticket 或 bk_token。
//   - tokenClient: 用于访问用户令牌服务的客户端对象。
//   - allowSetUserInHeader: 是否允许通过请求头直接设置已认证用户。
//   - abortWhenUnauthorized: 是否在认证失败时中止请求。
func middleware(
	authBackend AuthBackend,
	backendType string,
	tokenClient usertoken.TokenClient,
	allowSetUserInHeader bool,
	abortWhenUnauthorized bool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		result := authenticate(
			c.Request.Context(),
			c.Request,
			authBackend,
			backendType,
			tokenClient,
			allowSetUserInHeader,
		)
		ctx := injectResult(c.Request.Context(), result)
		c.Request = c.Request.WithContext(ctx)

		if abortWhenUnauthorized && !result.RequestUser.IsAuthenticated() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHENTICATED",
					"message": "unauthorized",
					"system":  "bkms",
					"module":  "bkms-server",
					"details": []map[string]any{},
				},
			})
			return
		}
		c.Next()
	}
}

// 完成用户认证的核心函数，供中间件调用。
func authenticate(
	ctx context.Context,
	request *http.Request,
	authBackend AuthBackend,
	backendType string,
	tokenClient usertoken.TokenClient,
	allowSetUserInHeader bool,
) Result {
	result := Result{RequestUser: AnonymousUser{}, LoginURL: authBackend.GetLoginUrl()}
	if allowSetUserInHeader {
		if user, err := getUserFromHeaders(request); err == nil {
			result.RequestUser = user
			return result
		}
	}

	userCredential := authBackend.GetUserCredential(request)
	accessToken := getAccessToken(request)

	ctx, cancel := context.WithTimeout(ctx, authTimeout)
	defer cancel()

	var userInfo *backends.UserInfo
	cred := UserCredential{}
	var err error
	// 首先尝试使用 credentials 认证，如果未提供，则使用 access token。
	switch {
	case userCredential != "":
		userInfo, err = authBackend.GetUserInfo(ctx, userCredential)
		cred = newCredential(backendType, userCredential)
	case accessToken != "":
		var username string
		username, err = tokenClient.GetUserInfo(ctx, accessToken)
		userInfo = &backends.UserInfo{ID: username}
		cred = UserCredential{AccessToken: accessToken}
	default:
		err = errors.New("no token or access token can be found")
	}

	if err != nil {
		result.RequestUser = AnonymousUser{Cred: cred}
		result.ErrorMsg = fmt.Sprintf("calling backend service: %s", err)
		return result
	}

	result.RequestUser = User{ID: userInfo.ID, Cred: cred}
	return result
}

// 用于保存来自于请求头的用户信息，供测试和开发环境使用。
type headerUserInfo struct {
	UserID string `json:"userId"`
}

// 从请求头中获取用户信息，仅在允许通过请求头设置用户的环境中使用。
func getUserFromHeaders(request *http.Request) (User, error) {
	rawUserInfo := request.Header.Get(BKAuthKey)
	if rawUserInfo == "" {
		return User{}, errors.Errorf("%s header is empty", BKAuthKey)
	}
	userInfo := headerUserInfo{}
	if err := json.Unmarshal([]byte(rawUserInfo), &userInfo); err != nil {
		return User{}, errors.Wrapf(err, "unmarshal %s header", BKAuthKey)
	}
	if userInfo.UserID == "" {
		return User{}, errors.Errorf("%s header does not contain an authenticated user", BKAuthKey)
	}

	// Get the optional credential field
	rawCredential := request.Header.Get(BKCredentialKey)
	if rawCredential == "" {
		return User{ID: userInfo.UserID, Cred: UserCredential{}}, nil
	}
	credential := UserCredential{}
	if err := json.Unmarshal([]byte(rawCredential), &credential); err != nil {
		return User{}, errors.Wrapf(err, "unmarshal %s header", BKCredentialKey)
	}
	return User{ID: userInfo.UserID, Cred: credential}, nil
}

func newCredential(backendType, userCredential string) UserCredential {
	if backendType == BackendBkTicket {
		return UserCredential{BkTicket: userCredential}
	}
	return UserCredential{BkToken: userCredential}
}

// Inject the auth user to the context
func injectResult(ctx context.Context, result Result) context.Context {
	ctx = context.WithValue(ctx, resultContextKey{}, result)
	// Set user to the context if it's authenticated
	if user, ok := result.RequestUser.(User); ok {
		ctx = context.WithValue(ctx, ctxkey.AuthUser, user)
	}
	return ctx
}

// getAccessToken 获取 Authorization 请求头中的 Bearer access token。
func getAccessToken(request *http.Request) string {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}
