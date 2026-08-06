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

// Package account provides BlueKing account and user-token APIs.
package account

import "github.com/gin-gonic/gin"

// Handler defines the account Gin API handlers.
type Handler interface {
	CreateToken(c *gin.Context)
	RefreshToken(c *gin.Context)
	ValidateToken(c *gin.Context)
	GetCurrentUser(c *gin.Context)
}

// Register registers account routes.
func Register(rg *gin.RouterGroup, h Handler, optionalAuth gin.HandlerFunc) {
	// These paths intentionally retain historical URL compatibility.
	// They may move under /account after clients no longer depend on the old URLs.
	//
	// 签发、刷新以及验证用户 Token 的 API
	rg.GET("/user_token/token", h.CreateToken)
	rg.GET("/user_token/refresh", h.RefreshToken)
	rg.GET("/user_token/validate", h.ValidateToken)

	// 简单的当前用户信息的 API
	rg.GET("/simple_account/info", optionalAuth, h.GetCurrentUser)
}
