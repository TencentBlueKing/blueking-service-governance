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
	"fmt"
	"net/http"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
)

const (
	bkLoginGatewayName = "bk-login"
	// defaultRequestTimeout 与认证中间件超时保持一致，这是请求入口同步校验，不是后台集成调用。
	defaultRequestTimeout = 10 * time.Second
)

// BkTokenApigwAuthBackend 通过蓝鲸 API 网关校验 bk_token 并获取用户信息。
type BkTokenApigwAuthBackend struct {
	define.BkApiClient
	// LoginPageURL 是已解析的登录页根地址，仅用于 GetLoginUrl。
	LoginPageURL string
}

// GetLoginUrl 获取登录地址。
func (b *BkTokenApigwAuthBackend) GetLoginUrl() string {
	return fmt.Sprintf("%s/plain/", b.LoginPageURL)
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

// GetUserInfo 通过 API 网关校验 bk_token 并获取用户信息。
func (b *BkTokenApigwAuthBackend) GetUserInfo(ctx context.Context, userCred string) (*UserInfo, error) {
	op := b.NewOperation(
		bkapi.OperationConfig{
			Name:   "get_userinfo",
			Method: http.MethodGet,
			Path:   "/api/v3/open/bk-tokens/userinfo/",
		},
		bkapi.OptSetRequestQueryParams(map[string]string{"bk_token": userCred}),
	)

	var result apigwUserInfoResponse
	resp, err := op.SetContext(ctx).SetResult(&result).Request()
	if err != nil {
		return nil, errors.Wrap(err, "get user info from bk-login")
	}
	defer resp.Body.Close()

	if !httpresp.IsSuccess(resp) {
		if result.Error != nil {
			return nil, errors.New(result.Error.Message)
		}
		return nil, errors.Errorf("call bk-login get_userinfo failed, http code: %d", resp.StatusCode)
	}
	if result.Data.BkUsername == "" {
		return nil, errors.New("bk-login returned empty bk_username")
	}

	return &UserInfo{ID: result.Data.BkUsername, TenantID: result.Data.TenantID}, nil
}

// NewBkTokenApigwAuthBackend 创建通过 bk-login 网关校验 bk_token 的认证后端。
func NewBkTokenApigwAuthBackend(
	apiURLTmpl, stage, bkAppCode, bkAppSecret, loginPageURL string,
) (*BkTokenApigwAuthBackend, error) {
	apiClient, err := bkapi.NewBkApiClient(bkLoginGatewayName, bkapi.ClientConfig{
		BkApiUrlTmpl: apiURLTmpl,
		Stage:        stage,
		AppCode:      bkAppCode,
		AppSecret:    bkAppSecret,
		ClientOptions: []define.BkApiClientOption{
			bkapi.OptJsonResultProvider(),
			bkapi.OptTimeout(defaultRequestTimeout),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "new bk-login client")
	}
	return &BkTokenApigwAuthBackend{
		BkApiClient:  apiClient,
		LoginPageURL: loginPageURL,
	}, nil
}
