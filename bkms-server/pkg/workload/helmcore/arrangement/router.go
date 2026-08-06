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

// Package arrangement 定义应用编排相关 Gin v2 API 路由。
package arrangement

import "github.com/gin-gonic/gin"

// PlaceholderVarHandler 包含占位符变量路由所需的视图方法。
type PlaceholderVarHandler interface {
	// ListPlaceholderVars 获取应用占位符变量列表。
	ListPlaceholderVars(c *gin.Context)
}

// Register 注册 Gin arrangement 路由。
func Register(rg *gin.RouterGroup, h PlaceholderVarHandler) {
	// 获取编排可用的应用占位符变量列表
	rg.GET("/placeholder-vars", h.ListPlaceholderVars)
}
