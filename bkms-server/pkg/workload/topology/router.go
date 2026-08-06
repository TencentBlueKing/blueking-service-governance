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

package topology

import "github.com/gin-gonic/gin"

// Handler contains views required by topology Gin routes.
type Handler interface {
	GetResourceTopology(c *gin.Context)
	GetTopologyNodeDetail(c *gin.Context)
	ListTopologyNodeEvents(c *gin.Context)
	GetTopologyNodeManifest(c *gin.Context)
}

// Register registers Gin topology routes.
func Register(rg *gin.RouterGroup, h Handler) {
	rg.GET("/apps/:appID/envs/:envName/resource-topology", h.GetResourceTopology)
	rg.GET("/apps/:appID/envs/:envName/resource-topology/nodes/:nodeID", h.GetTopologyNodeDetail)
	rg.GET("/apps/:appID/envs/:envName/resource-topology/nodes/:nodeID/events", h.ListTopologyNodeEvents)
	rg.GET("/apps/:appID/envs/:envName/resource-topology/nodes/:nodeID/manifest", h.GetTopologyNodeManifest)
}
