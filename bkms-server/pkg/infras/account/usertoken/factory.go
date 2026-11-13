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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// 签发 token 时使用的用户凭证名称，同时也是浏览器中保存该凭证的 cookie 名
const (
	CredentialBkTicket = "bk_ticket"
	CredentialBkToken  = "bk_token"
)

// NewTokenClient 根据账号配置选择 token 服务，选择规则见 config.AccountConfig.UseSSMTokenService。
//
// 登录态为 bk_token 但未配置 SSM 地址时回退到 API 网关 token 服务（需要 bk_ticket），
// 是为了兼容此前未显式配置 backendType（默认 bk_token）的存量部署。
// 返回的 client 同时用于签发和校验 token，两种服务签发的 token 互不认可，切换后需重新获取。
func NewTokenClient(ctx context.Context, account config.AccountConfig, app config.BkAppConfig) TokenClient {
	if account.UseSSMTokenService() {
		return NewSSMTokenClient(account.SSMBaseURL, app.Code, app.Secret)
	}
	if account.BackendType == config.AuthBackendBkToken {
		log.Warn(ctx, "account.ssmBaseURL is not configured, user token falls back to the API gateway token service")
	}
	return NewAPIGatewayTokenClient(account.AuthBaseURL, app.Code, app.Secret)
}
