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

package accesspoint

import "github.com/gin-gonic/gin"

// Handler 包含北极星接入点路由所需的视图方法
type Handler interface {
	ListAccessPoints(c *gin.Context)
	CreateAccessPoint(c *gin.Context)
	UpdateAccessPoint(c *gin.Context)
	DeleteAccessPoint(c *gin.Context)
	TestConnectivity(c *gin.Context)
	ListNamespaces(c *gin.Context)

	ListClusters(c *gin.Context)
	ListOperatorValues(c *gin.Context)
}

// Register 注册北极星接入点路由。
//
// 调用方只在允许空间自建接入点时注册：不允许时只有平台托管接入点，所有集群自动使用它，
// 前端沿用现有界面，不需要接入点管理、集群绑定与组件相关的接口
func Register(rg *gin.RouterGroup, h Handler) {
	// 接入点列表
	rg.GET("/workspaces/:workspaceID/polaris-access-points", h.ListAccessPoints)
	// 创建接入点
	rg.POST("/workspaces/:workspaceID/polaris-access-points", h.CreateAccessPoint)
	// 更新接入点
	rg.PATCH("/workspaces/:workspaceID/polaris-access-points/:accessPointID", h.UpdateAccessPoint)
	// 删除接入点
	rg.DELETE("/workspaces/:workspaceID/polaris-access-points/:accessPointID", h.DeleteAccessPoint)
	// 连通测试
	rg.POST("/workspaces/:workspaceID/polaris-access-points/connectivity-test", h.TestConnectivity)
	// 接入点上可选的北极星命名空间
	rg.GET("/workspaces/:workspaceID/polaris-access-points/:accessPointID/namespaces", h.ListNamespaces)
	// 集群绑定视图
	rg.GET("/workspaces/:workspaceID/polaris-clusters", h.ListClusters)
	// 已绑定集群上 bcs-polaris-operator 的期望 values；安装与状态查询使用集群插件接口
	rg.GET("/workspaces/:workspaceID/polaris-operator-values", h.ListOperatorValues)
}
