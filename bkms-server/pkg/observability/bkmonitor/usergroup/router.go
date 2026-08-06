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

package usergroup

import "github.com/gin-gonic/gin"

// Handler contains views required by user group Gin routes.
type Handler interface {
	ListUserGroups(c *gin.Context)
	GetUserGroup(c *gin.Context)
	CreateUserGroup(c *gin.Context)
	UpdateUserGroup(c *gin.Context)
	DeleteUserGroup(c *gin.Context)
}

// Register 注册用户分组相关路由。
func Register(rg *gin.RouterGroup, h Handler) {
	bk := rg.Group("/workspaces/:workspaceID/bkmonitor")
	{
		// 查询用户分组列表
		bk.GET("/user-groups", h.ListUserGroups)
		// 查询单个用户分组详情
		bk.GET("/user-groups/:groupID", h.GetUserGroup)
		// 创建用户分组
		bk.POST("/user-groups", h.CreateUserGroup)
		// 更新用户分组
		bk.PUT("/user-groups/:groupID", h.UpdateUserGroup)
		// 删除用户分组
		bk.DELETE("/user-groups/:groupID", h.DeleteUserGroup)
	}
}
