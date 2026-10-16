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

package releasestrategy

import "github.com/gin-gonic/gin"

// Handler contains views required by workspace release strategy routes.
type Handler interface {
	ListReleaseStrategies(c *gin.Context)
	GetReleaseStrategy(c *gin.Context)
	CreateReleaseStrategy(c *gin.Context)
	UpdateReleaseStrategy(c *gin.Context)
	DeleteReleaseStrategy(c *gin.Context)
}

// Register registers workspace release strategy routes.
func Register(rg *gin.RouterGroup, h Handler) {
	ws := rg.Group("/workspaces/:workspaceID")
	ws.GET("/release-strategies", h.ListReleaseStrategies)
	ws.POST("/release-strategies", h.CreateReleaseStrategy)
	ws.GET("/release-strategies/:strategyID", h.GetReleaseStrategy)
	ws.PUT("/release-strategies/:strategyID", h.UpdateReleaseStrategy)
	ws.DELETE("/release-strategies/:strategyID", h.DeleteReleaseStrategy)
}
