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

package build

import "github.com/gin-gonic/gin"

// Handler 定义 Helm Chart Gin 路由需要的视图方法。
type Handler interface {
	// CreateHelmChartBuild 触发 Helm Chart 构建（从 Git 源码构建，落库 + 异步轮询）。
	CreateHelmChartBuild(c *gin.Context)
	// GetHelmChartSemver 查询 Helm Chart semver counter 当前值。
	GetHelmChartSemver(c *gin.Context)
	// ListAppHelmCharts 获取 Helm Chart 制品列表。
	ListAppHelmCharts(c *gin.Context)
	// ListHelmChartBuildRecords 获取 Helm Chart 构建记录列表。
	ListHelmChartBuildRecords(c *gin.Context)
	// GetHelmChartFiles 获取指定 Helm Chart 版本的全部文件（递归文件树 + 文本文件内容）。
	GetHelmChartFiles(c *gin.Context)
	// ListChartVersions 获取 Helm Chart 版本列表。
	ListChartVersions(c *gin.Context)
	// GetValuesFile 获取 Helm Chart Values 文件。
	GetValuesFile(c *gin.Context)
	// StreamHelmChartBuildLogs 流式推送 Helm Chart 构建日志（SSE）。
	StreamHelmChartBuildLogs(c *gin.Context)
	// DownloadHelmChartBuildLogs 下载 Helm Chart 构建日志。
	DownloadHelmChartBuildLogs(c *gin.Context)
}

// Register 注册 Helm Chart Gin 路由。
func Register(rg *gin.RouterGroup, h Handler) {
	// 触发 Helm Chart 构建
	rg.POST("/apps/:appID/charts/builds", h.CreateHelmChartBuild)
	// 查询 Helm Chart semver counter 当前值
	rg.GET("/apps/:appID/charts/semver", h.GetHelmChartSemver)
	// 获取 Helm Chart 制品列表
	rg.GET("/apps/:appID/charts", h.ListAppHelmCharts)
	// 获取 Helm Chart 构建记录列表
	rg.GET("/apps/:appID/charts/builds", h.ListHelmChartBuildRecords)
	// 获取指定 Helm Chart 版本的全部文件
	rg.GET("/apps/:appID/charts/:chartVersion/files", h.GetHelmChartFiles)
	// 获取 Helm Chart 版本列表
	rg.GET("/apps/:appID/charts/versions", h.ListChartVersions)
	// 获取 Helm Chart Values 文件
	rg.GET("/apps/:appID/charts/:chartVersion/valuesfile", h.GetValuesFile)
	// Helm Chart 构建日志
	rg.GET("/apps/:appID/charts/builds/:buildID/logs/stream", h.StreamHelmChartBuildLogs)
	rg.GET("/apps/:appID/charts/builds/:buildID/logs/download", h.DownloadHelmChartBuildLogs)
}
