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

package appcfg

import "github.com/gin-gonic/gin"

// Handler contains views required by app config file Gin routes.
type Handler interface {
	CreateAppConfigFile(c *gin.Context)
	UpdateAppConfigFile(c *gin.Context)
	ListAppConfigFiles(c *gin.Context)
	DeleteAppConfigFile(c *gin.Context)
	GetAppConfigFileDetails(c *gin.Context)
	UpdateAppConfigFileContent(c *gin.Context)
	UpdateAppConfigFileOverlayContent(c *gin.Context)
	PreviewOverlayMerge(c *gin.Context)
	ListAppConfigFileVersions(c *gin.Context)
	GetAppConfigFileVersion(c *gin.Context)
	CompareAppConfigFileVersions(c *gin.Context)
	RollbackAppConfigFileVersion(c *gin.Context)
	DeleteAppConfigFileVersion(c *gin.Context)
}

// Register registers Gin app config file routes.
// ⚠️ 以下路由为存量接口，仅处理 framework 类型文件。新功能（含 plain 文件）
// 请使用 filedef.RegisterRoutes 注册的 /app-config-file-defs 路由。
func Register(rg *gin.RouterGroup, h Handler) {
	apps := rg.Group("/apps/:appID")

	// App config file resource routes.
	// ⚠️ 存量路由组：仅操作 framework 文件，不含 plain。
	files := apps.Group("/app-config-files")
	// 创建 framework 配置文件。新文件请使用 POST /app-config-file-defs。
	files.POST("", h.CreateAppConfigFile)
	// 更新 framework 文件属性。新文件请使用 PUT /app-config-file-defs/:id。
	files.PUT("/:id", h.UpdateAppConfigFile)
	// 列出 framework 配置文件（已自动过滤 plain）。完整列表请使用 GET /app-config-file-defs。
	files.GET("", h.ListAppConfigFiles)
	// 删除 framework 文件（禁止删除 plain 环境实例）。
	files.DELETE("/:id", h.DeleteAppConfigFile)
	// 查看 framework 文件详情。def 视角详情请使用 GET /app-config-file-defs/:id。
	files.GET("/:id/details", h.GetAppConfigFileDetails)
	// 更新 framework normal 文件内容。新接口请使用 PUT /app-config-file-defs/:id/content。
	files.PUT("/:id/content", h.UpdateAppConfigFileContent)
	// 更新 framework overlay 文件内容。新接口请使用 PUT /app-config-file-defs/:id/content。
	files.PUT("/:id/overlay-content", h.UpdateAppConfigFileOverlayContent)
	// 预览 framework overlay 合并结果。挂载预览请使用 GET /mount-preview?envName=。
	files.POST("/:id/preview-overlay-merge", h.PreviewOverlayMerge)

	// App config file version routes. The `:id` path param here is a version record ID.
	// ⚠️ 版本管理接口通用于所有 ConfigKind，暂不迁移。后续新版本路由可根据需要调整。
	versions := apps.Group("/app-config-file/versions")
	// 列出版本记录。
	versions.GET("", h.ListAppConfigFileVersions)
	// 查看单条版本记录。
	versions.GET("/:id", h.GetAppConfigFileVersion)
	// 比较两条版本记录。
	versions.POST("/compare", h.CompareAppConfigFileVersions)
	// 回滚到指定版本。
	versions.POST("/:id/rollback", h.RollbackAppConfigFileVersion)
	// 删除历史版本记录。
	versions.DELETE("/:id", h.DeleteAppConfigFileVersion)
}
