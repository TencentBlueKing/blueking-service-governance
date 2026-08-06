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

package env

import (
	"github.com/gin-gonic/gin"
)

// EnvHandler contains views required by env Gin routes.
type EnvHandler interface {
	CreateEnv(c *gin.Context)
	CreateFeatureEnv(c *gin.Context)
	ListEnvs(c *gin.Context)
	ListAppEnvs(c *gin.Context)
	ListFeatureEnvs(c *gin.Context)
	GetEnv(c *gin.Context)
	UpdateEnvBasicInfo(c *gin.Context)
	UpdateEnvCluster(c *gin.Context)
	DeleteEnv(c *gin.Context)
	ListEnvTrafficLanes(c *gin.Context)
}

// Register registers Gin env routes.
func Register(rg *gin.RouterGroup, h EnvHandler) {
	// 创建部署环境
	rg.POST("/workspaces/:workspaceID/envs", h.CreateEnv)
	// 获取空间下的环境列表
	rg.GET("/workspaces/:workspaceID/envs", h.ListEnvs)
	// 创建应用特性环境
	rg.POST("/apps/:appID/feat-envs", h.CreateFeatureEnv)
	// 获取应用特性环境管理列表
	rg.GET("/apps/:appID/feat-envs", h.ListFeatureEnvs)
	// 获取应用可用环境列表
	rg.GET("/apps/:appID/envs", h.ListAppEnvs)
	// 获取单个环境详情
	rg.GET("/envs/:envID", h.GetEnv)
	// 更新部署环境基本信息
	rg.PUT("/envs/:envID/basic-info", h.UpdateEnvBasicInfo)
	// 更新部署环境集群配置
	rg.PUT("/envs/:envID/cluster", h.UpdateEnvCluster)
	// 删除环境
	rg.DELETE("/envs/:envID", h.DeleteEnv)
	// 获取指定环境下的泳道列表
	rg.GET("/workspaces/:workspaceID/envs/:envName/traffic-lanes", h.ListEnvTrafficLanes)
}
