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

package frameworktmpl

import "github.com/gin-gonic/gin"

// Handler is the interface for framework config template API handlers.
type Handler interface {
	ListTemplates(c *gin.Context)
	GetTemplate(c *gin.Context)
	CreateTemplate(c *gin.Context)
	UpdateTemplate(c *gin.Context)
	DeleteTemplate(c *gin.Context)
	SetDefaultTemplate(c *gin.Context)
}

// Register registers workspace framework config template routes.
func Register(rg *gin.RouterGroup, h Handler) {
	rg.GET("/workspaces/:workspaceID/framework-templates", h.ListTemplates)
	rg.POST("/workspaces/:workspaceID/framework-templates", h.CreateTemplate)
	rg.GET("/workspaces/:workspaceID/framework-templates/:templateID", h.GetTemplate)
	rg.PUT("/workspaces/:workspaceID/framework-templates/:templateID", h.UpdateTemplate)
	rg.DELETE("/workspaces/:workspaceID/framework-templates/:templateID", h.DeleteTemplate)
	rg.PUT("/workspaces/:workspaceID/framework-templates/:templateID/default", h.SetDefaultTemplate)
}
