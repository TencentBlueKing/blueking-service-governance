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

package depservice

import "github.com/gin-gonic/gin"

// APIHandler contains views required by dependency-service Gin routes.
type APIHandler interface {
	CreateRedisInstance(c *gin.Context)
	ListRedisInstances(c *gin.Context)
	GetRedisInstance(c *gin.Context)
	DeleteRedisInstance(c *gin.Context)
	CreateBinding(c *gin.Context)
	ListBindings(c *gin.Context)
	GetBinding(c *gin.Context)
	UpdateBinding(c *gin.Context)
	DeleteBinding(c *gin.Context)
}

// Register registers Gin dependency-service routes.
func Register(rg *gin.RouterGroup, h APIHandler) {
	rg.POST("/workspaces/:workspaceID/deps/redis", h.CreateRedisInstance)
	rg.GET("/workspaces/:workspaceID/deps/redis", h.ListRedisInstances)
	rg.GET("/workspaces/:workspaceID/deps/redis/:instanceID", h.GetRedisInstance)
	rg.DELETE("/workspaces/:workspaceID/deps/redis/:instanceID", h.DeleteRedisInstance)

	rg.POST("/apps/:appID/deps/:serviceName/bindings", h.CreateBinding)
	rg.GET("/apps/:appID/deps/:serviceName/bindings", h.ListBindings)
	rg.GET("/apps/:appID/deps/:serviceName/bindings/:name", h.GetBinding)
	rg.PUT("/apps/:appID/deps/:serviceName/bindings/:name", h.UpdateBinding)
	rg.DELETE("/apps/:appID/deps/:serviceName/bindings/:name", h.DeleteBinding)
}
