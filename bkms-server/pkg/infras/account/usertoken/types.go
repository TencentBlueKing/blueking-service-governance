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

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// TokenClient is the interface for managing user access tokens.
type TokenClient interface {
	// CredentialName returns the name of the user credential GetToken needs, which is also the name
	// of the cookie that holds the credential in the browser.
	CredentialName() string

	// RequiresUsername reports whether GetToken needs the username besides the credential.
	RequiresUsername() bool

	// GetToken gets an access token of the given user. Implementations may ignore the
	// arguments their token service does not need, see the doc of each implementation.
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

// AccessToken contains the access token for a user.
type AccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func parseUserAccessToken(respObj ClientResp) (*AccessToken, error) {
	var data TokenResultData
	if err := mapstructure.Decode(respObj.Data, &data); err != nil {
		return nil, errors.Wrap(err, "parsing token data")
	}
	if data.AccessToken == "" {
		return nil, errors.New("access token not found")
	}
	return &AccessToken{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		ExpiresIn:    data.ExpiresIn,
	}, nil
}

// toSuccessRespObj turn the response body into a structured object, return error if the
// given content is malformed or the "code" in the content is not "0"("0" mean success).
//
// INFO: The API gateway token service responds 200 even if it's an error, so the status code is
// not checked here, callers whose service uses error status codes should report it themselves.
func toSuccessRespObj(body []byte) (ClientResp, error) {
	// Try to parse the response and check the "code"
	respObj := ClientResp{}
	if err := json.Unmarshal(body, &respObj); err != nil {
		return respObj, errors.Wrap(err, "parsing response result")
	}
	// The code might be int(1308406) or string("0"), cast it to int anyway.
	code := cast.ToInt(respObj.Code)
	if code != 0 {
		return respObj, errors.Errorf("api error: %d-%s", code, respObj.Message)
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
