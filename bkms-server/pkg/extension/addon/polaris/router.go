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

package polaris

import "github.com/gin-gonic/gin"

// PolarisConfigHandler contains views required by polaris-config Gin routes.
type PolarisConfigHandler interface {
	ListAppPolarisConfigs(c *gin.Context)
	CreateAppPolarisConfig(c *gin.Context)
	PatchAppPolarisConfig(c *gin.Context)
	DeleteAppPolarisConfig(c *gin.Context)
	ListAppPolarisConfigVars(c *gin.Context)
	ValidateAppPolarisConfig(c *gin.Context)
}

// Register registers Gin polaris-config routes.
func Register(rg *gin.RouterGroup, h PolarisConfigHandler) {
	// 获取应用的北极星配置列表
	rg.GET("/apps/:appID/deps/polaris-configs", h.ListAppPolarisConfigs)
	// 创建北极星配置
	rg.POST("/apps/:appID/deps/polaris-configs", h.CreateAppPolarisConfig)
	// 更新北极星配置
	rg.PATCH("/apps/:appID/deps/polaris-configs/:configName", h.PatchAppPolarisConfig)
	// 删除北极星配置
	rg.DELETE("/apps/:appID/deps/polaris-configs/:configName", h.DeleteAppPolarisConfig)

	// 获取北极星配置变量列表
	rg.GET("/apps/:appID/deps/polaris-configs/:configName/vars", h.ListAppPolarisConfigVars)
	// 校验北极星配置（创建前预校验）
	rg.POST("/apps/:appID/deps/polaris-configs/validate", h.ValidateAppPolarisConfig)
}
