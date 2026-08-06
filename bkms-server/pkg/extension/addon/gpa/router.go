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

package gpa

import "github.com/gin-gonic/gin"

// GPAConfigHandler contains views required by GPA config Gin routes.
type GPAConfigHandler interface {
	GetAppEnvGPAConfig(c *gin.Context)
	UpsertAppEnvGPAConfig(c *gin.Context)
	DeleteAppEnvGPAConfig(c *gin.Context)
	ToggleAppEnvGPAConfig(c *gin.Context)
}

// Register registers Gin GPA config routes.
// 配置在应用 + 环境维度唯一，URL 不携带配置名称。
// 对外路由以 autoscaler 暴露（屏蔽底层 GPA 实现细节）。
func Register(rg *gin.RouterGroup, h GPAConfigHandler) {
	// 查询该应用在该环境的 GPA 配置（含 K8s 运行状态）
	rg.GET("/apps/:appID/envs/:envName/autoscaler", h.GetAppEnvGPAConfig)
	// 创建或更新该应用在该环境的 GPA 配置并下发 CRD（upsert 语义）
	rg.PUT("/apps/:appID/envs/:envName/autoscaler", h.UpsertAppEnvGPAConfig)
	// 删除该应用在该环境的 GPA 配置并清理 CRD
	rg.DELETE("/apps/:appID/envs/:envName/autoscaler", h.DeleteAppEnvGPAConfig)
	// 开关该应用在该环境的 GPA（关闭时删除 CR，开启时按 DB 配置重新下发）
	rg.PATCH("/apps/:appID/envs/:envName/autoscaler/toggle", h.ToggleAppEnvGPAConfig)
}
