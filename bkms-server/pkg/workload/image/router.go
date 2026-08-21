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

package image

import "github.com/gin-gonic/gin"

// Handler contains views required by image Gin routes.
type Handler interface {
	ListAppImages(c *gin.Context)
	RefreshAppImages(c *gin.Context)
	PromoteAppImage(c *gin.Context)
	ListAppImageUsages(c *gin.Context)
	DeleteAppImage(c *gin.Context)
	ListImageTagDeployRecords(c *gin.Context)
	ListDeployableImageTags(c *gin.Context)
	ListPlatformBuildImages(c *gin.Context)
	ListPlatformBuildImageTags(c *gin.Context)
	ListCustomBuildImages(c *gin.Context)
	ListCustomBuildImageTags(c *gin.Context)
	RefreshCustomBuildImageTags(c *gin.Context)
}

// Register registers Gin image routes.
func Register(rg *gin.RouterGroup, h Handler) {
	rg.GET("/platform-build-images", h.ListPlatformBuildImages)
	rg.GET("/platform-build-images/:imageID/tags", h.ListPlatformBuildImageTags)

	// 自定义构建镜像归属工作空间而非应用，故挂在 workspace 维度；
	// TAG 查询与刷新以完整镜像名称作为参数，不使用记录 ID 作为路径参数
	customImages := rg.Group("/workspaces/:workspaceID/custom-build-images")
	customImages.GET("", h.ListCustomBuildImages)
	customImages.GET("/tags", h.ListCustomBuildImageTags)
	customImages.POST("/tags/refresh", h.RefreshCustomBuildImageTags)

	apps := rg.Group("/apps/:appID")
	apps.GET("/images", h.ListAppImages)
	apps.POST("/images/refresh", h.RefreshAppImages)
	apps.PATCH("/images/:tag/promote", h.PromoteAppImage)
	apps.GET("/images/:tag/usages", h.ListAppImageUsages)
	apps.DELETE("/images/:tag", h.DeleteAppImage)
	apps.GET("/images/:tag/deploy-records", h.ListImageTagDeployRecords)
	apps.GET("/envs/:envName/deployable-image-tags", h.ListDeployableImageTags)
}
