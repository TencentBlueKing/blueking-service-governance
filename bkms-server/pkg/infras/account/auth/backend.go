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

// Package auth 定义了认证所使用的后端接口类型，可支持 bk_ticket、bk_token 等多种认证方式
package auth

import (
	"context"
	"net/http"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth/backends"
)

// AuthBackend 是用户认证接口
type AuthBackend interface {
	// GetLoginUrl 方法获取登录地址
	GetLoginUrl() string

	// GetUserCredential 方法尝试从当前请求中获取用户票据，返回票据内容
	GetUserCredential(request *http.Request) string

	// GetUserInfo 方法通过票据获取用户信息
	GetUserInfo(ctx context.Context, userCred string) (*backends.UserInfo, error)
}

// getBackend 根据配置创建认证后端实例。
func getBackend(cfg Config) (AuthBackend, string) {
	switch cfg.BackendType {
	case BackendBkTicket:
		return backends.NewBkTicketAuthBackend(cfg.LoginURL), BackendBkTicket
	case BackendBkToken:
		return backends.NewBkTokenAuthBackend(cfg.LoginURL), BackendBkToken
	default:
		// Default to bk_token
		return backends.NewBkTokenAuthBackend(cfg.LoginURL), BackendBkToken
	}
}
