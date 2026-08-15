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

package bkintegrations

import (
	"github.com/gin-gonic/gin"
)

// Handler contains views required by external platform integration Gin routes.
type Handler interface {
	// --- BSCP 相关 API ---

	// ListBSCPBizs 获取用户的 BSCP 业务列表
	ListBSCPBizs(c *gin.Context)
	// ListBSCPServices 获取 BSCP 业务下的服务列表
	ListBSCPServices(c *gin.Context)
	// ListBSCPConfigs 获取 BSCP 服务下的配置列表
	ListBSCPConfigs(c *gin.Context)
	// GetBSCPConfig 获取 BSCP 配置项内容
	GetBSCPConfig(c *gin.Context)

	// --- BKCC（蓝鲸配置平台）相关 API ---

	// ListBKCCAuthorizedBusinesses 获取用户有权限的 bkcc 业务信息
	ListBKCCAuthorizedBusinesses(c *gin.Context)

	// --- BkCI（蓝盾）相关 API ---

	// ListBkCIOAuthGitProjects 获取 OAuth 授权给蓝盾的 Git 项目列表
	ListBkCIOAuthGitProjects(c *gin.Context)
	// GetBkCIOAuthUrl 获取 OAuth 授权的 Url
	GetBkCIOAuthUrl(c *gin.Context)
	// ListBkCIPipelines 获取工作空间对应的蓝盾项目下的流水线列表
	ListBkCIPipelines(c *gin.Context)
	// GetBkCIPipelineVariables 获取蓝盾流水线的变量列表
	GetBkCIPipelineVariables(c *gin.Context)
	// GetBkCIPipeline 获取蓝盾流水线详情
	GetBkCIPipeline(c *gin.Context)
	// ListBkCIRepositoryBranches 获取代码仓库分支列表
	ListBkCIRepositoryBranches(c *gin.Context)
	// ListBkCIRepositoryTags 获取代码仓库标签列表
	ListBkCIRepositoryTags(c *gin.Context)

	// --- BCS（蓝鲸容器服务）相关 API ---

	// ListBCSAuthorizedProjects 获取有权限的 BCS 项目列表
	ListBCSAuthorizedProjects(c *gin.Context)
	// GetBCSProject 根据项目 ID 获取项目详情
	GetBCSProject(c *gin.Context)
	// ListClustersByProject 获取项目下的集群列表
	ListClustersByProject(c *gin.Context)
	// ListNamespacesByCluster 获取集群下的命名空间列表
	ListNamespacesByCluster(c *gin.Context)

	// --- KubeInsight 相关 API ---

	// GetLatestEnvReport 获取最新环境巡检报告
	GetLatestEnvReport(c *gin.Context)

	// --- BkMonitor（蓝鲸监控）相关 API ---

	// GetApmServiceName 获取 Apm 服务名称
	GetApmServiceName(c *gin.Context)
	// ListApms 从蓝鲸监控 API 获取该工作空间的所有 Apm 详情
	ListApms(c *gin.Context)
	// CreateEnvApm 为环境创建 APM 并绑定
	CreateEnvApm(c *gin.Context)
	// BindApmToEnv 将环境绑定到指定 APM 上
	BindApmToEnv(c *gin.Context)
	// GetEnvApm 查询环境绑定的 APM
	GetEnvApm(c *gin.Context)
	// GetInstanceTimeSeries 查询实例监控指标时序数据
	GetInstanceTimeSeries(c *gin.Context)

	// --- BkHCM（蓝鲸海垫）相关 API ---

	// ListBkHCMRegions 查询云地域列表
	ListBkHCMRegions(c *gin.Context)
	// ListBkHCMSubnets 查询子网列表
	ListBkHCMSubnets(c *gin.Context)
	// ListBkHCMVPCs 查询 VPC 列表
	ListBkHCMVPCs(c *gin.Context)
	// ListBkHCMZones 查询可用区列表
	ListBkHCMZones(c *gin.Context)
	// CreateBkHCMLoadBalancerApplication 创建负载均衡申请
	CreateBkHCMLoadBalancerApplication(c *gin.Context)
}

// Register registers Gin external platform integration routes.
func Register(rg *gin.RouterGroup, h Handler) {
	// --- BSCP 相关路由 ---
	rg.GET("/bscp/bizs", h.ListBSCPBizs)
	rg.GET("/bscp/bizs/:bizID/services", h.ListBSCPServices)
	rg.GET("/bscp/bizs/:bizID/services/:serviceID/configs", h.ListBSCPConfigs)
	rg.GET("/bscp/bizs/:bizID/services/:serviceID/configs/:configID", h.GetBSCPConfig)

	// --- BKCC 相关路由 ---
	rg.GET("/bkcc/businesses/authorized", h.ListBKCCAuthorizedBusinesses)

	// --- BkCI 相关路由 ---
	rg.GET("/workspaces/:workspaceID/bkci-git-projects", h.ListBkCIOAuthGitProjects)
	rg.GET("/workspaces/:workspaceID/bkci-oauth-url", h.GetBkCIOAuthUrl)
	rg.GET("/workspaces/:workspaceID/bkci-pipelines", h.ListBkCIPipelines)
	rg.GET("/workspaces/:workspaceID/bkci-pipelines/:pipelineID/variables", h.GetBkCIPipelineVariables)
	rg.GET("/workspaces/:workspaceID/bkci-pipelines/:pipelineID", h.GetBkCIPipeline)
	rg.GET("/workspaces/:workspaceID/bkci-repositories/branches", h.ListBkCIRepositoryBranches)
	rg.GET("/workspaces/:workspaceID/bkci-repositories/tags", h.ListBkCIRepositoryTags)

	// --- BCS 相关路由 ---
	rg.GET("/bcs/projects/authorized", h.ListBCSAuthorizedProjects)
	rg.GET("/bcs/projects/:projectID", h.GetBCSProject)
	rg.GET("/bcs/projects/:projectID/clusters", h.ListClustersByProject)
	rg.GET("/bcs/projects/:projectID/clusters/:clusterID/namespaces", h.ListNamespacesByCluster)

	// --- KubeInsight 相关路由 ---
	rg.GET("/kube-insight/reports", h.GetLatestEnvReport)

	// --- BkMonitor 相关路由 ---
	rg.GET("/apps/:appID/envs/:envName/bkmonitor/apm-service-name", h.GetApmServiceName)
	rg.GET("/workspaces/:workspaceID/bkmonitor/apms", h.ListApms)
	rg.POST("/envs/:envID/bkmonitor/apms", h.CreateEnvApm)
	rg.PUT("/envs/:envID/bkmonitor/apms/:apmID", h.BindApmToEnv)
	rg.GET("/envs/:envID/bkmonitor/apms", h.GetEnvApm)
	rg.GET("/apps/:appID/envs/:envName/bkmonitor/instance-time-series", h.GetInstanceTimeSeries)

	// --- BkHCM 相关路由 ---
	rg.POST("/bkhcm/regions", h.ListBkHCMRegions)
	rg.POST("/bkhcm/bizs/:bkBizID/subnets", h.ListBkHCMSubnets)
	rg.POST("/bkhcm/bizs/:bkBizID/vpcs", h.ListBkHCMVPCs)
	rg.POST("/bkhcm/regions/:region/zones", h.ListBkHCMZones)
	rg.POST("/bkhcm/load-balancers", h.CreateBkHCMLoadBalancerApplication)
}
