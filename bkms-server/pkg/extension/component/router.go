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

package component

import "github.com/gin-gonic/gin"

// ComponentHandler contains views required by component Gin routes.
type ComponentHandler interface {
	// 组件定义
	ListComponentDefs(c *gin.Context)
	CreateComponentDef(c *gin.Context)
	PatchComponentDef(c *gin.Context)
	DeleteComponentDef(c *gin.Context)

	// 列出组件内置变量列表
	ListBuiltinVars(c *gin.Context)

	// 预览组件定义（试运行），使用内置变量与属性默认值渲染输出
	PreviewComponentDef(c *gin.Context)
	PreviewComponentInst(c *gin.Context)
}

// Register registers Gin component routes.
func Register(rg *gin.RouterGroup, h ComponentHandler) {
	// 获取组件定义
	rg.GET("/component-defs", h.ListComponentDefs)
	// 创建组件定义
	rg.POST("/component-defs", h.CreateComponentDef)
	// 更新组件定义
	rg.PATCH("/component-defs/:compDefName", h.PatchComponentDef)
	// 删除组件定义
	rg.DELETE("/component-defs/:compDefName", h.DeleteComponentDef)

	// 获取组件输出模板系统变量
	rg.GET("/component-defs/builtin-vars", h.ListBuiltinVars)

	// 预览组件定义（试运行），使用内置变量与属性默认值渲染输出
	rg.POST("/component-defs/preview", h.PreviewComponentDef)
	// 预览组件实例（试运行），按 type 与默认版本拉取组件定义并预览
	rg.POST("/component-insts/preview", h.PreviewComponentInst)
}
