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

package platconfig

import "github.com/gin-gonic/gin"

// Handler contains views required by platform config Gin routes.
type Handler interface {
	GetFrontendFeatures(c *gin.Context)
}

// Register registers platform config routes.
//
// 前端在确定当前租户之前就需要读取特性开关，因此传入的路由组不能带租户校验中间件。
func Register(rg *gin.RouterGroup, h Handler) {
	rg.GET("/platform/frontend-features", h.GetFrontendFeatures)
}
