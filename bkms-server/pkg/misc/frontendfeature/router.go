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

// Package frontendfeature 向前端下发由部署配置决定的功能开关，前端据此决定页面入口是否展示。
package frontendfeature

import "github.com/gin-gonic/gin"

// Handler 包含前端功能开关路由所需的视图方法。
type Handler interface {
	// GetFrontendFeatures 获取前端功能开关
	GetFrontendFeatures(c *gin.Context)
}

// Register 注册前端功能开关路由。
func Register(rg *gin.RouterGroup, h Handler) {
	rg.GET("/frontend-features", h.GetFrontendFeatures)
}
