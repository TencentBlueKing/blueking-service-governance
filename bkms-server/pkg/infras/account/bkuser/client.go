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
	"net/http"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
)

// UserClient is the minimal client contract required by tenant verification.
type UserClient interface {
	GetUser(ctx context.Context, bkUsername string) (*User, error)
}

// defaultRequestTimeout bounds the synchronous request-path bk-user lookup.
// Keep it aligned with auth/token timeouts instead of the longer 30s-60s
// budgets commonly used by background cloudapi integrations.
const defaultRequestTimeout = 10 * time.Second

// Client requests bk-user APIs using application authorization through bk-apigateway SDK.
type Client struct {
	apiClient define.BkApiClient
}

// NewClient builds a bk-user client from the gateway URL template, stage and app credentials.
func NewClient(apiURLTmpl, stage, bkAppCode, bkAppSecret string) (*Client, error) {
	apiClient, err := bkapi.NewBkApiClient(bkUserGatewayName, bkapi.ClientConfig{
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
		return nil, errors.Wrap(err, "new bk-user client")
	}
	return &Client{apiClient: apiClient}, nil
}

// GetUser gets the tenant-aware user info for the given bk_username.
func (c *Client) GetUser(ctx context.Context, bkUsername string) (*User, error) {
	op := c.apiClient.NewOperation(
		bkapi.OperationConfig{
			Name:   "get_user",
			Method: http.MethodGet,
			Path:   "/api/v3/open/tenant/users/{bk_username}/",
		},
		bkapi.OptSetRequestPathParams(map[string]string{"bk_username": bkUsername}),
	)

	var result getUserResponse
	resp, err := op.SetContext(ctx).SetResult(&result).Request()
	if err != nil {
		return nil, errors.Wrapf(err, "get bk-user user %s", bkUsername)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	requestURL := bkUserGatewayName
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		requestURL = resp.Request.URL.String()
	}
	if !httpresp.IsSuccess(resp) {
		if result.Error != nil {
			return nil, errors.Errorf(
				"bk-user %s returned status %d: code=%s, message=%s",
				requestURL, resp.StatusCode, result.Error.Code, result.Error.Message,
			)
		}
		return nil, errors.Errorf("bk-user %s returned status %d", requestURL, resp.StatusCode)
	}
	if result.Error != nil {
		return nil, errors.Errorf(
			"bk-user %s returned error: code=%s, message=%s",
			requestURL, result.Error.Code, result.Error.Message,
		)
	}
	if result.Data.TenantID == "" {
		return nil, errors.Errorf("bk-user user %s has empty tenant_id", bkUsername)
	}
	return &result.Data, nil
}
