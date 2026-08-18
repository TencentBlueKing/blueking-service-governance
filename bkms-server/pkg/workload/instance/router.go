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

package instance

import "github.com/gin-gonic/gin"

// Handler contains views required by app instance Gin routes.
type Handler interface {
	ListAppInstances(c *gin.Context)
	WatchAppInstances(c *gin.Context)
	UpdateAppInstances(c *gin.Context)
	ScaleAppInstances(c *gin.Context)
	BatchDeleteAppInstances(c *gin.Context)
	UpdateAppInstancePolaris(c *gin.Context)
	CreateAppInstanceWebConsole(c *gin.Context)
	ListAppInstanceLogs(c *gin.Context)
	ListEvents(c *gin.Context)
	ListTrpcAdminCmds(c *gin.Context)
	ExecuteTrpcAdminCmd(c *gin.Context)
	ExecuteTafAdminCmd(c *gin.Context)
	PortForward(c *gin.Context)
}

// Register 注册应用实例相关 Gin 路由。
func Register(rg *gin.RouterGroup, h Handler) {
	// 获取应用实例列表
	rg.GET("/apps/:appID/envs/:envName/instances", h.ListAppInstances)
	// 订阅应用实例投影变更；与 :instanceID 同层，Gin 静态段优先匹配，不依赖注册顺序
	rg.GET("/apps/:appID/envs/:envName/instances/watch", h.WatchAppInstances)
	// 更新应用实例（支持单/多/全量实例更新）
	rg.PUT("/apps/:appID/envs/:envName/instances", h.UpdateAppInstances)
	// 扩缩容应用实例数量
	rg.PUT("/apps/:appID/envs/:envName/instances/operations/scale", h.ScaleAppInstances)
	// 批量删除指定的应用实例，同时缩容副本数量
	rg.POST("/apps/:appID/envs/:envName/instances/operations/batch_delete", h.BatchDeleteAppInstances)
	// 更新应用实例的北极星注解（权重 / 隔离）
	rg.PUT("/apps/:appID/envs/:envName/instances/operations/polaris", h.UpdateAppInstancePolaris)
	// 创建应用运行实例（Pod）WebConsole
	rg.POST("/apps/:appID/envs/:envName/instances/:instanceID/web-console", h.CreateAppInstanceWebConsole)
	// 获取应用运行实例（Pod）日志
	rg.GET("/apps/:appID/envs/:envName/instances/:instanceID/logs", h.ListAppInstanceLogs)
	// 获取指定环境的事件列表
	rg.GET("/apps/:appID/envs/:envName/events", h.ListEvents)
	// 查询 Trpc 管理命令
	rg.GET("/apps/:appID/envs/:envName/instances/admin-cmds", h.ListTrpcAdminCmds)
	// 执行 Trpc 管理命令
	rg.POST("/apps/:appID/envs/:envName/instances/admin-cmds", h.ExecuteTrpcAdminCmd)
	// 执行 TAF 管理命令
	rg.POST("/apps/:appID/envs/:envName/instances/taf-admin-cmds", h.ExecuteTafAdminCmd)
	// 应用实例端口转发
	rg.GET("/apps/:appID/envs/:envName/instances/:instanceID/port-forward/connect", h.PortForward)
}
