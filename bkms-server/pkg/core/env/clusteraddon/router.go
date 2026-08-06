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

package clusteraddon

import "github.com/gin-gonic/gin"

// ClusterAddonHandler contains views required by cluster-addon Gin routes.
type ClusterAddonHandler interface {
	ListClusterAddons(c *gin.Context)
	UpsertClusterAddon(c *gin.Context)
	DeleteClusterAddon(c *gin.Context)
}

// Register registers Gin cluster-addon routes.
func Register(rg *gin.RouterGroup, h ClusterAddonHandler) {
	// 查询可安装的集群插件列表
	rg.GET("/envs/:envID/cluster-addons", h.ListClusterAddons)
	// 部署/更新集群插件
	rg.POST("/envs/:envID/cluster-addons/:addonName", h.UpsertClusterAddon)
	// 卸载集群插件
	rg.DELETE("/envs/:envID/cluster-addons/:addonName", h.DeleteClusterAddon)
}
