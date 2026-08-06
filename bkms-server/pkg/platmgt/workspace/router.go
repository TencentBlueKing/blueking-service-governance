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

// Package workspace provides platform workspace query capabilities for platform administrators.
package workspace

import "github.com/gin-gonic/gin"

// Handler contains views required by platform workspace Gin routes.
type Handler interface {
	ListPlatWorkspaces(c *gin.Context)
	GetPlatWorkspaceStats(c *gin.Context)
	GetPlatWorkspace(c *gin.Context)
}

// Register registers Gin platform workspace routes.
func Register(rg *gin.RouterGroup, h Handler, middleware gin.HandlerFunc) {
	group := rg.Group("/plat-mgt/workspaces")
	group.Use(middleware)
	group.GET("", h.ListPlatWorkspaces)
	group.GET("/statistics", h.GetPlatWorkspaceStats)
	group.GET("/:workspaceID", h.GetPlatWorkspace)
}
