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

// Package bscpcfg 提供应用配置管理的对外入口。
package bscpcfg

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// Handler 包含应用配置管理路由所需的视图方法。
type Handler interface {
	// InitMetadata 初始化配置管理
	InitMetadata(c *gin.Context)
	// GetMetadata 获取配置管理元信息
	GetMetadata(c *gin.Context)
	// PatchMetadata 修改配置管理元信息
	PatchMetadata(c *gin.Context)
	// DeleteMetadata 删除配置管理元信息（级联删除所有环境绑定）
	DeleteMetadata(c *gin.Context)
	// ListEnvBindings 获取所有环境的绑定列表
	ListEnvBindings(c *gin.Context)

	// CreateEnvBinding 创建环境绑定
	CreateEnvBinding(c *gin.Context)
	// DeleteEnvBinding 删除环境绑定
	DeleteEnvBinding(c *gin.Context)
	// GetEnvBinding 获取指定环境的绑定详情
	GetEnvBinding(c *gin.Context)
}

// Register 注册应用配置管理的 Gin v2 路由。
func Register(rg *gin.RouterGroup, h Handler) {
	g := rg.Group("/apps/:appID/bscpcfg", ginutils.RequireBSCP())

	// Metadata 操作
	g.POST("/metadata", h.InitMetadata)
	g.PATCH("/metadata", h.PatchMetadata)
	g.GET("/metadata", h.GetMetadata)
	g.DELETE("/metadata", h.DeleteMetadata)

	// EnvBinding 操作
	g.GET("/envs", h.ListEnvBindings)
	g.POST("/envs/:envName/binding", h.CreateEnvBinding)
	g.DELETE("/envs/:envName/binding", h.DeleteEnvBinding)
	g.GET("/envs/:envName/binding", h.GetEnvBinding)
}
