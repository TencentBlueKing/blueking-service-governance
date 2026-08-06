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

// Package basic 提供基础接口（Ping、Version）的 Gin API 实现。
package basic

import (
	"github.com/gin-gonic/gin"
)

// BasicHandler 包含 basic 路由所需的视图方法。
type BasicHandler interface {
	// Ping 联通性测试接口
	Ping(c *gin.Context)
	// Version 提供服务版本信息
	Version(c *gin.Context)
}

// Register 注册 Gin basic 路由。
// 这些接口不需要鉴权，因此不使用传入的 RouterGroup 上的鉴权中间件。
func Register(rg *gin.RouterGroup, h BasicHandler) {
	// 联通性测试接口
	rg.GET("/ping", h.Ping)
	// 服务版本信息接口
	rg.GET("/version", h.Version)
}
