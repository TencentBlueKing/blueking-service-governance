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

package auth

const (
	// BKAuthKey for http header or trpc metadata
	BKAuthKey = "X-Bk-Authed-User-Info"
	// BKCredentialKey for http header or trpc metadata
	BKCredentialKey = "X-Bk-Authed-User-Credential" //nolint:gosec // G101: header name, not a credential

	// CredKeyAccessToken 认证凭据类型：access_token
	CredKeyAccessToken = "access_token"
	// CredKeyBkTicket 认证凭据类型：bk_ticket
	CredKeyBkTicket = "bk_ticket"
	// CredKeyBkToken 认证凭据类型：bk_token
	CredKeyBkToken = "bk_token"
)

// RequestUser 表示请求上下文中的用户身份。
type RequestUser interface {
	GetID() string
	Credential() UserCredential
	IsAuthenticated() bool
}

// User 为完成认证的用户信息。
type User struct {
	// ID 是用户 ID。
	ID string `json:"id"`

	// Cred 是用户认证凭据。
	Cred UserCredential `json:"credential,omitempty"`
}

// GetID 返回用户 ID。
func (u User) GetID() string { return u.ID }

// Credential 返回用户认证凭据。
func (u User) Credential() UserCredential { return u.Cred }

// IsAuthenticated 对 User 恒为 true。
func (u User) IsAuthenticated() bool { return true }

// AnonymousUser 表示未认证的匿名用户。
type AnonymousUser struct {
	Cred UserCredential `json:"credential,omitempty"`
}

// GetID 返回空用户 ID。
func (u AnonymousUser) GetID() string { return "" }

// Credential 返回已捕获但尚未通过认证的凭据。
func (u AnonymousUser) Credential() UserCredential { return u.Cred }

// IsAuthenticated 对 AnonymousUser 恒为 false。
func (u AnonymousUser) IsAuthenticated() bool { return false }

// UserCredential 是用来认证用户的凭证数据，一般情况下，CLI 等工具主要用 AccessToken 字段
// 进行认证，Web 端则使用 bk_ticket 或 bk_token 字段（视平台配置而定）。长期凭证 AccessToken 与短
// 期凭证 bk_ticket/bk_token 不会同时存在。
type UserCredential struct {
	// AccessToken 是用户的访问令牌，有效时间较长，以天为单位。
	AccessToken string `json:"accessToken,omitempty"`

	// BkTicket 是一种短期用户凭证，可用作用户认证凭证，有效时间以小时为单位。
	BkTicket string `json:"bkTicket,omitempty"`

	// BkToken 是一种短期用户凭证，和 BkTicket 功能类似。
	BkToken string `json:"bkToken,omitempty"`

	// UnknownValues 用来存在当前未知的凭证数据，大部分情况下本字段不会被使用，仅在特殊情况比如单元
	// 测试时可能会被用到。
	UnknownValues []string `json:"unknownValues,omitempty"`
}

// IsEmpty 判断 UserCredential 是否为空。
func (c UserCredential) IsEmpty() bool {
	return c.AccessToken == "" && c.BkTicket == "" && c.BkToken == "" && len(c.UnknownValues) == 0
}

// CredKey 返回当前有效凭据的 key。
func (c UserCredential) CredKey() string {
	switch {
	case c.AccessToken != "":
		return CredKeyAccessToken
	case c.BkTicket != "":
		return CredKeyBkTicket
	case c.BkToken != "":
		return CredKeyBkToken
	default:
		return ""
	}
}

// CredValue 返回当前有效凭据的值。
func (c UserCredential) CredValue() string {
	switch {
	case c.AccessToken != "":
		return c.AccessToken
	case c.BkTicket != "":
		return c.BkTicket
	case c.BkToken != "":
		return c.BkToken
	default:
		return ""
	}
}
