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

package usertoken

import (
	"context"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
)

// NewSSMTokenClient creates a new SSMTokenClient instance.
//
// - baseURL is the base URL of the blueking SSM service, SSM can only be requested directly
// instead of through the API gateway.
// - bkAppCode and bkAppSecret are the blueking app identity, required for calling APIs.
func NewSSMTokenClient(baseURL, bkAppCode, bkAppSecret string) *SSMTokenClient {
	return &SSMTokenClient{
		BaseURL:     baseURL,
		BkAppCode:   bkAppCode,
		BkAppSecret: bkAppSecret,
	}
}

// SSMTokenClient is the client for managing user tokens by the blueking SSM service,
// it exchanges the bk_token credential for an access token.
type SSMTokenClient struct {
	// BaseURL is the base URL of the blueking SSM service.
	BaseURL string

	BkAppCode   string
	BkAppSecret string
}

// CredentialName returns "bk_token".
func (s *SSMTokenClient) CredentialName() string { return CredentialBkToken }

// RequiresUsername returns false, SSM identifies the user by the bk_token only.
func (s *SSMTokenClient) RequiresUsername() bool { return false }

// GetToken gets an access token of the user who owns the "bk_token" in credentials.
//
// SSM identifies the user by "bk_token" only and doesn't isolate tokens by environment,
// so username, envName and needNewToken are ignored.
func (s *SSMTokenClient) GetToken(
	ctx context.Context, _ string, credentials map[string]string, _ string, _ bool,
) (*AccessToken, error) {
	respObj, err := s.post(ctx, "/api/v1/auth/access-tokens", map[string]string{
		"grant_type":  "authorization_code",
		"id_provider": "bk_login",
		"bk_token":    credentials[CredentialBkToken],
	})
	if err != nil {
		return nil, errors.Wrap(err, "request ssm token api")
	}
	return parseUserAccessToken(respObj)
}

// RefreshToken refreshes an access token by using the refresh token, envName and needNewToken
// are ignored for the same reason as GetToken.
func (s *SSMTokenClient) RefreshToken(ctx context.Context, refreshToken, _ string, _ bool) (*AccessToken, error) {
	respObj, err := s.post(ctx, "/api/v1/auth/access-tokens/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
	if err != nil {
		return nil, errors.Wrap(err, "request ssm refresh token api")
	}
	return parseUserAccessToken(respObj)
}

// GetUserInfo get the username by the access token. It returns error when the token has
// been expired or is invalid.
func (s *SSMTokenClient) GetUserInfo(ctx context.Context, token string) (string, error) {
	respObj, err := s.post(ctx, "/api/v1/auth/access-tokens/verify", map[string]string{
		"access_token": token,
	})
	if err != nil {
		return "", errors.Wrap(err, "request ssm verify token api")
	}

	var data ssmVerifyResultData
	if err = mapstructure.Decode(respObj.Data, &data); err != nil {
		return "", errors.Wrap(err, "parse ssm verify token data")
	}
	if data.Identity.Username == "" {
		return "", errors.New("username not provided")
	}
	return data.Identity.Username, nil
}

func (s *SSMTokenClient) post(ctx context.Context, path string, body map[string]string) (ClientResp, error) {
	requestURL, err := url.JoinPath(s.BaseURL, path)
	if err != nil {
		return ClientResp{}, errors.Wrapf(err, "construct url with path %s", path)
	}
	resp, err := httpcli.NewRestyClient(ctx).R().
		SetContext(ctx).
		SetHeaders(map[string]string{"X-Bk-App-Code": s.BkAppCode, "X-Bk-App-Secret": s.BkAppSecret}).
		SetBody(body).
		Post(requestURL)
	if err != nil {
		return ClientResp{}, errors.Wrapf(err, "post %s", requestURL)
	}
	respObj, err := toSuccessRespObj(resp.Body())
	if err != nil && resp.IsError() {
		return respObj, errors.Wrapf(err, "post %s: status %d", requestURL, resp.StatusCode())
	}
	return respObj, err
}

// ssmVerifyResultData holds the data of the verifying token response.
type ssmVerifyResultData struct {
	Identity struct {
		Username string `mapstructure:"username"`
		UserType string `mapstructure:"user_type"`
	} `mapstructure:"identity"`
}
