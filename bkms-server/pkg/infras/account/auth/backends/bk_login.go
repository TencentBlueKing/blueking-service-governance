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

package backends

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
)

const (
	bkLoginGatewayName     = "bk-login"
	bkApiNamePlaceholder   = "{api_name}"
	defaultBkLoginAPIStage = "prod"
)

// BkTokenApigwAuthBackend 通过蓝鲸 API 网关校验 bk_token 并获取用户信息。
type BkTokenApigwAuthBackend struct {
	// ApigwBaseURL 是 bk-login 网关前缀，不含接口路径。
	ApigwBaseURL string
	// BkAppCode 应用 ID，用于 API 网关应用认证
	BkAppCode string
	// BkAppSecret 应用密钥，用于 API 网关应用认证
	BkAppSecret string
	// LoginPageURL 是已解析的登录页根地址，仅用于 GetLoginUrl。
	LoginPageURL string
}

// GetLoginUrl 获取登录地址。
func (b *BkTokenApigwAuthBackend) GetLoginUrl() string {
	return fmt.Sprintf("%s/plain/", b.LoginPageURL)
}

// BuildBkLoginGatewayURL renders the bk-login API gateway base URL from template and stage.
func BuildBkLoginGatewayURL(apiURLTmpl, stage string) (string, error) {
	gatewayURL := strings.ReplaceAll(apiURLTmpl, bkApiNamePlaceholder, bkLoginGatewayName)
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = defaultBkLoginAPIStage
	}

	parsedURL, err := url.Parse(gatewayURL)
	if err != nil {
		return "", errors.Wrap(err, "parse bk-login gateway url")
	}
	parsedURL.Path, err = url.JoinPath(parsedURL.Path, stage)
	if err != nil {
		return "", errors.Wrap(err, "join bk-login gateway stage")
	}
	return parsedURL.String(), nil
}

// GetUserCredential 获取用户票据
func (b *BkTokenApigwAuthBackend) GetUserCredential(request *http.Request) string {
	if userToken := request.Header.Get("X-User-Bk-Token"); userToken != "" {
		return userToken
	}
	cookie, err := request.Cookie("bk_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// apigwAuthHeader 构造 API 网关要求的应用认证请求头 X-Bkapi-Authorization。
func (b *BkTokenApigwAuthBackend) apigwAuthHeader() (string, error) {
	auth := map[string]string{
		"bk_app_code":   b.BkAppCode,
		"bk_app_secret": b.BkAppSecret,
	}
	data, err := json.Marshal(auth)
	if err != nil {
		return "", errors.Wrap(err, "marshal apigw auth header")
	}
	return string(data), nil
}

// apigwUserInfoResponse 是 API 网关 bk-login userinfo 接口的响应结构。
type apigwUserInfoResponse struct {
	Data struct {
		BkUsername  string `json:"bk_username"`
		TenantID    string `json:"tenant_id"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// GetUserInfo 通过 API 网关校验 bk_token 并获取用户信息。
func (b *BkTokenApigwAuthBackend) GetUserInfo(ctx context.Context, userCred string) (*UserInfo, error) {
	authHeader, err := b.apigwAuthHeader()
	if err != nil {
		return nil, errors.Wrap(err, "build apigw auth header")
	}

	url := strings.TrimRight(b.ApigwBaseURL, "/") + "/api/v3/open/bk-tokens/userinfo/"
	client := httpcli.NewRestyClient(ctx)
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("X-Bkapi-Authorization", authHeader).
		SetQueryParams(map[string]string{"bk_token": userCred}).
		ForceContentType("application/json").
		Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "get user info from apigw %s", url)
	}

	if resp.StatusCode() != http.StatusOK {
		var failed apigwUserInfoResponse
		if unmarshalErr := json.Unmarshal(resp.Body(), &failed); unmarshalErr == nil && failed.Error != nil {
			return nil, errors.Errorf(
				"apigw %s returned status %d: code=%s, message=%s",
				url, resp.StatusCode(), failed.Error.Code, failed.Error.Message,
			)
		}
		return nil, errors.Errorf(
			"apigw %s returned status %d: %s", url, resp.StatusCode(), resp.String(),
		)
	}

	var result apigwUserInfoResponse
	if err = json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, errors.Wrapf(err, "unmarshal apigw response from %s", url)
	}
	if result.Error != nil {
		return nil, errors.Errorf(
			"apigw %s returned error: code=%s, message=%s",
			url, result.Error.Code, result.Error.Message,
		)
	}
	// bk-login userinfo 以 data.bk_username 作为登录用户标识。
	if result.Data.BkUsername == "" {
		return nil, errors.Errorf("apigw %s returned empty bk_username", url)
	}

	return &UserInfo{ID: result.Data.BkUsername, TenantID: result.Data.TenantID}, nil
}

// NewBkTokenApigwAuthBackend 创建 BkTokenApigwAuthBackend 实例。
func NewBkTokenApigwAuthBackend(apigwBaseURL, bkAppCode, bkAppSecret, loginPageURL string) *BkTokenApigwAuthBackend {
	return &BkTokenApigwAuthBackend{
		ApigwBaseURL: apigwBaseURL,
		BkAppCode:    bkAppCode,
		BkAppSecret:  bkAppSecret,
		LoginPageURL: loginPageURL,
	}
}
