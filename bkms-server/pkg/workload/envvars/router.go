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

package envvars

import (
	"github.com/gin-gonic/gin"
)

// ScopedEnvVarHandler contains views required by envvars Gin routes.
type ScopedEnvVarHandler interface {
	CreateScopedEnvVar(c *gin.Context)
	UpdateScopedEnvVar(c *gin.Context)
	DeleteScopedEnvVar(c *gin.Context)
	ListPublicScopedEnvVars(c *gin.Context)
	PreviewPublicScopedEnvVar(c *gin.Context)
	ImportPublicScopedEnvVar(c *gin.Context)
	ExportPublicScopedEnvVars(c *gin.Context)
	ListDetailedEnvScopedEnvVars(c *gin.Context)
	PreviewEnvScopedEnvVar(c *gin.Context)
	ImportEnvScopedEnvVar(c *gin.Context)
	ExportEnvScopedEnvVars(c *gin.Context)
	ListDetailedAppEnvVars(c *gin.Context)
	PreviewAppDefinedEnvVar(c *gin.Context)
	ImportAppDefinedEnvVar(c *gin.Context)
	ExportAppEnvVars(c *gin.Context)
	ListEnvAvailableEnvVars(c *gin.Context)
	ListAppDefinedEnvVars(c *gin.Context)
	CreateAppDefinedEnvVar(c *gin.Context)
	UpdateAppDefinedEnvVar(c *gin.Context)
	DeleteAppDefinedEnvVar(c *gin.Context)
	ListAppEnvVars(c *gin.Context)
	ListEnvBgEnvVars(c *gin.Context)
	ListAppBgEnvVars(c *gin.Context)
}

// Register registers Gin envvars routes.
func Register(rg *gin.RouterGroup, h ScopedEnvVarHandler) {
	// 创建作用域级别的环境变量（ScopedEnvVar）
	rg.POST("/workspaces/:workspaceID/scoped-env-vars", h.CreateScopedEnvVar)
	// 更新作用域级别的环境变量（ScopedEnvVar）
	rg.PUT("/workspaces/:workspaceID/scoped-env-vars/:scopedEnvVarID", h.UpdateScopedEnvVar)
	// 删除作用域级别的环境变量（ScopedEnvVar）
	rg.DELETE("/workspaces/:workspaceID/scoped-env-vars/:scopedEnvVarID", h.DeleteScopedEnvVar)
	// 获取指定空间下的公共环境变量列表
	rg.GET("/workspaces/:workspaceID/scoped-env-vars/public-vars", h.ListPublicScopedEnvVars)
	// 预览导入公共环境变量
	rg.POST("/workspaces/:workspaceID/scoped-env-vars/public-vars/preview", h.PreviewPublicScopedEnvVar)
	// 正式导入公共环境变量
	rg.POST("/workspaces/:workspaceID/scoped-env-vars/public-vars/import", h.ImportPublicScopedEnvVar)
	// 导出公共环境变量
	rg.GET("/workspaces/:workspaceID/scoped-env-vars/public-vars/export", h.ExportPublicScopedEnvVars)

	// 获取指定环境下作用域为当前环境的环境变量详情
	rg.GET("/scoped-env-vars/detailed-list/:envID", h.ListDetailedEnvScopedEnvVars)
	// 预览导入单环境环境变量
	rg.POST("/scoped-env-vars/preview/:envID", h.PreviewEnvScopedEnvVar)
	// 正式导入单环境环境变量
	rg.POST("/scoped-env-vars/import/:envID", h.ImportEnvScopedEnvVar)
	// 导出单环境环境变量
	rg.GET("/scoped-env-vars/export/:envID", h.ExportEnvScopedEnvVars)

	// 获取指定应用的环境变量详情，包含可能的 Key 冲突信息
	rg.GET("/apps/:appID/env-vars/detailed-list", h.ListDetailedAppEnvVars)
	// 预览导入应用直接定义的环境变量
	rg.POST("/apps/:appID/env-vars/preview", h.PreviewAppDefinedEnvVar)
	// 正式导入应用直接定义的环境变量
	rg.POST("/apps/:appID/env-vars/import", h.ImportAppDefinedEnvVar)
	// 导出应用环境变量
	rg.GET("/apps/:appID/env-vars/export", h.ExportAppEnvVars)
	// 获取应用直接定义的环境变量列表
	rg.GET("/apps/:appID/env-vars", h.ListAppDefinedEnvVars)
	// 创建应用直接定义的环境变量
	rg.POST("/apps/:appID/env-vars", h.CreateAppDefinedEnvVar)
	// 更新应用直接定义的环境变量
	rg.PUT("/apps/:appID/env-vars/:key", h.UpdateAppDefinedEnvVar)
	// 删除应用直接定义的环境变量
	rg.DELETE("/apps/:appID/env-vars/:key", h.DeleteAppDefinedEnvVar)
	// 查询指定环境下所有可用的环境变量列表
	rg.GET("/envs/:envID/available-env-vars", h.ListEnvAvailableEnvVars)
	// 获取应用部署到某个环境后可用的环境变量
	rg.GET("/apps/:appID/envs/:envName/env-variables", h.ListAppEnvVars)
	// 查询指定环境的背景环境变量列表
	rg.GET("/envs/:envID/bg-env-vars", h.ListEnvBgEnvVars)
	// 查询应用在某个环境下的背景环境变量列表
	rg.GET("/apps/:appID/envs/:envName/bg-env-vars", h.ListAppBgEnvVars)
}
