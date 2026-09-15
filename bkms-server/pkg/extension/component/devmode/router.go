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

package devmode

import "github.com/gin-gonic/gin"

// DevModeHandler contains views required by devmode Gin routes.
type DevModeHandler interface {
	DevModePublishPreflight(c *gin.Context)
	CreateDevModePublishRecords(c *gin.Context)
	ListDevModePublishRecords(c *gin.Context)
}

// RegisterRoutes registers Gin devmode routes.
func RegisterRoutes(rg *gin.RouterGroup, h DevModeHandler) {
	// 开发模式 Publish 预检
	rg.POST("/devmode/:appID/envs/:envName/preflight", h.DevModePublishPreflight)
	// 上报开发模式发布结果
	rg.POST("/devmode/:appID/envs/:envName/publish-records", h.CreateDevModePublishRecords)
	// 查询开发模式发布记录列表
	rg.GET("/devmode/:appID/envs/:envName/publish-records", h.ListDevModePublishRecords)
}
