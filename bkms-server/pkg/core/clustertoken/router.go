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

package clustertoken

import "github.com/gin-gonic/gin"

// Handler 包含集群 Token 与集群接入路由所需的视图方法
type Handler interface {
	ListTokens(c *gin.Context)
	CreateToken(c *gin.Context)
	ReplaceToken(c *gin.Context)
	SetDefaultToken(c *gin.Context)
	DeleteToken(c *gin.Context)

	ListCandidates(c *gin.Context)
	ListClusters(c *gin.Context)
	AccessClusters(c *gin.Context)
	SwitchClusterToken(c *gin.Context)
	RemoveCluster(c *gin.Context)
}

// Register 注册集群 Token 与集群接入路由。
//
// 调用方只在 bcs.allowWorkspaceClusterTokens 打开时注册：关闭时所有集群使用平台 Token，不需要这些接口
func Register(rg *gin.RouterGroup, h Handler) {
	// Token 列表
	rg.GET("/workspaces/:workspaceID/cluster-tokens", h.ListTokens)
	// 新建 Token
	rg.POST("/workspaces/:workspaceID/cluster-tokens", h.CreateToken)
	// 更换 Token 值
	rg.PUT("/workspaces/:workspaceID/cluster-tokens/:tokenID", h.ReplaceToken)
	// 设为默认 Token
	rg.PUT("/workspaces/:workspaceID/cluster-tokens/:tokenID/default", h.SetDefaultToken)
	// 删除 Token
	rg.DELETE("/workspaces/:workspaceID/cluster-tokens/:tokenID", h.DeleteToken)

	// 可接入的集群
	rg.GET("/workspaces/:workspaceID/cluster-candidates", h.ListCandidates)
	// 已接入的集群
	rg.GET("/workspaces/:workspaceID/clusters", h.ListClusters)
	// 接入集群
	rg.POST("/workspaces/:workspaceID/clusters", h.AccessClusters)
	// 更换集群使用的 Token
	rg.PUT("/workspaces/:workspaceID/clusters/:clusterID/token", h.SwitchClusterToken)
	// 移除集群
	rg.DELETE("/workspaces/:workspaceID/clusters/:clusterID", h.RemoveCluster)
}
