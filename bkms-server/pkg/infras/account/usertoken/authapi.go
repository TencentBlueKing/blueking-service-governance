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
	"fmt"
	"log/slog"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// NewAPIGatewayTokenClient Creates a new APIGatewayTokenClient instance.
//
// - baseURL is the blueking API gateway's base URL, it will be used for building the full URL of the token API.
// - bkAppCode and bkAppSecret are the blueking app identity, required for calling APIs.
func NewAPIGatewayTokenClient(baseURL, bkAppCode, bkAppSecret string) *APIGatewayTokenClient {
	return &APIGatewayTokenClient{
		BaseURL:     baseURL,
		BkAppCode:   bkAppCode,
		BkAppSecret: bkAppSecret,
	}
}

// APIGatewayTokenClient is the client for managing user tokens by the token service behind the
// API gateway, it exchanges the bk_ticket credential and username for an access token.
type APIGatewayTokenClient struct {
	// BaseURL is the base URL of the blueking API gateway.
	BaseURL string

	BkAppCode   string
	BkAppSecret string
}

// CredentialName returns "bk_ticket".
func (b *APIGatewayTokenClient) CredentialName() string { return CredentialBkTicket }

// RequiresUsername returns true, the token service needs the username along with the bk_ticket.
func (b *APIGatewayTokenClient) RequiresUsername() bool { return true }

// GetToken gets an access token of the given user.
//
// - username is the username of the user.
// - credentials is a map which holds the user credentials, which include "bk_ticket" etc.
// - envName is the environment name for requesting token, token in different environments are isolated.
// - needNewToken indicates whether to request a new token, default to false to reuse the existing token.
//
// The method returns an access token object and error.
func (b *APIGatewayTokenClient) GetToken(
	ctx context.Context, username string, credentials map[string]string, envName string, needNewToken bool,
) (*AccessToken, error) {
	requestURL, err := url.JoinPath(b.BaseURL, "/auth_api/token/")
	if err != nil {
		return nil, fmt.Errorf("constructing token api url: %w", err)
	}
	client := httpcli.NewRestyClient(ctx)

	// Prepare the request body and headers
	headers := map[string]string{"X-Bk-App-Code": b.BkAppCode, "X-Bk-App-Secret": b.BkAppSecret}
	baseBody := map[string]string{
		"app_code":   b.BkAppCode,
		"app_secret": b.BkAppSecret,
		"env_name":   envName,
		"grant_type": "authorization_code",
		"rtx":        username,
		// Set "need_new_token" to "1" only when caller explicitly requests a new token.
		"need_new_token": boolToBoolIntString(needNewToken),
	}
	body := lo.Assign(baseBody, credentials)

	// Send the request
	resp, err := client.R().
		SetContext(ctx).
		SetHeaders(headers).
		SetBody(body).
		Post(requestURL)
	if err != nil {
		return nil, fmt.Errorf("requesting token api: %w", err)
	}
	respObj, err := toSuccessRespObj(resp.Body())
	if err != nil {
		return nil, err
	}
	return parseUserAccessToken(respObj)
}

// RefreshToken refreshes an access token by using the refresh token.
//
// - refreshToken is the refresh token of the user.
// - envName is the environment name for requesting token, token in different environments are isolated.
// - needNewToken indicates whether to request a new token, default to false to reuse the existing token.
func (b *APIGatewayTokenClient) RefreshToken(
	ctx context.Context, refreshToken, envName string, needNewToken bool,
) (*AccessToken, error) {
	requestURL, err := url.JoinPath(b.BaseURL, "/auth_api/refresh_token/")
	if err != nil {
		return nil, fmt.Errorf("constructing refresh token api url: %w", err)
	}
	client := httpcli.NewRestyClient(ctx)

	// Prepare the request query and headers
	headers := map[string]string{"X-Bk-App-Code": b.BkAppCode, "X-Bk-App-Secret": b.BkAppSecret}
	query := map[string]string{
		"app_code":       b.BkAppCode,
		"app_secret":     b.BkAppSecret,
		"env_name":       envName,
		"grant_type":     "refresh_token",
		"refresh_token":  refreshToken,
		"need_new_token": boolToBoolIntString(needNewToken),
	}

	// Send the request
	resp, err := client.R().
		SetContext(ctx).
		SetHeaders(headers).
		SetQueryParams(query).
		Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("requesting refresh_token api: %w", err)
	}
	respObj, err := toSuccessRespObj(resp.Body())
	if err != nil {
		return nil, err
	}
	return parseUserAccessToken(respObj)
}

// Convert a bool value to "1" or "0" string, which is required by the token API.
func boolToBoolIntString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

// GetUserInfo get the user information by the access token. It returns error when the token has
// been expired or is invalid.
func (b *APIGatewayTokenClient) GetUserInfo(ctx context.Context, token string) (string, error) {
	requestURL, err := url.JoinPath(b.BaseURL, "/auth_api/check_token/")
	if err != nil {
		return "", fmt.Errorf("constructing check token api url: %w", err)
	}
	client := httpcli.NewRestyClient(ctx)

	// Prepare the request body and headers
	headers := map[string]string{"X-Bk-App-Code": b.BkAppCode, "X-Bk-App-Secret": b.BkAppSecret}

	// Send the request
	log.Debug(ctx, "Get user info using the token client.")
	resp, err := client.R().
		SetContext(ctx).
		SetHeaders(headers).
		SetQueryParam("access_token", token).
		Get(requestURL)
	if resp != nil {
		log.DebugAttrs(ctx, "Get user info response", slog.String("body", resp.String()))
	}

	if err != nil {
		return "", fmt.Errorf("requesting token api: %w", err)
	}
	respObj, err := toSuccessRespObj(resp.Body())
	if err != nil {
		return "", err
	}

	// Get the username and return
	var data UserInfoResultData
	if err = mapstructure.Decode(respObj.Data, &data); err != nil {
		return "", fmt.Errorf("parsing user info data error")
	}
	username := data.IDProviders.Rtx.Username
	if username == "" {
		return "", fmt.Errorf("username not provided")
	}
	return username, nil
}

// UserInfoResultData holds the data of the getting user info response.
type UserInfoResultData struct {
	IDProviders struct {
		Rtx struct {
			Username string `mapstructure:"username,required"`
		} `mapstructure:"rtx"`
	} `mapstructure:"id_providers"`
}
