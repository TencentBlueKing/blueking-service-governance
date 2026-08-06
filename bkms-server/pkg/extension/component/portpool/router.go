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

package portpool

import "github.com/gin-gonic/gin"

// PortPoolHandler contains views required by port-pool Gin routes.
type PortPoolHandler interface {
	ListPortPools(c *gin.Context)
	CreatePortPool(c *gin.Context)
	UpdatePortPool(c *gin.Context)
	DeletePortPool(c *gin.Context)
}

// Register registers Gin port-pool routes.
func Register(rg *gin.RouterGroup, h PortPoolHandler) {
	// 获取端口池列表
	rg.GET("/envs/:envID/port-pools", h.ListPortPools)
	// 创建端口池
	rg.POST("/envs/:envID/port-pools", h.CreatePortPool)
	// 更新端口池
	rg.PUT("/envs/:envID/port-pools/:name", h.UpdatePortPool)
	// 删除端口池
	rg.DELETE("/envs/:envID/port-pools/:name", h.DeletePortPool)
}
