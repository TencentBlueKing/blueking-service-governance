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

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// 签发 token 时使用的用户凭证名称，同时也是浏览器中保存该凭证的 cookie 名
const (
	CredentialBkTicket = "bk_ticket"
	CredentialBkToken  = "bk_token"
)

// ClientConfig 是创建 TokenClient 所需的配置
type ClientConfig struct {
	// AuthBackendType 是用户登录认证使用的后端类型，bk_ticket 或 bk_token
	AuthBackendType string
	// APIGatewayBaseURL 是 API 网关 token 服务的基础 URL
	APIGatewayBaseURL string
	// SSMBaseURL 是蓝鲸 SSM 服务的基础 URL
	SSMBaseURL  string
	BkAppCode   string
	BkAppSecret string
}

// NewTokenClient 根据登录认证方式选择 token 服务，返回 client 以及签发 token 时所需的用户凭证名称。
//
// 登录态为 bk_token 且配置了 SSM 地址时使用 SSM，否则使用 API 网关 token 服务（需要 bk_ticket）。
// 未配置 SSM 时回退到 API 网关，是为了兼容此前未显式配置 backendType（默认 bk_token）的存量部署。
func NewTokenClient(ctx context.Context, cfg ClientConfig) (TokenClient, string) {
	if cfg.AuthBackendType == CredentialBkToken {
		if cfg.SSMBaseURL != "" {
			return NewSSMTokenClient(cfg.SSMBaseURL, cfg.BkAppCode, cfg.BkAppSecret), CredentialBkToken
		}
		log.Warn(ctx, "account.ssmBaseURL is not configured, user token falls back to the API gateway token service")
	}
	return NewAPIGatewayTokenClient(cfg.APIGatewayBaseURL, cfg.BkAppCode, cfg.BkAppSecret), CredentialBkTicket
}
