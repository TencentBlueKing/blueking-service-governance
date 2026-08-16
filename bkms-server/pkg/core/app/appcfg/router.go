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
	UpdateAppConfigFileEnvConfig(c *gin.Context)
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
func Register(rg *gin.RouterGroup, h Handler) {
	apps := rg.Group("/apps/:appID")

	// 应用配置文件资源路由。
	files := apps.Group("/app-config-files")
	// 创建应用配置文件。
	files.POST("", h.CreateAppConfigFile)
	// 更新应用配置文件元数据，例如名称、base 引用等。
	files.PUT("/:id", h.UpdateAppConfigFile)
	// 切换统一配置/按环境配置模式及挂载环境范围。
	files.PUT("/:id/env-config-policy", h.UpdateAppConfigFileEnvConfig)
	// 列出当前应用下的应用配置文件。
	files.GET("", h.ListAppConfigFiles)
	// 按 ID 删除应用配置文件。
	files.DELETE("/:id", h.DeleteAppConfigFile)
	// 查询详情、可编辑字段信息与当前内容。
	files.GET("/:id/details", h.GetAppConfigFileDetails)
	// Update the content field for a normal app config file.
	files.PUT("/:id/content", h.UpdateAppConfigFileContent)
	// Update the overlayContent field for an overlay app config file.
	files.PUT("/:id/overlay-content", h.UpdateAppConfigFileOverlayContent)
	// Preview the merged result between base content and overlay content.
	files.POST("/:id/preview-overlay-merge", h.PreviewOverlayMerge)

	// App config file version routes. The `:id` path param here is a version record ID.
	versions := apps.Group("/app-config-file/versions")
	// List historical version records.
	versions.GET("", h.ListAppConfigFileVersions)
	// Query a single version record by version ID.
	versions.GET("/:id", h.GetAppConfigFileVersion)
	// Compare two version records of the same app config file.
	versions.POST("/compare", h.CompareAppConfigFileVersions)
	// Roll back the current app config file to the target version record.
	versions.POST("/:id/rollback", h.RollbackAppConfigFileVersion)
	// Delete a historical version record by version ID.
	versions.DELETE("/:id", h.DeleteAppConfigFileVersion)
}
