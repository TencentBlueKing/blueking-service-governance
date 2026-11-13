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

package frameworkcfgtmpl

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

// Register 注册工作空间级别的框架配置模板相关路由。
func Register(rg *gin.RouterGroup, h Handler) {
	tmpl := rg.Group("/workspaces/:workspaceID/framework-config-templates")
	// 获取当前工作空间下的框架配置模板列表
	tmpl.GET("", h.ListTemplates)
	// 在当前工作空间下创建新的框架配置模板
	tmpl.POST("", h.CreateTemplate)
	// 根据模板 ID 获取单个框架配置模板详情
	tmpl.GET("/:templateID", h.GetTemplate)
	// 根据模板 ID 更新指定的框架配置模板
	tmpl.PUT("/:templateID", h.UpdateTemplate)
	// 根据模板 ID 删除指定的框架配置模板
	tmpl.DELETE("/:templateID", h.DeleteTemplate)
	// 将指定模板设为当前工作空间的默认框架配置模板
	tmpl.PUT("/:templateID/default", h.SetDefaultTemplate)
}
