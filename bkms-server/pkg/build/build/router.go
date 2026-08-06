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

package build

import "github.com/gin-gonic/gin"

// Handler contains views required by build Gin routes.
type Handler interface {
	UpdateBuildConfig(c *gin.Context)
	ListBuildRecords(c *gin.Context)
	CreateBuild(c *gin.Context)
	GetRecommendedImageTag(c *gin.Context)
	StreamBuildLogs(c *gin.Context)
	DownloadBuildLogs(c *gin.Context)
}

// Register registers Gin build routes.
func Register(rg *gin.RouterGroup, h Handler) {
	apps := rg.Group("/apps/:appID")
	apps.PUT("/build-configs", h.UpdateBuildConfig)
	apps.GET("/builds", h.ListBuildRecords)
	apps.POST("/builds", h.CreateBuild)
	apps.GET("/recommended-image-tag", h.GetRecommendedImageTag)

	// build logs
	apps.GET("/builds/:buildID/logs/stream", h.StreamBuildLogs)
	apps.GET("/builds/:buildID/logs/download", h.DownloadBuildLogs)
}
