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

package networking

import (
	"github.com/gin-gonic/gin"
)

// Handler contains views required by networking Gin routes.
type Handler interface {
	CreateAppService(c *gin.Context)
	ListAppServices(c *gin.Context)
	DeleteAppService(c *gin.Context)
	UpdateAppService(c *gin.Context)
	ListTrafficLaneCandidateApps(c *gin.Context)
}

// Register registers Gin networking routes.
func Register(rg *gin.RouterGroup, h Handler) {
	// 创建应用下的 Service
	rg.POST("/apps/:appID/services", h.CreateAppService)
	// 获取应用下的 Services
	rg.GET("/apps/:appID/services", h.ListAppServices)
	// 删除应用下的 Service
	rg.DELETE("/apps/:appID/services/:name", h.DeleteAppService)
	// 更新应用下的 Service
	rg.PUT("/apps/:appID/services/:name", h.UpdateAppService)
	// 查询空间下的候选应用列表(用于泳道关联)
	rg.GET("/workspaces/:workspaceID/traffic-lanes/candidate-apps", h.ListTrafficLaneCandidateApps)
}
