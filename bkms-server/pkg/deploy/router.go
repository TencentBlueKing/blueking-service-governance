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

package deploy

import "github.com/gin-gonic/gin"

// HelmDeployHandler 定义 Helm 应用部署 Gin 路由需要的视图方法
type HelmDeployHandler interface {
	// Helm Deploy

	// ListHelmDeployRecords 获取 Helm 应用部署记录列表
	ListHelmDeployRecords(c *gin.Context)
	// PreviewHelmDeploy 预览 Helm 应用部署
	PreviewHelmDeploy(c *gin.Context)
	// CreateHelmDeploy 部署 Helm 应用
	CreateHelmDeploy(c *gin.Context)
	// PreviewRollbackHelmDeploy 预览 Helm 部署版本回滚
	PreviewRollbackHelmDeploy(c *gin.Context)
	// RollbackHelmDeploy 回滚到指定 Helm 部署版本
	RollbackHelmDeploy(c *gin.Context)
	// DeleteHelmDeploy 删除 Helm 应用部署
	DeleteHelmDeploy(c *gin.Context)

	// AppModel Deploy
	//
	// NOTE: 下面的 Trpc 与 TAF 两组方法是完全镜面复用的同一套逻辑（仅资源类型不同），
	// 如需修改其中一组 Handler 的行为，需要记得同步调整另一组，避免双方逻辑出现漂移。

	// ListTrpcDeployRecords 获取 Trpc 应用部署记录列表
	ListTrpcDeployRecords(c *gin.Context)
	// PreCheckTrpcDeployEnvVars Trpc 部署前环境变量校验
	PreCheckTrpcDeployEnvVars(c *gin.Context)
	// CreateTrpcDeploy 部署 Trpc 应用
	CreateTrpcDeploy(c *gin.Context)
	// DeleteTrpcDeploy 删除 Trpc 应用部署
	DeleteTrpcDeploy(c *gin.Context)
	// ListTrpcResourceSnapshots 获取 Trpc 应用资源快照列表
	ListTrpcResourceSnapshots(c *gin.Context)
	// GetTrpcResourceSnapshot 获取 Trpc 类型应用部署记录下某个资源的快照详情
	GetTrpcResourceSnapshot(c *gin.Context)
	// GetLatestTrpcDeployStatus 获取 Trpc 应用最新一次部署的状态
	GetLatestTrpcDeployStatus(c *gin.Context)
	// ListTafDeployRecords 获取 TAF 应用部署记录列表
	ListTafDeployRecords(c *gin.Context)
	// PreCheckTafDeployEnvVars TAF 部署前环境变量校验
	PreCheckTafDeployEnvVars(c *gin.Context)
	// CreateTafDeploy 部署 TAF 应用
	CreateTafDeploy(c *gin.Context)
	// DeleteTafDeploy 删除 TAF 应用部署
	DeleteTafDeploy(c *gin.Context)
	// ListTafResourceSnapshots 获取 TAF 应用资源快照列表
	ListTafResourceSnapshots(c *gin.Context)
	// GetTafResourceSnapshot 获取 TAF 类型应用部署记录下某个资源的快照详情
	GetTafResourceSnapshot(c *gin.Context)
	// GetLatestTafDeployStatus 获取 TAF 应用最新一次部署的状态
	GetLatestTafDeployStatus(c *gin.Context)
}

// Register 注册 Helm 应用部署 Gin v2 路由
func Register(rg *gin.RouterGroup, h HelmDeployHandler) {
	// 获取 Helm 应用部署记录列表
	// [bkms-cli 使用] 避免破坏性修改
	rg.GET("/apps/:appID/envs/:envName/helm-deploys", h.ListHelmDeployRecords)
	// 部署 Helm 应用预览
	rg.GET("/apps/:appID/envs/:envName/helm-deploys/preview", h.PreviewHelmDeploy)
	// 部署 Helm 应用
	// [bkms-cli 使用] 避免破坏性修改
	rg.POST("/apps/:appID/envs/:envName/helm-deploys", h.CreateHelmDeploy)
	// Helm 部署版本回滚预览
	rg.GET("/apps/:appID/envs/:envName/helm-deploys/:deployID/preview", h.PreviewRollbackHelmDeploy)
	// Helm 回滚到指定的部署版本
	rg.PUT("/apps/:appID/envs/:envName/helm-deploys/:deployID", h.RollbackHelmDeploy)
	// 删除 Helm 应用部署
	rg.DELETE("/apps/:appID/envs/:envName/helm-deploys/:deployID", h.DeleteHelmDeploy)

	// 获取 Trpc 应用部署记录列表
	// [bkms-cli 使用] 避免破坏性修改
	rg.GET("/apps/:appID/envs/:envName/trpc-deploys", h.ListTrpcDeployRecords)
	// Trpc 部署前环境变量校验
	rg.GET("/apps/:appID/envs/:envName/trpc-deploys/env-var-precheck", h.PreCheckTrpcDeployEnvVars)
	// 创建 Trpc 应用部署
	// [bkms-cli 使用] 避免破坏性修改
	rg.POST("/apps/:appID/envs/:envName/trpc-deploys", h.CreateTrpcDeploy)
	// 删除 Trpc 应用部署（下架当前环境最新版本）
	rg.DELETE("/apps/:appID/envs/:envName/trpc-deploys", h.DeleteTrpcDeploy)
	// 列出 Trpc 应用某次部署下发的资源清单快照（元数据）
	rg.GET("/apps/:appID/envs/:envName/trpc-deploys/:deployID/resource-snapshots", h.ListTrpcResourceSnapshots)
	// 获取 TAF 类型应用部署记录下某个资源的快照详情（具体 k8s 资源的 Manifest 等）
	rg.GET(
		"/apps/:appID/envs/:envName/trpc-deploys/:deployID/resource-snapshots/:snapshotID",
		h.GetTrpcResourceSnapshot,
	)
	// 获取 Trpc 应用最新一次部署的状态
	rg.GET("/apps/:appID/envs/:envName/trpc-deploys/latest-status", h.GetLatestTrpcDeployStatus)

	// 获取 TAF 应用部署记录列表
	// [bkms-cli 使用] 避免破坏性修改
	rg.GET("/apps/:appID/envs/:envName/taf-deploys", h.ListTafDeployRecords)
	// TAF 部署前环境变量校验
	rg.GET("/apps/:appID/envs/:envName/taf-deploys/env-var-precheck", h.PreCheckTafDeployEnvVars)
	// 创建 TAF 应用部署
	// [bkms-cli 使用] 避免破坏性修改
	rg.POST("/apps/:appID/envs/:envName/taf-deploys", h.CreateTafDeploy)
	// 删除 TAF 应用部署（下架当前环境最新版本）
	rg.DELETE("/apps/:appID/envs/:envName/taf-deploys", h.DeleteTafDeploy)
	// 列出 TAF 应用某次部署下发的资源清单快照（元数据）
	rg.GET("/apps/:appID/envs/:envName/taf-deploys/:deployID/resource-snapshots", h.ListTafResourceSnapshots)
	// 获取 TAF 类型应用部署记录下某个资源的快照详情（具体 k8s 资源的 Manifest 等）
	rg.GET("/apps/:appID/envs/:envName/taf-deploys/:deployID/resource-snapshots/:snapshotID", h.GetTafResourceSnapshot)
	// 获取 TAF 应用最新一次部署的状态
	rg.GET("/apps/:appID/envs/:envName/taf-deploys/latest-status", h.GetLatestTafDeployStatus)
}
