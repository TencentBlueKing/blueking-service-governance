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

package rainbowtconf

import "github.com/gin-gonic/gin"

// Handler contains views required by rainbow tconf Gin routes.
type Handler interface {
	GetRainbowTConf(c *gin.Context)
	PutRainbowTConf(c *gin.Context)
}

// Register registers Gin rainbow tconf routes.
func Register(rg *gin.RouterGroup, h Handler) {
	// 获取应用的七彩石入口
	rg.GET("/apps/:appID/rainbow-tconf", h.GetRainbowTConf)
	// 设置应用的七彩石入口（首次配置或修改）
	rg.PUT("/apps/:appID/rainbow-tconf", h.PutRainbowTConf)
}
