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

package appcfgfiledef

import "github.com/gin-gonic/gin"

// RegisterRoutes 注册新版配置文件定义（file definition）视角的路由。
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	apps := rg.Group("/apps/:appID")

	// 挂载预览（应用级）
	apps.GET("/mount-preview", h.GetMountPreview)

	defs := apps.Group("/app-config-file-defs")

	// 列出应用下所有配置文件定义及其默认文件内容
	defs.GET("/defaults", h.ListDefaultFilesWithDef)
	// 创建配置文件定义 + 默认文件
	defs.POST("", h.CreateAppConfigFileDef)
	// 获取配置文件定义详情
	defs.GET("/:id", h.GetAppCfgFileDetail)
	// 更新配置文件定义信息
	defs.PUT("/:id", h.UpdateAppCfgFileDef)
	// 删除配置文件定义及其所有关联文件和版本
	defs.DELETE("/:id", h.DeleteAppCfgFileDef)
	// 列出配置文件定义下的所有环境实例
	defs.GET("/:id/env-instances", h.ListEnvInstances)
	// 更新配置内容
	defs.PUT("/:id/content", h.UpdateContent)
	// 恢复环境到默认，只允许对 plain 类型的环境实例使用
	defs.DELETE("/:id/envs/:envName", h.ResetEnvToDefault)
}
