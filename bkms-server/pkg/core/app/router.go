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

package app

import "github.com/gin-gonic/gin"

// AppHandler contains views required by app Gin routes.
type AppHandler interface {
	// 应用管理
	// 创建应用
	CreateApp(c *gin.Context)
	// 获取创建应用时使用的自动 ID 后缀
	GetAppIDAutoSuffix(c *gin.Context)
	// 查询单个应用详情
	GetApp(c *gin.Context)
	// 查询 app 列表，只返回基本信息
	ListApps(c *gin.Context)
	// 删除单个应用
	DeleteApp(c *gin.Context)
	// 更新应用显示名称
	UpdateAppDisplayName(c *gin.Context)
	// 更新应用 Helm Chart 配置
	UpdateHelmSpec(c *gin.Context)
	// 更新应用 Trpc 配置
	UpdateAppTrpcSpec(c *gin.Context)
	// 更新应用 Taf 配置
	UpdateAppTafSpec(c *gin.Context)

	// 应用组件实例管理
	// 添加应用组件
	CreateAppComponent(c *gin.Context)
	// 更新应用组件
	PatchAppComponent(c *gin.Context)
	// 删除应用组件
	DeleteAppComponent(c *gin.Context)
	// 应用部署状态
	GetAppDeployStatuses(c *gin.Context)
}

// Register registers Gin app routes.
func Register(rg *gin.RouterGroup, h AppHandler) {
	// 应用管理
	// 创建应用
	rg.POST("/workspaces/:workspaceID/apps", h.CreateApp)
	// 获取创建应用时使用的自动 ID 后缀
	rg.GET("/apps/auto-id-suffix", h.GetAppIDAutoSuffix)
	// 查询单个应用详情
	rg.GET("/apps/:appID", h.GetApp)
	// 查询 app 列表，只返回基本信息
	rg.GET("/workspaces/:workspaceID/apps", h.ListApps)
	// 删除单个应用
	rg.DELETE("/apps/:appID", h.DeleteApp)
	// 更新应用显示名称
	rg.PUT("/apps/:appID/display-name", h.UpdateAppDisplayName)
	// 更新应用 Helm Chart 配置
	rg.PUT("/apps/:appID/helm-spec", h.UpdateHelmSpec)
	// 更新应用 Trpc 配置
	rg.PUT("/apps/:appID/trpc-spec", h.UpdateAppTrpcSpec)
	// 更新应用 Taf 配置
	rg.PUT("/apps/:appID/taf-spec", h.UpdateAppTafSpec)

	// 应用组件实例管理
	// 添加应用组件
	rg.POST("/apps/:appID/components", h.CreateAppComponent)
	// 更新应用组件
	rg.PATCH("/apps/:appID/components/:compName", h.PatchAppComponent)
	// 删除应用组件
	rg.DELETE("/apps/:appID/components/:compName", h.DeleteAppComponent)
	// 查询应用在各环境及各泳道上的部署状态
	rg.GET("/apps/:appID/deploy-statuses", h.GetAppDeployStatuses)
}
