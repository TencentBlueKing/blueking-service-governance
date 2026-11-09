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

package apitracking

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// 客户端类型
const (
	// ClientCLI bkms-cli
	ClientCLI = "cli"
	// ClientWeb 前端，bk_ticket / bk_token 认证
	ClientWeb = "web"

	// ClientToken 非 CLI 的 access token 调用方
	ClientToken = "token"
	// ClientUnknown 认证失败且非 CLI
	ClientUnknown = "unknown"
)

// maxUserAgentLen UA 最大长度
const maxUserAgentLen = 512

// clientType 按认证凭据区分非 CLI 客户端：cookie 为 web，access token 为 token。
func clientType(cred auth.UserCredential) string {
	switch cred.CredKey() {
	case auth.CredKeyBkTicket, auth.CredKeyBkToken:
		return ClientWeb
	case auth.CredKeyAccessToken:
		return ClientToken
	default:
		return ClientUnknown
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
