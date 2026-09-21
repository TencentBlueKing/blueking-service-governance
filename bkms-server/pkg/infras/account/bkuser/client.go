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

// Package bkuser provides a thin client for bk-user tenant-aware user APIs.
package bkuser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
)

// UserClient is the minimal client contract required by tenant verification.
type UserClient interface {
	GetUser(ctx context.Context, bkUsername string) (*User, error)
}

// Client requests bk-user APIs using application authorization.
type Client struct {
	baseURL     string
	bkAppCode   string
	bkAppSecret string
}

// User is the subset of bk-user user fields currently consumed by BKMS.
type User struct {
	TenantID    string `json:"tenant_id"`
	BkUsername  string `json:"bk_username"`
	LoginName   string `json:"login_name"`
	DisplayName string `json:"display_name"`
	TimeZone    string `json:"time_zone"`
	Language    string `json:"language"`
	Status      string `json:"status"`
}

type getUserResponse struct {
	Data  User `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type appAuthorizationHeader struct {
	BkAppCode   string `json:"bk_app_code"`
	BkAppSecret string `json:"bk_app_secret"`
}

// NewClient builds a bk-user client from the gateway base URL and app credentials.
func NewClient(baseURL, bkAppCode, bkAppSecret string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		bkAppCode:   bkAppCode,
		bkAppSecret: bkAppSecret,
	}
}

func (c *Client) authHeader() (string, error) {
	authHeader, err := json.Marshal(appAuthorizationHeader{
		BkAppCode:   c.bkAppCode,
		BkAppSecret: c.bkAppSecret,
	})
	if err != nil {
		return "", errors.Wrap(err, "marshal bk-user auth header")
	}
	return string(authHeader), nil
}

// GetUser gets the tenant-aware user info for the given bk_username.
func (c *Client) GetUser(ctx context.Context, bkUsername string) (*User, error) {
	authHeader, err := c.authHeader()
	if err != nil {
		return nil, errors.Wrap(err, "build bk-user auth header")
	}

	requestURL, err := url.JoinPath(c.baseURL, "/api/v3/open/tenant/users/"+url.PathEscape(bkUsername)+"/")
	if err != nil {
		return nil, errors.Wrapf(err, "build bk-user user url for %s", bkUsername)
	}

	client := httpcli.NewRestyClient(ctx)
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("X-Bkapi-Authorization", authHeader).
		ForceContentType("application/json").
		Get(requestURL)
	if err != nil {
		return nil, errors.Wrapf(err, "get bk-user user %s", bkUsername)
	}

	if resp.StatusCode() != http.StatusOK {
		var failed getUserResponse
		if unmarshalErr := json.Unmarshal(resp.Body(), &failed); unmarshalErr == nil && failed.Error != nil {
			return nil, errors.Errorf(
				"bk-user %s returned status %d: code=%s, message=%s",
				requestURL,
				resp.StatusCode(),
				failed.Error.Code,
				failed.Error.Message,
			)
		}
		return nil, errors.Errorf("bk-user %s returned status %d: %s", requestURL, resp.StatusCode(), resp.String())
	}

	var result getUserResponse
	if err = json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, errors.Wrapf(err, "unmarshal bk-user response from %s", requestURL)
	}
	if result.Error != nil {
		return nil, errors.Errorf(
			"bk-user %s returned error: code=%s, message=%s",
			requestURL,
			result.Error.Code,
			result.Error.Message,
		)
	}
	if result.Data.TenantID == "" {
		return nil, errors.Errorf("bk-user user %s has empty tenant_id", bkUsername)
	}
	return &result.Data, nil
}
