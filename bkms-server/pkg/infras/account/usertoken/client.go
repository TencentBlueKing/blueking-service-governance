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

// Package usertoken contains the token client which request the token service.
package usertoken

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// TokenClient is the interface for managing user access tokens.
type TokenClient interface {
	// GetToken gets an access token of the given user.
	GetToken(
		ctx context.Context, username string, credentials map[string]string, envName string, needNewToken bool,
	) (*AccessToken, error)

	// RefreshToken refreshes an access token by using a refresh token.
	RefreshToken(
		ctx context.Context, refreshToken, envName string, needNewToken bool,
	) (*AccessToken, error)

	// GetUserInfo get the user information by the access token.
	GetUserInfo(ctx context.Context, token string) (string, error)
}

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

// APIGatewayTokenClient is the client for managing user tokens.
type APIGatewayTokenClient struct {
	// BaseURL is the base URL of the blueking API gateway.
	BaseURL string

	BkAppCode   string
	BkAppSecret string
}

// AccessToken contains the access token for a user.
type AccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// UserAccessToken is the compatibility alias for the former gateway package type name.
type UserAccessToken = AccessToken

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

func parseUserAccessToken(respObj ClientResp) (*AccessToken, error) {
	var data TokenResultData
	if err := mapstructure.Decode(respObj.Data, &data); err != nil {
		return nil, fmt.Errorf("parsing token data error")
	}
	if data.AccessToken == "" {
		return nil, fmt.Errorf("access token not found")
	}
	return &AccessToken{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		ExpiresIn:    data.ExpiresIn,
	}, nil
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

// toSuccessRespObj turn the response body into a structured object, return error if the
// given content is malformed or the "code" in the content is not "0"("0" mean success).
//
// INFO: The response use 200 status code even if it's an error, so we won't check the status code.
func toSuccessRespObj(body []byte) (ClientResp, error) {
	// Try to parse the response and check the "code"
	respObj := ClientResp{}
	if err := json.Unmarshal(body, &respObj); err != nil {
		return respObj, fmt.Errorf("parsing response result: %w", err)
	}
	// The code might be int(1308406) or string("0"), cast it to int anyway.
	code := cast.ToInt(respObj.Code)
	if code != 0 {
		return respObj, fmt.Errorf("api error: %d-%s", code, respObj.Message)
	}
	return respObj, nil
}

// ClientResp is the common structure of API responses.
type ClientResp struct {
	Message string `json:"message"`
	// Code might be int or string
	Code     any    `json:"code"`
	CodeName string `json:"code_name"`
	Result   bool   `json:"result"`
	Data     any    `json:"data"`
}

// TokenResultData holds the data of the getting token response.
type TokenResultData struct {
	AccessToken  string `mapstructure:"access_token"`
	RefreshToken string `mapstructure:"refresh_token"`
	Scope        string `mapstructure:"scope"`
	ExpiresIn    int    `mapstructure:"expires_in"`
	UserID       string `mapstructure:"user_id"`
	UserType     string `mapstructure:"user_type"`
}

// UserInfoResultData holds the data of the getting user info response.
type UserInfoResultData struct {
	IDProviders struct {
		Rtx struct {
			Username string `mapstructure:"username,required"`
		} `mapstructure:"rtx"`
	} `mapstructure:"id_providers"`
}
